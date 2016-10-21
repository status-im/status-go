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

package vm

import (
	"fmt"
	"math/big"
	"os"
	"unicode"

	"github.com/ethereum/go-ethereum/common"
)

type Storage map[common.Hash]common.Hash

func (self Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range self {
		cpy[key] = value
	}

	return cpy
}

// LogConfig are the configuration options for structured logger the EVM
type LogConfig struct {
	DisableMemory  bool // disable memory capture
	DisableStack   bool // disable stack capture
	DisableStorage bool // disable storage capture
	FullStorage    bool // show full storage (slow)
}

// StructLog is emitted to the Environment each cycle and lists information about the current internal state
// prior to the execution of the statement.
type StructLog struct {
	Pc      uint64
	Op      OpCode
	Gas     *big.Int
	GasCost *big.Int
	Memory  []byte
	Stack   []*big.Int
	Storage map[common.Hash]common.Hash
	Depth   int
	Err     error
}

// Tracer is used to collect execution traces from an EVM transaction
// execution. CaptureState is called for each step of the VM with the
// current VM state.
// Note that reference types are actual VM data structures; make copies
// if you need to retain them beyond the current call.
type Tracer interface {
	CaptureState(env Environment, pc uint64, op OpCode, gas, cost *big.Int, memory *Memory, stack *Stack, contract *Contract, depth int, err error)
}

// StructLogger is an EVM state logger and implements Tracer.
//
// StructLogger can capture state based on the given Log configuration and also keeps
// a track record of modified storage which is used in reporting snapshots of the
// contract their storage.
type StructLogger struct {
	cfg LogConfig

	logs          []StructLog
	changedValues map[common.Address]Storage
}

// NewLogger returns a new logger
func NewStructLogger(cfg *LogConfig) *StructLogger {
	logger := &StructLogger{
		changedValues: make(map[common.Address]Storage),
	}
	if cfg != nil {
		logger.cfg = *cfg
	}
	return logger
}

// captureState logs a new structured log message and pushes it out to the environment
//
// captureState also tracks SSTORE ops to track dirty values.
func (l *StructLogger) CaptureState(env Environment, pc uint64, op OpCode, gas, cost *big.Int, memory *Memory, stack *Stack, contract *Contract, depth int, err error) {
	// initialise new changed values storage container for this contract
	// if not present.
	if l.changedValues[contract.Address()] == nil {
		l.changedValues[contract.Address()] = make(Storage)
	}

	// capture SSTORE opcodes and determine the changed value and store
	// it in the local storage container. NOTE: we do not need to do any
	// range checks here because that's already handler prior to calling
	// this function.
	switch op {
	case SSTORE:
		var (
			value   = common.BigToHash(stack.data[stack.len()-2])
			address = common.BigToHash(stack.data[stack.len()-1])
		)
		l.changedValues[contract.Address()][address] = value
	}

	// copy a snapstot of the current memory state to a new buffer
	var mem []byte
	if !l.cfg.DisableMemory {
		mem = make([]byte, len(memory.Data()))
		copy(mem, memory.Data())
	}

	// copy a snapshot of the current stack state to a new buffer
	var stck []*big.Int
	if !l.cfg.DisableStack {
		stck = make([]*big.Int, len(stack.Data()))
		for i, item := range stack.Data() {
			stck[i] = new(big.Int).Set(item)
		}
	}

	// Copy the storage based on the settings specified in the log config. If full storage
	// is disabled (default) we can use the simple Storage.Copy method, otherwise we use
	// the state object to query for all values (slow process).
	var storage Storage
	if !l.cfg.DisableStorage {
		if l.cfg.FullStorage {
			storage = make(Storage)
			// Get the contract account and loop over each storage entry. This may involve looping over
			// the trie and is a very expensive process.
			env.Db().GetAccount(contract.Address()).ForEachStorage(func(key, value common.Hash) bool {
				storage[key] = value
				// Return true, indicating we'd like to continue.
				return true
			})
		} else {
			// copy a snapshot of the current storage to a new container.
			storage = l.changedValues[contract.Address()].Copy()
		}
	}
	// create a new snaptshot of the EVM.
	log := StructLog{pc, op, new(big.Int).Set(gas), cost, mem, stck, storage, env.Depth(), err}

	l.logs = append(l.logs, log)
}

// StructLogs returns a list of captured log entries
func (l *StructLogger) StructLogs() []StructLog {
	return l.logs
}

// StdErrFormat formats a slice of StructLogs to human readable format
func StdErrFormat(logs []StructLog) {
	fmt.Fprintf(os.Stderr, "VM STAT %d OPs\n", len(logs))
	for _, log := range logs {
		fmt.Fprintf(os.Stderr, "PC %08d: %s GAS: %v COST: %v", log.Pc, log.Op, log.Gas, log.GasCost)
		if log.Err != nil {
			fmt.Fprintf(os.Stderr, " ERROR: %v", log.Err)
		}
		fmt.Fprintf(os.Stderr, "\n")

		fmt.Fprintln(os.Stderr, "STACK =", len(log.Stack))

		for i := len(log.Stack) - 1; i >= 0; i-- {
			fmt.Fprintf(os.Stderr, "%04d: %x\n", len(log.Stack)-i-1, common.LeftPadBytes(log.Stack[i].Bytes(), 32))
		}

		const maxMem = 10
		addr := 0
		fmt.Fprintln(os.Stderr, "MEM =", len(log.Memory))
		for i := 0; i+16 <= len(log.Memory) && addr < maxMem; i += 16 {
			data := log.Memory[i : i+16]
			str := fmt.Sprintf("%04d: % x  ", addr*16, data)
			for _, r := range data {
				if r == 0 {
					str += "."
				} else if unicode.IsPrint(rune(r)) {
					str += fmt.Sprintf("%s", string(r))
				} else {
					str += "?"
				}
			}
			addr++
			fmt.Fprintln(os.Stderr, str)
		}

		fmt.Fprintln(os.Stderr, "STORAGE =", len(log.Storage))
		for h, item := range log.Storage {
			fmt.Fprintf(os.Stderr, "%x: %x\n", h, item)
		}
		fmt.Fprintln(os.Stderr)
	}
}
