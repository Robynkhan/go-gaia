// Copyright 2016 The go-fairblock Authors
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

// Package les implements the Light Fairblock Subprotocol.
package les

import (
	"fmt"
	"sync"
	"time"

	"github.com/fairblock/go-fairblock/accounts"
	"github.com/fairblock/go-fairblock/common"
	"github.com/fairblock/go-fairblock/common/hexutil"
	"github.com/fairblock/go-fairblock/consensus"
	"github.com/fairblock/go-fairblock/core"
	"github.com/fairblock/go-fairblock/core/bloombits"
	"github.com/fairblock/go-fairblock/core/types"
	"github.com/fairblock/go-fairblock/fbc"
	"github.com/fairblock/go-fairblock/fbc/downloader"
	"github.com/fairblock/go-fairblock/fbc/filters"
	"github.com/fairblock/go-fairblock/fbc/gasprice"
	"github.com/fairblock/go-fairblock/fbcdb"
	"github.com/fairblock/go-fairblock/event"
	"github.com/fairblock/go-fairblock/internal/fbcapi"
	"github.com/fairblock/go-fairblock/light"
	"github.com/fairblock/go-fairblock/log"
	"github.com/fairblock/go-fairblock/node"
	"github.com/fairblock/go-fairblock/p2p"
	"github.com/fairblock/go-fairblock/p2p/discv5"
	"github.com/fairblock/go-fairblock/params"
	rpc "github.com/fairblock/go-fairblock/rpc"
)

type LightFairblock struct {
	odr         *LesOdr
	relay       *LesTxRelay
	chainConfig *params.ChainConfig
	// Channel for shutting down the service
	shutdownChan chan bool
	// Handlers
	peers           *peerSet
	txPool          *light.TxPool
	blockchain      *light.LightChain
	protocolManager *ProtocolManager
	serverPool      *serverPool
	reqDist         *requestDistributor
	retriever       *retrieveManager
	// DB interfaces
	chainDb fbcdb.Database // Block chain database

	bloomRequests                              chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer, chtIndexer, bloomTrieIndexer *core.ChainIndexer

	ApiBackend *LesApiBackend

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	networkId     uint64
	netRPCService *fbcapi.PublicNetAPI

	wg sync.WaitGroup
}

func New(ctx *node.ServiceContext, config *fbc.Config) (*LightFairblock, error) {
	chainDb, err := fbc.CreateDB(ctx, config, "lightchaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newPeerSet()
	quitSync := make(chan struct{})

	lfbc := &LightFairblock{
		chainConfig:      chainConfig,
		chainDb:          chainDb,
		eventMux:         ctx.EventMux,
		peers:            peers,
		reqDist:          newRequestDistributor(peers, quitSync),
		accountManager:   ctx.AccountManager,
		engine:           fbc.CreateConsensusEngine(ctx, config, chainConfig, chainDb),
		shutdownChan:     make(chan bool),
		networkId:        config.NetworkId,
		bloomRequests:    make(chan chan *bloombits.Retrieval),
		bloomIndexer:     fbc.NewBloomIndexer(chainDb, light.BloomTrieFrequency),
		chtIndexer:       light.NewChtIndexer(chainDb, true),
		bloomTrieIndexer: light.NewBloomTrieIndexer(chainDb, true),
	}

	lfbc.relay = NewLesTxRelay(peers, lfbc.reqDist)
	lfbc.serverPool = newServerPool(chainDb, quitSync, &lfbc.wg)
	lfbc.retriever = newRetrieveManager(peers, lfbc.reqDist, lfbc.serverPool)
	lfbc.odr = NewLesOdr(chainDb, lfbc.chtIndexer, lfbc.bloomTrieIndexer, lfbc.bloomIndexer, lfbc.retriever)
	if lfbc.blockchain, err = light.NewLightChain(lfbc.odr, lfbc.chainConfig, lfbc.engine); err != nil {
		return nil, err
	}
	lfbc.bloomIndexer.Start(lfbc.blockchain)
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		lfbc.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	lfbc.txPool = light.NewTxPool(lfbc.chainConfig, lfbc.blockchain, lfbc.relay)
	if lfbc.protocolManager, err = NewProtocolManager(lfbc.chainConfig, true, ClientProtocolVersions, config.NetworkId, lfbc.eventMux, lfbc.engine, lfbc.peers, lfbc.blockchain, nil, chainDb, lfbc.odr, lfbc.relay, quitSync, &lfbc.wg); err != nil {
		return nil, err
	}
	lfbc.ApiBackend = &LesApiBackend{lfbc, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	lfbc.ApiBackend.gpo = gasprice.NewOracle(lfbc.ApiBackend, gpoParams)
	return lfbc, nil
}

func lesTopic(genesisHash common.Hash, protocolVersion uint) discv5.Topic {
	var name string
	switch protocolVersion {
	case lpv1:
		name = "LES"
	case lpv2:
		name = "LES2"
	default:
		panic(nil)
	}
	return discv5.Topic(name + "@" + common.Bytes2Hex(genesisHash.Bytes()[0:8]))
}

type LightDummyAPI struct{}

// Fairblockbase is the address that mining rewards will be send to
func (s *LightDummyAPI) Fairblockbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Coinbase is the address that mining rewards will be send to (alias for Fairblockbase)
func (s *LightDummyAPI) Coinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Hashrate returns the POW hashrate
func (s *LightDummyAPI) Hashrate() hexutil.Uint {
	return 0
}

// Mining returns an indication if this node is currently mining.
func (s *LightDummyAPI) Mining() bool {
	return false
}

// APIs returns the collection of RPC services the fairblock package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *LightFairblock) APIs() []rpc.API {
	return append(fbcapi.GetAPIs(s.ApiBackend), []rpc.API{
		{
			Namespace: "fbc",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "fbc",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "fbc",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, true),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *LightFairblock) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *LightFairblock) BlockChain() *light.LightChain      { return s.blockchain }
func (s *LightFairblock) TxPool() *light.TxPool              { return s.txPool }
func (s *LightFairblock) Engine() consensus.Engine           { return s.engine }
func (s *LightFairblock) LesVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *LightFairblock) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *LightFairblock) EventMux() *event.TypeMux           { return s.eventMux }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *LightFairblock) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

// Start implements node.Service, starting all internal goroutines needed by the
// Fairblock protocol implementation.
func (s *LightFairblock) Start(srvr *p2p.Server) error {
	s.startBloomHandlers()
	log.Warn("Light client mode is an experimental feature")
	s.netRPCService = fbcapi.NewPublicNetAPI(srvr, s.networkId)
	// search the topic belonging to the oldest supported protocol because
	// servers always advertise all supported protocols
	protocolVersion := ClientProtocolVersions[len(ClientProtocolVersions)-1]
	s.serverPool.start(srvr, lesTopic(s.blockchain.Genesis().Hash(), protocolVersion))
	s.protocolManager.Start()
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Fairblock protocol.
func (s *LightFairblock) Stop() error {
	s.odr.Stop()
	if s.bloomIndexer != nil {
		s.bloomIndexer.Close()
	}
	if s.chtIndexer != nil {
		s.chtIndexer.Close()
	}
	if s.bloomTrieIndexer != nil {
		s.bloomTrieIndexer.Close()
	}
	s.blockchain.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()

	s.eventMux.Stop()

	time.Sleep(time.Millisecond * 200)
	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
