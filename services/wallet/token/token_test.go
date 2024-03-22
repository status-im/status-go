package token

import (
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/network"
	mediaserver "github.com/status-im/status-go/server"
	"github.com/status-im/status-go/services/accounts/accountsevent"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/community"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/t/utils"
	"github.com/status-im/status-go/transactions/fake"
	"github.com/status-im/status-go/walletdatabase"
)

func setupTestTokenDB(t *testing.T) (*Manager, func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	return &Manager{
			db:                db,
			RPCClient:         nil,
			ContractMaker:     nil,
			networkManager:    nil,
			stores:            nil,
			communityTokensDB: nil,
			communityManager:  nil,
		}, func() {
			require.NoError(t, db.Close())
		}
}

func upsertCommunityToken(t *testing.T, token *Token, manager *Manager) {
	require.NotNil(t, token.CommunityData)

	err := manager.UpsertCustom(*token)
	require.NoError(t, err)

	// Community ID is only discovered by calling contract, so must be updated manually
	_, err = manager.db.Exec("UPDATE tokens SET community_id = ? WHERE address = ?", token.CommunityData.ID, token.Address)
	require.NoError(t, err)
}

func TestCustoms(t *testing.T) {
	manager, stop := setupTestTokenDB(t)
	defer stop()

	rst, err := manager.GetCustoms(false)
	require.NoError(t, err)
	require.Nil(t, rst)

	token := Token{
		Address:  common.Address{1},
		Name:     "Zilliqa",
		Symbol:   "ZIL",
		Decimals: 12,
		ChainID:  777,
	}

	err = manager.UpsertCustom(token)
	require.NoError(t, err)

	rst, err = manager.GetCustoms(false)
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, token, *rst[0])

	err = manager.DeleteCustom(777, token.Address)
	require.NoError(t, err)

	rst, err = manager.GetCustoms(false)
	require.NoError(t, err)
	require.Equal(t, 0, len(rst))
}

func TestCommunityTokens(t *testing.T) {
	manager, stop := setupTestTokenDB(t)
	defer stop()

	rst, err := manager.GetCustoms(true)
	require.NoError(t, err)
	require.Nil(t, rst)

	token := Token{
		Address:  common.Address{1},
		Name:     "Zilliqa",
		Symbol:   "ZIL",
		Decimals: 12,
		ChainID:  777,
	}

	err = manager.UpsertCustom(token)
	require.NoError(t, err)

	communityToken := Token{
		Address:  common.Address{2},
		Name:     "Communitia",
		Symbol:   "COM",
		Decimals: 12,
		ChainID:  777,
		CommunityData: &community.Data{
			ID: "random_community_id",
		},
	}

	upsertCommunityToken(t, &communityToken, manager)

	rst, err = manager.GetCustoms(false)
	require.NoError(t, err)
	require.Equal(t, 2, len(rst))
	require.Equal(t, token, *rst[0])
	require.Equal(t, communityToken, *rst[1])

	rst, err = manager.GetCustoms(true)
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, communityToken, *rst[0])
}

func toTokenMap(tokens []*Token) storeMap {
	tokenMap := storeMap{}

	for _, token := range tokens {
		addTokMap := tokenMap[token.ChainID]
		if addTokMap == nil {
			addTokMap = make(addressTokenMap)
		}

		addTokMap[token.Address] = token
		tokenMap[token.ChainID] = addTokMap
	}

	return tokenMap
}

