package multistandardbalance

import (
	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"

	"github.com/status-im/status-go/pkg/services/wallet/walletevent"
)

// EventRequestedBalanceRefreshFinished is sent on the wallet feed once every
// fetch of a refresh the user asked for has ended, whether it succeeded or not.
const EventRequestedBalanceRefreshFinished walletevent.EventType = "wallet-requested-balance-refresh-finished"

type EventBalanceFetchStarted struct {
	ChainID uint64
	Config  multistandardfetcher.FetchConfig
}

type EventBalanceFetchFailedToStart struct {
	ChainID uint64
	Config  multistandardfetcher.FetchConfig
	Error   error
}

type EventBalanceFetchFinished struct {
	Key            BalancesKey
	ResultType     multistandardfetcher.ResultType
	BalanceChanged bool
	OldState       State
	NewState       State
}

type EventBalanceFetchError struct {
	Key        BalancesKey
	ResultType multistandardfetcher.ResultType
	Error      error
}
