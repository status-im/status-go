package multistandardbalance

import "github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"

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
