package tokenlists

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"golang.org/x/exp/maps"

	walletCommon "github.com/status-im/status-go/services/wallet/common"
	defaulttokenlists "github.com/status-im/status-go/services/wallet/token/token-lists/default-lists"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
)

// This is a temporary solution to resolve symbol collisions and tokens decimal issues.

func (t *TokenLists) solveCollision() {
	t.tokensListsMu.RLock()
	defer t.tokensListsMu.RUnlock()

	// Remove duplicate tokens from the token lists if they have different symbols for the same chainId + address pair
	for _, tokenList := range t.tokensLists {
		tokenList.Tokens = removeDuplicateSymbolOnTheSameChain(tokenList.Tokens)
	}

	// Remove duplicate tokens from the token lists if they have different symbols for the same chainId + address pair (main source of collisions is uniswap token list, then status)
	referenceTokenList := t.tokensLists[defaulttokenlists.UniswapTokenListID].Tokens
	t.tokensLists[defaulttokenlists.StatusTokenListID].Tokens = removeTokenIfAppearsInTheReferenceList(t.tokensLists[defaulttokenlists.StatusTokenListID].Tokens, referenceTokenList)
	t.tokensLists[defaulttokenlists.AaveTokenListID].Tokens = removeTokenIfAppearsInTheReferenceList(t.tokensLists[defaulttokenlists.AaveTokenListID].Tokens, referenceTokenList)
	referenceTokenList = t.tokensLists[defaulttokenlists.StatusTokenListID].Tokens
	t.tokensLists[defaulttokenlists.AaveTokenListID].Tokens = removeTokenIfAppearsInTheReferenceList(t.tokensLists[defaulttokenlists.AaveTokenListID].Tokens, referenceTokenList)

	// Use uniswap tokens map as reference for solving collisions, that's why it is processed first
	uniswapTokensMap := solveDecimalsCollision(t.tokensLists[defaulttokenlists.UniswapTokenListID].Tokens, nil)
	t.tokensLists[defaulttokenlists.UniswapTokenListID].Tokens = make([]*tokenTypes.Token, 0)
	for _, tokens := range uniswapTokensMap {
		t.tokensLists[defaulttokenlists.UniswapTokenListID].Tokens = append(t.tokensLists[defaulttokenlists.UniswapTokenListID].Tokens, tokens...)
	}

	// Use Status tokens list and process tokens using uniswap tokens map as reference
	statusTokensMap := solveDecimalsCollision(t.tokensLists[defaulttokenlists.StatusTokenListID].Tokens, uniswapTokensMap)
	t.tokensLists[defaulttokenlists.StatusTokenListID].Tokens = make([]*tokenTypes.Token, 0)
	for symbol, tokens := range statusTokensMap {
		t.tokensLists[defaulttokenlists.StatusTokenListID].Tokens = append(t.tokensLists[defaulttokenlists.StatusTokenListID].Tokens, tokens...)
		uniswapTokensMap[symbol] = append(uniswapTokensMap[symbol], tokens...)
	}

	// Use Aave tokens list and process tokens using uniswap and status tokens map as reference
	aaveTokensMap := solveDecimalsCollision(t.tokensLists[defaulttokenlists.AaveTokenListID].Tokens, uniswapTokensMap)
	t.tokensLists[defaulttokenlists.AaveTokenListID].Tokens = make([]*tokenTypes.Token, 0)
	for _, tokens := range aaveTokensMap {
		t.tokensLists[defaulttokenlists.AaveTokenListID].Tokens = append(t.tokensLists[defaulttokenlists.AaveTokenListID].Tokens, tokens...)
	}
}

func getSymbolChainPair(token *tokenTypes.Token) string {
	return fmt.Sprintf("%d-%s", token.ChainID, token.Symbol)
}

func makeUniqueSymbol(token *tokenTypes.Token) string {
	// Based on proposal here https://discord.com/channels/1210237582470807632/1363907888082321448/1364611566753812601
	l1Chain := "EVM"
	if token.ChainID == walletCommon.BSCMainnet || token.ChainID == walletCommon.BSCTestnet {
		l1Chain = "BSC"
	}
	return fmt.Sprintf("%s (%s)", token.Symbol, l1Chain)
}

func removeDuplicateSymbolOnTheSameChain(tokens []*tokenTypes.Token) []*tokenTypes.Token {
	tokensByIdMap := make(map[string]*tokenTypes.Token) // map[tokenID]*tokenTypes.Token
	for _, token := range tokens {
		id := getSymbolChainPair(token)
		if _, ok := tokensByIdMap[id]; !ok {
			tokensByIdMap[id] = token
		}
	}
	return maps.Values(tokensByIdMap)
}

