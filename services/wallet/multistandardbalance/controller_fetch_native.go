package multistandardbalance

import (
	"context"
	"time"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"

	gocommon "github.com/status-im/status-go/common"

	"go.uber.org/zap"
)

func (c *Controller) handleNativeResult(ctx context.Context, chainID uint64, result multistandardfetcher.NativeResult) {
	key := BalancesKey{Account: result.Account, ChainID: chainID}
	resultType := multistandardfetcher.ResultTypeNative

	if result.Err != nil {
		c.logger.Error("failed to get native balance", zap.String("address", gocommon.TruncateWithDot(key.Account.String())), zap.Uint64("chainID", key.ChainID), zap.Error(result.Err))
		c.sendEventBalanceFetchError(key, resultType, result.Err)
		return
	}

	c.lastBlockManager.SetLatestBlockNumber(chainID, result.AtBlockNumber.Uint64())

	state := State{
		AtBlockNumber: result.AtBlockNumber,
		AtBlockHash:   result.AtBlockHash,
		FetchedAt:     time.Now().Unix(),
	}

	balance := result.Result
	balanceChanged, oldState, err := c.storage.UpdateNativeBalance(ctx, key, balance, state)
	if err != nil {
		c.logger.Error("failed to update native balance", zap.String("address", gocommon.TruncateWithDot(key.Account.String())), zap.Uint64("chainID", key.ChainID), zap.Error(err))
		c.sendEventBalanceFetchError(key, resultType, err)
		return
	}
	c.logger.Debug("finished updating native balance", zap.String("address", gocommon.TruncateWithDot(key.Account.String())), zap.Uint64("chainID", key.ChainID), zap.Bool("balanceChanged", balanceChanged))
	c.sendEventBalanceFetchFinished(key, resultType, balanceChanged, oldState, state)
}
