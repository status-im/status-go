// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package miner

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/teslapatrick/go-ethereum/accounts"
	"github.com/teslapatrick/go-ethereum/common"
	"github.com/teslapatrick/go-ethereum/core"
	"github.com/teslapatrick/go-ethereum/core/state"
	"github.com/teslapatrick/go-ethereum/core/types"
	"github.com/teslapatrick/go-ethereum/core/vm"
	"github.com/teslapatrick/go-ethereum/ethdb"
	"github.com/teslapatrick/go-ethereum/event"
	"github.com/teslapatrick/go-ethereum/logger"
	"github.com/teslapatrick/go-ethereum/logger/glog"
	"github.com/teslapatrick/go-ethereum/params"
	"github.com/teslapatrick/go-ethereum/pow"
	"gopkg.in/fatih/set.v0"
)

const (
	resultQueueSize  = 10
	miningLogAtDepth = 5
)

// Agent can register themself with the worker
type Agent interface {
	Work() chan<- *Work
	SetReturnCh(chan<- *Result)
	Stop()
	Start()
	GetHashRate() int64
}

// Work is the workers current environment and holds
// all of the current state information
type Work struct {
	config *params.ChainConfig
	signer types.Signer

	state         *state.StateDB // apply state changes here
	ancestors     *set.Set       // ancestor set (used for checking uncle parent validity)
	family        *set.Set       // family set (used for checking uncle invalidity)
	uncles        *set.Set       // uncle set
	tcount        int            // tx count in cycle
	ownedAccounts *set.Set
	lowGasTxs     types.Transactions
	failedTxs     types.Transactions

	Block *types.Block // the new block

	header   *types.Header
	txs      []*types.Transaction
	receipts []*types.Receipt

	createdAt time.Time
}

type Result struct {
	Work  *Work
	Block *types.Block
}

// worker is the main object which takes care of applying messages to the new state
type worker struct {
	config *params.ChainConfig

	mu sync.Mutex

	// update loop
	mux    *event.TypeMux
	events *event.TypeMuxSubscription
	wg     sync.WaitGroup

	agents map[Agent]struct{}
	recv   chan *Result
	pow    pow.PoW

	eth     Backend
	chain   *core.BlockChain
	proc    core.Validator
	chainDb ethdb.Database

	coinbase common.Address
	gasPrice *big.Int
	extra    []byte

	currentMu sync.Mutex
	current   *Work

	uncleMu        sync.Mutex
	possibleUncles map[common.Hash]*types.Block

	txQueueMu sync.Mutex
	txQueue   map[common.Hash]*types.Transaction

	unconfirmed *unconfirmedBlocks // set of locally mined blocks pending canonicalness confirmations

	// atomic status counters
	mining int32
	atWork int32

	fullValidation bool
}

func newWorker(config *params.ChainConfig, coinbase common.Address, eth Backend, mux *event.TypeMux) *worker {
	worker := &worker{
		config:         config,
		eth:            eth,
		mux:            mux,
		chainDb:        eth.ChainDb(),
		recv:           make(chan *Result, resultQueueSize),
		gasPrice:       new(big.Int),
		chain:          eth.BlockChain(),
		proc:           eth.BlockChain().Validator(),
		possibleUncles: make(map[common.Hash]*types.Block),
		coinbase:       coinbase,
		txQueue:        make(map[common.Hash]*types.Transaction),
		agents:         make(map[Agent]struct{}),
		unconfirmed:    newUnconfirmedBlocks(eth.BlockChain(), 5),
		fullValidation: false,
	}
	worker.events = worker.mux.Subscribe(core.ChainHeadEvent{}, core.ChainSideEvent{}, core.TxPreEvent{})
	go worker.update()

	go worker.wait()
	worker.commitNewWork()

	return worker
}

func (self *worker) setEtherbase(addr common.Address) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.coinbase = addr
}

func (self *worker) setExtra(extra []byte) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.extra = extra
}

func (self *worker) pending() (*types.Block, *state.StateDB) {
	self.currentMu.Lock()
	defer self.currentMu.Unlock()

	if atomic.LoadInt32(&self.mining) == 0 {
		return types.NewBlock(
			self.current.header,
			self.current.txs,
			nil,
			self.current.receipts,
		), self.current.state.Copy()
	}
	return self.current.Block, self.current.state.Copy()
}

