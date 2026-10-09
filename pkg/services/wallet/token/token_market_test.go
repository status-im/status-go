package token

import (
	"context"
	"encoding/json"
	"maps"
	"math/rand"
	"slices"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/db/walletdb"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

// setupMarketTestManager loads the embedded lists for every wallet chain, with
// testnet mode as given.
func setupMarketTestManager(t testing.TB, testnet bool) *Manager {
	appDB, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	walletDB, err := testutils.SetupTestMemorySQLDB(walletdb.DbInitializer{})
	require.NoError(t, err)
	settingsDB, err := settings.MakeNewDB(appDB)
	require.NoError(t, err)
	networks := json.RawMessage(`{}`)
	require.NoError(t, settingsDB.CreateSettings(settings.Settings{Networks: &networks, TestNetworksEnabled: testnet}, params.NodeConfig{}))
	m := &Manager{walletDB: walletDB, settings: settingsDB, tokenBalancesStorage: balanceStorage{walletDB: walletDB}}
	facade, err := newTKLReadManager(m, walletcommon.AllChainIDsAsUint64(), time.Time{})
	require.NoError(t, err)
	require.NoError(t, facade.Start(context.Background(), false, nil))
	m.tokensManager = facade
	t.Cleanup(func() {
		require.NoError(t, facade.Stop())
		require.NoError(t, appDB.Close())
		require.NoError(t, walletDB.Close())
	})
	mode, err := settingsDB.GetTestNetworksEnabled()
	require.NoError(t, err)
	require.Equal(t, testnet, mode)
	return m
}

// legacyAddTokensSharingCrossChainIDs is addTokensSharingCrossChainIDsToUsedTokenKeys
// before the narrow query: a scan of every catalogue token.
func legacyAddTokensSharingCrossChainIDs(tm *Manager, usedTokensKeys map[string]interface{}, testnetMode bool) error {
	tokens, err := tm.GetTokensByKeys(slices.Collect(maps.Keys(usedTokensKeys)))
	if err != nil {
		return err
	}
	crossChainIDs := make([]string, 0)
	for _, token := range tokens {
		if token.CrossChainID != "" {
			crossChainIDs = append(crossChainIDs, token.CrossChainID)
		}
	}
	if len(crossChainIDs) == 0 {
		return nil
	}
	tokensByCrossChainIDs := make(map[string][]*tokentypes.Token)
	for _, token := range tm.tokensManager.UniqueTokens() {
		if token.CrossChainID == "" ||
			testnetMode && walletcommon.ChainID(token.ChainID).IsMainnet() ||
			!testnetMode && !walletcommon.ChainID(token.ChainID).IsMainnet() {
			continue
		}
		tokensByCrossChainIDs[token.CrossChainID] = append(tokensByCrossChainIDs[token.CrossChainID], &tokentypes.Token{Token: token})
	}
	for _, crossChainID := range crossChainIDs {
		for _, token := range tokensByCrossChainIDs[crossChainID] {
			usedTokensKeys[token.Key()] = nil
		}
	}
	return nil
}

// legacyTestnetTokensForMarketData is the testnet branch of
// GetTokensByKeysForFetchingMarketData before the narrow query.
func legacyTestnetTokensForMarketData(tm *Manager, tokenKeys []string) ([]*tokentypes.Token, error) {
	mainnetTokenKeysByCrossChainIDs := make(map[string][]string, 0)
	tokens := make([]*tokentypes.Token, 0)
	for _, token := range tm.tokensManager.UniqueTokens() {
		if token.CrossChainID != "" && walletcommon.ChainID(token.ChainID).IsMainnet() {
			mainnetTokenKeysByCrossChainIDs[token.CrossChainID] = append(mainnetTokenKeysByCrossChainIDs[token.CrossChainID], token.Key())
		}
		if !slices.Contains(tokenKeys, token.Key()) {
			continue
		}
		tokens = append(tokens, &tokentypes.Token{Token: token})
	}
	mainnetTokenKeys := make([]string, 0)
	for _, token := range tokens {
		crossChainID := token.CrossChainID
		if crossChainID == "" {
			continue
		}
		if crossChainID == walletcommon.StatusTestTokenCrossChainID {
			crossChainID = walletcommon.StatusMainnetTokenCrossChainID
		}
		mainnetTokenKeys = append(mainnetTokenKeys, mainnetTokenKeysByCrossChainIDs[crossChainID]...)
	}
	mainnetTokens, err := tm.GetTokensByKeys(mainnetTokenKeys)
	if err != nil {
		return nil, err
	}
	return append(tokens, mainnetTokens...), nil
}

// marketKeySets are the used-key sets the parity tests replay: single tokens,
// a spread sample, everything, random subsets, unknown, aliased and
// mixed-case keys, and community customs.
func marketKeySets(t *testing.T, tm *Manager) [][]string {
	all, err := tm.GetAllTokens()
	require.NoError(t, err)
	require.Greater(t, len(all), 1000)
	keys := make([]string, len(all))
	for i, token := range all {
		keys[i] = token.Key()
	}
	sets := [][]string{nil, {}, keys, {"1-0x0000000000000000000000000000000000000000"}, {"1-0xEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE"},
		{"1-0xC02AAA39B223FE8D0A0E5C4F27EAD9083C756CC2", "1-0x744d70fdbe2ba4cf95131626614a1763df805b9e"},
		{"999999-0x0000000000000000000000000000000000000001", "not-a-key", ""}}
	sample := []string{}
	for i := 0; i < len(keys); i += len(keys) / 60 {
		sample = append(sample, keys[i])
	}
	sets = append(sets, sample, append(slices.Clone(sample), sample...))
	for _, token := range all {
		switch token.CrossChainID {
		case walletcommon.StatusTestTokenCrossChainID, walletcommon.StatusMainnetTokenCrossChainID:
			sets = append(sets, []string{token.Key()})
		}
	}
	random := rand.New(rand.NewSource(7)) // nolint: gosec
	for range 25 {
		subset := make([]string, 0, 20)
		for range 20 {
			subset = append(subset, keys[random.Intn(len(keys))])
		}
		sets = append(sets, subset)
	}
	return sets
}

func addCommunityCustomWithCrossChainID(t *testing.T, tm *Manager, chainID uint64, crossChainID string) string {
	token := &tokentypes.Token{
		Token:         &types.Token{ChainID: chainID, Address: common.HexToAddress("0xc0ffee"), Symbol: "COMM", Name: "Community", Decimals: 18, CrossChainID: crossChainID},
		CommunityData: &tokentypes.CommunityData{ID: "community"},
	}
	upsertCommunityToken(t, token, tm)
	return token.Key()
}

func TestAddTokensSharingCrossChainIDsMatchesCatalogueScan(t *testing.T) {
	for _, testnet := range []bool{false, true} {
		tm := setupMarketTestManager(t, testnet)
		sets := marketKeySets(t, tm)
		sets = append(sets, []string{addCommunityCustomWithCrossChainID(t, tm, walletcommon.EthereumMainnet, "usd-coin")})
		for _, set := range sets {
			legacy := map[string]interface{}{}
			current := map[string]interface{}{}
			for _, key := range set {
				legacy[key] = nil
				current[key] = nil
			}
			require.NoError(t, legacyAddTokensSharingCrossChainIDs(tm, legacy, testnet))
			require.NoError(t, tm.addTokensSharingCrossChainIDsToUsedTokenKeys(current, testnet))
			require.Equal(t, legacy, current, "testnet=%v keys=%v", testnet, set)
		}
	}
}

func tokensByKey(t *testing.T, tokens []*tokentypes.Token) map[string]tokentypes.Token {
	result := make(map[string]tokentypes.Token, len(tokens))
	for _, token := range tokens {
		if previous, ok := result[token.Key()]; ok {
			require.Equal(t, previous, *token, token.Key())
		}
		result[token.Key()] = *token
	}
	return result
}

func TestTestnetTokensForMarketDataMatchCatalogueScan(t *testing.T) {
	tm := setupMarketTestManager(t, true)
	sets := marketKeySets(t, tm)
	sets = append(sets, []string{addCommunityCustomWithCrossChainID(t, tm, walletcommon.EthereumSepolia, walletcommon.StatusTestTokenCrossChainID)})
	stt := 0
	for _, set := range sets {
		legacy, err := legacyTestnetTokensForMarketData(tm, set)
		require.NoError(t, err)
		current, err := tm.GetTokensByKeysForFetchingMarketData(set)
		require.NoError(t, err)
		require.Equal(t, tokensByKey(t, legacy), tokensByKey(t, current), set)
		// The market fallback picks the first mainnet sibling of each token.
		for _, token := range current {
			if token.CrossChainID == walletcommon.StatusTestTokenCrossChainID {
				stt++
			}
			require.Equal(t, tokentypes.EthereumMainnetSibling(token, legacy), tokentypes.EthereumMainnetSibling(token, current), token.Key())
		}
	}
	require.NotZero(t, stt, "the fixture must exercise the STT to SNT remap")
}
