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
	query := func() {
		ctx, cancel := context.WithTimeout(f.ctx, timeout)
		defer cancel()
		logs, err := getLogs(ctx, client, f.criteria())
		if err != nil {
			log.Error("Error fetch logs", "criteria", f.crit, "error", err)
			return
		}
		if err := f.add(logs); err != nil {
			log.Error("Error adding logs", "logs", logs, "error", err)
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
			log.Debug("Filter was stopped", "ID", f.id, "crit", f.crit)
			return
		}
	}
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
