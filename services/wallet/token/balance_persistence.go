package token

import (
	"context"
	"database/sql"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type TokenMarketValues struct {
	MarketCap       float64 `json:"marketCap"`
	HighDay         float64 `json:"highDay"`
	LowDay          float64 `json:"lowDay"`
	ChangePctHour   float64 `json:"changePctHour"`
	ChangePctDay    float64 `json:"changePctDay"`
	ChangePct24hour float64 `json:"changePct24hour"`
	Change24hour    float64 `json:"change24hour"`
	Price           float64 `json:"price"`
	HasError        bool    `json:"hasError"`
}

type StorageToken struct {
	Token
	BalancesPerChain        map[uint64]ChainBalance      `json:"balancesPerChain"`
	Description             string                       `json:"description"`
	AssetWebsiteURL         string                       `json:"assetWebsiteUrl"`
	BuiltOn                 string                       `json:"builtOn"`
	MarketValuesPerCurrency map[string]TokenMarketValues `json:"marketValuesPerCurrency"`
}

type ChainBalance struct {
	RawBalance     string         `json:"rawBalance"`
	Balance        *big.Float     `json:"balance"`
	Balance1DayAgo string         `json:"balance1DayAgo"`
	Address        common.Address `json:"address"`
	ChainID        uint64         `json:"chainId"`
	HasError       bool           `json:"hasError"`
}

type TokenBalancesStorage interface {
	SaveTokens(tokens map[common.Address][]StorageToken) error
	GetTokens() (map[common.Address][]StorageToken, error)
}

type Persistence struct {
	db *sql.DB
}

func NewPersistence(db *sql.DB) *Persistence {
	return &Persistence{db: db}
}

func (p *Persistence) SaveTokens(tokens map[common.Address][]StorageToken) (err error) {
	tx, err := p.db.BeginTx(context.Background(), &sql.TxOptions{})
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
			for chainID, b := range t.BalancesPerChain {
				if b.HasError {
					continue
				}
				_, err = tx.Exec(`INSERT INTO token_balances(user_address,token_name,token_symbol,token_address,token_decimals,token_description,token_url,balance,raw_balance,chain_id) VALUES (?,?,?,?,?,?,?,?,?,?)`, address.Hex(), t.Name, t.Symbol, b.Address.Hex(), t.Decimals, t.Description, t.AssetWebsiteURL, b.Balance.String(), b.RawBalance, chainID)
				if err != nil {
					return err
				}
			}

		}
	}

	return nil
}

func (p *Persistence) GetTokens() (map[common.Address][]StorageToken, error) {
	rows, err := p.db.Query(`SELECT user_address, token_name, token_symbol, token_address, token_decimals, token_description, token_url, balance, raw_balance, chain_id FROM token_balances `)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	acc := make(map[common.Address]map[string]StorageToken)

	for rows.Next() {
		var addressStr, balance, rawBalance, tokenAddress string
		token := StorageToken{}
		var chainID uint64

		err := rows.Scan(&addressStr, &token.Name, &token.Symbol, &tokenAddress, &token.Decimals, &token.Description, &token.AssetWebsiteURL, &balance, &rawBalance, &chainID)
		if err != nil {
			return nil, err
		}

		token.Address = common.HexToAddress(tokenAddress)
		token.ChainID = chainID
		address := common.HexToAddress(addressStr)

		if acc[address] == nil {
			acc[address] = make(map[string]StorageToken)
		}

		if acc[address][token.Name].Name == "" {
			token.BalancesPerChain = make(map[uint64]ChainBalance)
			acc[address][token.Name] = token
		}

		tokenAcc := acc[address][token.Name]

		balanceFloat := new(big.Float)
		_, _, err = balanceFloat.Parse(balance, 10)
		if err != nil {
			return nil, err
		}

		tokenAcc.BalancesPerChain[chainID] = ChainBalance{
			RawBalance: rawBalance,
			Balance:    balanceFloat,
			Address:    common.HexToAddress(tokenAddress),
			ChainID:    chainID,
		}
	}

	result := make(map[common.Address][]StorageToken)

	for address, tks := range acc {
		for _, t := range tks {
			result[address] = append(result[address], t)
		}
	}
	return result, nil
}
