package token

import (
	"context"
	"database/sql"
	"errors"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/contracts"
	"github.com/status-im/status-go/contracts/ierc20"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/wallet/async"
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
	// PegSymbol indicates that the token is pegged to some fiat currency, using the
	// ISO 4217 alphabetic code. For example, an empty string means it is not
	// pegged, while "USD" means it's pegged to the United States Dollar.
	PegSymbol string `json:"pegSymbol"`
}

func (t *Token) IsNative() bool {
	return t.Address == nativeChainAddress
}

// Manager is used for accessing token store. It changes the token store based on overridden tokens
type Manager struct {
	db             *sql.DB
	RPCClient      *rpc.Client
	networkManager *network.Manager
	stores         []store
	tokenList      []*Token
	tokenMap       storeMap
}

func NewTokenManager(
	db *sql.DB,
	RPCClient *rpc.Client,
	networkManager *network.Manager,
) *Manager {
	// Order of stores is important when merging token lists. The former prevale
	tokenManager := &Manager{db, RPCClient, networkManager, []store{newUniswapStore(), newDefaultStore()}, nil, nil}

	return tokenManager
}

// overrideTokensInPlace overrides tokens in the store with the ones from the networks
// BEWARE: overridden tokens will have their original address removed and replaced by the one in networks
func overrideTokensInPlace(networks []params.Network, tokens []*Token) {
	for _, network := range networks {
		if len(network.TokenOverrides) == 0 {
			continue
		}

		for _, overrideToken := range network.TokenOverrides {
			for _, token := range tokens {
				if token.Symbol == overrideToken.Symbol {
					token.Address = overrideToken.Address
				}
			}
		}
	}
}

func mergeTokenLists(sliceLists [][]*Token) []*Token {
	allKeys := make(map[string]bool)
	res := []*Token{}
	for _, list := range sliceLists {
		for _, token := range list {
			key := strconv.FormatUint(token.ChainID, 10) + token.Address.String()
			if _, value := allKeys[key]; !value {
				allKeys[key] = true
				res = append(res, token)
			}
		}
	}
	return res
}

func (tm *Manager) inStore(address common.Address, chainID uint64) bool {
	if address == nativeChainAddress {
		return true
	}

	if !tm.areTokensFetched() {
		tm.fetchTokens()
	}

	tokensMap, ok := tm.tokenMap[chainID]
	if !ok {
		return false
	}
	_, ok = tokensMap[address]

	return ok
}

func (tm *Manager) areTokensFetched() bool {
	for _, store := range tm.stores {
		if !store.areTokensFetched() {
			return false
		}
	}

	return true
}

func (tm *Manager) fetchTokens() {
	tm.tokenList = nil
	tm.tokenMap = nil

	networks, err := tm.networkManager.Get(false)
	if err != nil {
		return
	}

	for _, store := range tm.stores {
		tokens, err := store.GetTokens()
		if err != nil {
			log.Error("can't fetch tokens from store", "error", err)
			continue
		}
		validTokens := make([]*Token, 0)
		for _, token := range tokens {
			for _, network := range networks {
				if network.ChainID == token.ChainID {
					validTokens = append(validTokens, token)
					break
				}
			}
		}

		tm.tokenList = mergeTokenLists([][]*Token{tm.tokenList, validTokens})
	}

	tm.tokenMap = toTokenMap(tm.tokenList)
}

func (tm *Manager) FindToken(network *params.Network, tokenSymbol string) *Token {
	if tokenSymbol == network.NativeCurrencySymbol {
		return tm.ToToken(network)
	}

	tokens, err := tm.GetTokens(network.ChainID)
	if err != nil {
		return nil
	}
	customTokens, err := tm.GetCustomsByChainID(network.ChainID)
	if err != nil {
		return nil
	}
	allTokens := append(tokens, customTokens...)
	for _, token := range allTokens {
		if token.Symbol == tokenSymbol {
			return token
		}
	}
	return nil
}