func TestTokenOverride(t *testing.T) {
	networks := []params.Network{
		{
			ChainID:   1,
			ChainName: "TestChain1",
			TokenOverrides: []params.TokenOverride{
				{
					Symbol:  "SNT",
					Address: common.Address{11},
				},
			},
		}, {
			ChainID:   2,
			ChainName: "TestChain2",
			TokenOverrides: []params.TokenOverride{
				{
					Symbol:  "STT",
					Address: common.Address{33},
				},
			},
		},
	}

	tokenList := []*Token{
		&Token{
			Address: common.Address{1},
			Symbol:  "SNT",
			ChainID: 1,
		},
		&Token{
			Address: common.Address{2},
			Symbol:  "TNT",
			ChainID: 1,
		},
		&Token{
			Address: common.Address{3},
			Symbol:  "STT",
			ChainID: 2,
		},
		&Token{
			Address: common.Address{4},
			Symbol:  "TTT",
			ChainID: 2,
		},
	}
	testStore := &DefaultStore{
		tokenList,
	}

	overrideTokensInPlace(networks, tokenList)
	tokens := testStore.GetTokens()
	tokenMap := toTokenMap(tokens)
	_, found := tokenMap[1][common.Address{1}]
	require.False(t, found)
	require.Equal(t, common.Address{11}, tokenMap[1][common.Address{11}].Address)
	require.Equal(t, common.Address{2}, tokenMap[1][common.Address{2}].Address)
	_, found = tokenMap[2][common.Address{3}]
	require.False(t, found)
	require.Equal(t, common.Address{33}, tokenMap[2][common.Address{33}].Address)
	require.Equal(t, common.Address{4}, tokenMap[2][common.Address{4}].Address)
}

