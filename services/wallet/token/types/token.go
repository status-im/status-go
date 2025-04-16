package tokentypes

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/community"

	wallet_common "github.com/status-im/status-go/services/wallet/common"
)

type Token struct {
	Address common.Address `json:"address"`
	Name    string         `json:"name"`
	Symbol  string         `json:"symbol"`
	// DONT USE THE FIELD BELOW
	TmpSymbol string `json:"-"` // TODO: this is just a temporary solution to solve the collision, remove this when switching to CoinGecko tokens list
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
	if t.ChainID == wallet_common.BSCMainnet ||
		t.ChainID == wallet_common.BSCTestnet {
		return strings.EqualFold(t.Symbol, "BNB")
	}
	return strings.EqualFold(t.Symbol, "ETH")
}
