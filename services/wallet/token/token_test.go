package token

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/params"
)

func setupTestTokenDB(t *testing.T) (*Manager, func()) {
	db, err := appdatabase.InitializeDB(":memory:", "wallet-token-tests", 1)
	require.NoError(t, err)
	return &Manager{db, nil, nil, nil, nil, nil, false}, func() {
		require.NoError(t, db.Close())
	}
}

func TestCustoms(t *testing.T) {
	manager, stop := setupTestTokenDB(t)
	defer stop()

	rst, err := manager.GetCustoms()
	require.NoError(t, err)
	require.Nil(t, rst)

	token := Token{
		Address:  common.Address{1},
		Name:     "Zilliqa",
		Symbol:   "ZIL",
		Decimals: 12,
		Color:    "#fa6565",
		ChainID:  777,
	}

	err = manager.UpsertCustom(token)
	require.NoError(t, err)

	rst, err = manager.GetCustoms()
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, token, *rst[0])

	err = manager.DeleteCustom(777, token.Address)
	require.NoError(t, err)

	rst, err = manager.GetCustoms()
	require.NoError(t, err)
	require.Equal(t, 0, len(rst))
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

	overrideTokensInPlace(networks, testStore.tokenList)
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

func TestMergeTokenLists(t *testing.T) {
	tokenList1 := []*Token{
		&Token{
			Address: common.Address{1},
			Symbol:  "SNT",
			ChainID: 1,
		},
	}
	tokenList1Copy := []*Token{
		&Token{
			Address: common.Address{1},
			Symbol:  "SNT",
			ChainID: 1,
		},
	}
	tokenList2 := []*Token{
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
	tokenList1Plus2 := []*Token{
		&Token{
			Address: common.Address{1},
			Symbol:  "SNT",
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
	tokenListEmpty := []*Token{}

	mergedList := mergeTokenLists([][]*Token{tokenListEmpty, tokenListEmpty})
	require.Equal(t, 0, len(mergedList))

	mergedList = mergeTokenLists([][]*Token{tokenListEmpty, tokenList1})
	require.True(t, reflect.DeepEqual(mergedList, tokenList1))

	mergedList = mergeTokenLists([][]*Token{tokenList1, tokenList1Copy})
	require.True(t, reflect.DeepEqual(mergedList, tokenList1))

	mergedList = mergeTokenLists([][]*Token{tokenList1, tokenList2})
	require.True(t, reflect.DeepEqual(mergedList, tokenList1Plus2))
}