func TestMarkAsPreviouslyOwnedToken(t *testing.T) {
	manager, stop := setupTestTokenDB(t)
	defer stop()

	owner := common.HexToAddress("0x1234567890abcdef")
	token := &Token{
		Address:  common.HexToAddress("0xabcdef1234567890"),
		Name:     "TestToken",
		Symbol:   "TT",
		Decimals: 18,
		ChainID:  1,
	}

	isFirst, err := manager.MarkAsPreviouslyOwnedToken(nil, owner)
	require.Error(t, err)
	require.False(t, isFirst)

	isFirst, err = manager.MarkAsPreviouslyOwnedToken(token, common.Address{})
	require.Error(t, err)
	require.False(t, isFirst)

	isFirst, err = manager.MarkAsPreviouslyOwnedToken(token, owner)
	require.NoError(t, err)
	require.True(t, isFirst)

	// Verify that the token balance was inserted correctly
	var count int
	err = manager.db.QueryRow(`SELECT count(*) FROM token_balances`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	token.Name = "123"

	isFirst, err = manager.MarkAsPreviouslyOwnedToken(token, owner)
	require.NoError(t, err)
	require.False(t, isFirst)

	// Not updated because already exists
	err = manager.db.QueryRow(`SELECT count(*) FROM token_balances`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	token.ChainID = 2

	isFirst, err = manager.MarkAsPreviouslyOwnedToken(token, owner)
	require.NoError(t, err)

	// Same token on different chains counts as different token
	err = manager.db.QueryRow(`SELECT count(*) FROM token_balances`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.True(t, isFirst)
}

func TestGetTokenHistoricalBalance(t *testing.T) {
	manager, stop := setupTestTokenDB(t)
	defer stop()

	account := common.HexToAddress("0x1234567890abcdef")
	chainID := uint64(1)
	testSymbol := "TEST"
	block := int64(1)
	timestamp := int64(1629878400) // Replace with desired timestamp
	historyBalance := big.NewInt(0)

	// Test case when no rows are returned
	balance, err := manager.GetTokenHistoricalBalance(account, chainID, testSymbol, timestamp)
	require.NoError(t, err)
	require.Nil(t, balance)

	// Test case when a row is returned
	historyBalance.SetInt64(int64(100))
	_, err = manager.db.Exec("INSERT INTO balance_history (currency, chain_id, address, timestamp, balance, block) VALUES (?, ?, ?, ?, ?, ?)", testSymbol, chainID, account, timestamp-100, (*bigint.SQLBigIntBytes)(historyBalance), block)
	require.NoError(t, err)

	expectedBalance := big.NewInt(100)
	balance, err = manager.GetTokenHistoricalBalance(account, chainID, testSymbol, timestamp)
	require.NoError(t, err)
	require.Equal(t, expectedBalance, balance)

	// Test multiple values. Must return the most recent one
	historyBalance.SetInt64(int64(100))
	_, err = manager.db.Exec("INSERT INTO balance_history (currency, chain_id, address, timestamp, balance, block) VALUES (?, ?, ?, ?, ?, ?)", testSymbol, chainID, account, timestamp-200, (*bigint.SQLBigIntBytes)(historyBalance), block)
	require.NoError(t, err)

	historyBalance.SetInt64(int64(50))
	symbol := "TEST2"
	_, err = manager.db.Exec("INSERT INTO balance_history (currency, chain_id, address, timestamp, balance, block) VALUES (?, ?, ?, ?, ?, ?)", symbol, chainID, account, timestamp-1, (*bigint.SQLBigIntBytes)(historyBalance), block)
	require.NoError(t, err)

	historyBalance.SetInt64(int64(50))
	chainID = uint64(2)
	_, err = manager.db.Exec("INSERT INTO balance_history (currency, chain_id, address, timestamp, balance, block) VALUES (?, ?, ?, ?, ?, ?)", testSymbol, chainID, account, timestamp-1, (*bigint.SQLBigIntBytes)(historyBalance), block)
	require.NoError(t, err)

	chainID = uint64(1)
	balance, err = manager.GetTokenHistoricalBalance(account, chainID, testSymbol, timestamp)
	require.NoError(t, err)
	require.Equal(t, expectedBalance, balance)
}

func Test_removeTokenBalanceOnEventAccountRemoved(t *testing.T) {
	appDB, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	walletDB, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	accountsDB, err := accounts.NewDB(appDB)
	require.NoError(t, err)

	address := common.HexToAddress("0x1234")
	accountFeed := event.Feed{}
	chainID := uint64(1)
	txServiceMockCtrl := gomock.NewController(t)
	server, _ := fake.NewTestServer(txServiceMockCtrl)
	client := gethrpc.DialInProc(server)
	rpcClient, _ := rpc.NewClient(client, chainID, params.UpstreamRPCConfig{}, nil, nil)
	rpcClient.UpstreamChainID = chainID
	nm := network.NewManager(appDB)
	mediaServer, err := mediaserver.NewMediaServer(appDB, nil, nil, walletDB)
	require.NoError(t, err)

	manager := NewTokenManager(walletDB, rpcClient, nil, nm, appDB, mediaServer, nil, &accountFeed, accountsDB)

	// Insert balances for address
	marked, err := manager.MarkAsPreviouslyOwnedToken(&Token{
		Address:  common.HexToAddress("0x1234"),
		Symbol:   "Dummy",
		Decimals: 18,
		ChainID:  1,
	}, address)
	require.NoError(t, err)
	require.True(t, marked)

	tokenByAddress, err := manager.GetPreviouslyOwnedTokens()
	require.NoError(t, err)
	require.Len(t, tokenByAddress, 1)

	// Start service
	manager.startAccountsWatcher()

	// Watching accounts must start before sending event.
	// To avoid running goroutine immediately and let the controller subscribe first,
	// use any delay.
	group := sync.WaitGroup{}
	group.Add(1)
	go func() {
		defer group.Done()
		time.Sleep(1 * time.Millisecond)

		accountFeed.Send(accountsevent.Event{
			Type:     accountsevent.EventTypeRemoved,
			Accounts: []common.Address{address},
		})

		require.NoError(t, utils.Eventually(func() error {
			tokenByAddress, err := manager.GetPreviouslyOwnedTokens()
			if err == nil && len(tokenByAddress) == 0 {
				return nil
			}
			return errors.New("Token not removed")
		}, 100*time.Millisecond, 10*time.Millisecond))
	}()

	group.Wait()

	// Stop service
	txServiceMockCtrl.Finish()
	server.Stop()
	manager.stopAccountsWatcher()
}
