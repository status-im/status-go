package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"golang.org/x/exp/maps"

	tokenlists "github.com/status-im/status-go/services/wallet/token/token-lists"
	defaulttokenlists "github.com/status-im/status-go/services/wallet/token/token-lists/default-lists"
	"github.com/status-im/status-go/services/wallet/token/token-lists/fetcher"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
)

func main() {
	fmt.Println("Analyzing token lists")
	fetchedTokensLists := []fetcher.FetchedTokenList{
		defaulttokenlists.StatusTokenList,
		defaulttokenlists.AaveTokenList,
		defaulttokenlists.UniswapTokenList,
	}

	fmt.Println("Analyzing token lists")
	fmt.Println("=====================================")
	fmt.Println("Total number of token lists: ", len(fetchedTokensLists))
	if len(fetchedTokensLists) != len(defaulttokenlists.TokensSources)+1 { // +1 for the Status token list
		fmt.Println("Warning: The number of token lists does not match the number of sources")
		return
	}
	fmt.Println("=====================================")

	tokensLists, err := rebuildTokensMap(fetchedTokensLists)
	if err != nil {
		fmt.Println("Error rebuilding tokens map: ", err)
		return
	}

	fmt.Println("")
	tokensPerList := make(map[string]map[string]*tokenTypes.Token) // map[store][tokenID]*tokenTypes.Token
	tokensByIdMap := make(map[string][]*tokenTypes.Token)          // map[tokenID][]*tokenTypes.Token
	tokensBySymbolMap := make(map[string][]*tokenTypes.Token)      // map[tokenSymbol][]*tokenTypes.Token
	for _, tList := range tokensLists {
		fmt.Printf("Analizying token list: %s\n", tList.Name)
		fmt.Printf("Total number of tokens: %d\n", len(tList.Tokens))

		tokensPerList[tList.Name] = make(map[string]*tokenTypes.Token)
		tokensPerChainID := make(map[uint64][]*tokenTypes.Token) // map[chainID]*tokenTypes.Token
		for _, chainToken := range tList.Tokens {
			if _, ok := tokensPerChainID[chainToken.ChainID]; !ok {
				tokensPerChainID[chainToken.ChainID] = make([]*tokenTypes.Token, 0, len(tList.Tokens))
			}
			tokensPerChainID[chainToken.ChainID] = append(tokensPerChainID[chainToken.ChainID], chainToken)

			id := getTokenID(chainToken)
			if _, ok := tokensPerList[tList.Name][id]; !ok {
				tokensPerList[tList.Name][id] = chainToken
			} else {
				fmt.Printf("Duplicate token for id: %s\n", id)
			}

			if _, ok := tokensByIdMap[id]; !ok {
				tokensByIdMap[id] = make([]*tokenTypes.Token, 0)
			}
			tokensByIdMap[id] = append(tokensByIdMap[id], chainToken)

			if _, ok := tokensBySymbolMap[chainToken.Symbol]; !ok {
				tokensBySymbolMap[chainToken.Symbol] = make([]*tokenTypes.Token, 0)
			}
			tokensBySymbolMap[chainToken.Symbol] = append(tokensBySymbolMap[chainToken.Symbol], chainToken)
		}

		for chainID, chainTokens := range tokensPerChainID {
			fmt.Printf("Total number of tokens for chain %d: %d\n", chainID, len(chainTokens))
		}
		fmt.Println("")
	}

	fmt.Println("=====================================")
	fmt.Println("Cross-analyzing tokens")
	fmt.Println("=====================================")
	fmt.Println("")
	fmt.Println("Cross-analyzing tokens by id (finds different symbols for the same chainId+address pairs)")
	for tokenID, tokens := range tokensByIdMap {
		symbolMap := make(map[string]struct{}) // map[symbol]struct{}
		for _, token := range tokens {
			if _, ok := symbolMap[token.Symbol]; !ok {
				symbolMap[token.Symbol] = struct{}{}
			}
		}
		if len(symbolMap) > 1 {
			fmt.Printf("Token with id '%s' has multiple symbols: %+v\n", tokenID, maps.Keys(symbolMap))
		}
	}

	fmt.Println("")
	fmt.Println("Cross-analyzing tokens by symbol (finds different addresses for the same symbol on the same chain)")
	for tokenSymbol, tokens := range tokensBySymbolMap {
		chainIDAddressesMap := make(map[uint64]map[string]struct{}) // map[chainID]map[address]
		for _, token := range tokens {
			if _, ok := chainIDAddressesMap[token.ChainID]; !ok {
				chainIDAddressesMap[token.ChainID] = make(map[string]struct{})
			}
			chainIDAddressesMap[token.ChainID][token.Address.Hex()] = struct{}{}
		}
		for chainID, addresses := range chainIDAddressesMap {
			if len(addresses) > 1 {
				fmt.Printf("Token with symbol '%s' has multiple addresses for chain %d: %+v\n", tokenSymbol, chainID, maps.Keys(addresses))
			}
			if len(addresses) == 0 {
				fmt.Printf("Token with symbol '%s' has no address for chain %d\n", tokenSymbol, chainID)
			}
		}
	}

	fmt.Println("")
	fmt.Println("Cross-analyzing tokens by symbol (finds different decimals for the same symbol across chains)")
	for _, tokens := range tokensBySymbolMap {
		decimalsChainIdMapBySymbol := make(map[string]map[uint]map[uint64]struct{}) // map[symbol]map[decimals]map[chainID]
		for _, token := range tokens {
			if _, ok := decimalsChainIdMapBySymbol[token.Symbol]; !ok {
				decimalsChainIdMapBySymbol[token.Symbol] = make(map[uint]map[uint64]struct{})
			}
			if _, ok := decimalsChainIdMapBySymbol[token.Symbol][token.Decimals]; !ok {
				decimalsChainIdMapBySymbol[token.Symbol][token.Decimals] = make(map[uint64]struct{})
			}
			decimalsChainIdMapBySymbol[token.Symbol][token.Decimals][token.ChainID] = struct{}{}
		}
		for symbol, chainsByDecimalsMap := range decimalsChainIdMapBySymbol {
			if len(chainsByDecimalsMap) > 1 {
				fmt.Printf("Token with symbol '%s' has different decimals across chains\n", symbol)
				for decimal, chainsMap := range chainsByDecimalsMap {
					fmt.Printf("Token with symbol '%s' has decimals %d for chains %+v\n", symbol, decimal, maps.Keys(chainsMap))
				}
			}
		}
	}
	fmt.Println("=====================================")
}

func rebuildTokensMap(fetchedLists []fetcher.FetchedTokenList) (map[string]*tokenlists.TokensList, error) {
	tokensLists := make(map[string]*tokenlists.TokensList)
	for _, fetchedTokenList := range fetchedLists {
		var list tokenlists.TokensList
		decoder := json.NewDecoder(strings.NewReader(fetchedTokenList.JsonData))
		if err := decoder.Decode(&list); err != nil {
			return nil, err
		}

		list.Source = fetchedTokenList.SourceURL
		list.FetchedTimestamp = fetchedTokenList.Fetched.Format(time.RFC3339)

		tokensLists[fetchedTokenList.ID] = &list
	}

	return tokensLists, nil
}

func getTokenID(token *tokenTypes.Token) string {
	return fmt.Sprintf("%d - %s", token.ChainID, token.Address.Hex())
}