func (self *worker) pendingBlock() *types.Block {
	self.currentMu.Lock()
	defer self.currentMu.Unlock()

	if atomic.LoadInt32(&self.mining) == 0 {
		return types.NewBlock(
			self.current.header,
			self.current.txs,
			nil,
			self.current.receipts,
		)
	}
	return self.current.Block
}

func (self *worker) start() {
	self.mu.Lock()
	defer self.mu.Unlock()

	atomic.StoreInt32(&self.mining, 1)

	// spin up agents
	for agent := range self.agents {
		agent.Start()
	}
}

func (self *worker) stop() {
	self.wg.Wait()

	self.mu.Lock()
	defer self.mu.Unlock()
	if atomic.LoadInt32(&self.mining) == 1 {
		// Stop all agents.
		for agent := range self.agents {
			agent.Stop()
			// Remove CPU agents.
			if _, ok := agent.(*CpuAgent); ok {
				delete(self.agents, agent)
			}
		}
	}

	atomic.StoreInt32(&self.mining, 0)
	atomic.StoreInt32(&self.atWork, 0)
}

func (self *worker) register(agent Agent) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.agents[agent] = struct{}{}
	agent.SetReturnCh(self.recv)
}

func (self *worker) unregister(agent Agent) {
	self.mu.Lock()
	defer self.mu.Unlock()
	delete(self.agents, agent)
	agent.Stop()
}

func (self *worker) update() {
	for event := range self.events.Chan() {
		// A real event arrived, process interesting content
		switch ev := event.Data.(type) {
		case core.ChainHeadEvent:
			self.commitNewWork()
		case core.ChainSideEvent:
			self.uncleMu.Lock()
			self.possibleUncles[ev.Block.Hash()] = ev.Block
			self.uncleMu.Unlock()
		case core.TxPreEvent:
			// Apply transaction to the pending state if we're not mining
			if atomic.LoadInt32(&self.mining) == 0 {
				self.currentMu.Lock()

				acc, _ := types.Sender(self.current.signer, ev.Tx)
				txs := map[common.Address]types.Transactions{acc: {ev.Tx}}
				txset := types.NewTransactionsByPriceAndNonce(txs)

				self.current.commitTransactions(self.mux, txset, self.gasPrice, self.chain)
				self.currentMu.Unlock()
			}
		}
	}
}

func (self *worker) wait() {
	for {
		mustCommitNewWork := true
		for result := range self.recv {
			atomic.AddInt32(&self.atWork, -1)

			if result == nil {
				continue
			}
			block := result.Block
			work := result.Work

			if self.fullValidation {
				if _, err := self.chain.InsertChain(types.Blocks{block}); err != nil {
					glog.V(logger.Error).Infoln("mining err", err)
					continue
				}
				go self.mux.Post(core.NewMinedBlockEvent{Block: block})
			} else {
				work.state.Commit(self.config.IsEIP158(block.Number()))
				parent := self.chain.GetBlock(block.ParentHash(), block.NumberU64()-1)
				if parent == nil {
					glog.V(logger.Error).Infoln("Invalid block found during mining")
					continue
				}

				auxValidator := self.eth.BlockChain().AuxValidator()
				if err := core.ValidateHeader(self.config, auxValidator, block.Header(), parent.Header(), true, false); err != nil && err != core.BlockFutureErr {
					glog.V(logger.Error).Infoln("Invalid header on mined block:", err)
					continue
				}

				stat, err := self.chain.WriteBlock(block)
				if err != nil {
					glog.V(logger.Error).Infoln("error writing block to chain", err)
					continue
				}

				// update block hash since it is now available and not when the receipt/log of individual transactions were created
				for _, r := range work.receipts {
					for _, l := range r.Logs {
						l.BlockHash = block.Hash()
					}
				}
				for _, log := range work.state.Logs() {
					log.BlockHash = block.Hash()
				}

				// check if canon block and write transactions
				if stat == core.CanonStatTy {
					// This puts transactions in a extra db for rpc
					core.WriteTransactions(self.chainDb, block)
					// store the receipts
					core.WriteReceipts(self.chainDb, work.receipts)
					// Write map map bloom filters
					core.WriteMipmapBloom(self.chainDb, block.NumberU64(), work.receipts)
					// implicit by posting ChainHeadEvent
					mustCommitNewWork = false
				}

				// broadcast before waiting for validation
				go func(block *types.Block, logs []*types.Log, receipts []*types.Receipt) {
					self.mux.Post(core.NewMinedBlockEvent{Block: block})
					self.mux.Post(core.ChainEvent{Block: block, Hash: block.Hash(), Logs: logs})

					if stat == core.CanonStatTy {
						self.mux.Post(core.ChainHeadEvent{Block: block})
						self.mux.Post(logs)
					}
					if err := core.WriteBlockReceipts(self.chainDb, block.Hash(), block.NumberU64(), receipts); err != nil {
						glog.V(logger.Warn).Infoln("error writing block receipts:", err)
					}
				}(block, work.state.Logs(), work.receipts)
			}
			// Insert the block into the set of pending ones to wait for confirmations
			self.unconfirmed.Insert(block.NumberU64(), block.Hash())

			if mustCommitNewWork {
				self.commitNewWork()
			}
		}
	}
}

