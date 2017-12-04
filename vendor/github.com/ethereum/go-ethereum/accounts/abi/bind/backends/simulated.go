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

package backends

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
)

// This nil assignment ensures compile time that SimulatedBackend implements bind.ContractBackend.
var _ bind.ContractBackend = (*SimulatedBackend)(nil)

var errBlockNumberUnsupported = errors.New("SimulatedBackend cannot access blocks other than the latest block")
var errGasEstimationFailed = errors.New("gas required exceeds allowance or always failing transaction")

// SimulatedBackend implements bind.ContractBackend, simulating a blockchain in
// the background. Its main purpose is to allow easily testing contract bindings.
type SimulatedBackend struct {
	database   ethdb.Database   // In memory database to store our testing data
	blockchain *core.BlockChain // Ethereum blockchain to handle the consensus

	mu           sync.Mutex
	pendingBlock *types.Block   // Currently pending block that will be imported on request
	pendingState *state.StateDB // Currently pending state that will be the active on on request

	config *params.ChainConfig
}

// NewSimulatedBackend creates a new binding backend using a simulated blockchain
// for testing purposes.
func NewSimulatedBackend(alloc core.GenesisAlloc) *SimulatedBackend {
	database, _ := ethdb.NewMemDatabase()
	genesis := core.Genesis{Config: params.AllEthashProtocolChanges, Alloc: alloc}
	genesis.MustCommit(database)
	blockchain, _ := core.NewBlockChain(database, genesis.Config, ethash.NewFaker(), vm.Config{})
	backend := &SimulatedBackend{database: database, blockchain: blockchain, config: genesis.Config}
	backend.rollback()
	return backend
}

// Commit imports all the pending transactions as a single block and starts a
// fresh new state.
func (b *SimulatedBackend) Commit() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, err := b.blockchain.InsertChain([]*types.Block{b.pendingBlock}); err != nil {
		panic(err) // This cannot happen unless the simulator is wrong, fail in that case
	}
	b.rollback()
}

// Rollback aborts all pending transactions, reverting to the last committed state.
func (b *SimulatedBackend) Rollback() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.rollback()
}

func (b *SimulatedBackend) rollback() {
	blocks, _ := core.GenerateChain(b.config, b.blockchain.CurrentBlock(), b.database, 1, func(int, *core.BlockGen) {})
	b.pendingBlock = blocks[0]
	b.pendingState, _ = state.New(b.pendingBlock.Root(), state.NewDatabase(b.database))
}

// CodeAt returns the code associated with a certain account in the blockchain.
func (b *SimulatedBackend) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if blockNumber != nil && blockNumber.Cmp(b.blockchain.CurrentBlock().Number()) != 0 {
		return nil, errBlockNumberUnsupported
	}
	statedb, _ := b.blockchain.State()
	return statedb.GetCode(contract), nil
}

// BalanceAt returns the wei balance of a certain account in the blockchain.
func (b *SimulatedBackend) BalanceAt(ctx context.Context, contract common.Address, blockNumber *big.Int) (*big.Int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if blockNumber != nil && blockNumber.Cmp(b.blockchain.CurrentBlock().Number()) != 0 {
		return nil, errBlockNumberUnsupported
	}
	statedb, _ := b.blockchain.State()
	return statedb.GetBalance(contract), nil
}

// NonceAt returns the nonce of a certain account in the blockchain.
func (b *SimulatedBackend) NonceAt(ctx context.Context, contract common.Address, blockNumber *big.Int) (uint64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if blockNumber != nil && blockNumber.Cmp(b.blockchain.CurrentBlock().Number()) != 0 {
		return 0, errBlockNumberUnsupported
	}
	statedb, _ := b.blockchain.State()
	return statedb.GetNonce(contract), nil
}

// StorageAt returns the value of key in the storage of an account in the blockchain.
func (b *SimulatedBackend) StorageAt(ctx context.Context, contract common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if blockNumber != nil && blockNumber.Cmp(b.blockchain.CurrentBlock().Number()) != 0 {
		return nil, errBlockNumberUnsupported
	}
	statedb, _ := b.blockchain.State()
	val := statedb.GetState(contract, key)
	return val[:], nil
}

// TransactionReceipt returns the receipt of a transaction.
func (b *SimulatedBackend) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	receipt, _, _, _ := core.GetReceipt(b.database, txHash)
	return receipt, nil
}

// PendingCodeAt returns the code associated with an account in the pending state.
func (b *SimulatedBackend) PendingCodeAt(ctx context.Context, contract common.Address) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.pendingState.GetCode(contract), nil
}

// CallContract executes a contract call.
func (b *SimulatedBackend) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if blockNumber != nil && blockNumber.Cmp(b.blockchain.CurrentBlock().Number()) != 0 {
		return nil, errBlockNumberUnsupported
	}
	state, err := b.blockchain.State()
	if err != nil {
		return nil, err
	}
	rval, _, _, err := b.callContract(ctx, call, b.blockchain.CurrentBlock(), state)
	return rval, err
}

// PendingCallContract executes a contract call on the pending state.
func (b *SimulatedBackend) PendingCallContract(ctx context.Context, call ethereum.CallMsg) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	defer b.pendingState.RevertToSnapshot(b.pendingState.Snapshot())

	rval, _, _, err := b.callContract(ctx, call, b.pendingBlock, b.pendingState)
	return rval, err
}

