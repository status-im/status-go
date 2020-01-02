package wallet

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/status-im/status-go/services/wallet/ierc20"
)

// GetTokensBalances takes list of accounts and tokens and returns mapping of token balances for each account.
func GetTokensBalances(parent context.Context, client *ethclient.Client, accounts, tokens []common.Address) (map[common.Address]map[common.Address]*big.Int, error) {
	var (
		group    = NewAtomicGroup(parent)
		mu       sync.Mutex
		response = map[common.Address]map[common.Address]*big.Int{}
	)
	// requested current head to request balance on the same block number
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	header, err := client.HeaderByNumber(ctx, nil)
	cancel()
	if err != nil {
		return nil, err
	}
	for _, token := range tokens {
		caller, err := ierc20.NewIERC20Caller(token, client)
		token := token
		if err != nil {
			return nil, err
		}
		for _, account := range accounts {
			account := account
			group.Add(func(parent context.Context) error {
				ctx, cancel := context.WithTimeout(parent, 3*time.Second)
				balance, err := caller.BalanceOf(&bind.CallOpts{
					BlockNumber: header.Number,
					Context:     ctx,
				}, account)
				cancel()
				if err != nil {
					return err
				}
				mu.Lock()
				_, exist := response[account]
				if !exist {
					response[account] = map[common.Address]*big.Int{}
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
