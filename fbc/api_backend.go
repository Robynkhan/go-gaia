// Copyright 2015 The go-fairblock Authors
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

package fbc

import (
	"context"
	"math/big"

	"github.com/fairblock/go-fairblock/accounts"
	"github.com/fairblock/go-fairblock/common"
	"github.com/fairblock/go-fairblock/common/math"
	"github.com/fairblock/go-fairblock/core"
	"github.com/fairblock/go-fairblock/core/bloombits"
	"github.com/fairblock/go-fairblock/core/state"
	"github.com/fairblock/go-fairblock/core/types"
	"github.com/fairblock/go-fairblock/core/vm"
	"github.com/fairblock/go-fairblock/fbc/downloader"
	"github.com/fairblock/go-fairblock/fbc/gasprice"
	"github.com/fairblock/go-fairblock/fbcdb"
	"github.com/fairblock/go-fairblock/event"
	"github.com/fairblock/go-fairblock/params"
	"github.com/fairblock/go-fairblock/rpc"
)

// FbcApiBackend implements fbcapi.Backend for full nodes
type FbcApiBackend struct {
	fbc *Fairblock
	gpo *gasprice.Oracle
}

func (b *FbcApiBackend) ChainConfig() *params.ChainConfig {
	return b.fbc.chainConfig
}

func (b *FbcApiBackend) CurrentBlock() *types.Block {
	return b.fbc.blockchain.CurrentBlock()
}

func (b *FbcApiBackend) SetHead(number uint64) {
	b.fbc.protocolManager.downloader.Cancel()
	b.fbc.blockchain.SetHead(number)
}

func (b *FbcApiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.fbc.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.fbc.blockchain.CurrentBlock().Header(), nil
	}
	return b.fbc.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *FbcApiBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.fbc.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.fbc.blockchain.CurrentBlock(), nil
	}
	return b.fbc.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *FbcApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.fbc.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.fbc.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *FbcApiBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.fbc.blockchain.GetBlockByHash(blockHash), nil
}

func (b *FbcApiBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	return core.GetBlockReceipts(b.fbc.chainDb, blockHash, core.GetBlockNumber(b.fbc.chainDb, blockHash)), nil
}

func (b *FbcApiBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.fbc.blockchain.GetTdByHash(blockHash)
}

func (b *FbcApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.fbc.BlockChain(), nil)
	return vm.NewEVM(context, state, b.fbc.chainConfig, vmCfg), vmError, nil
}

func (b *FbcApiBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.fbc.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *FbcApiBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.fbc.BlockChain().SubscribeChainEvent(ch)
}

func (b *FbcApiBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.fbc.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *FbcApiBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.fbc.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *FbcApiBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.fbc.BlockChain().SubscribeLogsEvent(ch)
}

func (b *FbcApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.fbc.txPool.AddLocal(signedTx)
}

func (b *FbcApiBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.fbc.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *FbcApiBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.fbc.txPool.Get(hash)
}

func (b *FbcApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.fbc.txPool.State().GetNonce(addr), nil
}

func (b *FbcApiBackend) Stats() (pending int, queued int) {
	return b.fbc.txPool.Stats()
}

func (b *FbcApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.fbc.TxPool().Content()
}

func (b *FbcApiBackend) SubscribeTxPreEvent(ch chan<- core.TxPreEvent) event.Subscription {
	return b.fbc.TxPool().SubscribeTxPreEvent(ch)
}

func (b *FbcApiBackend) Downloader() *downloader.Downloader {
	return b.fbc.Downloader()
}

func (b *FbcApiBackend) ProtocolVersion() int {
	return b.fbc.FbcVersion()
}

func (b *FbcApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *FbcApiBackend) ChainDb() fbcdb.Database {
	return b.fbc.ChainDb()
}

func (b *FbcApiBackend) EventMux() *event.TypeMux {
	return b.fbc.EventMux()
}

func (b *FbcApiBackend) AccountManager() *accounts.Manager {
	return b.fbc.AccountManager()
}

func (b *FbcApiBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.fbc.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *FbcApiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.fbc.bloomRequests)
	}
}
