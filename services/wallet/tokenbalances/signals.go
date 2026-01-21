package tokenbalances

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/signal"
)

const (
	SignalFetchStarted       = signal.SignalType("wallet.token-balances.fetch-started")
	SignalFetchFailedToStart = signal.SignalType("wallet.token-balances.fetch-failed-to-start")
	SignalFetchFinished      = signal.SignalType("wallet.token-balances.fetch-finished")
	SignalFetchError         = signal.SignalType("wallet.token-balances.fetch-error")
)

type SignalFetchStartedPayload struct {
	ChainID  uint64           `json:"chainId"`
	Accounts []common.Address `json:"accounts"`
}

type SignalFetchFailedToStartPayload struct {
	ChainID  uint64           `json:"chainId"`
	Accounts []common.Address `json:"accounts"`
	Error    error            `json:"error"`
}

type SignalFetchErrorPayload struct {
	ChainID uint64         `json:"chainId"`
	Account common.Address `json:"account"`
	Error   error          `json:"error"`
}

type SignalFetchFinishedPayload struct {
	ChainID        uint64         `json:"chainId"`
	Account        common.Address `json:"account"`
	BalanceChanged bool           `json:"balanceChanged"`
}