// push sends a new work task to currently live miner agents.
func (self *worker) push(work *Work) {
	if atomic.LoadInt32(&self.mining) != 1 {
		return
	}
	for agent := range self.agents {
		atomic.AddInt32(&self.atWork, 1)
		if ch := agent.Work(); ch != nil {
			ch <- work
		}
	}
}

// makeCurrent creates a new environment for the current cycle.
func (self *worker) makeCurrent(parent *types.Block, header *types.Header) error {
	state, err := self.chain.StateAt(parent.Root())
	if err != nil {
		return err
	}
	work := &Work{
		config:    self.config,
		signer:    types.NewEIP155Signer(self.config.ChainId),
		state:     state,
		ancestors: set.New(),
		family:    set.New(),
		uncles:    set.New(),
		header:    header,
		createdAt: time.Now(),
	}

	// when 08 is processed ancestors contain 07 (quick block)
	for _, ancestor := range self.chain.GetBlocksFromHash(parent.Hash(), 7) {
		for _, uncle := range ancestor.Uncles() {
			work.family.Add(uncle.Hash())
		}
		work.family.Add(ancestor.Hash())
		work.ancestors.Add(ancestor.Hash())
	}
	wallets := self.eth.AccountManager().Wallets()
	accounts := make([]accounts.Account, 0, len(wallets))
	for _, wallet := range wallets {
		accounts = append(accounts, wallet.Accounts()...)
	}
	// Keep track of transactions which return errors so they can be removed
	work.tcount = 0
	work.ownedAccounts = accountAddressesSet(accounts)
	self.current = work
	return nil
}

func (w *worker) setGasPrice(p *big.Int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// calculate the minimal gas price the miner accepts when sorting out transactions.
	const pct = int64(90)
	w.gasPrice = gasprice(p, pct)

	w.mux.Post(core.GasPriceChanged{Price: w.gasPrice})
}

