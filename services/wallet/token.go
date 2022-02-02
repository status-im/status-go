package wallet

import (
	"context"
	"database/sql"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/contracts/ierc20"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/chain"
)

var requestTimeout = 20 * time.Second

type Token struct {
	Address common.Address `json:"address"`
	Name    string         `json:"name"`
	Symbol  string         `json:"symbol"`
	Color   string         `json:"color"`
	// Decimals defines how divisible the token is. For example, 0 would be
	// indivisible, whereas 18 would allow very small amounts of the token
	// to be traded.
	Decimals uint   `json:"decimals"`
	ChainID  uint64 `json:"chainId"`
}

type TokenManager struct {
	db *sql.DB
}

func (tm *TokenManager) getTokens(chainID uint64) ([]*Token, error) {
	tokensMap, ok := tokenStore[chainID]
	if !ok {
		return nil, errors.New("no tokens for this network")
	}

	res := make([]*Token, 0, len(tokensMap))

	for _, token := range tokensMap {
		res = append(res, token)
	}

	return res, nil
}

func (tm *TokenManager) getCustoms() ([]*Token, error) {
	rows, err := tm.db.Query("SELECT address, name, symbol, decimals, color, network_id FROM tokens")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rst []*Token
	for rows.Next() {
		token := &Token{}
		err := rows.Scan(&token.Address, &token.Name, &token.Symbol, &token.Decimals, &token.Color, &token.ChainID)
		if err != nil {
			return nil, err
		}

		rst = append(rst, token)
	}

	return rst, nil
}

func (tm *TokenManager) upsertCustom(token Token) error {
	insert, err := tm.db.Prepare("INSERT OR REPLACE INTO TOKENS (network_id, address, name, symbol, decimals, color) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	_, err = insert.Exec(token.ChainID, token.Address, token.Name, token.Symbol, token.Decimals, token.Color)
	return err
}

func (tm *TokenManager) deleteCustom(chainID uint64, address common.Address) error {
	_, err := tm.db.Exec(`DELETE FROM TOKENS WHERE address = ? and network_id = ?`, address, chainID)
	return err
}

func (tm *TokenManager) getBalances(parent context.Context, clients []*chain.Client, accounts, tokens []common.Address) (map[common.Address]map[common.Address]*hexutil.Big, error) {
	var (
		group    = async.NewAtomicGroup(parent)
		mu       sync.Mutex
		response = map[common.Address]map[common.Address]*hexutil.Big{}
	)
	for _, client := range clients {
		for tokenIdx := range tokens {
			caller, err := ierc20.NewIERC20Caller(tokens[tokenIdx], client)
			if err != nil {
				return nil, err
			}
			for accountIdx := range accounts {
				// Below, we set account and token from idx on purpose to avoid override
				account := accounts[accountIdx]
				token := tokens[tokenIdx]
				group.Add(func(parent context.Context) error {
					ctx, cancel := context.WithTimeout(parent, requestTimeout)
					defer cancel()
					balance, err := caller.BalanceOf(&bind.CallOpts{
						Context: ctx,
					}, account)
					// We don't want to return an error here and prevent
					// the rest from completing
					if err != nil {
						log.Error("can't fetch erc20 token balance", "account", account, "token", token, "error", err)

						return nil
					}
					mu.Lock()
					if _, ok := response[account]; !ok {
						response[account] = map[common.Address]*hexutil.Big{}
					}

					if _, ok := response[account][token]; !ok {
						zeroHex := hexutil.Big(*big.NewInt(0))
						response[account][token] = &zeroHex
					}
					sum := big.NewInt(0).Add(response[account][token].ToInt(), balance)
					sumHex := hexutil.Big(*sum)
					response[account][token] = &sumHex
					mu.Unlock()
					return nil
				})
			}
		}
	}
	select {
	case <-group.WaitAsync():
	case <-parent.Done():
		return nil, parent.Err()
	}
	return response, group.Error()
}
