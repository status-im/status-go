package multistandardbalance

import (
	"context"
	"slices"
	"time"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/pkg/pubsub"

	"go.uber.org/zap"
)

func (c *Controller) buildFetchConfigs(balancesToFetch map[BalancesKey][]multistandardfetcher.ResultType) map[uint64]multistandardfetcher.FetchConfig {
	fetchConfigs := make(map[uint64]multistandardfetcher.FetchConfig)

	for balancesKey, resultTypes := range balancesToFetch {
		if _, exists := fetchConfigs[balancesKey.ChainID]; !exists {
			fetchConfigs[balancesKey.ChainID] = multistandardfetcher.FetchConfig{
				Native:  make([]AccountAddress, 0),
				ERC20:   make(map[AccountAddress][]ContractAddress),
				ERC721:  make(map[AccountAddress][]ContractAddress),
				ERC1155: make(map[AccountAddress][]CollectibleID),
			}
		}

		hasNative := slices.Contains(resultTypes, multistandardfetcher.ResultTypeNative)
		hasERC20 := slices.Contains(resultTypes, multistandardfetcher.ResultTypeERC20)
		hasERC721 := slices.Contains(resultTypes, multistandardfetcher.ResultTypeERC721)
		hasERC1155 := slices.Contains(resultTypes, multistandardfetcher.ResultTypeERC1155)

		if hasNative {
			fetchConfig := fetchConfigs[balancesKey.ChainID]
			fetchConfig.Native = append(fetchConfig.Native, balancesKey.Account)
			fetchConfigs[balancesKey.ChainID] = fetchConfig
		}
		if hasERC20 {
			fetchConfig := fetchConfigs[balancesKey.ChainID]
			tokens, err := c.tokenListProvider.GetTokenContractAddresses(balancesKey.ChainID)
			if err != nil {
				c.logger.Error("failed to get token contract addresses", zap.Error(err))
				continue
			}
			fetchConfig.ERC20[balancesKey.Account] = tokens
			fetchConfigs[balancesKey.ChainID] = fetchConfig
		}
		if hasERC721 || hasERC1155 {
			fetchConfig := fetchConfigs[balancesKey.ChainID]
			erc721List, erc1155List, err := c.collectibleListProvider.GetCollectiblesList(balancesKey.ChainID, balancesKey.Account)
			if err != nil {
				c.logger.Error("failed to get owned collectibles", zap.Error(err))
				continue
			}
			if hasERC721 {
				for _, erc721ID := range erc721List {
					fetchConfig.ERC721[balancesKey.Account] = append(fetchConfig.ERC721[balancesKey.Account], erc721ID.ContractAddress)
				}
			}
			if hasERC1155 {
				for _, erc1155ID := range erc1155List {
					fetchConfig.ERC1155[balancesKey.Account] = append(fetchConfig.ERC1155[balancesKey.Account], erc1155ID)
				}
			}
			fetchConfigs[balancesKey.ChainID] = fetchConfig
		}
	}

	return fetchConfigs
}

func (c *Controller) handleFetchResult(ctx context.Context, chainID uint64, fetchResult multistandardfetcher.FetchResult) {
	c.logger.Debug("processing result", zap.Uint64("chainID", chainID), zap.String("fetchResult.ResultType", string(fetchResult.ResultType)))
	switch fetchResult.ResultType {
	case multistandardfetcher.ResultTypeNative:
		result, ok := fetchResult.Result.(multistandardfetcher.NativeResult)
		if !ok {
			c.logger.Error("failed to parse native result", zap.Any("result", fetchResult.Result))
			return
		}
		c.handleNativeResult(ctx, chainID, result)
	case multistandardfetcher.ResultTypeERC20:
		result, ok := fetchResult.Result.(multistandardfetcher.ERC20Result)
		if !ok {
			c.logger.Error("failed to parse ERC20 result", zap.Any("result", fetchResult.Result))
			return
		}
		c.handleERC20Result(ctx, chainID, result)
	case multistandardfetcher.ResultTypeERC721:
		result, ok := fetchResult.Result.(multistandardfetcher.ERC721Result)
		if !ok {
			c.logger.Error("failed to parse ERC721 result", zap.Any("result", fetchResult.Result))
			return
		}
		c.handleERC721Result(ctx, chainID, result)
	case multistandardfetcher.ResultTypeERC1155:
		result, ok := fetchResult.Result.(multistandardfetcher.ERC1155Result)
		if !ok {
			c.logger.Error("failed to parse ERC1155 result", zap.Any("result", fetchResult.Result))
			return
		}
		c.handleERC1155Result(ctx, chainID, result)
	}
}

