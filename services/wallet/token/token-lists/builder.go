package tokenlists

import (
	"encoding/json"
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
		//Filter non-EVM tokens to avoid decode errors
		filteredData, err := filterNonEVMTokens([]byte(fetchedTokenList.JsonData))
		if err != nil {
			return err
		}

		if err := json.Unmarshal(filteredData, &list); err != nil {
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

	t.ensureEmptyTokenListsAreNotNil()

	return nil
}

func (t *TokenLists) ensureEmptyTokenListsAreNotNil() {
	t.tokensListsMu.RLock()
	defer t.tokensListsMu.RUnlock()
	for _, list := range t.tokensLists {
		if len(list.Tokens) == 0 {
			list.Tokens = make([]*tokenTypes.Token, 0)
		}
	}
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

// filterNonEVMTokens filters out tokens from non-EVM chains (e.g. Solana)
func filterNonEVMTokens(body []byte) ([]byte, error) {
	var rawList struct {
		Name      string                   `json:"name"`
		Timestamp string                   `json:"timestamp"`
		Version   map[string]interface{}   `json:"version"`
		Tags      map[string]interface{}   `json:"tags"`
		LogoURI   string                   `json:"logoURI"`
		Keywords  []string                 `json:"keywords"`
		Tokens    []map[string]interface{} `json:"tokens"`
	}

	if err := json.Unmarshal(body, &rawList); err != nil {
		return body, err
	}

	filteredTokens := make([]map[string]interface{}, 0, len(rawList.Tokens))

	for _, token := range rawList.Tokens {
		chainID, ok := token["chainId"].(float64)
		if !ok {
			continue
		}

		// Only include tokens from application's supported EVM chains
		if common.IsSupportedChainID(uint64(chainID)) {
			filteredTokens = append(filteredTokens, token)
		}
	}

	rawList.Tokens = filteredTokens

	filteredBody, err := json.Marshal(rawList)
	if err != nil {
		return nil, err
	}

	return filteredBody, nil
}

func filterTokens(tokens []*tokenTypes.Token) []*tokenTypes.Token {
	var filteredTokens []*tokenTypes.Token
	for _, token := range tokens {
		// remove native token on respective chains as they are added via a different list
		if token.IsNative() {
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
