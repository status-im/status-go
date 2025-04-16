package token

//go:generate mockgen -source=token.go -destination=mock/token/tokenmanager.go

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/contracts"
	"github.com/status-im/status-go/contracts/community-tokens/assets"
	eth_node_types "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/services/accounts/accountsevent"
	"github.com/status-im/status-go/services/communitytokens/communitytokensdatabase"
	"github.com/status-im/status-go/services/utils"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/community"
	"github.com/status-im/status-go/services/wallet/token/balancefetcher"
	tokenlists "github.com/status-im/status-go/services/wallet/token/token-lists"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const (
	EventCommunityTokenReceived walletevent.EventType = "wallet-community-token-received"
)

type ReceivedToken struct {
	tokenTypes.Token
	Amount  float64     `json:"amount"`
	TxHash  common.Hash `json:"txHash"`
	IsFirst bool        `json:"isFirst"`
}

type List struct {
	Name                string              `json:"name"`
	Tokens              []*tokenTypes.Token `json:"tokens"`
	Source              string              `json:"source"`
	Version             string              `json:"version"`
	LastUpdateTimestamp int64               `json:"lastUpdateTimestamp"`
}

type ListWrapper struct {
	UpdatedAt int64   `json:"updatedAt"`
	Data      []*List `json:"data"`
}

type addressTokenMap = map[common.Address]*tokenTypes.Token
type storeMap = map[uint64]addressTokenMap

type ManagerInterface interface {
	balancefetcher.BalanceFetcher
	LookupTokenIdentity(chainID uint64, address common.Address, native bool) *tokenTypes.Token
	LookupToken(chainID *uint64, tokenSymbol string) (token *tokenTypes.Token, isNative bool)
	GetTokenHistoricalBalance(account common.Address, chainID uint64, symbol string, timestamp int64) (*big.Int, error)
	GetTokensByChainIDs(chainIDs []uint64) ([]*tokenTypes.Token, error)
}

// Manager is used for accessing token store. It changes the token store based on overridden tokens
type Manager struct {
	balancefetcher.BalanceFetcher
	db                   *sql.DB
	RPCClient            rpc.ClientInterface
	ContractMaker        *contracts.ContractMaker
	networkManager       network.ManagerInterface
	communityTokensDB    *communitytokensdatabase.Database
	communityManager     *community.Manager
	mediaServer          *server.MediaServer
	walletFeed           *event.Feed
	accountFeed          *event.Feed
	accountWatcher       *accountsevent.Watcher
	accountsDB           *accounts.Database
	tokenBalancesStorage TokenBalancesStorage

	tokenLists *tokenlists.TokenLists
}

func NewTokenManager(
	db *sql.DB,
	RPCClient rpc.ClientInterface,
	communityManager *community.Manager,
	networkManager network.ManagerInterface,
	appDB *sql.DB,
	mediaServer *server.MediaServer,
	walletFeed *event.Feed,
	accountFeed *event.Feed,
	accountsDB *accounts.Database,
	tokenBalancesStorage TokenBalancesStorage,
) *Manager {
	maker, _ := contracts.NewContractMaker(RPCClient)

	tokensLists, err := tokenlists.NewTokenLists(appDB, db)
	if err != nil {
		logutils.ZapLogger().Error("Failed to create token lists", zap.Error(err))
		return nil
	}

	return &Manager{
		BalanceFetcher:       balancefetcher.NewDefaultBalanceFetcher(maker),
		db:                   db,
		RPCClient:            RPCClient,
		ContractMaker:        maker,
		networkManager:       networkManager,
		communityManager:     communityManager,
		communityTokensDB:    communitytokensdatabase.NewCommunityTokensDatabase(appDB),
		mediaServer:          mediaServer,
		walletFeed:           walletFeed,
		accountFeed:          accountFeed,
		accountsDB:           accountsDB,
		tokenBalancesStorage: tokenBalancesStorage,
		tokenLists:           tokensLists,
	}
}

func (tm *Manager) Start(ctx context.Context, autoRefreshInterval time.Duration, autoRefreshCheckInterval time.Duration) {
	tm.startAccountsWatcher()

	// For now we don't have the list of tokens lists remotely set so we're uisng the harcoded default lists. Once we have it
	//we will just need to update the empty string with the correct URL.
	tm.tokenLists.Start(ctx, "", autoRefreshInterval, autoRefreshCheckInterval)
}

func (tm *Manager) startAccountsWatcher() {
	if tm.accountWatcher != nil || tm.accountFeed == nil || tm.accountsDB == nil {
		return
	}

	tm.accountWatcher = accountsevent.NewWatcher(tm.accountsDB, tm.accountFeed, tm.onAccountsChange)
	tm.accountWatcher.Start()
}

