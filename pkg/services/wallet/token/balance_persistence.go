package token

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

type balanceStorage struct {
	walletDB *sql.DB
}

func (p *balanceStorage) saveBalances(tokens map[common.Address][]tokentypes.StorageToken) (err error) {
	tx, err := p.walletDB.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	for address, addressTokens := range tokens {
		for _, t := range addressTokens {
			if t.HasError {
				continue
			}
			_, err = tx.Exec(`
			INSERT INTO token_balances(
				user_address,
				token_address,
				token_description,
				token_url,
				balance,
				raw_balance,
				chain_id)
			VALUES
				(?,?,?,?,?,?,?)`,
				address.Hex(),
				t.TokenAddress.Hex(),
				t.Description,
				t.AssetWebsiteURL,
				t.Balance.String(),
				t.RawBalance,
				t.TokenChainID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *balanceStorage) removeTokenBalances(account common.Address) error {
	_, err := p.walletDB.Exec("DELETE FROM token_balances WHERE user_address = ?", account.String())
	return err
}

func (p *balanceStorage) getBalances() (map[common.Address][]tokentypes.StorageToken, error) {
	rows, err := p.walletDB.Query(`
	SELECT
		user_address,
		token_address,
		token_description,
		token_url,
		balance,
		raw_balance,
		chain_id
	FROM
		token_balances`)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	result := make(map[common.Address][]tokentypes.StorageToken)

	for rows.Next() {
		var userAddressStr, tokenAddressStr, balanceStr string
		token := tokentypes.StorageToken{}

		err := rows.Scan(&userAddressStr, &tokenAddressStr, &token.Description, &token.AssetWebsiteURL, &balanceStr, &token.RawBalance, &token.TokenChainID)
		if err != nil {
			return nil, err
		}

		userAddress := common.HexToAddress(userAddressStr)
		token.TokenAddress = common.HexToAddress(tokenAddressStr)

		balanceFloat := new(big.Float)
		_, _, err = balanceFloat.Parse(balanceStr, 10)
		if err != nil {
			return nil, err
		}

		token.Balance = balanceFloat

		result[userAddress] = append(result[userAddress], token)
	}

	return result, nil
}

func (p *balanceStorage) getCachedBalancesByChain(accounts []common.Address, tokens []*tokentypes.Token) (map[uint64]map[common.Address]map[common.Address]*hexutil.Big, error) {
	ret := make(map[uint64]map[common.Address]map[common.Address]*hexutil.Big)

	if len(accounts) == 0 || len(tokens) == 0 {
		return ret, nil
	}

	accountStrings := make([]string, len(accounts))
	for i, account := range accounts {
		accountStrings[i] = fmt.Sprintf("'%s'", account.Hex())
	}

	tokenAddressStrings := make([]string, len(tokens))
	chainIDStrings := make([]string, len(tokens))
	for i, token := range tokens {
		tokenAddressStrings[i] = fmt.Sprintf("'%s'", token.Address.Hex())
		chainIDStrings[i] = fmt.Sprintf("%d", token.ChainID)
	}

	//nolint: gosec
	query := `
	SELECT
		chain_id,
		user_address,
		token_address,
		raw_balance
	FROM
		token_balances
	WHERE
		user_address IN (` + strings.Join(accountStrings, ",") + `)
		AND token_address IN (` + strings.Join(tokenAddressStrings, ",") + `)
		AND chain_id IN (` + strings.Join(chainIDStrings, ",") + `)`

	rows, err := p.walletDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

func (p *balanceStorage) getUsedTokensKeys(testnetMode bool) (map[string]interface{}, error) {
	query := `SELECT token_address, chain_id FROM token_balances`
	rows, err := p.walletDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokenKeys := make(map[string]interface{}, 0)

	for rows.Next() {
		var tokenAddressStr string
		var chainID uint64

		err := rows.Scan(&tokenAddressStr, &chainID)
		if err != nil {
			return nil, err
		}

		if testnetMode && walletcommon.ChainID(chainID).IsMainnet() ||
			!testnetMode && !walletcommon.ChainID(chainID).IsMainnet() {
			continue
		}

		tokenKey := types.TokenKey(chainID, common.HexToAddress(tokenAddressStr))

		if _, ok := tokenKeys[tokenKey]; !ok {
			tokenKeys[tokenKey] = nil
		}
	}

	return tokenKeys, nil
}
