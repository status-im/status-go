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

package eth

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"runtime"
	"time"

	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/net/context"
)

const defaultTraceTimeout = 5 * time.Second

// PublicEthereumAPI provides an API to access Ethereum full node-related
// information.
type PublicEthereumAPI struct {
	e *Ethereum
}

// NewPublicEthereumAPI creates a new Etheruem protocol API for full nodes.
func NewPublicEthereumAPI(e *Ethereum) *PublicEthereumAPI {
	return &PublicEthereumAPI{e}
}

// Etherbase is the address that mining rewards will be send to
func (s *PublicEthereumAPI) Etherbase() (common.Address, error) {
	return s.e.Etherbase()
}

// Coinbase is the address that mining rewards will be send to (alias for Etherbase)
func (s *PublicEthereumAPI) Coinbase() (common.Address, error) {
	return s.Etherbase()
}

// Hashrate returns the POW hashrate
func (s *PublicEthereumAPI) Hashrate() *rpc.HexNumber {
	return rpc.NewHexNumber(s.e.Miner().HashRate())
}

// PublicMinerAPI provides an API to control the miner.
// It offers only methods that operate on data that pose no security risk when it is publicly accessible.
type PublicMinerAPI struct {
	e     *Ethereum
	agent *miner.RemoteAgent
}

// NewPublicMinerAPI create a new PublicMinerAPI instance.
func NewPublicMinerAPI(e *Ethereum) *PublicMinerAPI {
	agent := miner.NewRemoteAgent()
	e.Miner().Register(agent)

	return &PublicMinerAPI{e, agent}
}

// Mining returns an indication if this node is currently mining.
func (s *PublicMinerAPI) Mining() bool {
	return s.e.IsMining()
}

// SubmitWork can be used by external miner to submit their POW solution. It returns an indication if the work was
// accepted. Note, this is not an indication if the provided work was valid!
func (s *PublicMinerAPI) SubmitWork(nonce rpc.HexNumber, solution, digest common.Hash) bool {
	return s.agent.SubmitWork(nonce.Uint64(), digest, solution)
}

// GetWork returns a work package for external miner. The work package consists of 3 strings
// result[0], 32 bytes hex encoded current block header pow-hash
// result[1], 32 bytes hex encoded seed hash used for DAG
// result[2], 32 bytes hex encoded boundary condition ("target"), 2^256/difficulty
func (s *PublicMinerAPI) GetWork() (work [3]string, err error) {
	if !s.e.IsMining() {
		if err := s.e.StartMining(0, ""); err != nil {
			return work, err
		}
	}
	if work, err = s.agent.GetWork(); err == nil {
		return
	}
	glog.V(logger.Debug).Infof("%v", err)
	return work, fmt.Errorf("mining not ready")
}

// SubmitHashrate can be used for remote miners to submit their hash rate. This enables the node to report the combined
// hash rate of all miners which submit work through this node. It accepts the miner hash rate and an identifier which
// must be unique between nodes.
func (s *PublicMinerAPI) SubmitHashrate(hashrate rpc.HexNumber, id common.Hash) bool {
	s.agent.SubmitHashrate(id, hashrate.Uint64())
	return true
}

// PrivateMinerAPI provides private RPC methods to control the miner.
// These methods can be abused by external users and must be considered insecure for use by untrusted users.
type PrivateMinerAPI struct {
	e *Ethereum
}

// NewPrivateMinerAPI create a new RPC service which controls the miner of this node.
func NewPrivateMinerAPI(e *Ethereum) *PrivateMinerAPI {
	return &PrivateMinerAPI{e: e}
}

// Start the miner with the given number of threads. If threads is nil the number of
// workers started is equal to the number of logical CPU's that are usable by this process.
func (s *PrivateMinerAPI) Start(threads *rpc.HexNumber) (bool, error) {
	s.e.StartAutoDAG()

	if threads == nil {
		threads = rpc.NewHexNumber(runtime.NumCPU())
	}

	err := s.e.StartMining(threads.Int(), "")
	if err == nil {
		return true, nil
	}
	return false, err
}

// Stop the miner
func (s *PrivateMinerAPI) Stop() bool {
	s.e.StopMining()
	return true
}

// SetExtra sets the extra data string that is included when this miner mines a block.
func (s *PrivateMinerAPI) SetExtra(extra string) (bool, error) {
	if err := s.e.Miner().SetExtra([]byte(extra)); err != nil {
		return false, err
	}
	return true, nil
}

