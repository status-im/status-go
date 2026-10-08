package token

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/manager"
	sdktypes "github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"
)

// sdkCatalogue confines SDK DTOs to the rollback backend during migration.
// Lifecycle methods are forwarded unchanged; reads expose wallet-owned types.
type sdkCatalogue struct{ manager.Manager }

var _ tokenlist.Catalogue = (*sdkCatalogue)(nil)

func catalogueToken(token *sdktypes.Token) *tokenlist.Token {
	if token == nil {
		return nil
	}
	converted := tokenlist.Token(*token)
	return &converted
}

func catalogueTokens(tokens []*sdktypes.Token) []*tokenlist.Token {
	if tokens == nil {
		return nil
	}
	result := make([]*tokenlist.Token, len(tokens))
	for i, token := range tokens {
		result[i] = catalogueToken(token)
	}
	return result
}

func catalogueList(list *sdktypes.TokenList) *tokenlist.TokenList {
	if list == nil {
		return nil
	}
	return &tokenlist.TokenList{
		ID: list.ID, Name: list.Name, Timestamp: list.Timestamp,
		FetchedTimestamp: list.FetchedTimestamp, Source: list.Source,
		Version: tokenlist.Version(list.Version), Tags: list.Tags,
		LogoURI: list.LogoURI, Keywords: list.Keywords, Tokens: catalogueTokens(list.Tokens),
	}
}

func (m *sdkCatalogue) UniqueTokens() []*tokenlist.Token {
	return catalogueTokens(m.Manager.UniqueTokens())
}

func (m *sdkCatalogue) GetTokenByChainAddress(chain uint64, address common.Address) (*tokenlist.Token, bool) {
	token, found := m.Manager.GetTokenByChainAddress(chain, address)
	return catalogueToken(token), found
}

func (m *sdkCatalogue) GetTokensByChain(chain uint64) []*tokenlist.Token {
	return catalogueTokens(m.Manager.GetTokensByChain(chain))
}

func (m *sdkCatalogue) GetTokensByKeys(keys []string) ([]*tokenlist.Token, error) {
	tokens, err := m.Manager.GetTokensByKeys(keys)
	return catalogueTokens(tokens), err
}

func (m *sdkCatalogue) TokenLists() []*tokenlist.TokenList {
	lists := m.Manager.TokenLists()
	if lists == nil {
		return nil
	}
	result := make([]*tokenlist.TokenList, len(lists))
	for i, list := range lists {
		result[i] = catalogueList(list)
	}
	return result
}

func (m *sdkCatalogue) TokenList(id string) (*tokenlist.TokenList, bool) {
	list, found := m.Manager.TokenList(id)
	return catalogueList(list), found
}
