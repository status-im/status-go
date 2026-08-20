package token

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/status-im/status-go/internal/db/walletdatabase"
	"github.com/status-im/status-go/internal/testutils"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

func TestSaveTokens(t *testing.T) {
	db, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	require.NotNil(t, db)

	persistence := balanceStorage{walletDB: db}
	require.NotNil(t, persistence)

	storageTokens := make(map[common.Address][]tokentypes.StorageToken)
	address1 := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	address2 := common.HexToAddress("0x5e4e65926ba27467555eb562121fac00d24e9dd2")

	tokenAddress11 := common.HexToAddress("0xDb8d79C775452a3929b86ac5DEaB3e9d38e1c006")
	tokenAddress12 := common.HexToAddress("0xDb8d79C775452a3929b86ac5DEaB3e9d38e1c007")
	tokenAddress21 := common.HexToAddress("0xDb8d79C775452a3929b86ac5DEaB3e9d38e1c008")
	tokenAddress31 := common.HexToAddress("0xDb8d79C775452a3929b86ac5DEaB3e9d38e1c009")

	var chain1 uint64 = 1
	var chain2 uint64 = 2

	// token 1 on chain 1
	token11 := tokentypes.StorageToken{
		TokenAddress:    tokenAddress11,
		TokenChainID:    chain1,
		RawBalance:      "1",
		Balance:         big.NewFloat(0.1),
		Description:     "description-1",
		AssetWebsiteURL: "url-1",
	}

	// token 1 on chain 2
	token12 := tokentypes.StorageToken{
		TokenAddress:    tokenAddress12,
		TokenChainID:    chain2,
		RawBalance:      "2",
		Balance:         big.NewFloat(0.2),
		Description:     "description-1",
		AssetWebsiteURL: "url-1",
	}

	// token 2 on chain 1
	token21 := tokentypes.StorageToken{
		TokenAddress:    tokenAddress21,
		TokenChainID:    chain1,
		RawBalance:      "3",
		Balance:         big.NewFloat(0.3),
		Description:     "description-2",
		AssetWebsiteURL: "url-2",
	}

	// token 3 on chain 1
	token31 := tokentypes.StorageToken{
		TokenAddress:    tokenAddress31,
		TokenChainID:    chain1,
		RawBalance:      "4",
		Balance:         big.NewFloat(0.4),
		Description:     "description-3",
		AssetWebsiteURL: "url-3",
	}

	storageTokens[address1] = []tokentypes.StorageToken{token11, token12, token21}

	storageTokens[address2] = []tokentypes.StorageToken{token31}

	err = persistence.saveBalances(storageTokens)
	require.NoError(t, err)

	actualTokens, err := persistence.getBalances()
	require.NoError(t, err)
	require.NotNil(t, actualTokens)
	require.Len(t, actualTokens[address1], 3)
	require.Len(t, actualTokens[address2], 1)

	var actualToken1, actualToken2, actualToken3 tokentypes.StorageToken
	for _, token := range actualTokens[address1] {
		if token.TokenAddress == tokenAddress11 && token.TokenChainID == chain1 {
			actualToken1 = token
		} else if token.TokenAddress == tokenAddress12 && token.TokenChainID == chain2 {
			actualToken2 = token
		} else if token.TokenAddress == tokenAddress21 && token.TokenChainID == chain1 {
			actualToken3 = token
		}
	}

	actualToken4 := actualTokens[address2][0]

	sameTokens := func(token1, token2 tokentypes.StorageToken) bool {
		return token1.TokenAddress == token2.TokenAddress &&
			token1.TokenChainID == token2.TokenChainID &&
			token1.RawBalance == token2.RawBalance &&
			token1.Balance.String() == token2.Balance.String() &&
			token1.HasError == token2.HasError &&
			token1.Description == token2.Description &&
			token1.AssetWebsiteURL == token2.AssetWebsiteURL &&
			token1.BuiltOn == token2.BuiltOn &&
			len(token1.MarketValuesPerCurrency) == len(token2.MarketValuesPerCurrency)
	}

	require.True(t, sameTokens(actualToken1, token11))
	require.True(t, sameTokens(actualToken2, token12))
	require.True(t, sameTokens(actualToken3, token21))
	require.True(t, sameTokens(actualToken4, token31))
}

func TestGetCachedBalancesByChain(t *testing.T) {
	walletDB, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	persistence := balanceStorage{walletDB: walletDB}
	require.NotNil(t, persistence)

	storageTokens := make(map[common.Address][]tokentypes.StorageToken)
	address1 := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	address2 := common.HexToAddress("0x5e4e65926ba27467555eb562121fac00d24e9dd2")

	tokenAddress1 := common.HexToAddress("0xDb8d79C775452a3929b86ac5DEaB3e9d38e1c006")
	tokenAddress2 := common.HexToAddress("0xDb8d79C775452a3929b86ac5DEaB3e9d38e1c005")

	var chain1 uint64 = 1
	var chain2 uint64 = 2

	token1 := tokentypes.StorageToken{
		TokenAddress:    tokenAddress1,
		TokenChainID:    chain1,
		RawBalance:      "1",
		Balance:         big.NewFloat(0.000000000000000001),
		Description:     "description-1",
		AssetWebsiteURL: "url-1",
	}

	token2 := tokentypes.StorageToken{
		TokenAddress:    tokenAddress2,
		TokenChainID:    chain2,
		RawBalance:      "1000000000000000000",
		Balance:         big.NewFloat(1),
		Description:     "description-2",
		AssetWebsiteURL: "url-2",
	}

	storageTokens[address1] = []tokentypes.StorageToken{token1}
	storageTokens[address2] = []tokentypes.StorageToken{token2}

	err = persistence.saveBalances(storageTokens)
	require.NoError(t, err)

	// Verify that the token balance was inserted correctly
	var count int
	err = walletDB.QueryRow(`SELECT count(*) FROM token_balances`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	tokens := []*tokentypes.Token{{Token: &types.Token{ChainID: chain1, Address: tokenAddress1}}}

	nonExistingAddress := common.HexToAddress("0xaAC17F958D2ee523a2206206994597C13D831ec8")
	result, err := persistence.getCachedBalancesByChain([]common.Address{nonExistingAddress}, tokens)
	require.NoError(t, err)
	require.Len(t, result, 0)

	result, err = persistence.getCachedBalancesByChain([]common.Address{address1}, tokens)
	require.NoError(t, err)
	require.Len(t, result, 1)

	require.Equal(t, result[chain1][address1][tokenAddress1].ToInt(), big.NewInt(1))

	tokens = append(tokens, &tokentypes.Token{Token: &types.Token{ChainID: chain2, Address: tokenAddress2}})

	result, err = persistence.getCachedBalancesByChain([]common.Address{address1, address2}, tokens)
	require.NoError(t, err)
	require.Len(t, result, 2)

	require.Equal(t, result[chain1][address1][tokenAddress1].ToInt(), big.NewInt(1))
	require.Equal(t, result[chain2][address2][tokenAddress2].ToInt(), big.NewInt(1000000000000000000))
}