// SetGasPrice sets the minimum accepted gas price for the miner.
func (s *PrivateMinerAPI) SetGasPrice(gasPrice rpc.HexNumber) bool {
	s.e.Miner().SetGasPrice(gasPrice.BigInt())
	return true
}

// SetEtherbase sets the etherbase of the miner
func (s *PrivateMinerAPI) SetEtherbase(etherbase common.Address) bool {
	s.e.SetEtherbase(etherbase)
	return true
}

// StartAutoDAG starts auto DAG generation. This will prevent the DAG generating on epoch change
// which will cause the node to stop mining during the generation process.
func (s *PrivateMinerAPI) StartAutoDAG() bool {
	s.e.StartAutoDAG()
	return true
}

// StopAutoDAG stops auto DAG generation
func (s *PrivateMinerAPI) StopAutoDAG() bool {
	s.e.StopAutoDAG()
	return true
}

// MakeDAG creates the new DAG for the given block number
func (s *PrivateMinerAPI) MakeDAG(blockNr rpc.BlockNumber) (bool, error) {
	if err := ethash.MakeDAG(uint64(blockNr.Int64()), ""); err != nil {
		return false, err
	}
	return true, nil
}

// PrivateAdminAPI is the collection of Etheruem full node-related APIs
// exposed over the private admin endpoint.
type PrivateAdminAPI struct {
	eth *Ethereum
}

// NewPrivateAdminAPI creates a new API definition for the full node private
// admin methods of the Ethereum service.
func NewPrivateAdminAPI(eth *Ethereum) *PrivateAdminAPI {
	return &PrivateAdminAPI{eth: eth}
}

// ExportChain exports the current blockchain into a local file.
func (api *PrivateAdminAPI) ExportChain(file string) (bool, error) {
	// Make sure we can create the file to export into
	out, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return false, err
	}
	defer out.Close()

	// Export the blockchain
	if err := api.eth.BlockChain().Export(out); err != nil {
		return false, err
	}
	return true, nil
}

func hasAllBlocks(chain *core.BlockChain, bs []*types.Block) bool {
	for _, b := range bs {
		if !chain.HasBlock(b.Hash()) {
			return false
		}
	}

	return true
}

// ImportChain imports a blockchain from a local file.
func (api *PrivateAdminAPI) ImportChain(file string) (bool, error) {
	// Make sure the can access the file to import
	in, err := os.Open(file)
	if err != nil {
		return false, err
	}
	defer in.Close()

	// Run actual the import in pre-configured batches
	stream := rlp.NewStream(in, 0)

	blocks, index := make([]*types.Block, 0, 2500), 0
	for batch := 0; ; batch++ {
		// Load a batch of blocks from the input file
		for len(blocks) < cap(blocks) {
			block := new(types.Block)
			if err := stream.Decode(block); err == io.EOF {
				break
			} else if err != nil {
				return false, fmt.Errorf("block %d: failed to parse: %v", index, err)
			}
			blocks = append(blocks, block)
			index++
		}
		if len(blocks) == 0 {
			break
		}

		if hasAllBlocks(api.eth.BlockChain(), blocks) {
			blocks = blocks[:0]
			continue
		}
		// Import the batch and reset the buffer
		if _, err := api.eth.BlockChain().InsertChain(blocks); err != nil {
			return false, fmt.Errorf("batch %d: failed to insert: %v", batch, err)
		}
		blocks = blocks[:0]
	}
	return true, nil
}

// PublicDebugAPI is the collection of Etheruem full node APIs exposed
// over the public debugging endpoint.
type PublicDebugAPI struct {
	eth *Ethereum
}

// NewPublicDebugAPI creates a new API definition for the full node-
// related public debug methods of the Ethereum service.
func NewPublicDebugAPI(eth *Ethereum) *PublicDebugAPI {
	return &PublicDebugAPI{eth: eth}
}

// DumpBlock retrieves the entire state of the database at a given block.
func (api *PublicDebugAPI) DumpBlock(number uint64) (state.Dump, error) {
	block := api.eth.BlockChain().GetBlockByNumber(number)
	if block == nil {
		return state.Dump{}, fmt.Errorf("block #%d not found", number)
	}
	stateDb, err := api.eth.BlockChain().StateAt(block.Root())
	if err != nil {
		return state.Dump{}, err
	}
	return stateDb.RawDump(), nil
}

// PrivateDebugAPI is the collection of Etheruem full node APIs exposed over
// the private debugging endpoint.
type PrivateDebugAPI struct {
	config *core.ChainConfig
	eth    *Ethereum
}

