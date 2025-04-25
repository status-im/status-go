package tokenlists

import (
	"encoding/json"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/services/wallet/common"
	defaulttokenlists "github.com/status-im/status-go/services/wallet/token/token-lists/default-lists"
	"github.com/status-im/status-go/services/wallet/token/token-lists/fetcher"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
)

func (t *TokenLists) rebuildTokensMap(fetchedLists []fetcher.FetchedTokenList) error {
	for _, fetchedTokenList := range fetchedLists {
		// TODO: all lists that we support for now follow the same schema
		// so we can just decode them all the same way, but once we add new list that doesn't follow the same schema
		// we need to add a switch here, based on the `fetchedTokenList.ID` to map them to `TokensList` struct
		var list TokensList
		decoder := json.NewDecoder(strings.NewReader(fetchedTokenList.JsonData))
		if err := decoder.Decode(&list); err != nil {
			return err
		}

		list.Source = fetchedTokenList.SourceURL
		list.FetchedTimestamp = fetchedTokenList.Fetched.Format(time.RFC3339)

		list.Tokens = filterTokens(list.Tokens)

		processTokenPegs(list.Tokens)

		t.tokensListsMu.Lock()
		t.tokensLists[fetchedTokenList.ID] = &list
		t.tokensListsMu.Unlock()
	}

	// TODO: remove this once we switch to CoinGecko tokens list
	// temporary soltion to avoid token collisions
	t.solveCollision()

	return nil
}

func getDefaultTokensLists() []fetcher.FetchedTokenList {
	return []fetcher.FetchedTokenList{
		defaulttokenlists.StatusTokenList,
		defaulttokenlists.AaveTokenList,
		defaulttokenlists.UniswapTokenList,
	}
}

func getTheLatestFetchTimeOfDefaultTokenLists() time.Time {
	defaultTokenLists := getDefaultTokensLists()
	lastTokensUpdate := defaulttokenlists.StatusTokenList.Fetched
	for _, list := range defaultTokenLists {
		if list.Fetched.After(lastTokensUpdate) {
			lastTokensUpdate = list.Fetched
		}
	}
	return lastTokensUpdate
}

// buildInitialTokensListsMap builds the initial tokens map from the default token lists.
func (t *TokenLists) buildInitialTokensListsMap() error {
	lastTokensUpdate := getTheLatestFetchTimeOfDefaultTokenLists()
	err := t.settings.SaveSettingField(settings.LastTokensUpdate, lastTokensUpdate)
	if err != nil {
		logutils.ZapLogger().Error("failed to save last tokens update time", zap.Error(err))
		return err
	}

	return t.rebuildTokensMap(getDefaultTokensLists())
}

// rebuildTokensListsMap rebuilds the tokens map from the fetched token lists.
func (t *TokenLists) rebuildTokensListsMap() error {
	fetchedTokensLists, err := t.tokenListsFetcher.GetAllTokenLists()
	if err != nil {
		logutils.ZapLogger().Error("Failed to get all token lists", zap.Error(err))
		return err
	}
	var tokensListsForProcessing []fetcher.FetchedTokenList
	// first include the default token lists if not present in fetched lists
	for _, defaultList := range getDefaultTokensLists() {
		var found bool
		for _, fetchedList := range fetchedTokensLists {
			if fetchedList.ID == defaultList.ID {
				found = true
				break
			}
		}
		if !found {
			tokensListsForProcessing = append(tokensListsForProcessing, defaultList)
		}
	}
	// then include the fetched lists
	tokensListsForProcessing = append(tokensListsForProcessing, fetchedTokensLists...)

	return t.rebuildTokensMap(tokensListsForProcessing)
}

func filterTokens(tokens []*tokenTypes.Token) []*tokenTypes.Token {
	var filteredTokens []*tokenTypes.Token
	for _, token := range tokens {
		found := false
		for _, chainID := range common.AllChainIDs() {
			if token.ChainID == uint64(chainID) {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		filteredTokens = append(filteredTokens, token)
	}
	return filteredTokens
}

func processTokenPegs(tokens []*tokenTypes.Token) {
	for _, token := range tokens {
		token.PegSymbol = tokenTypes.GetTokenPegSymbol(token.Symbol)
	}
}
