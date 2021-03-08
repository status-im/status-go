package wallet

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/services/wallet/ierc20"
)

var requestTimeout = 20 * time.Second

// GetTokensBalances takes list of accounts and tokens and returns mapping of token balances for each account.
func GetTokensBalances(parent context.Context, client *walletClient, accounts, tokens []common.Address) (map[common.Address]map[common.Address]*hexutil.Big, error) {
	var (
		group    = NewAtomicGroup(parent)
		mu       sync.Mutex
		response = map[common.Address]map[common.Address]*hexutil.Big{}
	)
	for _, token := range tokens {
		caller, err := ierc20.NewIERC20Caller(token, client)
		token := token
		if err != nil {
			return nil, err
		}
		for _, account := range accounts {
			// Why we are doing this?
			account := account
			group.Add(func(parent context.Context) error {
				ctx, cancel := context.WithTimeout(parent, requestTimeout)
				balance, err := caller.BalanceOf(&bind.CallOpts{
					Context: ctx,
				}, account)
				cancel()
				// We don't want to return an error here and prevent
				// the rest from completing
				if err != nil {
					log.Error("can't fetch erc20 token balance", "account", account, "token", token, "error", err)

					return nil
				}
				mu.Lock()
				_, exist := response[account]
				if !exist {
					response[account] = map[common.Address]*hexutil.Big{}
				}
				response[account][token] = balance
				mu.Unlock()
				return nil
			})
		}
	}
	select {
	case <-group.WaitAsync():
	case <-parent.Done():
		return nil, parent.Err()
	}
	return response, group.Error()
}
