package multistandardbalance

import (
	"context"
	"time"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/logutils"
)

func (c *Controller) handleERC20Result(ctx context.Context, chainID uint64, result multistandardfetcher.ERC20Result) {
	key := BalancesKey{Account: result.Account, ChainID: chainID}
	resultType := multistandardfetcher.ResultTypeERC20

	if result.Err != nil {
		c.logger.Error("failed to get ERC20 balance", zap.String("address", logutils.TruncateWithDot(key.Account.String())), zap.Uint64("chainID", key.ChainID), zap.Error(result.Err))
		c.sendEventBalanceFetchError(key, resultType, result.Err)
		return
	}

	c.lastBlockManager.SetLatestBlockNumber(chainID, result.AtBlockNumber.Uint64())

	state := State{
		AtBlockNumber: result.AtBlockNumber,
		AtBlockHash:   result.AtBlockHash,
		FetchedAt:     time.Now().Unix(),
	}

	balances := result.Results
	balanceChanged, oldState, err := c.storage.UpdateERC20Balances(ctx, key, balances, state)
	if err != nil {
		c.logger.Error("failed to update ERC20 balance", zap.String("address", logutils.TruncateWithDot(key.Account.String())), zap.Uint64("chainID", key.ChainID), zap.Error(err))
		c.sendEventBalanceFetchError(key, resultType, err)
		return
	}
	c.logger.Debug("finished updating ERC20 balance", zap.String("address", logutils.TruncateWithDot(key.Account.String())), zap.Uint64("chainID", key.ChainID), zap.Bool("balanceChanged", balanceChanged))
	c.sendEventBalanceFetchFinished(key, resultType, balanceChanged, oldState, state)
}