func (tm *Manager) FindSNT(chainID uint64) *Token {
	tokens, err := tm.GetTokens(chainID)
	if err != nil {
		return nil
	}

	for _, token := range tokens {
		if token.Symbol == "SNT" || token.Symbol == "STT" {
			return token
		}
	}

	return nil
}

func (tm *Manager) GetAllTokensAndNativeCurrencies() ([]*Token, error) {
	allTokens, err := tm.GetAllTokens()
	if err != nil {
		return nil, err
	}

	networks, err := tm.RPCClient.NetworkManager.Get(false)
	if err != nil {
		return nil, err
	}

	for _, network := range networks {
		allTokens = append(allTokens, tm.ToToken(network))
	}

	return allTokens, nil
}

func (tm *Manager) GetAllTokens() ([]*Token, error) {
	if !tm.areTokensFetched() {
		tm.fetchTokens()
	}

	tokens, err := tm.GetCustoms()
	if err != nil {
		log.Error("can't fetch custom tokens", "error", err)
	}

	tokens = append(tm.tokenList, tokens...)

	overrideTokensInPlace(tm.networkManager.GetConfiguredNetworks(), tokens)

	return tokens, nil
}

func (tm *Manager) GetTokens(chainID uint64) ([]*Token, error) {
	if !tm.areTokensFetched() {
		tm.fetchTokens()
	}

	tokensMap, ok := tm.tokenMap[chainID]
	if !ok {
		return nil, errors.New("no tokens for this network")
	}

	res := make([]*Token, 0, len(tokensMap))

	for _, token := range tokensMap {
		res = append(res, token)
	}

	return res, nil
}

