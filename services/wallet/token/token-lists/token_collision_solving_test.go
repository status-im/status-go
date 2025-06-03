package tokenlists

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/token/token-lists/fetcher"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
)

func TestRemoveDuplicates(t *testing.T) {
	tokens := []*tokenTypes.Token{
		{
			ChainID: 1,
			Address: common.HexToAddress("0x1"),
			Symbol:  "ETH",
		},
		{
			ChainID: 1,
			Address: common.HexToAddress("0x2"),
			Symbol:  "ETH",
		},
		{
			ChainID: 2,
			Address: common.HexToAddress("0x1"),
			Symbol:  "ETH",
		},
	}

	tokens1 := []*tokenTypes.Token{
		{
			ChainID: 1,
			Address: common.HexToAddress("0x1"),
			Symbol:  "ETH 1",
		},
		{
			ChainID: 1,
			Address: common.HexToAddress("0x2"),
			Symbol:  "ETH 2",
		},
		{
			ChainID: 2,
			Address: common.HexToAddress("0x1"),
			Symbol:  "ETH 3",
		},
	}

	deDuplicatedTokens := removeDuplicateSymbolOnTheSameChain(tokens)
	require.Len(t, deDuplicatedTokens, 2)
	require.True(t, tokens[0] == deDuplicatedTokens[0] || tokens[0] == deDuplicatedTokens[1])
	require.True(t, tokens[2] == deDuplicatedTokens[0] || tokens[2] == deDuplicatedTokens[1])

	filteredTokens := removeTokenIfAppearsInTheReferenceList(tokens1, deDuplicatedTokens)
	require.Len(t, filteredTokens, 1)
	require.Equal(t, filteredTokens[0], tokens1[1])
}

func TestTokenExistence(t *testing.T) {
	tokensMap := map[string][]*tokenTypes.Token{
		"ETH1": {
			{
				ChainID:   1,
				Address:   common.HexToAddress("0x1"),
				Symbol:    "ETH1",
				TmpSymbol: "ETH1",
				Decimals:  18,
			},
			{
				ChainID:   2,
				Address:   common.HexToAddress("0x2"),
				Symbol:    "ETH1",
				TmpSymbol: "ETH1",
				Decimals:  18,
			},
		},
		"ETH2": {
			{
				ChainID:   1,
				Address:   common.HexToAddress("0x3"),
				Symbol:    "ETH2",
				TmpSymbol: "ETH2",
				Decimals:  6,
			},
		},
	}

	allReferenceTokens := getTokensForSymbolFromMap("", nil)
	require.Len(t, allReferenceTokens, 0)

	allReferenceTokens = getTokensForSymbolFromMap("ETH1", nil)
	require.Len(t, allReferenceTokens, 0)

	allReferenceTokens = getTokensForSymbolFromMap("ETH1", tokensMap)
	require.Len(t, allReferenceTokens, 1)
	require.Len(t, allReferenceTokens[0], 2)

	require.False(t, tokenWithChainIdExistsInList(5, allReferenceTokens[0]))
	require.True(t, tokenWithChainIdExistsInList(1, allReferenceTokens[0]))
}

