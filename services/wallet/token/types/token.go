package tokentypes

import (
	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"
)

type Token struct {
	*types.Token

	CommunityData *CommunityData `json:"communityData,omitempty"`
}

// TODO: think about removing CommunityData field and fetch it when needed, that way `tokenlists.Token` can be used directly.
type CommunityData struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Image string `json:"image,omitempty"`
}

type TokenList struct {
	*types.TokenList

	Tokens []*Token `json:"tokens"` // tokens from the `types.TokenList` are shadowed by this field
}
