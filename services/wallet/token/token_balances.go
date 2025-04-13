package token

//go:generate mockgen -source=token.go -destination=mock/token/tokenmanager.go

import (
	"database/sql"
	"fmt"
	"math/big"
	"strings"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/services/accounts/accountsevent"
	"github.com/status-im/status-go/services/wallet/bigint"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
)

func (tm *Manager) GetTokenHistoricalBalance(account common.Address, chainID uint64, tokenAddress common.Address, timestamp int64) (*big.Int, error) {
	var balance big.Int
	err := tm.db.QueryRow("SELECT balance FROM balance_history WHERE token_address = ? AND chain_id = ? AND address = ? AND timestamp < ? order by timestamp DESC LIMIT 1", tokenAddress, chainID, account, timestamp).Scan((*bigint.SQLBigIntBytes)(&balance))
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &balance, nil
}

func (tm *Manager) GetPreviouslyOwnedTokens() (map[common.Address][]tokenTypes.Token, error) {
	storageTokens, err := tm.tokenBalancesStorage.GetTokens()
	if err != nil {
		return nil, err
	}

	tokens := make(map[common.Address][]tokenTypes.Token)
	for account, storageToken := range storageTokens {
		for _, token := range storageToken {
			tokens[account] = append(tokens[account], token.Token)
		}
	}

	return tokens, nil
}

func (tm *Manager) removeTokenBalances(account common.Address) error {
	_, err := tm.db.Exec("DELETE FROM token_balances WHERE user_address = ?", account.String())
	return err
}

func (tm *Manager) onAccountsChange(changedAddresses []common.Address, eventType accountsevent.EventType, currentAddresses []common.Address) {
	if eventType == accountsevent.EventTypeRemoved {
		for _, account := range changedAddresses {
			err := tm.removeTokenBalances(account)
			if err != nil {
				logutils.ZapLogger().Error("token.Manager: can't remove token balances", zap.Error(err))
			}
		}
	}
}

func (tm *Manager) GetCachedBalancesByChain(accounts, tokenAddresses []common.Address, chainIDs []uint64) (map[uint64]map[common.Address]map[common.Address]*hexutil.Big, error) {
	accountStrings := make([]string, len(accounts))
	for i, account := range accounts {
		accountStrings[i] = fmt.Sprintf("'%s'", account.Hex())
	}

	tokenAddressStrings := make([]string, len(tokenAddresses))
	for i, tokenAddress := range tokenAddresses {
		tokenAddressStrings[i] = fmt.Sprintf("'%s'", tokenAddress.Hex())
	}

	chainIDStrings := make([]string, len(chainIDs))
	for i, chainID := range chainIDs {
		chainIDStrings[i] = fmt.Sprintf("%d", chainID)
	}

	//nolint: gosec
	query := `SELECT chain_id, user_address, token_address, raw_balance
			  	FROM token_balances
				WHERE user_address IN (` + strings.Join(accountStrings, ",") + `)
					AND token_address IN (` + strings.Join(tokenAddressStrings, ",") + `)
					AND chain_id IN (` + strings.Join(chainIDStrings, ",") + `)`

	rows, err := tm.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ret := make(map[uint64]map[common.Address]map[common.Address]*hexutil.Big)

	for rows.Next() {
		var chainID uint64
		var userAddressStr, tokenAddressStr string
		var rawBalance string

		err := rows.Scan(&chainID, &userAddressStr, &tokenAddressStr, &rawBalance)
		if err != nil {
			return nil, err
		}

		num := new(hexutil.Big)
		_, ok := num.ToInt().SetString(rawBalance, 10)
		if !ok {
			return ret, nil
		}

		if ret[chainID] == nil {
			ret[chainID] = make(map[common.Address]map[common.Address]*hexutil.Big)
		}

		if ret[chainID][common.HexToAddress(userAddressStr)] == nil {
			ret[chainID][common.HexToAddress(userAddressStr)] = make(map[common.Address]*hexutil.Big)
		}

		ret[chainID][common.HexToAddress(userAddressStr)][common.HexToAddress(tokenAddressStr)] = num
	}

	return ret, nil
}
