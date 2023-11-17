package token

import (
	"context"
	"database/sql"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/contracts"
	"github.com/status-im/status-go/contracts/community-tokens/assets"
	"github.com/status-im/status-go/contracts/ethscan"
	"github.com/status-im/status-go/contracts/ierc20"
	eth_node_types "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/utils"
	"github.com/status-im/status-go/services/wallet/async"
)

var requestTimeout = 20 * time.Second
var nativeChainAddress = common.HexToAddress("0x")

type Token struct {
	Address common.Address `json:"address"`
	Name    string         `json:"name"`
	Symbol  string         `json:"symbol"`
	// Decimals defines how divisible the token is. For example, 0 would be
	// indivisible, whereas 18 would allow very small amounts of the token
	// to be traded.
	Decimals uint   `json:"decimals"`
	ChainID  uint64 `json:"chainId"`
	// PegSymbol indicates that the token is pegged to some fiat currency, using the
	// ISO 4217 alphabetic code. For example, an empty string means it is not
	// pegged, while "USD" means it's pegged to the United States Dollar.
	PegSymbol string `json:"pegSymbol"`

	CommunityID *string `json:"communityId,omitempty"`
	Verified    bool    `json:"verified"`
	TokenListID string  `json:"tokenListId"`
}

func (t *Token) IsNative() bool {
	return t.Address == nativeChainAddress
}

type List struct {
	Name      string   `json:"name"`
	Tokens    []*Token `json:"tokens"`
	UpdatedAt int64    `json:"updatedAt"`
	Source    string   `json:"source"`
	Version   string   `json:"version"`
}

type addressTokenMap = map[common.Address]*Token
type storeMap = map[uint64]addressTokenMap

type ManagerInterface interface {
	LookupTokenIdentity(chainID uint64, address common.Address, native bool) *Token
	LookupToken(chainID *uint64, tokenSymbol string) (token *Token, isNative bool)
}

// Manager is used for accessing token store. It changes the token store based on overridden tokens
type Manager struct {
	db             *sql.DB
	RPCClient      *rpc.Client
	contractMaker  *contracts.ContractMaker
	networkManager *network.Manager
	stores         []store // Set on init, not changed afterwards

	tokens []*Token

	tokenLock sync.RWMutex
}