func (c *Controller) handleNativeResult(ctx context.Context, chainID uint64, result multistandardfetcher.NativeResult) {
	key := BalancesKey{Account: result.Account, ChainID: chainID}
	resultType := multistandardfetcher.ResultTypeNative

	if result.Err != nil {
		c.logger.Error("failed to get native balance", zap.String("address", gocommon.TruncateWithDot(key.Account.String())), zap.Uint64("chainID", key.ChainID), zap.Error(result.Err))
		pubsub.Publish(c.publisher, EventBalanceFetchError{
			Key:        key,
			ResultType: resultType,
			Error:      result.Err,
		})
		return
	}

	c.lastBlockManager.SetLatestBlockNumber(chainID, result.AtBlockNumber.Uint64())

	balance := result.Result
	state := State{
		AtBlockNumber: result.AtBlockNumber,
		AtBlockHash:   result.AtBlockHash,
		FetchedAt:     time.Now().Unix(),
	}

	balanceChanged, oldState, err := c.storage.UpdateNativeBalance(ctx, key, balance, state)
	if err != nil {
		c.logger.Error("failed to update native balance", zap.String("address", gocommon.TruncateWithDot(key.Account.String())), zap.Uint64("chainID", key.ChainID), zap.Error(err))
		pubsub.Publish(c.publisher, EventBalanceFetchError{
			Key:        key,
			ResultType: resultType,
			Error:      err,
		})
		return
	}
	c.logger.Debug("finished updating native balance", zap.String("address", gocommon.TruncateWithDot(key.Account.String())), zap.Uint64("chainID", key.ChainID), zap.Bool("balanceChanged", balanceChanged))
	pubsub.Publish(c.publisher, EventBalanceFetchFinished{
		Key:            key,
		ResultType:     resultType,
		BalanceChanged: balanceChanged,
		OldState:       oldState,
		NewState:       state,
	})
}

func (c *Controller) handleERC20Result(ctx context.Context, chainID uint64, result multistandardfetcher.ERC20Result) {
	key := BalancesKey{Account: result.Account, ChainID: chainID}
	resultType := multistandardfetcher.ResultTypeERC20

	if result.Err != nil {
		c.logger.Error("failed to get ERC20 balance", zap.String("address", gocommon.TruncateWithDot(key.Account.String())), zap.Uint64("chainID", key.ChainID), zap.Error(result.Err))
		pubsub.Publish(c.publisher, EventBalanceFetchError{
			Key:        key,
			ResultType: resultType,
			Error:      result.Err,
		})
		return
	}

	c.lastBlockManager.SetLatestBlockNumber(chainID, result.AtBlockNumber.Uint64())

	balances := result.Results
	state := State{
		AtBlockNumber: result.AtBlockNumber,
		AtBlockHash:   result.AtBlockHash,
		FetchedAt:     time.Now().Unix(),
	}

	balanceChanged, oldState, err := c.storage.UpdateERC20Balances(ctx, key, balances, state)
	if err != nil {
		c.logger.Error("failed to update ERC20 balance", zap.String("address", gocommon.TruncateWithDot(key.Account.String())), zap.Uint64("chainID", key.ChainID), zap.Error(err))
		return
	}
	c.logger.Debug("finished updating ERC20 balance", zap.String("address", gocommon.TruncateWithDot(key.Account.String())), zap.Uint64("chainID", key.ChainID), zap.Bool("balanceChanged", balanceChanged))
	pubsub.Publish(c.publisher, EventBalanceFetchFinished{
		Key:            key,
		ResultType:     resultType,
		BalanceChanged: balanceChanged,
		OldState:       oldState,
		NewState:       state,
	})
}