func (self *worker) commitNewWork() {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.uncleMu.Lock()
	defer self.uncleMu.Unlock()
	self.currentMu.Lock()
	defer self.currentMu.Unlock()

	tstart := time.Now()
	parent := self.chain.CurrentBlock()

	tstamp := tstart.Unix()
	if parent.Time().Cmp(new(big.Int).SetInt64(tstamp)) >= 0 {
		tstamp = parent.Time().Int64() + 1
	}
	// this will ensure we're not going off too far in the future
	if now := time.Now().Unix(); tstamp > now+4 {
		wait := time.Duration(tstamp-now) * time.Second
		glog.V(logger.Info).Infoln("We are too far in the future. Waiting for", wait)
		time.Sleep(wait)
	}

	num := parent.Number()
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     num.Add(num, common.Big1),
		Difficulty: core.CalcDifficulty(self.config, uint64(tstamp), parent.Time().Uint64(), parent.Number(), parent.Difficulty()),
		GasLimit:   core.CalcGasLimit(parent),
		GasUsed:    new(big.Int),
		Coinbase:   self.coinbase,
		Extra:      self.extra,
		Time:       big.NewInt(tstamp),
	}
	// If we are care about TheDAO hard-fork check whether to override the extra-data or not
	if daoBlock := self.config.DAOForkBlock; daoBlock != nil {
		// Check whether the block is among the fork extra-override range
		limit := new(big.Int).Add(daoBlock, params.DAOForkExtraRange)
		if header.Number.Cmp(daoBlock) >= 0 && header.Number.Cmp(limit) < 0 {
			// Depending whether we support or oppose the fork, override differently
			if self.config.DAOForkSupport {
				header.Extra = common.CopyBytes(params.DAOForkBlockExtra)
			} else if bytes.Equal(header.Extra, params.DAOForkBlockExtra) {
				header.Extra = []byte{} // If miner opposes, don't let it use the reserved extra-data
			}
		}
	}
	// Could potentially happen if starting to mine in an odd state.
	err := self.makeCurrent(parent, header)
	if err != nil {
		glog.V(logger.Info).Infoln("Could not create new env for mining, retrying on next block.")
		return
	}
	// Create the current work task and check any fork transitions needed
	work := self.current
	if self.config.DAOForkSupport && self.config.DAOForkBlock != nil && self.config.DAOForkBlock.Cmp(header.Number) == 0 {
		core.ApplyDAOHardFork(work.state)
	}

	pending, err := self.eth.TxPool().Pending()
	if err != nil {
		glog.Errorf("Could not fetch pending transactions: %v", err)
		return
	}

	txs := types.NewTransactionsByPriceAndNonce(pending)
	work.commitTransactions(self.mux, txs, self.gasPrice, self.chain)

	self.eth.TxPool().RemoveBatch(work.lowGasTxs)
	self.eth.TxPool().RemoveBatch(work.failedTxs)

	// compute uncles for the new block.
	var (
		uncles    []*types.Header
		badUncles []common.Hash
	)
	for hash, uncle := range self.possibleUncles {
		if len(uncles) == 2 {
			break
		}
		if err := self.commitUncle(work, uncle.Header()); err != nil {
			if glog.V(logger.Ridiculousness) {
				glog.V(logger.Detail).Infof("Bad uncle found and will be removed (%x)\n", hash[:4])
				glog.V(logger.Detail).Infoln(uncle)
			}
			badUncles = append(badUncles, hash)
		} else {
			glog.V(logger.Debug).Infof("committing %x as uncle\n", hash[:4])
			uncles = append(uncles, uncle.Header())
		}
	}
	for _, hash := range badUncles {
		delete(self.possibleUncles, hash)
	}

	if atomic.LoadInt32(&self.mining) == 1 {
		// commit state root after all state transitions.
		core.AccumulateRewards(work.state, header, uncles)
		header.Root = work.state.IntermediateRoot(self.config.IsEIP158(header.Number))
	}

	// create the new block whose nonce will be mined.
	work.Block = types.NewBlock(header, work.txs, uncles, work.receipts)

	// We only care about logging if we're actually mining.
	if atomic.LoadInt32(&self.mining) == 1 {
		glog.V(logger.Info).Infof("commit new work on block %v with %d txs & %d uncles. Took %v\n", work.Block.Number(), work.tcount, len(uncles), time.Since(tstart))
		self.unconfirmed.Shift(work.Block.NumberU64() - 1)
	}
	self.push(work)
}

func (self *worker) commitUncle(work *Work, uncle *types.Header) error {
	hash := uncle.Hash()
	if work.uncles.Has(hash) {
		return core.UncleError("Uncle not unique")
	}
	if !work.ancestors.Has(uncle.ParentHash) {
		return core.UncleError(fmt.Sprintf("Uncle's parent unknown (%x)", uncle.ParentHash[0:4]))
	}
	if work.family.Has(hash) {
		return core.UncleError(fmt.Sprintf("Uncle already in family (%x)", hash))
	}
	work.uncles.Add(uncle.Hash())
	return nil
}

