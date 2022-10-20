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
	"github.com/status-im/status-go/contracts"
	"github.com/status-im/status-go/contracts/ierc20"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/chain"
)

var requestTimeout = 20 * time.Second
var nativeChainAddress = common.HexToAddress("0x")

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
	db             *sql.DB
	RPCClient      *rpc.Client
	networkManager *network.Manager
}

func NewTokenManager(
	db *sql.DB,
	RPCClient *rpc.Client,
	networkManager *network.Manager,
) *TokenManager {
	tokenManager := &TokenManager{db, RPCClient, networkManager}
	// Check the networks' custom tokens to see if we must update the tokenStore
	networks := networkManager.GetConfiguredNetworks()
	for _, network := range networks {
		if len(network.TokenOverrides) == 0 {
			continue
		}

		for _, overrideToken := range network.TokenOverrides {
			tokensMap, ok := tokenStore[network.ChainID]
			if !ok {
				continue
			}
			for _, token := range tokensMap {
				if token.Symbol == overrideToken.Symbol {
					token.Address = overrideToken.Address
				}
			}
		}
	}

	return tokenManager
}

func (tm *TokenManager) findSNT(chainID uint64) *Token {
	tokensMap, ok := tokenStore[chainID]
	if !ok {
		return nil
	}

	for _, token := range tokensMap {
		if token.Symbol == "SNT" || token.Symbol == "STT" {
			return token
		}
	}

	return nil
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

func (tm *TokenManager) discoverToken(ctx context.Context, chainID uint64, address common.Address) (*Token, error) {
	backend, err := tm.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	caller, err := ierc20.NewIERC20Caller(address, backend)
	if err != nil {
		return nil, err
	}

	name, err := caller.Name(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}

	symbol, err := caller.Symbol(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}

	decimal, err := caller.Decimals(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}

	return &Token{
		Address:  address,
		Name:     name,
		Symbol:   symbol,
		Decimals: uint(decimal),
	}, nil
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

func (tm *TokenManager) getCustomsByChainID(chainID uint64) ([]*Token, error) {
	rows, err := tm.db.Query("SELECT address, name, symbol, decimals, color, network_id FROM tokens where network_id=?", chainID)
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

func (tm *TokenManager) isTokenVisible(chainID uint64, address common.Address) (bool, error) {
	rows, err := tm.db.Query("SELECT chain_id, address FROM visible_tokens WHERE chain_id = ? AND address = ?", chainID, address)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	return rows.Next(), nil
}

func (tm *TokenManager) toggle(chainID uint64, address common.Address) error {
	isVisible, err := tm.isTokenVisible(chainID, address)
	if err != nil {
		return err
	}

	if isVisible {
		_, err = tm.db.Exec(`DELETE FROM visible_tokens WHERE address = ? and chain_id = ?`, address, chainID)
		return err
	}

	insert, err := tm.db.Prepare("INSERT OR REPLACE INTO visible_tokens (chain_id, address) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer insert.Close()

	_, err = insert.Exec(chainID, address)
	return err
}

func (tm *TokenManager) getVisible(chainIDs []uint64) (map[uint64][]*Token, error) {
	customTokens, err := tm.getCustoms()
	if err != nil {
		return nil, err
	}

	rst := make(map[uint64][]*Token)
	for _, chainID := range chainIDs {
		network := tm.networkManager.Find(chainID)
		if network == nil {
			continue
		}

		rst[chainID] = make([]*Token, 0)
		rst[chainID] = append(rst[chainID], &Token{
			Address:  common.HexToAddress("0x"),
			Name:     network.NativeCurrencyName,
			Symbol:   network.NativeCurrencySymbol,
			Decimals: uint(network.NativeCurrencyDecimals),
			ChainID:  chainID,
		})
	}
	rows, err := tm.db.Query("SELECT chain_id, address FROM visible_tokens")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		address := common.HexToAddress("0x")
		chainID := uint64(0)
		err := rows.Scan(&chainID, &address)
		if err != nil {
			return nil, err
		}

		found := false
		for _, token := range tokenStore[chainID] {
			if token.Address == address {
				rst[chainID] = append(rst[chainID], token)
				found = true
				break
			}
		}

		if found {
			continue
		}

		for _, token := range customTokens {
			if token.Address == address {
				rst[chainID] = append(rst[chainID], token)
				break
			}
		}
	}

	for _, chainID := range chainIDs {
		if len(rst[chainID]) == 1 {
			token := tm.findSNT(chainID)
			if token != nil {
				rst[chainID] = append(rst[chainID], token)
			}
		}
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

func (tm *TokenManager) getTokenBalance(ctx context.Context, client *chain.Client, account common.Address, token common.Address) (*big.Int, error) {
	caller, err := ierc20.NewIERC20Caller(token, client)
	if err != nil {
		return nil, err
	}

	return caller.BalanceOf(&bind.CallOpts{
		Context: ctx,
	}, account)
}

func (tm *TokenManager) getChainBalance(ctx context.Context, client *chain.Client, account common.Address) (*big.Int, error) {
	return client.BalanceAt(ctx, account, nil)
}

func (tm *TokenManager) getBalance(ctx context.Context, client *chain.Client, account common.Address, token common.Address) (*big.Int, error) {
	if token == nativeChainAddress {
		return tm.getChainBalance(ctx, client, account)
	}

	return tm.getTokenBalance(ctx, client, account, token)
}

func (tm *TokenManager) getBalances(parent context.Context, clients []*chain.Client, accounts, tokens []common.Address) (map[common.Address]map[common.Address]*hexutil.Big, error) {
	var (
		group    = async.NewAtomicGroup(parent)
		mu       sync.Mutex
		response = map[common.Address]map[common.Address]*hexutil.Big{}
	)

	updateBalance := func(account common.Address, token common.Address, balance *big.Int) {
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
	}

	contractMaker := contracts.ContractMaker{RPCClient: tm.RPCClient}
	for clientIdx := range clients {
		client := clients[clientIdx]
		ethScanContract, err := contractMaker.NewEthScan(client.ChainID)

		if err == nil {
			fetchChainBalance := false
			var tokenChunks [][]common.Address
			chunkSize := 100
			for i := 0; i < len(tokens); i += chunkSize {
				end := i + chunkSize
				if end > len(tokens) {
					end = len(tokens)
				}

				tokenChunks = append(tokenChunks, tokens[i:end])
			}

			for _, token := range tokens {
				if token == nativeChainAddress {
					fetchChainBalance = true
				}
			}
			if fetchChainBalance {
				group.Add(func(parent context.Context) error {
					ctx, cancel := context.WithTimeout(parent, requestTimeout)
					defer cancel()
					res, err := ethScanContract.EtherBalances(&bind.CallOpts{
						Context: ctx,
					}, accounts)
					if err != nil {
						log.Error("can't fetch chain balance", err)
						return nil
					}
					for idx, account := range accounts {
						balance := new(big.Int)
						balance.SetBytes(res[idx].Data)
						updateBalance(account, common.HexToAddress("0x"), balance)
					}

					return nil
				})
			}

			for accountIdx := range accounts {
				account := accounts[accountIdx]
				for idx := range tokenChunks {
					chunk := tokenChunks[idx]
					group.Add(func(parent context.Context) error {
						ctx, cancel := context.WithTimeout(parent, requestTimeout)
						defer cancel()
						res, err := ethScanContract.TokensBalance(&bind.CallOpts{
							Context: ctx,
						}, account, chunk)
						if err != nil {
							log.Error("can't fetch erc20 token balance", "account", account, "error", err)
							return nil
						}

						for idx, token := range chunk {
							if !res[idx].Success {
								continue
							}
							balance := new(big.Int)
							balance.SetBytes(res[idx].Data)
							updateBalance(account, token, balance)
						}
						return nil
					})
				}
			}
		} else {
			for tokenIdx := range tokens {
				for accountIdx := range accounts {
					// Below, we set account, token and client from idx on purpose to avoid override
					account := accounts[accountIdx]
					token := tokens[tokenIdx]
					client := clients[clientIdx]
					group.Add(func(parent context.Context) error {
						ctx, cancel := context.WithTimeout(parent, requestTimeout)
						defer cancel()
						balance, err := tm.getBalance(ctx, client, account, token)

						if err != nil {
							log.Error("can't fetch erc20 token balance", "account", account, "token", token, "error", err)

							return nil
						}
						updateBalance(account, token, balance)
						return nil
					})
				}
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
