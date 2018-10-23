package rpcfilters

import (
	"context"
	"math/big"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// ContextCaller provides CallContext method as ethereums rpc.Client.
type ContextCaller interface {
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
}

func pollLogs(client ContextCaller, f *logsFilter, timeout, period time.Duration) {
	adjusted := false
	query := func() {
		ctx, cancel := context.WithTimeout(f.ctx, timeout)
		logs, err := getLogs(ctx, client, f.crit)
		cancel()
		if err != nil {
			log.Error("failed to get logs", "criteria", f.crit, "error", err)
		} else if !adjusted {
			adjustFromBlock(&f.crit)
			adjusted = true
		}
		if err := f.add(logs); err != nil {
			log.Error("error adding logs", "logs", logs, "error", err)
		}
	}
	query()
	latest := time.NewTicker(period)
	defer latest.Stop()
	for {
		select {
		case <-latest.C:
			query()
		case <-f.done:
			log.Debug("filter was stopped", "ID", f.id, "crit", f.crit)
			return
		}
	}
}

// adjustFromBlock adjusts crit.FromBlock to the latest to avoid querying same logs multiple times.
func adjustFromBlock(crit *ethereum.FilterQuery) {
	latest := big.NewInt(rpc.LatestBlockNumber.Int64())
	// don't adjust if filter is not interested in newer blocks
	if crit.ToBlock != nil && crit.ToBlock.Cmp(latest) == 1 {
		return
	}
	// don't adjust if from block is already pending
	if crit.FromBlock != nil && crit.FromBlock.Cmp(latest) == -1 {
		return
	}
	crit.FromBlock = latest
}

func getLogs(ctx context.Context, client ContextCaller, crit ethereum.FilterQuery) (rst []types.Log, err error) {
	return rst, client.CallContext(ctx, &rst, "eth_getLogs", toFilterArg(crit))
}

func toFilterArg(q ethereum.FilterQuery) interface{} {
	arg := map[string]interface{}{
		"fromBlock": toBlockNumArg(q.FromBlock),
		"toBlock":   toBlockNumArg(q.ToBlock),
		"address":   q.Addresses,
		"topics":    q.Topics,
	}
	return arg
}

func toBlockNumArg(number *big.Int) string {
	if number == nil || number.Int64() == rpc.LatestBlockNumber.Int64() {
		return "latest"
	} else if number.Int64() == rpc.PendingBlockNumber.Int64() {
		return "pending"
	}
	return hexutil.EncodeBig(number)
}