func (c *Controller) handleERC721Result(ctx context.Context, chainID uint64, result multistandardfetcher.ERC721Result) {
	key := BalancesKey{Account: result.Account, ChainID: chainID}
	resultType := multistandardfetcher.ResultTypeERC721

	if result.Err != nil {
		c.logger.Error("failed to get ERC721 balance", zap.String("address", gocommon.TruncateWithDot(key.Account.String())), zap.Uint64("chainID", key.ChainID), zap.Error(result.Err))
		pubsub.Publish(c.publisher, EventBalanceFetchError{
			Key:        key,
			ResultType: resultType,
			Error:      result.Err,
		})
		return
	}

	c.lastBlockManager.SetLatestBlockNumber(chainID, result.AtBlockNumber.Uint64())

	balances := result.Results
	state := State{
		AtBlockNumber: result.AtBlockNumber,
		AtBlockHash:   result.AtBlockHash,
		FetchedAt:     time.Now().Unix(),
	}

	balanceChanged, oldState, err := c.storage.UpdateERC721Balances(ctx, key, balances, state)
	if err != nil {
		c.logger.Error("failed to update ERC721 balance", zap.String("address", gocommon.TruncateWithDot(key.Account.String())), zap.Uint64("chainID", key.ChainID), zap.Error(err))
		return
	}
	c.logger.Debug("finished updating ERC721 balance", zap.String("address", gocommon.TruncateWithDot(key.Account.String())), zap.Uint64("chainID", key.ChainID), zap.Bool("balanceChanged", balanceChanged))
	pubsub.Publish(c.publisher, EventBalanceFetchFinished{
		Key:            key,
		ResultType:     resultType,
		BalanceChanged: balanceChanged,
		OldState:       oldState,
		NewState:       state,
	})
}

func (c *Controller) handleERC1155Result(ctx context.Context, chainID uint64, result multistandardfetcher.ERC1155Result) {
	key := BalancesKey{Account: result.Account, ChainID: chainID}
	resultType := multistandardfetcher.ResultTypeERC1155

	if result.Err != nil {
		c.logger.Error("failed to get ERC1155 balance", zap.String("address", gocommon.TruncateWithDot(key.Account.String())), zap.Uint64("chainID", key.ChainID), zap.Error(result.Err))
		pubsub.Publish(c.publisher, EventBalanceFetchError{
			Key:        key,
			ResultType: resultType,
			Error:      result.Err,
		})
		return
	}

	c.lastBlockManager.SetLatestBlockNumber(chainID, result.AtBlockNumber.Uint64())

	balances := result.Results
	state := State{
		AtBlockNumber: result.AtBlockNumber,
		AtBlockHash:   result.AtBlockHash,
		FetchedAt:     time.Now().Unix(),
	}

	balanceChanged, oldState, err := c.storage.UpdateERC1155Balances(ctx, key, balances, state)
	if err != nil {
		c.logger.Error("failed to update ERC1155 balance", zap.String("address", gocommon.TruncateWithDot(key.Account.String())), zap.Uint64("chainID", key.ChainID), zap.Error(err))
		pubsub.Publish(c.publisher, EventBalanceFetchError{
			Key:        key,
			ResultType: resultType,
			Error:      err,
		})
		return
	}
	c.logger.Debug("finished updating ERC1155 balance", zap.String("address", gocommon.TruncateWithDot(key.Account.String())), zap.Uint64("chainID", key.ChainID), zap.Bool("balanceChanged", balanceChanged))
	pubsub.Publish(c.publisher, EventBalanceFetchFinished{
		Key:            key,
		ResultType:     resultType,
		BalanceChanged: balanceChanged,
		OldState:       oldState,
		NewState:       state,
	})
}
