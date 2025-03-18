package main

import (
	"context"
	"fmt"
	"time"

	tokenlists "github.com/status-im/status-go/services/wallet/token/token-lists"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
)

func main() {
	tokensLists, err := tokenlists.NewTokenLists(nil, nil)
	if err != nil {
		fmt.Printf("Error creating token lists: %v\n", err)
		return
	}
	tokensLists.Start(context.Background(), "", time.Hour, time.Hour)
	allTokensLists := tokensLists.GetTokensLists()

	fmt.Println("")
	tokensPerList := make(map[string]map[string]*tokenTypes.Token) // map[store][tokenID]*tokenTypes.Token
	for _, tList := range allTokensLists {
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
		}

		for chainID, chainTokens := range tokensPerChainID {
			fmt.Printf("Total number of tokens for chain %d: %d\n", chainID, len(chainTokens))
		}
		fmt.Println("")
	}

	fmt.Println("Cross-analyzing stores")
	statusStoreName := "Status Token List"
	dupesFound := false
	for tokenID, token := range tokensPerList[statusStoreName] {
		for otherStoreName, otherTokensPerChain := range tokensPerList {
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

func getTokenID(token *tokenTypes.Token) string {
	return fmt.Sprintf("%d - %s", token.ChainID, token.Address.Hex())
}