func mergeTokens(sliceLists [][]*Token) []*Token {
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

func NewTokenManager(
	db *sql.DB,
	RPCClient *rpc.Client,
	networkManager *network.Manager,
) *Manager {
	maker, _ := contracts.NewContractMaker(RPCClient)
	stores := []store{newUniswapStore(), newDefaultStore()}
	tokens := make([]*Token, 0)

	networks, err := networkManager.GetAll()
	if err != nil {
		return nil
	}

	for _, store := range stores {
		validTokens := make([]*Token, 0)
		for _, token := range store.GetTokens() {
			token.Verified = true

			for _, network := range networks {
				if network.ChainID == token.ChainID {
					validTokens = append(validTokens, token)
					break
				}
			}
		}

		tokens = mergeTokens([][]*Token{tokens, validTokens})
	}

	return &Manager{
		db:             db,
		RPCClient:      RPCClient,
		contractMaker:  maker,
		networkManager: networkManager,
		stores:         stores,
		tokens:         tokens,
	}
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

func (tm *Manager) getTokens() []*Token {
	tm.tokenLock.RLock()
	defer tm.tokenLock.RUnlock()

	return tm.tokens
}

func (tm *Manager) SetTokens(tokens []*Token) {
	tm.tokenLock.Lock()
	defer tm.tokenLock.Unlock()
	tm.tokens = tokens
}

func (tm *Manager) FindToken(network *params.Network, tokenSymbol string) *Token {
	if tokenSymbol == network.NativeCurrencySymbol {
		return tm.ToToken(network)
	}

	return tm.GetToken(network.ChainID, tokenSymbol)
}

func (tm *Manager) LookupToken(chainID *uint64, tokenSymbol string) (token *Token, isNative bool) {
	if chainID == nil {
		networks, err := tm.networkManager.Get(false)
		if err != nil {
			return nil, false
		}

		for _, network := range networks {
			if tokenSymbol == network.NativeCurrencySymbol {
				return tm.ToToken(network), true
			}
			token := tm.GetToken(network.ChainID, tokenSymbol)
			if token != nil {
				return token, false
			}
		}
	} else {
		network := tm.networkManager.Find(*chainID)
		if network != nil && tokenSymbol == network.NativeCurrencySymbol {
			return tm.ToToken(network), true
		}
		return tm.GetToken(*chainID, tokenSymbol), false
	}
	return nil, false
}

// GetToken returns token by chainID and tokenSymbol. Use ToToken for native token
func (tm *Manager) GetToken(chainID uint64, tokenSymbol string) *Token {
	allTokens, err := tm.GetTokens(chainID)
	if err != nil {
		return nil
	}
	for _, token := range allTokens {
		if token.Symbol == tokenSymbol {
			return token
		}
	}
	return nil
}

func (tm *Manager) LookupTokenIdentity(chainID uint64, address common.Address, native bool) *Token {
	network := tm.networkManager.Find(chainID)
	if native {
		return tm.ToToken(network)
	}

	return tm.FindTokenByAddress(chainID, address)
}

func (tm *Manager) FindTokenByAddress(chainID uint64, address common.Address) *Token {
	allTokens, err := tm.GetTokens(chainID)
	if err != nil {
		return nil
	}
	for _, token := range allTokens {
		if token.Address == address {
			return token
		}
	}

	return nil
}

func (tm *Manager) FindOrCreateTokenByAddress(ctx context.Context, chainID uint64, address common.Address) *Token {
	// If token comes datasource, simply returns it
	for _, token := range tm.getTokens() {
		if token.ChainID != chainID {
			continue
		}
		if token.Address == address {
			return token
		}
	}

	// Create custom token if not known or try to link with a community
	customTokens, err := tm.GetCustoms(false)
	if err != nil {
		return nil
	}

	for _, token := range customTokens {
		if token.Address == address {
			tm.discoverTokenCommunityID(ctx, token, address)
			return token
		}
	}

	token, err := tm.DiscoverToken(ctx, chainID, address)
	if err != nil {
		return nil
	}

	err = tm.UpsertCustom(*token)
	if err != nil {
		return nil
	}

	tm.discoverTokenCommunityID(ctx, token, address)
	return token
}

func (tm *Manager) discoverTokenCommunityID(ctx context.Context, token *Token, address common.Address) {
	if token == nil || token.CommunityID != nil {
		// Token is invalid or is alrady discovered. Nothing to do here.
		return
	}
	backend, err := tm.RPCClient.EthClient(token.ChainID)
	if err != nil {
		return
	}
	caller, err := assets.NewAssetsCaller(address, backend)
	if err != nil {
		return
	}
	uri, err := caller.BaseTokenURI(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return
	}

	update, err := tm.db.Prepare("UPDATE tokens SET community_id=? WHERE network_id=? AND address=?")
	if err != nil {
		log.Error("Cannot prepare token update query", err)
		return
	}

	if uri == "" {
		// Update token community ID to prevent further checks
		_, err := update.Exec("", token.ChainID, token.Address)
		if err != nil {
			log.Error("Cannot update community id", err)
		}
		return
	}

	uri = strings.TrimSuffix(uri, "/")
	communityIDHex, err := utils.DeserializePublicKey(uri)
	if err != nil {
		return
	}
	communityID := eth_node_types.EncodeHex(communityIDHex)

	_, err = update.Exec(communityID, token.ChainID, token.Address)
	if err != nil {
		log.Error("Cannot update community id", err)
	}
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

func (tm *Manager) getNativeTokens() ([]*Token, error) {
	tokens := make([]*Token, 0)
	networks, err := tm.networkManager.Get(false)
	if err != nil {
		return nil, err
	}

	for _, network := range networks {
		tokens = append(tokens, tm.ToToken(network))
	}

	return tokens, nil
}

func (tm *Manager) GetAllTokens() ([]*Token, error) {
	allTokens, err := tm.GetCustoms(true)
	if err != nil {
		log.Error("can't fetch custom tokens", "error", err)
	}

	allTokens = append(tm.getTokens(), allTokens...)

	overrideTokensInPlace(tm.networkManager.GetConfiguredNetworks(), allTokens)

	native, err := tm.getNativeTokens()
	if err != nil {
		return nil, err
	}

	allTokens = append(allTokens, native...)

	return allTokens, nil
}

func (tm *Manager) GetTokens(chainID uint64) ([]*Token, error) {
	tokens, err := tm.GetAllTokens()
	if err != nil {
		return nil, err
	}

	res := make([]*Token, 0)

	for _, token := range tokens {
		if token.ChainID == chainID {
			res = append(res, token)
		}
	}

	return res, nil
}

func (tm *Manager) GetTokensByChainIDs(chainIDs []uint64) ([]*Token, error) {
	tokens, err := tm.GetAllTokens()
	if err != nil {
		return nil, err
	}

	res := make([]*Token, 0)

	for _, token := range tokens {
		for _, chainID := range chainIDs {
			if token.ChainID == chainID {
				res = append(res, token)
			}
		}
	}

	return res, nil
}

func (tm *Manager) GetList() []*List {
	res := make([]*List, 0)
	nativeTokens, err := tm.getNativeTokens()
	if err == nil {
		res = append(res, &List{
			Name:      "native",
			Tokens:    nativeTokens,
			UpdatedAt: time.Now().Unix(),
			Source:    "native",
			Version:   "1.0.0",
		})
	}

	customTokens, err := tm.GetCustoms(true)
	if err == nil && len(customTokens) > 0 {
		res = append(res, &List{
			Name:      "custom",
			Tokens:    customTokens,
			UpdatedAt: time.Now().Unix(),
			Source:    "custom",
			Version:   "1.0.0",
		})
	}

	for _, store := range tm.stores {
		res = append(res, &List{
			Name:      store.GetName(),
			Tokens:    store.GetTokens(),
			UpdatedAt: store.GetUpdatedAt(),
			Source:    store.GetSource(),
			Version:   store.GetVersion(),
		})
	}
	return res
}

func (tm *Manager) DiscoverToken(ctx context.Context, chainID uint64, address common.Address) (*Token, error) {
	caller, err := tm.contractMaker.NewERC20(chainID, address)
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
		ChainID:  chainID,
	}, nil
}

func (tm *Manager) getTokensFromDB(query string, args ...any) ([]*Token, error) {
	rows, err := tm.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rst []*Token
	for rows.Next() {
		token := &Token{}
		var communityIDDB sql.NullString
		err := rows.Scan(&token.Address, &token.Name, &token.Symbol, &token.Decimals, &token.ChainID, &communityIDDB)
		if err != nil {
			return nil, err
		}

		if communityIDDB.Valid {
			token.CommunityID = &communityIDDB.String
		}

		rst = append(rst, token)
	}

	return rst, nil
}

func (tm *Manager) GetCustoms(onlyCommunityCustoms bool) ([]*Token, error) {
	if onlyCommunityCustoms {
		return tm.getTokensFromDB("SELECT address, name, symbol, decimals, network_id, community_id FROM tokens WHERE community_id IS NOT NULL AND community_id != ''")
	}
	return tm.getTokensFromDB("SELECT address, name, symbol, decimals, network_id, community_id FROM tokens")
}

func (tm *Manager) ToToken(network *params.Network) *Token {
	return &Token{
		Address:  common.HexToAddress("0x"),
		Name:     network.NativeCurrencyName,
		Symbol:   network.NativeCurrencySymbol,
		Decimals: uint(network.NativeCurrencyDecimals),
		ChainID:  network.ChainID,
		Verified: true,
	}
}

func (tm *Manager) UpsertCustom(token Token) error {
	insert, err := tm.db.Prepare("INSERT OR REPLACE INTO TOKENS (network_id, address, name, symbol, decimals) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	_, err = insert.Exec(token.ChainID, token.Address, token.Name, token.Symbol, token.Decimals)
	return err
}

func (tm *Manager) DeleteCustom(chainID uint64, address common.Address) error {
	_, err := tm.db.Exec(`DELETE FROM TOKENS WHERE address = ? and network_id = ?`, address, chainID)
	return err
}

func (tm *Manager) GetTokenBalance(ctx context.Context, client chain.ClientInterface, account common.Address, token common.Address) (*big.Int, error) {
	caller, err := ierc20.NewIERC20Caller(token, client)
	if err != nil {
		return nil, err
	}

	return caller.BalanceOf(&bind.CallOpts{
		Context: ctx,
	}, account)
}

func (tm *Manager) GetTokenBalanceAt(ctx context.Context, client chain.ClientInterface, account common.Address, token common.Address, blockNumber *big.Int) (*big.Int, error) {
	caller, err := ierc20.NewIERC20Caller(token, client)
	if err != nil {
		return nil, err
	}

	balance, err := caller.BalanceOf(&bind.CallOpts{
		Context:     ctx,
		BlockNumber: blockNumber,
	}, account)

	if err != nil {
		if err != bind.ErrNoCode {
			return nil, err
		}
		balance = big.NewInt(0)
	}

	return balance, nil
}

func (tm *Manager) GetChainBalance(ctx context.Context, client chain.ClientInterface, account common.Address) (*big.Int, error) {
	return client.BalanceAt(ctx, account, nil)
}

func (tm *Manager) GetBalance(ctx context.Context, client chain.ClientInterface, account common.Address, token common.Address) (*big.Int, error) {
	if token == nativeChainAddress {
		return tm.GetChainBalance(ctx, client, account)
	}

	return tm.GetTokenBalance(ctx, client, account, token)
}

func (tm *Manager) GetBalancesByChain(parent context.Context, clients map[uint64]chain.ClientInterface, accounts, tokens []common.Address) (map[uint64]map[common.Address]map[common.Address]*hexutil.Big, error) {
	return tm.GetBalancesAtByChain(parent, clients, accounts, tokens, nil)
}

func (tm *Manager) GetBalancesAtByChain(parent context.Context, clients map[uint64]chain.ClientInterface, accounts, tokens []common.Address, atBlocks map[uint64]*big.Int) (map[uint64]map[common.Address]map[common.Address]*hexutil.Big, error) {
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

	for clientIdx := range clients {
		// Keep the reference to the client. DO NOT USE A LOOP, the client will be overridden in the coroutine
		client := clients[clientIdx]

		ethScanContract, availableAtBlock, err := tm.contractMaker.NewEthScan(client.NetworkID())
		if err != nil {
			log.Error("error scanning contract", "err", err)
			return nil, err
		}

		atBlock := atBlocks[client.NetworkID()]

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
					Context:     ctx,
					BlockNumber: atBlock,
				}, accounts)
				if err != nil {
					log.Error("can't fetch chain balance 5", err)
					return nil
				}
				for idx, account := range accounts {
					balance := new(big.Int)
					balance.SetBytes(res[idx].Data)
					updateBalance(client.NetworkID(), account, common.HexToAddress("0x"), balance)
				}

				return nil
			})
		}

		for accountIdx := range accounts {
			// Keep the reference to the account. DO NOT USE A LOOP, the account will be overridden in the coroutine
			account := accounts[accountIdx]
			for idx := range tokenChunks {
				// Keep the reference to the chunk. DO NOT USE A LOOP, the chunk will be overridden in the coroutine
				chunk := tokenChunks[idx]

				group.Add(func(parent context.Context) error {
					ctx, cancel := context.WithTimeout(parent, requestTimeout)
					defer cancel()
					var res []ethscan.BalanceScannerResult
					if atBlock == nil || big.NewInt(int64(availableAtBlock)).Cmp(atBlock) < 0 {
						res, err = ethScanContract.TokensBalance(&bind.CallOpts{
							Context:     ctx,
							BlockNumber: atBlock,
						}, account, chunk)
						if err != nil {
							log.Error("can't fetch erc20 token balance 6", "account", account, "error", err)
							return nil
						}

						if len(res) != len(chunk) {
							log.Error("can't fetch erc20 token balance 7", "account", account, "error response not complete")
							return nil
						}

						for idx, token := range chunk {
							if !res[idx].Success {
								continue
							}
							balance := new(big.Int)
							balance.SetBytes(res[idx].Data)
							updateBalance(client.NetworkID(), account, token, balance)
						}
						return nil
					}

					for _, token := range chunk {
						balance, err := tm.GetTokenBalanceAt(ctx, client, account, token, atBlock)
						if err != nil {
							if err != bind.ErrNoCode {
								log.Error("can't fetch erc20 token balance 8", "account", account, "token", token, "error on fetching token balance")

								return nil
							}
						}
						updateBalance(client.NetworkID(), account, token, balance)
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
