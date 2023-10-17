package token

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

func setupTestTokenDB(t *testing.T) (*Manager, func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	return &Manager{
			db:             db,
			RPCClient:      nil,
			contractMaker:  nil,
			networkManager: nil,
			stores:         nil,
		}, func() {
			require.NoError(t, db.Close())
		}
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
