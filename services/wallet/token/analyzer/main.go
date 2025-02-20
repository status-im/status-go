package main

import (
	"fmt"

	token "github.com/status-im/status-go/services/wallet/token"
)

func main() {
	statusStore := token.NewDefaultStore()
	otherStores := []token.Store{token.NewUniswapStore()}
	allStores := append([]token.Store{statusStore}, otherStores...)

	storePerName := map[string]token.Store{}
	for _, store := range allStores {
		if _, ok := storePerName[store.GetName()]; !ok {
			storePerName[store.GetName()] = store
		} else {
			fmt.Printf("Duplicate store names: %s\n", store.GetName())
		}
	}

	fmt.Println("")
	tokensPerStore := make(map[string]map[string]*token.Token) // map[store][tokenID]*token.Token
	for storeName, store := range storePerName {
		fmt.Printf("Analizying store: %s\n", storeName)
		tokens := store.GetTokens()
		fmt.Printf("Total number of tokens: %d\n", len(tokens))

		tokensPerStore[storeName] = make(map[string]*token.Token)
		tokensPerChainID := make(map[uint64][]*token.Token) // map[chainID]*token.Token
		for _, chainToken := range tokens {
			if _, ok := tokensPerChainID[chainToken.ChainID]; !ok {
				tokensPerChainID[chainToken.ChainID] = make([]*token.Token, 0, len(tokens))
			}
			tokensPerChainID[chainToken.ChainID] = append(tokensPerChainID[chainToken.ChainID], chainToken)

			id := getTokenID(chainToken)
			if _, ok := tokensPerStore[storeName][id]; !ok {
				tokensPerStore[storeName][id] = chainToken
			} else {
				fmt.Printf("Duplicate token for id: %s\n", id)
			}
		}

		for chainID, chainTokens := range tokensPerChainID {
			fmt.Printf("Total number of tokens for chain %d: %d\n", chainID, len(chainTokens))
		}
		fmt.Println("")
	}

	fmt.Println("Cross-analyzing stores")
	statusStoreName := statusStore.GetName()
	dupesFound := false
	for tokenID, token := range tokensPerStore[statusStore.GetName()] {
		for otherStoreName, otherTokensPerChain := range tokensPerStore {
			if otherStoreName == statusStoreName {
				continue
			}
			if _, ok := otherTokensPerChain[tokenID]; ok {
				dupesFound = true
				fmt.Printf("Token with id '%s' and symbol '%s' found in stores %s and %s\n", tokenID, token.Symbol, statusStoreName, otherStoreName)
			}
		}
	}
	if !dupesFound {
		fmt.Println("No duplicates found")
	}
}

func getTokenID(token *token.Token) string {
	return fmt.Sprintf("%d - %s", token.ChainID, token.Address.Hex())
}
