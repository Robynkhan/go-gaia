// Copyright 2014 The go-fairblock Authors
// This file is part of the go-fairblock library.
//
// The go-fairblock library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-fairblock library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-fairblock library. If not, see <http://www.gnu.org/licenses/>.

// Package fbc implements the Fairblock protocol.
package fbc

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/fairblock/go-fairblock/accounts"
	"github.com/fairblock/go-fairblock/common"
	"github.com/fairblock/go-fairblock/common/hexutil"
	"github.com/fairblock/go-fairblock/consensus"
	"github.com/fairblock/go-fairblock/consensus/clique"
	"github.com/fairblock/go-fairblock/consensus/fbcash"
	"github.com/fairblock/go-fairblock/core"
	"github.com/fairblock/go-fairblock/core/bloombits"
	"github.com/fairblock/go-fairblock/core/types"
	"github.com/fairblock/go-fairblock/core/vm"
	"github.com/fairblock/go-fairblock/fbc/downloader"
	"github.com/fairblock/go-fairblock/fbc/filters"
	"github.com/fairblock/go-fairblock/fbc/gasprice"
	"github.com/fairblock/go-fairblock/fbcdb"
	"github.com/fairblock/go-fairblock/event"
	"github.com/fairblock/go-fairblock/internal/fbcapi"
	"github.com/fairblock/go-fairblock/log"
	"github.com/fairblock/go-fairblock/miner"
	"github.com/fairblock/go-fairblock/node"
	"github.com/fairblock/go-fairblock/p2p"
	"github.com/fairblock/go-fairblock/params"
	"github.com/fairblock/go-fairblock/rlp"
	"github.com/fairblock/go-fairblock/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

// Fairblock implements the Fairblock full node service.
type Fairblock struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan  chan bool    // Channel for shutting down the fairblock
	stopDbUpgrade func() error // stop chain db sequential key upgrade

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb fbcdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	ApiBackend *FbcApiBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	fbcerbase common.Address

	networkId     uint64
	netRPCService *fbcapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and fbcerbase)
}

func (s *Fairblock) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// New creates a new Fairblock object (including the
// initialisation of the common Fairblock object)
func New(ctx *node.ServiceContext, config *Config) (*Fairblock, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run fbc.Fairblock in light sync mode, use les.LightFairblock")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	stopDbUpgrade := upgradeDeduplicateData(chainDb)
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	fbc := &Fairblock{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, config, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		stopDbUpgrade:  stopDbUpgrade,
		networkId:      config.NetworkId,
		gasPrice:       config.GasPrice,
		fbcerbase:      config.Fairblockbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks),
	}

	log.Info("Initialising Fairblock protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := core.GetBlockChainVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run gfbc upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	}

	vmConfig := vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
	fbc.blockchain, err = core.NewBlockChain(chainDb, fbc.chainConfig, fbc.engine, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		fbc.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	fbc.bloomIndexer.Start(fbc.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	fbc.txPool = core.NewTxPool(config.TxPool, fbc.chainConfig, fbc.blockchain)

	if fbc.protocolManager, err = NewProtocolManager(fbc.chainConfig, config.SyncMode, config.NetworkId, fbc.eventMux, fbc.txPool, fbc.engine, fbc.blockchain, chainDb); err != nil {
		return nil, err
	}
	fbc.miner = miner.New(fbc, fbc.chainConfig, fbc.EventMux(), fbc.engine)
	fbc.miner.SetExtra(makeExtraData(config.ExtraData))

	fbc.ApiBackend = &FbcApiBackend{fbc, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	fbc.ApiBackend.gpo = gasprice.NewOracle(fbc.ApiBackend, gpoParams)

	return fbc, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"gfbc",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (fbcdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*fbcdb.LDBDatabase); ok {
		db.Meter("fbc/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an Fairblock service
func CreateConsensusEngine(ctx *node.ServiceContext, config *Config, chainConfig *params.ChainConfig, db fbcdb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	}
	// Otherwise assume proof-of-work
	switch {
	case config.PowFake:
		log.Warn("Fbcash used in fake mode")
		return fbcash.NewFaker()
	case config.PowTest:
		log.Warn("Fbcash used in test mode")
		return fbcash.NewTester()
	case config.PowShared:
		log.Warn("Fbcash used in shared mode")
		return fbcash.NewShared()
	default:
		engine := fbcash.New(ctx.ResolvePath(config.FbcashCacheDir), config.FbcashCachesInMem, config.FbcashCachesOnDisk,
			config.FbcashDatasetDir, config.FbcashDatasetsInMem, config.FbcashDatasetsOnDisk)
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs returns the collection of RPC services the fairblock package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Fairblock) APIs() []rpc.API {
	apis := fbcapi.GetAPIs(s.ApiBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "fbc",
			Version:   "1.0",
			Service:   NewPublicFairblockAPI(s),
			Public:    true,
		}, {
			Namespace: "fbc",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "fbc",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "fbc",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *Fairblock) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Fairblock) Fairblockbase() (eb common.Address, err error) {
	s.lock.RLock()
	fbcerbase := s.fbcerbase
	s.lock.RUnlock()

	if fbcerbase != (common.Address{}) {
		return fbcerbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			return accounts[0].Address, nil
		}
	}
	return common.Address{}, fmt.Errorf("fbcerbase address must be explicitly specified")
}

// set in js console via admin interface or wrapper from cli flags
func (self *Fairblock) SetFairblockbase(fbcerbase common.Address) {
	self.lock.Lock()
	self.fbcerbase = fbcerbase
	self.lock.Unlock()

	self.miner.SetFairblockbase(fbcerbase)
}

func (s *Fairblock) StartMining(local bool) error {
	eb, err := s.Fairblockbase()
	if err != nil {
		log.Error("Cannot start mining without fbcerbase", "err", err)
		return fmt.Errorf("fbcerbase missing: %v", err)
	}
	if clique, ok := s.engine.(*clique.Clique); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("Fairblockbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		clique.Authorize(eb, wallet.SignHash)
	}
	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so noone will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	go s.miner.Start(eb)
	return nil
}

func (s *Fairblock) StopMining()         { s.miner.Stop() }
func (s *Fairblock) IsMining() bool      { return s.miner.Mining() }
func (s *Fairblock) Miner() *miner.Miner { return s.miner }

func (s *Fairblock) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *Fairblock) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *Fairblock) TxPool() *core.TxPool               { return s.txPool }
func (s *Fairblock) EventMux() *event.TypeMux           { return s.eventMux }
func (s *Fairblock) Engine() consensus.Engine           { return s.engine }
func (s *Fairblock) ChainDb() fbcdb.Database            { return s.chainDb }
func (s *Fairblock) IsListening() bool                  { return true } // Always listening
func (s *Fairblock) FbcVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Fairblock) NetVersion() uint64                 { return s.networkId }
func (s *Fairblock) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *Fairblock) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// Fairblock protocol implementation.
func (s *Fairblock) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = fbcapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		maxPeers -= s.config.LightPeers
		if maxPeers < srvr.MaxPeers/2 {
			maxPeers = srvr.MaxPeers / 2
		}
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Fairblock protocol.
func (s *Fairblock) Stop() error {
	if s.stopDbUpgrade != nil {
		s.stopDbUpgrade()
	}
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
