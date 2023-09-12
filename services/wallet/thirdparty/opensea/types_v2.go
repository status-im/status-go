package opensea

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	OpenseaV2ID           = "openseaV2"
	ethereumMainnetString = "ethereum"
	arbitrumMainnetString = "arbitrum"
	optimismMainnetString = "optimism"
	ethereumGoerliString  = "goerli"
	arbitrumGoerliString  = "arbitrum_goerli"
	optimismGoerliString  = "optimism_goerli"
)

func chainStringToChainID(chainString string) walletCommon.ChainID {
	chainID := walletCommon.UnknownChainID
	switch chainString {
	case ethereumMainnetString:
		chainID = walletCommon.EthereumMainnet
	case arbitrumMainnetString:
		chainID = walletCommon.ArbitrumMainnet
	case optimismMainnetString:
		chainID = walletCommon.OptimismMainnet
	case ethereumGoerliString:
		chainID = walletCommon.EthereumGoerli
	case arbitrumGoerliString:
		chainID = walletCommon.ArbitrumGoerli
	case optimismGoerliString:
		chainID = walletCommon.OptimismGoerli
	}
	return walletCommon.ChainID(chainID)
}

func chainIDToChainString(chainID walletCommon.ChainID) string {
	chainString := ""
	switch uint64(chainID) {
	case walletCommon.EthereumMainnet:
		chainString = ethereumMainnetString
	case walletCommon.ArbitrumMainnet:
		chainString = arbitrumMainnetString
	case walletCommon.OptimismMainnet:
		chainString = optimismMainnetString
	case walletCommon.EthereumGoerli:
		chainString = ethereumGoerliString
	case walletCommon.ArbitrumGoerli:
		chainString = arbitrumGoerliString
	case walletCommon.OptimismGoerli:
		chainString = optimismGoerliString
	}
	return chainString
}

type NFTContainer struct {
	NFTs       []NFT  `json:"nfts"`
	NextCursor string `json:"next"`
}

type NFT struct {
	TokenID       *bigint.BigInt `json:"identifier"`
	Collection    string         `json:"collection"`
	Contract      common.Address `json:"contract"`
	TokenStandard string         `json:"token_standard"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	ImageURL      string         `json:"image_url"`
	MetadataURL   string         `json:"metadata_url"`
}

type DetailedNFTContainer struct {
	NFT DetailedNFT `json:"nft"`
}

type DetailedNFT struct {
	TokenID       *bigint.BigInt `json:"identifier"`
	Collection    string         `json:"collection"`
	Contract      common.Address `json:"contract"`
	TokenStandard string         `json:"token_standard"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	ImageURL      string         `json:"image_url"`
	MetadataURL   string         `json:"metadata_url"`
	Owners        []OwnerV2      `json:"owners"`
	Traits        []TraitV2      `json:"traits"`
}

type OwnerV2 struct {
	Address  common.Address `json:"address"`
	Quantity *bigint.BigInt `json:"quantity"`
}

type TraitV2 struct {
	TraitType   string     `json:"trait_type"`
	DisplayType string     `json:"display_type"`
	MaxValue    string     `json:"max_value"`
	TraitCount  int        `json:"trait_count"`
	Order       string     `json:"order"`
	Value       TraitValue `json:"value"`
}

func (c *NFT) id(chainID walletCommon.ChainID) thirdparty.CollectibleUniqueID {
	return thirdparty.CollectibleUniqueID{
		ContractID: thirdparty.ContractID{
			ChainID: chainID,
			Address: c.Contract,
		},
		TokenID: c.TokenID,
	}
}

func (c *NFT) toCollectiblesData(chainID walletCommon.ChainID) thirdparty.CollectibleData {
	return thirdparty.CollectibleData{
		ID:           c.id(chainID),
		Provider:     OpenseaV2ID,
		Name:         c.Name,
		Description:  c.Description,
		ImageURL:     c.ImageURL,
		AnimationURL: c.ImageURL,
		Traits:       make([]thirdparty.CollectibleTrait, 0),
		TokenURI:     c.MetadataURL,
	}
}

func (c *NFT) toCommon(chainID walletCommon.ChainID) thirdparty.FullCollectibleData {
	return thirdparty.FullCollectibleData{
		CollectibleData: c.toCollectiblesData(chainID),
		CollectionData:  nil,
	}
}

func openseaV2ToCollectibleTraits(traits []TraitV2) []thirdparty.CollectibleTrait {
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

func (c *DetailedNFT) id(chainID walletCommon.ChainID) thirdparty.CollectibleUniqueID {
	return thirdparty.CollectibleUniqueID{
		ContractID: thirdparty.ContractID{
			ChainID: chainID,
			Address: c.Contract,
		},
		TokenID: c.TokenID,
	}
}

func (c *DetailedNFT) toCollectiblesData(chainID walletCommon.ChainID) thirdparty.CollectibleData {
	return thirdparty.CollectibleData{
		ID:           c.id(chainID),
		Provider:     OpenseaV2ID,
		Name:         c.Name,
		Description:  c.Description,
		ImageURL:     c.ImageURL,
		AnimationURL: c.ImageURL,
		Traits:       openseaV2ToCollectibleTraits(c.Traits),
		TokenURI:     c.MetadataURL,
	}
}

func (c *DetailedNFT) toCommon(chainID walletCommon.ChainID) thirdparty.FullCollectibleData {
	return thirdparty.FullCollectibleData{
		CollectibleData: c.toCollectiblesData(chainID),
		CollectionData:  nil,
	}
}
