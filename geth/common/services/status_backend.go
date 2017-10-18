package services

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/les/status"
)

// StatusBackend interface for Ethereum Backend service.
type StatusBackend interface {
	SetAccountsFilterHandler(fn status.AccountsFilterHandler)
	AccountManager() *status.AccountManager
	SendTransaction(ctx context.Context, args status.SendTxArgs, passphrase string) (common.Hash, error)
	EstimateGas(ctx context.Context, args status.SendTxArgs) (*hexutil.Big, error)
}
