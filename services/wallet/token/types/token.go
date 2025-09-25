package tokentypes

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

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

type StorageToken struct {
	TokenAddress            common.Address               `json:"tokenAddress"`
	TokenChainID            uint64                       `json:"tokenChainId"`
	RawBalance              string                       `json:"rawBalance"`
	Balance                 *big.Float                   `json:"balance"`
	HasError                bool                         `json:"hasError"`
	Description             string                       `json:"description"`
	AssetWebsiteURL         string                       `json:"assetWebsiteUrl"`
	BuiltOn                 string                       `json:"builtOn"`
	MarketValuesPerCurrency map[string]TokenMarketValues `json:"marketValuesPerCurrency"`
}

type TokenList struct {
	*types.TokenList

	Tokens []*Token `json:"tokens"` // tokens from the `types.TokenList` are shadowed by this field
}
