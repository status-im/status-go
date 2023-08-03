package opensea

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func chainStringToChainID(chainString string) walletCommon.ChainID {
	chainID := walletCommon.UnknownChainID
	switch chainString {
	case "ethereum":
		chainID = walletCommon.EthereumMainnet
	case "arbitrum":
		chainID = walletCommon.ArbitrumMainnet
	case "optimism":
		chainID = walletCommon.OptimismMainnet
	case "goerli":
		chainID = walletCommon.EthereumGoerli
	case "arbitrum_goerli":
		chainID = walletCommon.ArbitrumGoerli
	case "optimism_goerli":
		chainID = walletCommon.OptimismGoerli
	}
	return walletCommon.ChainID(chainID)
}

type TraitValue string

func (st *TraitValue) UnmarshalJSON(b []byte) error {
	var item interface{}
	if err := json.Unmarshal(b, &item); err != nil {
		return err
	}

	switch v := item.(type) {
	case float64:
		*st = TraitValue(strconv.FormatFloat(v, 'f', 2, 64))
	case int:
		*st = TraitValue(strconv.Itoa(v))
	case string:
		*st = TraitValue(v)

	}
	return nil
}

type AssetContainer struct {
	Assets         []Asset `json:"assets"`
	NextCursor     string  `json:"next"`
	PreviousCursor string  `json:"previous"`
}

type Contract struct {
	Address         string `json:"address"`
	ChainIdentifier string `json:"chain_identifier"`
}

type Trait struct {
	TraitType   string     `json:"trait_type"`
	Value       TraitValue `json:"value"`
	DisplayType string     `json:"display_type"`
	MaxValue    string     `json:"max_value"`
}

type PaymentToken struct {
	ID       int    `json:"id"`
	Symbol   string `json:"symbol"`
	Address  string `json:"address"`
	ImageURL string `json:"image_url"`
	Name     string `json:"name"`
	Decimals int    `json:"decimals"`
	EthPrice string `json:"eth_price"`
	UsdPrice string `json:"usd_price"`
}

type LastSale struct {
	PaymentToken PaymentToken `json:"payment_token"`
}

type SellOrder struct {
	CurrentPrice string `json:"current_price"`
}

type Asset struct {
	ID                int            `json:"id"`
	TokenID           *bigint.BigInt `json:"token_id"`
	Name              string         `json:"name"`
	Description       string         `json:"description"`
	Permalink         string         `json:"permalink"`
	ImageThumbnailURL string         `json:"image_thumbnail_url"`
	ImageURL          string         `json:"image_url"`
	AnimationURL      string         `json:"animation_url"`
	Contract          Contract       `json:"asset_contract"`
	Collection        Collection     `json:"collection"`
	Traits            []Trait        `json:"traits"`
	LastSale          LastSale       `json:"last_sale"`
	SellOrders        []SellOrder    `json:"sell_orders"`
	BackgroundColor   string         `json:"background_color"`
	TokenURI          string         `json:"token_metadata"`
}

type CollectionTrait struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type Collection struct {
	Name     string                     `json:"name"`
	Slug     string                     `json:"slug"`
	ImageURL string                     `json:"image_url"`
	Traits   map[string]CollectionTrait `json:"traits"`
}

type OwnedCollection struct {
	Collection
	OwnedAssetCount *bigint.BigInt `json:"owned_asset_count"`
}

type AssetContract struct {
	Collection Collection `json:"collection"`
}

func (c *Asset) id() thirdparty.CollectibleUniqueID {
	return thirdparty.CollectibleUniqueID{
		ContractID: thirdparty.ContractID{
			ChainID: chainStringToChainID(c.Contract.ChainIdentifier),
			Address: common.HexToAddress(c.Contract.Address),
		},
		TokenID: c.TokenID,
	}
}

func openseaToCollectibleTraits(traits []Trait) []thirdparty.CollectibleTrait {
	ret := make([]thirdparty.CollectibleTrait, 0, len(traits))
	caser := cases.Title(language.Und, cases.NoLower)
	for _, orig := range traits {
		dest := thirdparty.CollectibleTrait{
			TraitType:   strings.Replace(orig.TraitType, "_", " ", 1),
			Value:       caser.String(string(orig.Value)),
			DisplayType: orig.DisplayType,
			MaxValue:    orig.MaxValue,
		}

		ret = append(ret, dest)
	}
	return ret
}

func (c *Collection) toCollectionData(id thirdparty.ContractID) thirdparty.CollectionData {
	ret := thirdparty.CollectionData{
		ID:       id,
		Name:     c.Name,
		Slug:     c.Slug,
		ImageURL: c.ImageURL,
		Traits:   make(map[string]thirdparty.CollectionTrait),
	}
	for traitType, trait := range c.Traits {
		ret.Traits[traitType] = thirdparty.CollectionTrait{
			Min: trait.Min,
			Max: trait.Max,
		}
	}
	return ret
}

func (c *Asset) toCollectiblesData() thirdparty.CollectibleData {
	return thirdparty.CollectibleData{
		ID:              c.id(),
		Name:            c.Name,
		Description:     c.Description,
		Permalink:       c.Permalink,
		ImageURL:        c.ImageURL,
		AnimationURL:    c.AnimationURL,
		Traits:          openseaToCollectibleTraits(c.Traits),
		BackgroundColor: c.BackgroundColor,
		TokenURI:        c.TokenURI,
	}
}

func (c *Asset) toCommon() thirdparty.FullCollectibleData {
	collection := c.Collection.toCollectionData(c.id().ContractID)
	return thirdparty.FullCollectibleData{
		CollectibleData: c.toCollectiblesData(),
		CollectionData:  &collection,
	}
}
