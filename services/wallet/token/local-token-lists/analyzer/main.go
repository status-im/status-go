package main

import (
	"fmt"
	"time"

	"golang.org/x/exp/maps"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/parsers"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	walletcommon "github.com/status-im/status-go/services/wallet/common"
	defaulttokenlists "github.com/status-im/status-go/services/wallet/token/local-token-lists/default-lists"
)

func main() {
	fmt.Println("Analyzing token lists")
	fetchedTokensLists := []defaulttokenlists.DownloadedTokenList{
		defaulttokenlists.StatusTokenList,
		defaulttokenlists.UniswapTokenList,
		defaulttokenlists.CoingeckoEthereumTokenList,
		defaulttokenlists.CoingeckoOptimismTokenList,
		defaulttokenlists.CoingeckoArbitrumTokenList,
		defaulttokenlists.CoingeckoBscTokenList,
		defaulttokenlists.CoingeckoBaseTokenList,
		defaulttokenlists.CoingeckoLineaTokenList,
	}

	fmt.Println("Analyzing token lists")
	fmt.Println("=====================================")
	fmt.Println("Total number of token lists: ", len(fetchedTokensLists))
	if len(fetchedTokensLists) != len(defaulttokenlists.TokensSources) {
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
	tokensPerList := make(map[string]map[string]*types.Token) // map[tokenlistID][tokenID]*types.Token
	tokensByIdMap := make(map[string][]*types.Token)          // map[tokenID][]*types.Token
	tokensBySymbolMap := make(map[string][]*types.Token)      // map[tokenSymbol][]*types.Token
	for _, tList := range tokensLists {
		fmt.Printf("Analizying token list: %s\n", tList.Name)
		fmt.Printf("Total number of tokens: %d\n", len(tList.Tokens))

		tokensPerList[tList.Name] = make(map[string]*types.Token)
		tokensPerChainID := make(map[uint64][]*types.Token) // map[chainID]*types.Token
		for _, chainToken := range tList.Tokens {
			if _, ok := tokensPerChainID[chainToken.ChainID]; !ok {
				tokensPerChainID[chainToken.ChainID] = make([]*types.Token, 0, len(tList.Tokens))
			}
			tokensPerChainID[chainToken.ChainID] = append(tokensPerChainID[chainToken.ChainID], chainToken)

			tokenKey := chainToken.Key()
			if _, ok := tokensPerList[tList.Name][tokenKey]; !ok {
				tokensPerList[tList.Name][tokenKey] = chainToken
			} else {
				fmt.Printf("Duplicate token for token key: %s\n", tokenKey)
			}

			if _, ok := tokensByIdMap[tokenKey]; !ok {
				tokensByIdMap[tokenKey] = make([]*types.Token, 0)
			}
			tokensByIdMap[tokenKey] = append(tokensByIdMap[tokenKey], chainToken)

			if _, ok := tokensBySymbolMap[chainToken.Symbol]; !ok {
				tokensBySymbolMap[chainToken.Symbol] = make([]*types.Token, 0)
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

	var tokensWithDifferentDecimals []*types.Token
	fmt.Println("")
	fmt.Println("Cross-analyzing tokens by symbol (finds different decimals for the same symbol across chains)")
	for _, tokens := range tokensBySymbolMap {
		decimalsChainIdMapBySymbol := make(map[string]map[uint]map[uint64][]*types.Token) // map[symbol]map[decimals]map[chainID][]*types.Token
		for _, token := range tokens {
			if _, ok := decimalsChainIdMapBySymbol[token.Symbol]; !ok {
				decimalsChainIdMapBySymbol[token.Symbol] = make(map[uint]map[uint64][]*types.Token)
			}
			if _, ok := decimalsChainIdMapBySymbol[token.Symbol][token.Decimals]; !ok {
				decimalsChainIdMapBySymbol[token.Symbol][token.Decimals] = make(map[uint64][]*types.Token)
			}
			decimalsChainIdMapBySymbol[token.Symbol][token.Decimals][token.ChainID] = append(decimalsChainIdMapBySymbol[token.Symbol][token.Decimals][token.ChainID], token)
		}
		for symbol, chainsByDecimalsMap := range decimalsChainIdMapBySymbol {
			if len(chainsByDecimalsMap) > 1 {
				fmt.Printf("Token with symbol '%s' has different decimals across chains\n", symbol)
				for decimal, chainsMap := range chainsByDecimalsMap {
					fmt.Printf("Token with symbol '%s' has decimals %d for chains %+v\n", symbol, decimal, maps.Keys(chainsMap))
					for _, chainTokens := range chainsMap {
						tokensWithDifferentDecimals = append(tokensWithDifferentDecimals, chainTokens...)
					}
				}
			}
		}
	}
	fmt.Println("=====================================")
}

func rebuildTokensMap(fetchedLists []defaulttokenlists.DownloadedTokenList) (map[string]*types.TokenList, error) {
	tokensLists := make(map[string]*types.TokenList)
	for _, fetchedTokenList := range fetchedLists {
		var parser parsers.TokenListParser
		parser = &parsers.StandardTokenListParser{}
		if fetchedTokenList.ID == walletcommon.StatusTokenListID {
			parser = &parsers.StatusTokenListParser{}
		}

		list, err := parser.Parse(fetchedTokenList.JsonData, walletcommon.AllChainIDsAsUint64())
		if err != nil {
			fmt.Printf("Failed to parse token list %s: %v\n", fetchedTokenList.ID, err)
			continue
		}
		list.Source = fetchedTokenList.SourceURL
		list.FetchedTimestamp = fetchedTokenList.Fetched.Format(time.RFC3339)

		// remove tokens if not in supported chains
		for i := 0; i < len(list.Tokens); {
			token := list.Tokens[i]
			if !walletcommon.IsSupportedChainID(token.ChainID) {
				list.Tokens = append(list.Tokens[:i], list.Tokens[i+1:]...)
			} else {
				i++
			}
		}

		tokensLists[fetchedTokenList.ID] = list
	}

	return tokensLists, nil
}
