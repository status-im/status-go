package tokentypes

import (
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	walletcommon "github.com/status-im/status-go/services/wallet/common"
)

const (
	tokenKeySeparator = "-" // export this from wallet-sdk
)

type Token struct {
	*types.Token

	CollectibleTokenID *hexutil.Big   `json:"collectibleTokenID,omitempty"`
	CommunityData      *CommunityData `json:"communityData,omitempty"`

	Soulbound       bool `json:"soulbound,omitempty"`
	PrivilegesLevel *int `json:"privilegesLevel,omitempty"`
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

// EthereumMainnetSibling returns the Ethereum mainnet token sharing the same CrossChainID
func EthereumMainnetSibling(token *Token, candidates []*Token) *Token {
	if token == nil || token.CrossChainID == "" ||
		token.ChainID == walletcommon.EthereumMainnet {
		return nil
	}

	for _, candidate := range candidates {
		if candidate != nil &&
			candidate.ChainID == walletcommon.EthereumMainnet &&
			candidate.CrossChainID == token.CrossChainID {
			return candidate
		}
	}

	return nil
}

func ParseCollectibleKey(tokenKey string) (chainID uint64, contractAddress common.Address, collectibleTokenID *big.Int, success bool) {
	success = false

	parts := strings.Split(tokenKey, tokenKeySeparator)
	if len(parts) != 3 {
		return
	}
	chainID, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return
	}
	contractAddress = common.HexToAddress(parts[1])
	collectibleTokenID, success = new(big.Int).SetString(parts[2], 10)
	return
}