func TestSolvingDecimalsCollision(t *testing.T) {
	tokens := []*tokenTypes.Token{
		{
			ChainID:  walletCommon.BSCMainnet,
			Address:  common.HexToAddress("0x1"),
			Symbol:   "ETH1",
			Decimals: 18,
		},
		{
			ChainID:  walletCommon.OptimismMainnet,
			Address:  common.HexToAddress("0x2"),
			Symbol:   "ETH1",
			Decimals: 6,
		},
		{
			ChainID:  walletCommon.ArbitrumMainnet,
			Address:  common.HexToAddress("0x3"),
			Symbol:   "ETH1",
			Decimals: 6,
		},
		{
			ChainID:  walletCommon.EthereumMainnet,
			Address:  common.HexToAddress("0x4"),
			Symbol:   "ETH2",
			Decimals: 18,
		},
		{
			ChainID:  walletCommon.OptimismMainnet,
			Address:  common.HexToAddress("0x5"),
			Symbol:   "ETH2",
			Decimals: 18,
		},
	}

	tokensReferenceMap := map[string][]*tokenTypes.Token{
		makeUniqueSymbol(tokens[2]): {
			{
				ChainID:   walletCommon.ArbitrumMainnet,
				Address:   common.HexToAddress("0x3"),
				Symbol:    makeUniqueSymbol(tokens[2]),
				TmpSymbol: tokens[2].Symbol,
				Decimals:  6,
			},
		},
		"ETH2": {
			{
				ChainID:   walletCommon.EthereumMainnet,
				Address:   common.HexToAddress("0x4"),
				Symbol:    "ETH2",
				TmpSymbol: tokens[3].Symbol,
				Decimals:  18,
			},
		},
	}

	// test with empty tokens list and reference map
	cleanedMap := solveDecimalsCollision(nil, nil)
	require.Len(t, cleanedMap, 0)

	// test with tokens list and empty reference map
	cleanedMap = solveDecimalsCollision(tokens, nil)
	expectedToken0 := *tokens[0]
	expectedToken0.TmpSymbol = expectedToken0.Symbol
	expectedToken0.Symbol = makeUniqueSymbol(tokens[0])
	expectedToken1 := *tokens[1]
	expectedToken1.TmpSymbol = expectedToken1.Symbol
	expectedToken1.Symbol = makeUniqueSymbol(tokens[1])
	expectedToken2 := *tokens[2]
	expectedToken2.Symbol = makeUniqueSymbol(tokens[2])
	require.Len(t, cleanedMap, 3)
	require.Len(t, cleanedMap[makeUniqueSymbol(tokens[0])], 1)
	require.Equal(t, cleanedMap[makeUniqueSymbol(tokens[0])][0], &expectedToken0)
	require.Len(t, cleanedMap[makeUniqueSymbol(tokens[1])], 2)
	require.True(t, *cleanedMap[makeUniqueSymbol(tokens[1])][0] == expectedToken1 || *cleanedMap[makeUniqueSymbol(tokens[1])][0] == expectedToken2)
	require.True(t, *cleanedMap[makeUniqueSymbol(tokens[1])][1] == expectedToken1 || *cleanedMap[makeUniqueSymbol(tokens[1])][1] == expectedToken2)
	require.Len(t, cleanedMap["ETH2"], 2)
	require.True(t, cleanedMap["ETH2"][0] == tokens[3] || cleanedMap["ETH2"][0] == tokens[4])
	require.True(t, cleanedMap["ETH2"][1] == tokens[3] || cleanedMap["ETH2"][1] == tokens[4])

	// test with tokens list and reference map
	cleanedMap = solveDecimalsCollision(tokens, tokensReferenceMap)
	require.Len(t, cleanedMap, 3)
	require.Len(t, cleanedMap[makeUniqueSymbol(tokens[0])], 1)
	require.Equal(t, cleanedMap[makeUniqueSymbol(tokens[0])][0], &expectedToken0)
	require.Len(t, cleanedMap[makeUniqueSymbol(tokens[1])], 1)
	require.Equal(t, cleanedMap[makeUniqueSymbol(tokens[1])][0], &expectedToken1)
	require.Len(t, cleanedMap["ETH2"], 1)
	require.Equal(t, cleanedMap["ETH2"][0], tokens[4])
}

func TestSolvingCollision(t *testing.T) {
	var tokensLists *TokenLists
	appDb, closeAppDb := setupTestAppDB(t)
	walletDb, closeWalletDb := fetcher.SetupTestWalletDB(t)
	server, closeServer := fetcher.GetTestServer()

	t.Cleanup(func() {
		tokensLists.Stop()
		closeAppDb()
		closeWalletDb()
		closeServer()
	})

	// init settings with auto-refresh disabled
	settingsDB, err := initSettings(appDb, false)
	require.NoError(t, err)
	require.NotNil(t, settingsDB)

	tokensLists, err = NewTokenLists(appDb, walletDb)
	require.NoError(t, err)

	lastUpdate, err := tokensLists.LastTokensUpdate()
	require.NoError(t, err)
	require.True(t, lastUpdate.IsZero())

	// before starting the auto-refresh, we should have no token lists
	allTokens := tokensLists.GetUniqueTokens()
	require.NoError(t, err)
	require.Len(t, allTokens, 0)

	// start the auto-refresh process with a 5 second interval, and 1 second check interval
	const autoRefreshInterval = 3 * time.Second
	const autoRefreshCheckInterval = 1 * time.Second

	tokensLists.Start(context.Background(), server.URL, autoRefreshInterval, autoRefreshCheckInterval)

	// the token lists should not be updated
	allTokens = tokensLists.GetUniqueTokens()
	tokensBySymbol := make(map[string][]*tokenTypes.Token) // map[symbol][]*tokenTypes.Token
	for _, token := range allTokens {
		tokensBySymbol[token.Symbol] = append(tokensBySymbol[token.Symbol], token)
	}

	// no tokens for the same symbol should have different decimals or the same chainId
	for _, tokens := range tokensBySymbol {
		require.True(t, len(tokens) > 0)

		decimalsMap := make(map[uint]struct{})  // map[decimals]struct{}
		chainIdMap := make(map[uint64]struct{}) // map[chainId]struct{}
		for _, token := range tokens {
			decimalsMap[token.Decimals] = struct{}{}
			_, ok := chainIdMap[token.ChainID]
			require.False(t, ok)
			chainIdMap[token.ChainID] = struct{}{}
		}
		require.Len(t, decimalsMap, 1)
	}
}