func (tm *Manager) DiscoverToken(ctx context.Context, chainID uint64, address common.Address) (*Token, error) {
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

func (tm *Manager) GetCustoms() ([]*Token, error) {
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

func (tm *Manager) GetCustomsByChainID(chainID uint64) ([]*Token, error) {
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

func (tm *Manager) IsTokenVisible(chainID uint64, address common.Address) (bool, error) {
	rows, err := tm.db.Query("SELECT chain_id, address FROM visible_tokens WHERE chain_id = ? AND address = ?", chainID, address)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	return rows.Next(), nil
}

func (tm *Manager) Toggle(chainID uint64, address common.Address) error {
	isVisible, err := tm.IsTokenVisible(chainID, address)
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

func (tm *Manager) ToToken(network *params.Network) *Token {
	return &Token{
		Address:  common.HexToAddress("0x"),
		Name:     network.NativeCurrencyName,
		Symbol:   network.NativeCurrencySymbol,
		Decimals: uint(network.NativeCurrencyDecimals),
		ChainID:  network.ChainID,
	}
}

func (tm *Manager) GetVisible(chainIDs []uint64) (map[uint64][]*Token, error) {
	customTokens, err := tm.GetCustoms()
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
		rst[chainID] = append(rst[chainID], tm.ToToken(network))
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
		tokens, err := tm.GetTokens(chainID)
		if err != nil {
			continue
		}

		for _, token := range tokens {
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
			token := tm.FindSNT(chainID)
			if token != nil {
				rst[chainID] = append(rst[chainID], token)
			}
		}
	}
	return rst, nil
}

func (tm *Manager) UpsertCustom(token Token) error {
	insert, err := tm.db.Prepare("INSERT OR REPLACE INTO TOKENS (network_id, address, name, symbol, decimals, color) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	_, err = insert.Exec(token.ChainID, token.Address, token.Name, token.Symbol, token.Decimals, token.Color)
	return err
}

func (tm *Manager) DeleteCustom(chainID uint64, address common.Address) error {
	_, err := tm.db.Exec(`DELETE FROM TOKENS WHERE address = ? and network_id = ?`, address, chainID)
	return err
}

func (tm *Manager) GetTokenBalance(ctx context.Context, client *chain.ClientWithFallback, account common.Address, token common.Address) (*big.Int, error) {
	caller, err := ierc20.NewIERC20Caller(token, client)
	if err != nil {
		return nil, err
	}

	return caller.BalanceOf(&bind.CallOpts{
		Context: ctx,
	}, account)
}

func (tm *Manager) GetTokenBalanceAt(ctx context.Context, client *chain.ClientWithFallback, account common.Address, token common.Address, blockNumber *big.Int) (*big.Int, error) {
	caller, err := ierc20.NewIERC20Caller(token, client)
	if err != nil {
		return nil, err
	}

	return caller.BalanceOf(&bind.CallOpts{
		Context:     ctx,
		BlockNumber: blockNumber,
	}, account)
}

func (tm *Manager) GetChainBalance(ctx context.Context, client *chain.ClientWithFallback, account common.Address) (*big.Int, error) {
	return client.BalanceAt(ctx, account, nil)
}

func (tm *Manager) GetBalance(ctx context.Context, client *chain.ClientWithFallback, account common.Address, token common.Address) (*big.Int, error) {
	if token == nativeChainAddress {
		return tm.GetChainBalance(ctx, client, account)
	}

	return tm.GetTokenBalance(ctx, client, account, token)
}

func (tm *Manager) GetBalances(parent context.Context, clients map[uint64]*chain.ClientWithFallback, accounts, tokens []common.Address) (map[common.Address]map[common.Address]*hexutil.Big, error) {
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
			chunkSize := 500
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
					if !tm.inStore(token, client.ChainID) {
						continue
					}
					group.Add(func(parent context.Context) error {
						ctx, cancel := context.WithTimeout(parent, requestTimeout)
						defer cancel()
						balance, err := tm.GetBalance(ctx, client, account, token)

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

func (tm *Manager) GetBalancesByChain(parent context.Context, clients map[uint64]*chain.ClientWithFallback, accounts, tokens []common.Address) (map[uint64]map[common.Address]map[common.Address]*hexutil.Big, error) {
	var (
		group    = async.NewAtomicGroup(parent)
		mu       sync.Mutex
		response = map[uint64]map[common.Address]map[common.Address]*hexutil.Big{}
	)

	updateBalance := func(chainID uint64, account common.Address, token common.Address, balance *big.Int) {
		mu.Lock()
		if _, ok := response[chainID]; !ok {
			response[chainID] = map[common.Address]map[common.Address]*hexutil.Big{}
		}

		if _, ok := response[chainID][account]; !ok {
			response[chainID][account] = map[common.Address]*hexutil.Big{}
		}

		if _, ok := response[chainID][account][token]; !ok {
			zeroHex := hexutil.Big(*big.NewInt(0))
			response[chainID][account][token] = &zeroHex
		}
		sum := big.NewInt(0).Add(response[chainID][account][token].ToInt(), balance)
		sumHex := hexutil.Big(*sum)
		response[chainID][account][token] = &sumHex
		mu.Unlock()
	}

	contractMaker := contracts.ContractMaker{RPCClient: tm.RPCClient}
	for clientIdx := range clients {
		client := clients[clientIdx]
		ethScanContract, err := contractMaker.NewEthScan(client.ChainID)
		if err != nil {
			log.Error("error scanning contract", "err", err)
			return nil, err
		}

		fetchChainBalance := false
		var tokenChunks [][]common.Address
		chunkSize := 500
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
					updateBalance(client.ChainID, account, common.HexToAddress("0x"), balance)
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

					if len(res) != len(chunk) {
						log.Error("can't fetch erc20 token balance", "account", account, "error response not complete")
						return nil
					}

					for idx, token := range chunk {

						if !res[idx].Success {
							continue
						}
						balance := new(big.Int)
						balance.SetBytes(res[idx].Data)
						updateBalance(client.ChainID, account, token, balance)
					}
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