func (tm *Manager) Stop() {
	tm.tokenLists.Stop()
	tm.stopAccountsWatcher()
}

func (tm *Manager) stopAccountsWatcher() {
	if tm.accountWatcher != nil {
		tm.accountWatcher.Stop()
		tm.accountWatcher = nil
	}
}

// overrideTokensInPlace overrides tokens in the store with the ones from the networks
// BEWARE: overridden tokens will have their original address removed and replaced by the one in networks
func overrideTokensInPlace(networks []params.Network, tokens []*tokenTypes.Token) {
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

func (tm *Manager) FindToken(network *params.Network, tokenSymbol string) *tokenTypes.Token {
	if tokenSymbol == network.NativeCurrencySymbol {
		return tm.ToToken(network)
	}

	return tm.GetToken(network.ChainID, tokenSymbol)
}

func (tm *Manager) LookupToken(chainID *uint64, tokenSymbol string) (token *tokenTypes.Token, isNative bool) {
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
func (tm *Manager) GetToken(chainID uint64, tokenSymbol string) *tokenTypes.Token {
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

func (tm *Manager) LookupTokenIdentity(chainID uint64, address common.Address, native bool) *tokenTypes.Token {
	network := tm.networkManager.Find(chainID)
	if native {
		return tm.ToToken(network)
	}

	return tm.FindTokenByAddress(chainID, address)
}

func (tm *Manager) FindTokenByAddress(chainID uint64, address common.Address) *tokenTypes.Token {
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

func (tm *Manager) FindOrCreateTokenByAddress(ctx context.Context, chainID uint64, address common.Address) *tokenTypes.Token {
	uniqueListsTokens := tm.tokenLists.GetUniqueTokens()
	// If token comes datasource, simply returns it
	for _, token := range uniqueListsTokens {
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

func (tm *Manager) MarkAsPreviouslyOwnedToken(token *tokenTypes.Token, owner common.Address) (bool, error) {
	logutils.ZapLogger().Info("Marking token as previously owned",
		zap.Any("token", token),
		zap.Stringer("owner", owner),
	)
	if token == nil {
		return false, errors.New("token is nil")
	}
	if (owner == common.Address{}) {
		return false, errors.New("owner is nil")
	}

	tokens, err := tm.tokenBalancesStorage.GetTokens()
	if err != nil {
		return false, err
	}

	if tokens[owner] == nil {
		tokens[owner] = make([]tokenTypes.StorageToken, 0)
	} else {
		for _, t := range tokens[owner] {
			if t.Address == token.Address && t.ChainID == token.ChainID && t.Symbol == token.Symbol {
				logutils.ZapLogger().Info("Token already marked as previously owned",
					zap.Any("token", token),
					zap.Stringer("owner", owner),
				)
				return false, nil
			}
		}
	}

	// append token to the list of tokens
	tokens[owner] = append(tokens[owner], tokenTypes.StorageToken{
		Token: *token,
		BalancesPerChain: map[uint64]tokenTypes.ChainBalance{
			token.ChainID: {
				RawBalance: "0",
				Balance:    &big.Float{},
				Address:    token.Address,
				ChainID:    token.ChainID,
			},
		},
	})

	// save the updated list of tokens
	err = tm.tokenBalancesStorage.SaveTokens(tokens)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (tm *Manager) discoverTokenCommunityID(ctx context.Context, token *tokenTypes.Token, address common.Address) {
	if token == nil || token.CommunityData != nil {
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
		logutils.ZapLogger().Error("Cannot prepare token update query", zap.Error(err))
		return
	}

	if uri == "" {
		// Update token community ID to prevent further checks
		_, err := update.Exec("", token.ChainID, token.Address)
		if err != nil {
			logutils.ZapLogger().Error("Cannot update community id", zap.Error(err))
		}
		return
	}

	uri = strings.TrimSuffix(uri, "/")
	communityIDHex, err := utils.DeserializePublicKey(uri)
	if err != nil {
		return
	}
	communityID := eth_node_types.EncodeHex(communityIDHex)

	token.CommunityData = &community.Data{
		ID: communityID,
	}

	_, err = update.Exec(communityID, token.ChainID, token.Address)
	if err != nil {
		logutils.ZapLogger().Error("Cannot update community id", zap.Error(err))
	}
}

func (tm *Manager) FindSNT(chainID uint64) *tokenTypes.Token {
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

func (tm *Manager) getNativeTokens() ([]*tokenTypes.Token, error) {
	tokens := make([]*tokenTypes.Token, 0)
	networks, err := tm.networkManager.Get(false)
	if err != nil {
		return nil, err
	}

	for _, network := range networks {
		tokens = append(tokens, tm.ToToken(network))
	}

	return tokens, nil
}

func (tm *Manager) GetAllTokens() ([]*tokenTypes.Token, error) {
	allTokens, err := tm.GetCustoms(true)
	if err != nil {
		logutils.ZapLogger().Error("can't fetch custom tokens", zap.Error(err))
	}

	uniqueListsTokens := tm.tokenLists.GetUniqueTokens()

	allTokens = append(uniqueListsTokens, allTokens...)

	overrideTokensInPlace(tm.networkManager.GetEmbeddedNetworks(), allTokens)

	native, err := tm.getNativeTokens()
	if err != nil {
		return nil, err
	}

	allTokens = append(allTokens, native...)

	return allTokens, nil
}

func (tm *Manager) GetTokens(chainID uint64) ([]*tokenTypes.Token, error) {
	tokens, err := tm.GetAllTokens()
	if err != nil {
		return nil, err
	}

	res := make([]*tokenTypes.Token, 0)

	for _, token := range tokens {
		if token.ChainID == chainID {
			res = append(res, token)
		}
	}

	return res, nil
}

func (tm *Manager) GetTokensByChainIDs(chainIDs []uint64) ([]*tokenTypes.Token, error) {
	tokens, err := tm.GetAllTokens()
	if err != nil {
		return nil, err
	}

	res := make([]*tokenTypes.Token, 0)

	for _, token := range tokens {
		for _, chainID := range chainIDs {
			if token.ChainID == chainID {
				res = append(res, token)
			}
		}
	}

	return res, nil
}

func (tm *Manager) GetList() *ListWrapper {
	data := make([]*List, 0)
	nativeTokens, err := tm.getNativeTokens()
	if err == nil {
		data = append(data, &List{
			Name:    "native",
			Tokens:  nativeTokens,
			Source:  "native",
			Version: "1.0.0",
		})
	}

	customTokens, err := tm.GetCustoms(true)
	if err == nil && len(customTokens) > 0 {
		data = append(data, &List{
			Name:    "custom",
			Tokens:  customTokens,
			Source:  "custom",
			Version: "1.0.0",
		})
	}

	tokensLists := tm.tokenLists.GetTokensLists()
	for _, tokensList := range tokensLists {
		timestamp, err := time.Parse(time.RFC3339, tokensList.Timestamp)
		if err != nil {
			logutils.ZapLogger().Error("Failed to parse timestamp", zap.Error(err))
			continue
		}
		data = append(data, &List{
			Name:                tokensList.Name,
			Tokens:              tokensList.Tokens,
			Source:              tokensList.Source,
			Version:             tokensList.GetVersion(),
			LastUpdateTimestamp: timestamp.Unix(),
		})
	}

	lastUpdate, err := tm.tokenLists.LastTokensUpdate()
	if err != nil {
		logutils.ZapLogger().Error("Failed to get last update timestamp", zap.Error(err))
	}
	var updatedAt int64
	if !lastUpdate.IsZero() {
		updatedAt = lastUpdate.Unix()
	}

	return &ListWrapper{
		Data:      data,
		UpdatedAt: updatedAt,
	}
}

func (tm *Manager) DiscoverToken(ctx context.Context, chainID uint64, address common.Address) (*tokenTypes.Token, error) {
	caller, err := tm.ContractMaker.NewERC20(chainID, address)
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

	return &tokenTypes.Token{
		Address:  address,
		Name:     name,
		Symbol:   symbol,
		Decimals: uint(decimal),
		ChainID:  chainID,
	}, nil
}

func (tm *Manager) GetCommunityTokenType(chainID uint64, tokenContractAddress string) (protobuf.CommunityTokenType, error) {
	if tm.communityTokensDB != nil {
		return tm.communityTokensDB.GetTokenType(chainID, tokenContractAddress)
	}
	return protobuf.CommunityTokenType_UNKNOWN_TOKEN_TYPE, nil
}

func (tm *Manager) GetCommunityTokenPrivilegesLevel(chainID uint64, tokenContractAddress string) (token.PrivilegesLevel, error) {
	if tm.communityTokensDB != nil {
		return tm.communityTokensDB.GetTokenPrivilegesLevel(chainID, tokenContractAddress)
	}
	return token.CommunityLevel, nil
}

func (tm *Manager) getTokensFromDB(query string, args ...any) ([]*tokenTypes.Token, error) {
	communityTokens := []*token.CommunityToken{}
	if tm.communityTokensDB != nil {
		// Error is skipped because it's only returning optional metadata
		communityTokens, _ = tm.communityTokensDB.GetCommunityERC20Metadata()
	}

	rows, err := tm.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rst []*tokenTypes.Token
	for rows.Next() {
		token := &tokenTypes.Token{}
		var communityIDDB sql.NullString
		err := rows.Scan(&token.Address, &token.Name, &token.Symbol, &token.Decimals, &token.ChainID, &communityIDDB)
		if err != nil {
			return nil, err
		}

		if communityIDDB.Valid {
			communityID := communityIDDB.String
			for _, communityToken := range communityTokens {
				if communityToken.CommunityID != communityID || uint64(communityToken.ChainID) != token.ChainID || communityToken.Symbol != token.Symbol {
					continue
				}
				token.Image = tm.mediaServer.MakeCommunityTokenImagesURL(communityID, token.ChainID, token.Symbol)
				break
			}

			token.CommunityData = &community.Data{
				ID: communityID,
			}
		}

		_ = tm.fillCommunityData(token)

		rst = append(rst, token)
	}

	return rst, nil
}

func (tm *Manager) GetCustoms(onlyCommunityCustoms bool) ([]*tokenTypes.Token, error) {
	if onlyCommunityCustoms {
		return tm.getTokensFromDB("SELECT address, name, symbol, decimals, network_id, community_id FROM tokens WHERE community_id IS NOT NULL AND community_id != ''")
	}
	return tm.getTokensFromDB("SELECT address, name, symbol, decimals, network_id, community_id FROM tokens")
}

func (tm *Manager) ToToken(network *params.Network) *tokenTypes.Token {
	return &tokenTypes.Token{
		// TODO: we need to change the address for the native token to the correct one, we cannot to that right now cause will affect other parts of the code
		// The following line is the right fix for `{"error":"Validation failed: \"srcToken\" contains an invalid value"}` error for Swap
		// Address:  common.HexToAddress("0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"), // for all the chains we support this is the address of the native (ETH) token
		Address:   common.HexToAddress("0x"),
		Name:      network.NativeCurrencyName,
		Symbol:    network.NativeCurrencySymbol,
		TmpSymbol: network.NativeCurrencySymbol,
		Decimals:  uint(network.NativeCurrencyDecimals),
		ChainID:   network.ChainID,
		Verified:  true,
	}
}

func (tm *Manager) UpsertCustom(token tokenTypes.Token) error {
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

func (tm *Manager) SignalCommunityTokenReceived(address common.Address, txHash common.Hash, value *big.Int, t *tokenTypes.Token, isFirst bool) {
	defer gocommon.LogOnPanic()
	if tm.walletFeed == nil || t == nil || t.CommunityData == nil {
		return
	}

	if len(t.CommunityData.Name) == 0 {
		_ = tm.fillCommunityData(t)
	}
	if len(t.CommunityData.Name) == 0 && tm.communityManager != nil {
		communityData, _ := tm.communityManager.FetchCommunityMetadata(t.CommunityData.ID)
		if communityData != nil {
			t.CommunityData.Name = communityData.CommunityName
			t.CommunityData.Color = communityData.CommunityColor
			t.CommunityData.Image = tm.communityManager.GetCommunityImageURL(t.CommunityData.ID)
		}
	}

	floatAmount, _ := new(big.Float).Quo(new(big.Float).SetInt(value), new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(t.Decimals)), nil))).Float64()
	t.Image = tm.mediaServer.MakeCommunityTokenImagesURL(t.CommunityData.ID, t.ChainID, t.Symbol)

	receivedToken := ReceivedToken{
		Token:   *t,
		Amount:  floatAmount,
		TxHash:  txHash,
		IsFirst: isFirst,
	}

	encodedMessage, err := json.Marshal(receivedToken)
	if err != nil {
		return
	}

	tm.walletFeed.Send(walletevent.Event{
		Type:    EventCommunityTokenReceived,
		ChainID: t.ChainID,
		Accounts: []common.Address{
			address,
		},
		Message: string(encodedMessage),
	})
}

func (tm *Manager) fillCommunityData(token *tokenTypes.Token) error {
	if token == nil || token.CommunityData == nil || tm.communityManager == nil {
		return nil
	}

	communityInfo, _, err := tm.communityManager.GetCommunityInfo(token.CommunityData.ID)
	if err != nil {
		return err
	}
	if err == nil && communityInfo != nil {
		// Fetched data from cache. Cache is refreshed during every wallet token list call.
		token.CommunityData.Name = communityInfo.CommunityName
		token.CommunityData.Color = communityInfo.CommunityColor
		token.CommunityData.Image = communityInfo.CommunityImage
	}
	return nil
}

func (tm *Manager) GetTokenHistoricalBalance(account common.Address, chainID uint64, symbol string, timestamp int64) (*big.Int, error) {
	var balance big.Int
	err := tm.db.QueryRow("SELECT balance FROM balance_history WHERE currency = ? AND chain_id = ? AND address = ? AND timestamp < ? order by timestamp DESC LIMIT 1", symbol, chainID, account, timestamp).Scan((*bigint.SQLBigIntBytes)(&balance))
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