// NewPrivateDebugAPI creates a new API definition for the full node-related
// private debug methods of the Ethereum service.
func NewPrivateDebugAPI(config *core.ChainConfig, eth *Ethereum) *PrivateDebugAPI {
	return &PrivateDebugAPI{config: config, eth: eth}
}

// BlockTraceResult is the returned value when replaying a block to check for
// consensus results and full VM trace logs for all included transactions.
type BlockTraceResult struct {
	Validated  bool                  `json:"validated"`
	StructLogs []ethapi.StructLogRes `json:"structLogs"`
	Error      string                `json:"error"`
}

// TraceArgs holds extra parameters to trace functions
type TraceArgs struct {
	*vm.LogConfig
	Tracer  *string
	Timeout *string
}

// TraceBlock processes the given block's RLP but does not import the block in to
// the chain.
func (api *PrivateDebugAPI) TraceBlock(blockRlp []byte, config *vm.LogConfig) BlockTraceResult {
	var block types.Block
	err := rlp.Decode(bytes.NewReader(blockRlp), &block)
	if err != nil {
		return BlockTraceResult{Error: fmt.Sprintf("could not decode block: %v", err)}
	}

	validated, logs, err := api.traceBlock(&block, config)
	return BlockTraceResult{
		Validated:  validated,
		StructLogs: ethapi.FormatLogs(logs),
		Error:      formatError(err),
	}
}

// TraceBlockFromFile loads the block's RLP from the given file name and attempts to
// process it but does not import the block in to the chain.
func (api *PrivateDebugAPI) TraceBlockFromFile(file string, config *vm.LogConfig) BlockTraceResult {
	blockRlp, err := ioutil.ReadFile(file)
	if err != nil {
		return BlockTraceResult{Error: fmt.Sprintf("could not read file: %v", err)}
	}
	return api.TraceBlock(blockRlp, config)
}

// TraceBlockByNumber processes the block by canonical block number.
func (api *PrivateDebugAPI) TraceBlockByNumber(number uint64, config *vm.LogConfig) BlockTraceResult {
	// Fetch the block that we aim to reprocess
	block := api.eth.BlockChain().GetBlockByNumber(number)
	if block == nil {
		return BlockTraceResult{Error: fmt.Sprintf("block #%d not found", number)}
	}

	validated, logs, err := api.traceBlock(block, config)
	return BlockTraceResult{
		Validated:  validated,
		StructLogs: ethapi.FormatLogs(logs),
		Error:      formatError(err),
	}
}

// TraceBlockByHash processes the block by hash.
func (api *PrivateDebugAPI) TraceBlockByHash(hash common.Hash, config *vm.LogConfig) BlockTraceResult {
	// Fetch the block that we aim to reprocess
	block := api.eth.BlockChain().GetBlockByHash(hash)
	if block == nil {
		return BlockTraceResult{Error: fmt.Sprintf("block #%x not found", hash)}
	}

	validated, logs, err := api.traceBlock(block, config)
	return BlockTraceResult{
		Validated:  validated,
		StructLogs: ethapi.FormatLogs(logs),
		Error:      formatError(err),
	}
}

// traceBlock processes the given block but does not save the state.
func (api *PrivateDebugAPI) traceBlock(block *types.Block, logConfig *vm.LogConfig) (bool, []vm.StructLog, error) {
	// Validate and reprocess the block
	var (
		blockchain = api.eth.BlockChain()
		validator  = blockchain.Validator()
		processor  = blockchain.Processor()
	)

	structLogger := vm.NewStructLogger(logConfig)

	config := vm.Config{
		Debug:  true,
		Tracer: structLogger,
	}

	if err := core.ValidateHeader(api.config, blockchain.AuxValidator(), block.Header(), blockchain.GetHeader(block.ParentHash(), block.NumberU64()-1), true, false); err != nil {
		return false, structLogger.StructLogs(), err
	}
	statedb, err := blockchain.StateAt(blockchain.GetBlock(block.ParentHash(), block.NumberU64()-1).Root())
	if err != nil {
		return false, structLogger.StructLogs(), err
	}

	receipts, _, usedGas, err := processor.Process(block, statedb, config)
	if err != nil {
		return false, structLogger.StructLogs(), err
	}
	if err := validator.ValidateState(block, blockchain.GetBlock(block.ParentHash(), block.NumberU64()-1), statedb, receipts, usedGas); err != nil {
		return false, structLogger.StructLogs(), err
	}
	return true, structLogger.StructLogs(), nil
}