func removeTokenIfAppearsInTheReferenceList(tokens []*tokenTypes.Token, referenceTokens []*tokenTypes.Token) (filteredTokens []*tokenTypes.Token) {
	for _, token := range tokens {
		if !tokenWithChainIdAndAddressExistsInList(token.ChainID, token.Address, referenceTokens) {
			filteredTokens = append(filteredTokens, token)
		}
	}
	return
}

func solveDecimalsCollision(tokens []*tokenTypes.Token, tokensReferenceMap map[string][]*tokenTypes.Token) map[string][]*tokenTypes.Token {
	// make sure all tokens have tmpSymbol set
	for _, token := range tokens {
		token.TmpSymbol = token.Symbol
	}

	decimalsChainIdMapBySymbol := make(map[string]map[uint]map[uint64]*tokenTypes.Token) // map[symbol]map[decimal]map[chainID]token
	tokensUniqueBySymbolChainPair := make(map[string][]*tokenTypes.Token)                // map[symbol]map[chainID]token

	for _, token := range tokens {
		if _, ok := decimalsChainIdMapBySymbol[token.Symbol]; !ok {
			decimalsChainIdMapBySymbol[token.Symbol] = make(map[uint]map[uint64]*tokenTypes.Token)
		}
		if _, ok := decimalsChainIdMapBySymbol[token.Symbol][token.Decimals]; !ok {
			decimalsChainIdMapBySymbol[token.Symbol][token.Decimals] = make(map[uint64]*tokenTypes.Token)
		}
		decimalsChainIdMapBySymbol[token.Symbol][token.Decimals][token.ChainID] = token
	}

	for _, chainsByDecimalsMap := range decimalsChainIdMapBySymbol {
		if len(chainsByDecimalsMap) == 0 {
			// should never be here
			continue
		}
		// add tokens with the same symbol and same decimals
		for _, chainsMap := range chainsByDecimalsMap {
			for _, token := range chainsMap {
				allReferenceTokens := getTokensForSymbolFromMap(token.Symbol, tokensReferenceMap)
				if len(allReferenceTokens) == 0 {
					if len(chainsByDecimalsMap) == 1 {
						tokensUniqueBySymbolChainPair[token.Symbol] = append(tokensUniqueBySymbolChainPair[token.Symbol], token)
					} else {
						tokenCopy := *token
						tokenCopy.Symbol = makeUniqueSymbol(token)
						tokensUniqueBySymbolChainPair[tokenCopy.Symbol] = append(tokensUniqueBySymbolChainPair[tokenCopy.Symbol], &tokenCopy)
					}
					continue
				}

				added := false
				for _, referenceTokens := range allReferenceTokens {
					if referenceTokens[0].Decimals == token.Decimals { // all tokens from the reference list have the same decimals
						if tokenWithChainIdExistsInList(token.ChainID, referenceTokens) {
							added = true
							break
						}
						tokenCopy := *token
						tokenCopy.Symbol = referenceTokens[0].Symbol //makeUniqueSymbol(token)
						tokensUniqueBySymbolChainPair[tokenCopy.Symbol] = append(tokensUniqueBySymbolChainPair[tokenCopy.Symbol], &tokenCopy)
						added = true
						break
					}
				}

				if added {
					// at this point the token that is being added has the same decimals as one of the tokens with the same symbol that was already added processing one of the previous token lists
					continue
				}

				// at this point the token that is being added has different decimals to the ones with the same symbols that ware already added processing one of the previous token lists
				tokenCopy := *token
				tokenCopy.Symbol = makeUniqueSymbol(token)
				tokensUniqueBySymbolChainPair[tokenCopy.Symbol] = append(tokensUniqueBySymbolChainPair[tokenCopy.Symbol], &tokenCopy)
			}
		}
	}

	return tokensUniqueBySymbolChainPair
}

func getTokensForSymbolFromMap(symbol string, tokensMap map[string][]*tokenTypes.Token) [][]*tokenTypes.Token {
	allReferenceTokens := make([][]*tokenTypes.Token, 0)
	for _, tokens := range tokensMap {
		for i := range tokens {
			if tokens[i].TmpSymbol == symbol {
				allReferenceTokens = append(allReferenceTokens, tokens)
				break
			}
		}
	}
	return allReferenceTokens
}

func tokenWithChainIdExistsInList(chainID uint64, tokensList []*tokenTypes.Token) bool {
	for _, t := range tokensList {
		if t.ChainID == chainID {
			return true
		}
	}
	return false
}

func tokenWithChainIdAndAddressExistsInList(chainID uint64, address common.Address, tokensList []*tokenTypes.Token) bool {
	for _, t := range tokensList {
		if t.ChainID == chainID && t.Address == address {
			return true
		}
	}
	return false
}
