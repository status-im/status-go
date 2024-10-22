package rpcfilters

import (
	"context"
	"math/big"
	"time"

	"go.uber.org/zap"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	getRpc "github.com/ethereum/go-ethereum/rpc"
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
)

// ContextCaller provides CallContext method as ethereums rpc.Client.
type ContextCaller interface {
	CallContext(ctx context.Context, result interface{}, chainID uint64, method string, args ...interface{}) error
}

func pollLogs(client ContextCaller, chainID uint64, f *logsFilter, timeout, period time.Duration) {
	defer gocommon.LogOnPanic()
	query := func() {
		ctx, cancel := context.WithTimeout(f.ctx, timeout)
		defer cancel()
		logs, err := getLogs(ctx, client, chainID, f.criteria())
		if err != nil {
			logutils.ZapLogger().Error("Error fetch logs", zap.Any("criteria", f.crit), zap.Error(err))
			return
		}
		if err := f.add(logs); err != nil {
			logutils.ZapLogger().Error("Error adding logs", zap.Any("logs", logs), zap.Error(err))
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
			logutils.ZapLogger().Debug("Filter was stopped", zap.String("ID", string(f.id)), zap.Any("crit", f.crit))
			return
		}
	}
}
func getLogs(ctx context.Context, client ContextCaller, chainID uint64, crit ethereum.FilterQuery) (rst []types.Log, err error) {
	return rst, client.CallContext(ctx, &rst, chainID, "eth_getLogs", toFilterArg(crit))
}

func toFilterArg(q ethereum.FilterQuery) interface{} {
	arg := map[string]interface{}{
		"fromBlock": toBlockNumArg(q.FromBlock),
		"toBlock":   toBlockNumArg(q.ToBlock),
		"address":   q.Addresses,
		"topics":    q.Topics,
	}
	if q.FromBlock == nil {
		arg["fromBlock"] = "0x0"
	}
	return arg
}

func toBlockNumArg(number *big.Int) string {
	if number == nil || number.Int64() == getRpc.LatestBlockNumber.Int64() {
		return "latest"
	} else if number.Int64() == getRpc.PendingBlockNumber.Int64() {
		return "pending"
	}
	return hexutil.EncodeBig(number)
}
