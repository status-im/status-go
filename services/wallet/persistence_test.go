package wallet

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/ethereum/go-ethereum/common"
)

func TestSaveTokens(t *testing.T) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})

	require.NoError(t, err)
	require.NotNil(t, db)

	persistence := NewPersistence(db)
	require.NotNil(t, persistence)

	tokens := make(map[common.Address][]Token)
	address1 := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	address2 := common.HexToAddress("0x5e4e65926ba27467555eb562121fac00d24e9dd2")

	tokenAddress1 := common.HexToAddress("0xDb8d79C775452a3929b86ac5DEaB3e9d38e1c006")
	tokenAddress2 := common.HexToAddress("0xDb8d79C775452a3929b86ac5DEaB3e9d38e1c005")

	var chain1 uint64 = 1
	var chain2 uint64 = 2

	token1 := Token{
		Name:             "token-1",
		Symbol:           "TT1",
		Decimals:         10,
		BalancesPerChain: make(map[uint64]ChainBalance),
		Description:      "description-1",
		AssetWebsiteURL:  "url-1",
	}

	token1.BalancesPerChain[chain1] = ChainBalance{
		RawBalance: "1",
		Balance:    big.NewFloat(0.1),
		Address:    tokenAddress1,
		ChainID:    chain1,
	}

	token1.BalancesPerChain[chain2] = ChainBalance{
		RawBalance: "2",
		Balance:    big.NewFloat(0.2),
		Address:    tokenAddress2,
		ChainID:    chain2,
	}

	token2 := Token{
		Name:             "token-2",
		Symbol:           "TT2",
		Decimals:         11,
		BalancesPerChain: make(map[uint64]ChainBalance),
		Description:      "description-2",
		AssetWebsiteURL:  "url-2",
	}

	token2.BalancesPerChain[chain1] = ChainBalance{
		RawBalance: "3",
		Balance:    big.NewFloat(0.3),
		Address:    tokenAddress1,
		ChainID:    chain1,
	}

	token3 := Token{
		Name:             "token-3",
		Symbol:           "TT3",
		Decimals:         11,
		BalancesPerChain: make(map[uint64]ChainBalance),
		Description:      "description-3",
		AssetWebsiteURL:  "url-3",
	}

	token3.BalancesPerChain[chain1] = ChainBalance{
		RawBalance: "4",
		Balance:    big.NewFloat(0.4),
		Address:    tokenAddress1,
		ChainID:    chain1,
	}

	tokens[address1] = []Token{token1, token2}

	tokens[address2] = []Token{token3}

	require.NoError(t, persistence.SaveTokens(tokens))

	actualTokens, err := persistence.GetTokens()
	require.NoError(t, err)
	require.NotNil(t, actualTokens)
	require.NotNil(t, actualTokens[address1])
	require.Len(t, actualTokens[address1], 2)

	var actualToken1, actualToken2, actualToken3 Token
	if actualTokens[address1][0].Name == "token-1" {
		actualToken1 = actualTokens[address1][0]
		actualToken2 = actualTokens[address1][1]
	} else {
		actualToken1 = actualTokens[address1][1]
		actualToken2 = actualTokens[address1][0]

	}

	require.NotNil(t, actualTokens[address2])
	require.Len(t, actualTokens[address2], 1)

	actualToken3 = actualTokens[address2][0]

	require.Equal(t, actualToken1.Name, token1.Name)
	require.Equal(t, actualToken1.Symbol, token1.Symbol)
	require.Equal(t, actualToken1.Decimals, token1.Decimals)
	require.Equal(t, actualToken1.Description, token1.Description)
	require.Equal(t, actualToken1.AssetWebsiteURL, token1.AssetWebsiteURL)

	require.Equal(t, actualToken1.BalancesPerChain[chain1].RawBalance, "1")
	require.NotNil(t, actualToken1.BalancesPerChain[chain1].Balance)
	require.Equal(t, actualToken1.BalancesPerChain[chain1].Balance.String(), "0.1")
	require.Equal(t, actualToken1.BalancesPerChain[chain1].Address, tokenAddress1)
	require.Equal(t, actualToken1.BalancesPerChain[chain1].ChainID, chain1)

	require.Equal(t, actualToken1.BalancesPerChain[chain2].RawBalance, "2")
	require.NotNil(t, actualToken1.BalancesPerChain[chain2].Balance)
	require.Equal(t, actualToken1.BalancesPerChain[chain2].Balance.String(), "0.2")
	require.Equal(t, actualToken1.BalancesPerChain[chain2].Address, tokenAddress2)
	require.Equal(t, actualToken1.BalancesPerChain[chain2].ChainID, chain2)

	require.Equal(t, actualToken2.Name, token2.Name)
	require.Equal(t, actualToken2.Symbol, token2.Symbol)
	require.Equal(t, actualToken2.Decimals, token2.Decimals)
	require.Equal(t, actualToken2.Description, token2.Description)
	require.Equal(t, actualToken2.AssetWebsiteURL, token2.AssetWebsiteURL)

	require.Equal(t, actualToken2.BalancesPerChain[chain1].RawBalance, "3")
	require.NotNil(t, actualToken2.BalancesPerChain[chain1].Balance)
	require.Equal(t, actualToken2.BalancesPerChain[chain1].Balance.String(), "0.3")
	require.Equal(t, actualToken2.BalancesPerChain[chain1].Address, tokenAddress1)
	require.Equal(t, actualToken2.BalancesPerChain[chain1].ChainID, chain1)

	require.Equal(t, actualToken3.Name, token3.Name)
	require.Equal(t, actualToken3.Symbol, token3.Symbol)
	require.Equal(t, actualToken3.Decimals, token3.Decimals)
	require.Equal(t, actualToken3.Description, token3.Description)
	require.Equal(t, actualToken3.AssetWebsiteURL, token3.AssetWebsiteURL)

	require.Equal(t, actualToken3.BalancesPerChain[chain1].RawBalance, "4")
	require.NotNil(t, actualToken3.BalancesPerChain[chain1].Balance)
	require.Equal(t, actualToken3.BalancesPerChain[chain1].Balance.String(), "0.4")
	require.Equal(t, actualToken3.BalancesPerChain[chain1].Address, tokenAddress1)
	require.Equal(t, actualToken3.BalancesPerChain[chain1].ChainID, chain1)
}
