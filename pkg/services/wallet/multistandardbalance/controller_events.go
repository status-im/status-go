package multistandardbalance

import (
	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"

	"github.com/status-im/status-go/pkg/pubsub"
)

func (c *Controller) sendEventBalanceFetchStarted(chainID uint64, config multistandardfetcher.FetchConfig) {
	pubsub.Publish(c.publisher, EventBalanceFetchStarted{
		ChainID: chainID,
		Config:  config,
	})
}

func (c *Controller) sendEventBalanceFetchFailedToStart(chainID uint64, config multistandardfetcher.FetchConfig, err error) {
	pubsub.Publish(c.publisher, EventBalanceFetchFailedToStart{
		ChainID: chainID,
		Config:  config,
		Error:   err,
	})
}

func (c *Controller) sendEventBalanceFetchFinished(key BalancesKey, resultType multistandardfetcher.ResultType, balanceChanged bool, oldState State, newState State) {
	pubsub.Publish(c.publisher, EventBalanceFetchFinished{
		Key:            key,
		ResultType:     resultType,
		BalanceChanged: balanceChanged,
		OldState:       oldState,
		NewState:       newState,
	})
}

func (c *Controller) sendEventBalanceFetchError(key BalancesKey, resultType multistandardfetcher.ResultType, err error) {
	pubsub.Publish(c.publisher, EventBalanceFetchError{
		Key:        key,
		ResultType: resultType,
		Error:      err,
	})
}
