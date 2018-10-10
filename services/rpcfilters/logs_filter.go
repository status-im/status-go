package rpcfilters

import (
	"context"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type logsFilter struct {
	mu                  sync.Mutex
	logs                []types.Log
	lastSeenBlockNumber uint64
	lastSeenBlockHash   common.Hash

	id    rpc.ID
	crit  ethereum.FilterQuery
	timer *time.Timer

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

func (f *logsFilter) add(data interface{}) {
	logs := data.([]types.Log)
	filtered, num, hash := filterLogs(logs, f.crit, f.lastSeenBlockNumber, f.lastSeenBlockHash)
	f.mu.Lock()
	f.lastSeenBlockNumber = num
	f.lastSeenBlockHash = hash
	if len(filtered) > 0 {
		f.logs = append(f.logs, filtered...)
	}
	f.mu.Unlock()
}

func (f *logsFilter) pop() interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.logs == nil {
		return []types.Log{}
	}
	rst := f.logs
	f.logs = nil
	return rst
}

func (f *logsFilter) stop() {
	close(f.done)
	if f.cancel != nil {
		f.cancel()
	}
}

func (f *logsFilter) deadline() *time.Timer {
	return f.timer
}

func includes(addresses []common.Address, a common.Address) bool {
	for _, addr := range addresses {
		if addr == a {
			return true
		}
	}

	return false
}

// filterLogs creates a slice of logs matching the given criteria.
func filterLogs(logs []types.Log, crit ethereum.FilterQuery, blockNum uint64, blockHash common.Hash) (
	ret []types.Log, num uint64, hash common.Hash) {
	num = blockNum
	hash = blockHash
Logs:
	for _, log := range logs {
		// skip logs from seen blocks
		if log.BlockNumber < blockNum {
			continue
		} else if log.BlockNumber == blockNum && log.BlockHash == blockHash {
			continue
		}
		// find highest block number that we didnt see before
		if log.BlockNumber >= num {
			num = log.BlockNumber
			hash = log.BlockHash
		}
		if crit.FromBlock != nil && crit.FromBlock.Int64() >= 0 && crit.FromBlock.Uint64() > log.BlockNumber {
			continue
		}
		if crit.ToBlock != nil && crit.ToBlock.Int64() >= 0 && crit.ToBlock.Uint64() < log.BlockNumber {
			continue
		}

		if len(crit.Addresses) > 0 && !includes(crit.Addresses, log.Address) {
			continue
		}
		// If the to filtered topics is greater than the amount of topics in logs, skip.
		if len(crit.Topics) > len(log.Topics) {
			continue Logs
		}
		for i, sub := range crit.Topics {
			match := len(sub) == 0 // empty rule set == wildcard
			for _, topic := range sub {
				if log.Topics[i] == topic {
					match = true
					break
				}
			}
			if !match {
				continue Logs
			}
		}
		ret = append(ret, log)
	}
	return
}