// callmsg is the message type used for call transations.
type callmsg struct {
	addr          common.Address
	to            *common.Address
	gas, gasPrice *big.Int
	value         *big.Int
	data          []byte
}

// accessor boilerplate to implement core.Message
func (m callmsg) From() (common.Address, error)         { return m.addr, nil }
func (m callmsg) FromFrontier() (common.Address, error) { return m.addr, nil }
func (m callmsg) Nonce() uint64                         { return 0 }
func (m callmsg) CheckNonce() bool                      { return false }
func (m callmsg) To() *common.Address                   { return m.to }
func (m callmsg) GasPrice() *big.Int                    { return m.gasPrice }
func (m callmsg) Gas() *big.Int                         { return m.gas }
func (m callmsg) Value() *big.Int                       { return m.value }
func (m callmsg) Data() []byte                          { return m.data }

// formatError formats a Go error into either an empty string or the data content
// of the error itself.
func formatError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

type timeoutError struct{}

func (t *timeoutError) Error() string {
	return "Execution time exceeded"
}

// TraceTransaction returns the structured logs created during the execution of EVM
// and returns them as a JSON object.
func (api *PrivateDebugAPI) TraceTransaction(ctx context.Context, txHash common.Hash, config *TraceArgs) (interface{}, error) {
	var tracer vm.Tracer
	if config != nil && config.Tracer != nil {
		timeout := defaultTraceTimeout
		if config.Timeout != nil {
			var err error
			if timeout, err = time.ParseDuration(*config.Timeout); err != nil {
				return nil, err
			}
		}

		var err error
		if tracer, err = ethapi.NewJavascriptTracer(*config.Tracer); err != nil {
			return nil, err
		}

		// Handle timeouts and RPC cancellations
		deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
		go func() {
			<-deadlineCtx.Done()
			tracer.(*ethapi.JavascriptTracer).Stop(&timeoutError{})
		}()
		defer cancel()
	} else if config == nil {
		tracer = vm.NewStructLogger(nil)
	} else {
		tracer = vm.NewStructLogger(config.LogConfig)
	}

	// Retrieve the tx from the chain and the containing block
	tx, blockHash, _, txIndex := core.GetTransaction(api.eth.ChainDb(), txHash)
	if tx == nil {
		return nil, fmt.Errorf("transaction %x not found", txHash)
	}
	block := api.eth.BlockChain().GetBlockByHash(blockHash)
	if block == nil {
		return nil, fmt.Errorf("block %x not found", blockHash)
	}
	// Create the state database to mutate and eventually trace
	parent := api.eth.BlockChain().GetBlock(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return nil, fmt.Errorf("block parent %x not found", block.ParentHash())
	}
	stateDb, err := api.eth.BlockChain().StateAt(parent.Root())
	if err != nil {
		return nil, err
	}
	// Mutate the state and trace the selected transaction
	for idx, tx := range block.Transactions() {
		// Assemble the transaction call message
		from, err := tx.FromFrontier()
		if err != nil {
			return nil, fmt.Errorf("sender retrieval failed: %v", err)
		}
		msg := callmsg{
			addr:     from,
			to:       tx.To(),
			gas:      tx.Gas(),
			gasPrice: tx.GasPrice(),
			value:    tx.Value(),
			data:     tx.Data(),
		}
		// Mutate the state if we haven't reached the tracing transaction yet
		if uint64(idx) < txIndex {
			vmenv := core.NewEnv(stateDb, api.config, api.eth.BlockChain(), msg, block.Header(), vm.Config{})
			_, _, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(tx.Gas()))
			if err != nil {
				return nil, fmt.Errorf("mutation failed: %v", err)
			}
			stateDb.DeleteSuicides()
			continue
		}
		// Otherwise trace the transaction and return
		vmenv := core.NewEnv(stateDb, api.config, api.eth.BlockChain(), msg, block.Header(), vm.Config{Debug: true, Tracer: tracer})
		ret, gas, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(tx.Gas()))
		if err != nil {
			return nil, fmt.Errorf("tracing failed: %v", err)
		}

		switch tracer := tracer.(type) {
		case *vm.StructLogger:
			return &ethapi.ExecutionResult{
				Gas:         gas,
				ReturnValue: fmt.Sprintf("%x", ret),
				StructLogs:  ethapi.FormatLogs(tracer.StructLogs()),
			}, nil
		case *ethapi.JavascriptTracer:
			return tracer.GetResult()
		}
	}
	return nil, errors.New("database inconsistency")
}