func (env *Work) commitTransactions(mux *event.TypeMux, txs *types.TransactionsByPriceAndNonce, gasPrice *big.Int, bc *core.BlockChain) {
	gp := new(core.GasPool).AddGas(env.header.GasLimit)

	var coalescedLogs []*types.Log

	for {
		// Retrieve the next transaction and abort if all done
		tx := txs.Peek()
		if tx == nil {
			break
		}
		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		//
		// We use the eip155 signer regardless of the current hf.
		from, _ := types.Sender(env.signer, tx)
		// Check whether the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		if tx.Protected() && !env.config.IsEIP155(env.header.Number) {
			glog.V(logger.Detail).Infof("Transaction (%x) is replay protected, but we haven't yet hardforked. Transaction will be ignored until we hardfork.\n", tx.Hash())

			txs.Pop()
			continue
		}

		// Ignore any transactions (and accounts subsequently) with low gas limits
		if tx.GasPrice().Cmp(gasPrice) < 0 && !env.ownedAccounts.Has(from) {
			// Pop the current low-priced transaction without shifting in the next from the account
			glog.V(logger.Info).Infof("Transaction (%x) below gas price (tx=%v ask=%v). All sequential txs from this address(%x) will be ignored\n", tx.Hash().Bytes()[:4], common.CurrencyToString(tx.GasPrice()), common.CurrencyToString(gasPrice), from[:4])

			env.lowGasTxs = append(env.lowGasTxs, tx)
			txs.Pop()

			continue
		}
		// Start executing the transaction
		env.state.StartRecord(tx.Hash(), common.Hash{}, env.tcount)

		err, logs := env.commitTransaction(tx, bc, gp)
		switch {
		case core.IsGasLimitErr(err):
			// Pop the current out-of-gas transaction without shifting in the next from the account
			glog.V(logger.Detail).Infof("Gas limit reached for (%x) in this block. Continue to try smaller txs\n", from[:4])
			txs.Pop()

		case err != nil:
			// Pop the current failed transaction without shifting in the next from the account
			glog.V(logger.Detail).Infof("Transaction (%x) failed, will be removed: %v\n", tx.Hash().Bytes()[:4], err)
			env.failedTxs = append(env.failedTxs, tx)
			txs.Pop()

		default:
			// Everything ok, collect the logs and shift in the next transaction from the same account
			coalescedLogs = append(coalescedLogs, logs...)
			env.tcount++
			txs.Shift()
		}
	}

	if len(coalescedLogs) > 0 || env.tcount > 0 {
		// make a copy, the state caches the logs and these logs get "upgraded" from pending to mined
		// logs by filling in the block hash when the block was mined by the local miner. This can
		// cause a race condition if a log was "upgraded" before the PendingLogsEvent is processed.
		cpy := make([]*types.Log, len(coalescedLogs))
		for i, l := range coalescedLogs {
			cpy[i] = new(types.Log)
			*cpy[i] = *l
		}
		go func(logs []*types.Log, tcount int) {
			if len(logs) > 0 {
				mux.Post(core.PendingLogsEvent{Logs: logs})
			}
			if tcount > 0 {
				mux.Post(core.PendingStateEvent{})
			}
		}(cpy, env.tcount)
	}
}

func (env *Work) commitTransaction(tx *types.Transaction, bc *core.BlockChain, gp *core.GasPool) (error, []*types.Log) {
	snap := env.state.Snapshot()

	receipt, _, err := core.ApplyTransaction(env.config, bc, gp, env.state, env.header, tx, env.header.GasUsed, vm.Config{})
	if err != nil {
		env.state.RevertToSnapshot(snap)
		return err, nil
	}
	env.txs = append(env.txs, tx)
	env.receipts = append(env.receipts, receipt)

	return nil, receipt.Logs
}

// TODO: remove or use
func (self *worker) HashRate() int64 {
	return 0
}

// gasprice calculates a reduced gas price based on the pct
// XXX Use big.Rat?
func gasprice(price *big.Int, pct int64) *big.Int {
	p := new(big.Int).Set(price)
	p.Div(p, big.NewInt(100))
	p.Mul(p, big.NewInt(pct))
	return p
}

func accountAddressesSet(accounts []accounts.Account) *set.Set {
	accountSet := set.New()
	for _, account := range accounts {
		accountSet.Add(account.Address)
	}
	return accountSet
}