// PendingNonceAt implements PendingStateReader.PendingNonceAt, retrieving
// the nonce currently pending for the account.
func (b *SimulatedBackend) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.pendingState.GetOrNewStateObject(account).Nonce(), nil
}

// SuggestGasPrice implements ContractTransactor.SuggestGasPrice. Since the simulated
// chain doens't have miners, we just return a gas price of 1 for any call.
func (b *SimulatedBackend) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return big.NewInt(1), nil
}

// EstimateGas executes the requested code against the currently pending block/state and
// returns the used amount of gas.
func (b *SimulatedBackend) EstimateGas(ctx context.Context, call ethereum.CallMsg) (*big.Int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Determine the lowest and highest possible gas limits to binary search in between
	var (
		lo  uint64 = params.TxGas - 1
		hi  uint64
		cap uint64
	)
	if call.Gas != nil && call.Gas.Uint64() >= params.TxGas {
		hi = call.Gas.Uint64()
	} else {
		hi = b.pendingBlock.GasLimit().Uint64()
	}
	cap = hi

	// Create a helper to check if a gas allowance results in an executable transaction
	executable := func(gas uint64) bool {
		call.Gas = new(big.Int).SetUint64(gas)

		snapshot := b.pendingState.Snapshot()
		_, _, failed, err := b.callContract(ctx, call, b.pendingBlock, b.pendingState)
		b.pendingState.RevertToSnapshot(snapshot)

		if err != nil || failed {
			return false
		}
		return true
	}
	// Execute the binary search and hone in on an executable gas limit
	for lo+1 < hi {
		mid := (hi + lo) / 2
		if !executable(mid) {
			lo = mid
		} else {
			hi = mid
		}
	}
	// Reject the transaction as invalid if it still fails at the highest allowance
	if hi == cap {
		if !executable(hi) {
			return nil, errGasEstimationFailed
		}
	}
	return new(big.Int).SetUint64(hi), nil
}

// callContract implemens common code between normal and pending contract calls.
// state is modified during execution, make sure to copy it if necessary.
func (b *SimulatedBackend) callContract(ctx context.Context, call ethereum.CallMsg, block *types.Block, statedb *state.StateDB) ([]byte, *big.Int, bool, error) {
	// Ensure message is initialized properly.
	if call.GasPrice == nil {
		call.GasPrice = big.NewInt(1)
	}
	if call.Gas == nil || call.Gas.Sign() == 0 {
		call.Gas = big.NewInt(50000000)
	}
	if call.Value == nil {
		call.Value = new(big.Int)
	}
	// Set infinite balance to the fake caller account.
	from := statedb.GetOrNewStateObject(call.From)
	from.SetBalance(math.MaxBig256)
	// Execute the call.
	msg := callmsg{call}

	evmContext := core.NewEVMContext(msg, block.Header(), b.blockchain, nil)
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	vmenv := vm.NewEVM(evmContext, statedb, b.config, vm.Config{})
	gaspool := new(core.GasPool).AddGas(math.MaxBig256)
	ret, gasUsed, _, failed, err := core.NewStateTransition(vmenv, msg, gaspool).TransitionDb()
	return ret, gasUsed, failed, err
}

// SendTransaction updates the pending block to include the given transaction.
// It panics if the transaction is invalid.
func (b *SimulatedBackend) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	sender, err := types.Sender(types.HomesteadSigner{}, tx)
	if err != nil {
		panic(fmt.Errorf("invalid transaction: %v", err))
	}
	nonce := b.pendingState.GetNonce(sender)
	if tx.Nonce() != nonce {
		panic(fmt.Errorf("invalid transaction nonce: got %d, want %d", tx.Nonce(), nonce))
	}

	blocks, _ := core.GenerateChain(b.config, b.blockchain.CurrentBlock(), b.database, 1, func(number int, block *core.BlockGen) {
		for _, tx := range b.pendingBlock.Transactions() {
			block.AddTx(tx)
		}
		block.AddTx(tx)
	})
	b.pendingBlock = blocks[0]
	b.pendingState, _ = state.New(b.pendingBlock.Root(), state.NewDatabase(b.database))
	return nil
}

// JumpTimeInSeconds adds skip seconds to the clock
func (b *SimulatedBackend) AdjustTime(adjustment time.Duration) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	blocks, _ := core.GenerateChain(b.config, b.blockchain.CurrentBlock(), b.database, 1, func(number int, block *core.BlockGen) {
		for _, tx := range b.pendingBlock.Transactions() {
			block.AddTx(tx)
		}
		block.OffsetTime(int64(adjustment.Seconds()))
	})
	b.pendingBlock = blocks[0]
	b.pendingState, _ = state.New(b.pendingBlock.Root(), state.NewDatabase(b.database))

	return nil
}

// callmsg implements core.Message to allow passing it as a transaction simulator.
type callmsg struct {
	ethereum.CallMsg
}

func (m callmsg) From() common.Address { return m.CallMsg.From }
func (m callmsg) Nonce() uint64        { return 0 }
func (m callmsg) CheckNonce() bool     { return false }
func (m callmsg) To() *common.Address  { return m.CallMsg.To }
func (m callmsg) GasPrice() *big.Int   { return m.CallMsg.GasPrice }
func (m callmsg) Gas() *big.Int        { return m.CallMsg.Gas }
func (m callmsg) Value() *big.Int      { return m.CallMsg.Value }
func (m callmsg) Data() []byte         { return m.CallMsg.Data }
