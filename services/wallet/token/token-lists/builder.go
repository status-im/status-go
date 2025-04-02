package tokenlists

import (
	"encoding/json"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts/settings"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty/market/coingecko"
	defaulttokenlists "github.com/status-im/status-go/services/wallet/token/token-lists/default-lists"
	"github.com/status-im/status-go/services/wallet/token/token-lists/fetcher"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
)

func mapFetchedOtherListToTokenList(fetchedTokenList fetcher.FetchedTokenList, tokenList *TokensList) error {
	tokenList.Source = fetchedTokenList.SourceURL
	tokenList.FetchedTimestamp = fetchedTokenList.Fetched.Format(time.RFC3339)

	decoder := json.NewDecoder(strings.NewReader(fetchedTokenList.JsonData))
	if err := decoder.Decode(tokenList); err != nil {
		return err
	}

	// remove tokens if the address or id is empty
	var tokens []*tokenTypes.Token
	for _, token := range tokenList.Tokens {
		if token.TokenGroupKey() != "" && token.Address != walletCommon.ZeroAddress() {
			tokens = append(tokens, token)
		}
	}

	tokenList.Tokens = tokens

	return nil
}

func mapFetchedCoingeckoListToTokenList(fetchedTokenList fetcher.FetchedTokenList, tokenList *TokensList) error {
	tokenList.Name = "CoinGecko"
	tokenList.Source = fetchedTokenList.SourceURL
	tokenList.FetchedTimestamp = fetchedTokenList.Fetched.Format(time.RFC3339)

	jsonData := fetchedTokenList.JsonData
	if jsonData == "" {
		jsonData = string(fetchedTokenList.JsonDataBytes)
	}
	decoder := json.NewDecoder(strings.NewReader(jsonData))
	var coingeckoList []coingecko.GeckoToken
	if err := decoder.Decode(&coingeckoList); err != nil {
		return err
	}

	for _, token := range coingeckoList {
		var (
			chainIDs []uint64
			address  string
		)
		// coingecko doesn't have testnet tokens
		if token.Platforms.Ethereum != "" {
			chainIDs = append(chainIDs, walletCommon.EthereumMainnet)
			address = token.Platforms.Ethereum
		}
		if token.Platforms.Optimism != "" {
			chainIDs = append(chainIDs, walletCommon.OptimismMainnet)
			address = token.Platforms.Optimism
		}
		if token.Platforms.Arbitrum != "" {
			chainIDs = append(chainIDs, walletCommon.ArbitrumMainnet)
			address = token.Platforms.Arbitrum
		}
		if token.Platforms.Base != "" {
			chainIDs = append(chainIDs, walletCommon.BaseMainnet)
			address = token.Platforms.Base
		}

		if len(chainIDs) == 0 || address == "" {
			continue
		}

		for _, chainID := range chainIDs {
			tokenList.Tokens = append(tokenList.Tokens, &tokenTypes.Token{
				GroupKey: token.ID,
				Address:  common.HexToAddress(address),
				Symbol:   token.Symbol,
				Name:     token.Name,
				ChainID:  chainID,
			})
		}
	}

	return nil
}

func (t *TokenLists) rebuildTokensMap(fetchedLists []fetcher.FetchedTokenList) error {
	for _, fetchedTokenList := range fetchedLists {
		// TODO: all lists that we support for now follow the same schema
		// so we can just decode them all the same way, but once we add new list that doesn't follow the same schema
		// we need to add a switch here, based on the `fetchedTokenList.ID` to map them to `TokensList` struct
		var list TokensList
		if fetchedTokenList.ID != defaulttokenlists.Coingecko {
			err := mapFetchedOtherListToTokenList(fetchedTokenList, &list)
			if err != nil {
				logutils.ZapLogger().Error("failed to map fetched token list", zap.Error(err))
				return err
			}
		} else {
			err := mapFetchedCoingeckoListToTokenList(fetchedTokenList, &list)
			if err != nil {
				logutils.ZapLogger().Error("failed to map fetched token list", zap.Error(err))
				return err
			}
		}

		t.tokensListsMu.Lock()
		t.tokensLists[fetchedTokenList.ID] = &list
		t.tokensListsMu.Unlock()
	}

	return nil
}

func getDefaultTokensLists() []fetcher.FetchedTokenList {
	return []fetcher.FetchedTokenList{
		defaulttokenlists.StatusTokenList,
		defaulttokenlists.CoingeckoTokenList,
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
