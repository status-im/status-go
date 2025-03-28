package tokentypes

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/community"
)

const TokenKeyPattern = "%d-%s"

type Token struct {
	GroupKey string         `json:"groupKey"`
	Address  common.Address `json:"address"`
	Name     string         `json:"name"`
	Symbol   string         `json:"symbol"`
	// Decimals defines how divisible the token is. For example, 0 would be
	// indivisible, whereas 18 would allow very small amounts of the token
	// to be traded.
	Decimals uint   `json:"decimals"`
	ChainID  uint64 `json:"chainId"`
	// PegSymbol indicates that the token is pegged to some fiat currency, using the
	// ISO 4217 alphabetic code. For example, an empty string means it is not
	// pegged, while "USD" means it's pegged to the United States Dollar.
	PegSymbol string `json:"pegSymbol"`
	Image     string `json:"image,omitempty"`

	CommunityData *community.Data `json:"community_data,omitempty"`
	Verified      bool            `json:"verified"`
}

type StorageToken struct {
	Token
	BalancesPerChain        map[uint64]ChainBalance      `json:"balancesPerChain"`
	Description             string                       `json:"description"`
	AssetWebsiteURL         string                       `json:"assetWebsiteUrl"`
	BuiltOn                 string                       `json:"builtOn"`
	MarketValuesPerCurrency map[string]TokenMarketValues `json:"marketValuesPerCurrency"`
}

func (t *Token) IsNative() bool {
	return strings.EqualFold(t.Symbol, "ETH")
}

// TokenKey returns the key of the token, which is chainId + address pair.
func (t *Token) TokenKey() string {
	return fmt.Sprintf(TokenKeyPattern, t.ChainID, t.Address.Hex())
}

// TokenKey returns the id of the group of tokens where this token belongs to.
// Since the grouping tokens across chains is only possible for tokens that have a common ID across chains, the CoinGecko tokens list is used and their ID parameter.
// Once we have a standard definded, that will guarantee that the name is unique across chains we won't be tied to the CoinGecko token list.
// This PR is about it https://github.com/status-im/status-go/pull/6486
func (t *Token) TokenGroupKey() string {
	return t.GroupKey
}

type tokenAlias Token

func (t *Token) UnmarshalJSON(data []byte) error {
	aux := struct {
		*tokenAlias
		ID *string `json:"id"` // present in CoinGecko tokens list
	}{
		tokenAlias: (*tokenAlias)(t),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Set the GroupKey to ID if it's present in the JSON data
	if aux.ID != nil {
		t.GroupKey = *aux.ID
	}

	return nil
}
