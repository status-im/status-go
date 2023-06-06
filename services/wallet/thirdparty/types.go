package thirdparty

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/bigint"
)

type HistoricalPrice struct {
	Timestamp int64   `json:"time"`
	Value     float64 `json:"close"`
}

type TokenMarketValues struct {
	MKTCAP          float64 `json:"MKTCAP"`
	HIGHDAY         float64 `json:"HIGHDAY"`
	LOWDAY          float64 `json:"LOWDAY"`
	CHANGEPCTHOUR   float64 `json:"CHANGEPCTHOUR"`
	CHANGEPCTDAY    float64 `json:"CHANGEPCTDAY"`
	CHANGEPCT24HOUR float64 `json:"CHANGEPCT24HOUR"`
	CHANGE24HOUR    float64 `json:"CHANGE24HOUR"`
}

type TokenDetails struct {
	ID                   string  `json:"Id"`
	Name                 string  `json:"Name"`
	Symbol               string  `json:"Symbol"`
	Description          string  `json:"Description"`
	TotalCoinsMined      float64 `json:"TotalCoinsMined"`
	AssetLaunchDate      string  `json:"AssetLaunchDate"`
	AssetWhitepaperURL   string  `json:"AssetWhitepaperUrl"`
	AssetWebsiteURL      string  `json:"AssetWebsiteUrl"`
	BuiltOn              string  `json:"BuiltOn"`
	SmartContractAddress string  `json:"SmartContractAddress"`
}

type MarketDataProvider interface {
	FetchPrices(symbols []string, currencies []string) (map[string]map[string]float64, error)
	FetchHistoricalDailyPrices(symbol string, currency string, limit int, allData bool, aggregate int) ([]HistoricalPrice, error)
	FetchHistoricalHourlyPrices(symbol string, currency string, limit int, aggregate int) ([]HistoricalPrice, error)
	FetchTokenMarketValues(symbols []string, currency string) (map[string]TokenMarketValues, error)
	FetchTokenDetails(symbols []string) (map[string]TokenDetails, error)
}

type NFTUniqueID struct {
	ContractAddress common.Address `json:"contractAddress"`
	TokenID         *bigint.BigInt `json:"tokenID"`
}

func (k *NFTUniqueID) HashKey() string {
	return k.ContractAddress.String() + "+" + k.TokenID.String()
}

type NFTMetadata struct {
	Name               string `json:"name"`
	Description        string `json:"description"`
	CollectionImageURL string `json:"collection_image"`
	ImageURL           string `json:"image"`
}

type NFTMetadataProvider interface {
	CanProvideNFTMetadata(chainID uint64, id NFTUniqueID, tokenURI string) (bool, error)
	FetchNFTMetadata(chainID uint64, id NFTUniqueID, tokenURI string) (*NFTMetadata, error)
}

type TokenBalance struct {
	TokenID *bigint.BigInt `json:"tokenId"`
	Balance *bigint.BigInt `json:"balance"`
}

type NFTOwner struct {
	OwnerAddress  common.Address `json:"ownerAddress"`
	TokenBalances []TokenBalance `json:"tokenBalances"`
}

type NFTContractOwnership struct {
	ContractAddress common.Address `json:"contractAddress"`
	Owners          []NFTOwner     `json:"owners"`
}

type NFTContractOwnershipProvider interface {
	FetchNFTOwnersByContractAddress(chainID uint64, contractAddress common.Address) (*NFTContractOwnership, error)
	IsChainSupported(chainID uint64) bool
}

type DataParsed struct {
	Name      string            `json:"name"`
	ID        string            `json:"id"`
	Inputs    map[string]string `json:"inputs"`
	Signature string            `json:"signature"`
}

type DecoderProvider interface {
	Run(data string) (*DataParsed, error)
}
