package token

//go:generate mockgen -source=token.go -destination=mock/token/tokenmanager.go

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
)

func (tm *Manager) getNativeTokens() ([]*tokenTypes.Token, error) {
	tokens := make([]*tokenTypes.Token, 0)
	networks, err := tm.networkManager.Get(false)
	if err != nil {
		return nil, err
	}

	for _, network := range networks {
		tokens = append(tokens, tm.ToToken(network))
	}

	return tokens, nil
}

// GetAllTokens returns all tokens from the token manager.
func (tm *Manager) GetAllTokens() ([]*tokenTypes.Token, error) {
	allTokens, err := tm.GetCustoms(true)
	if err != nil {
		logutils.ZapLogger().Error("can't fetch custom tokens", zap.Error(err))
	}

	uniqueListsTokens := tm.tokenLists.GetUniqueTokens()

	allTokens = append(uniqueListsTokens, allTokens...)

	overrideTokensInPlace(tm.networkManager.GetEmbeddedNetworks(), allTokens)

	native, err := tm.getNativeTokens()
	if err != nil {
		return nil, err
	}

	allTokens = append(allTokens, native...)

	return allTokens, nil
}

// GetTokens returns all tokens for a specific chain ID.
func (tm *Manager) GetTokens(chainID uint64) ([]*tokenTypes.Token, error) {
	tokens, err := tm.GetAllTokens()
	if err != nil {
		return nil, err
	}

	res := make([]*tokenTypes.Token, 0)

	for _, token := range tokens {
		if token.ChainID == chainID {
			res = append(res, token)
		}
	}

	return res, nil
}

// GetTokensByChainIDs returns all tokens for a list of chain IDs.
func (tm *Manager) GetTokensByChainIDs(chainIDs []uint64) ([]*tokenTypes.Token, error) {
	tokens, err := tm.GetAllTokens()
	if err != nil {
		return nil, err
	}

	res := make([]*tokenTypes.Token, 0)

	for _, token := range tokens {
		for _, chainID := range chainIDs {
			if token.ChainID == chainID {
				res = append(res, token)
			}
		}
	}

	return res, nil
}

// GetTokensGroupedByGroupKey returns all tokens grouped by groups they belong to (determined by the group key).
func (tm *Manager) GetTokensGroupedByGroupKey() (map[string][]*tokenTypes.Token, error) {
	tokens, err := tm.GetAllTokens()
	if err != nil {
		return nil, err
	}

	res := make(map[string][]*tokenTypes.Token)

	for _, token := range tokens {
		res[token.TokenGroupKey()] = append(res[token.TokenGroupKey()], token)
	}

	return res, nil
}

// GetTokensForGroupKey returns all tokens belonging to a specific group (determined by the group key).
func (tm *Manager) GetTokensForGroupKey(groupKey string) ([]*tokenTypes.Token, error) {
	tokens, err := tm.GetAllTokens()
	if err != nil {
		return nil, err
	}

	res := make([]*tokenTypes.Token, 0)
	for _, token := range tokens {
		if token.TokenGroupKey() == groupKey {
			res = append(res, token)
		}
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("no tokens found for group key %s", groupKey)
	}
	return res, nil
}

// GetToken returns a token for a specific chain ID that belongs to a specific group (determined by the group key).
func (tm *Manager) GetToken(chainID uint64, groupedTokensKey string) *tokenTypes.Token {
	tokens, err := tm.GetAllTokens()
	if err != nil {
		return nil
	}

	for _, token := range tokens {
		if token.ChainID == chainID && token.TokenGroupKey() == groupedTokensKey {
			return token
		}
	}
	return nil
}
