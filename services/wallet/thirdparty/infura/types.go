package infura

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
	case "ETHEREUM":
		chainID = walletCommon.EthereumMainnet
	case "ARBITRUM":
		chainID = walletCommon.ArbitrumMainnet
	case "GOERLI":
		chainID = walletCommon.EthereumGoerli
	case "SEPOLIA":
		chainID = walletCommon.EthereumSepolia
	}
	return walletCommon.ChainID(chainID)
}

type CollectibleOwner struct {
	ContractAddress common.Address `json:"tokenAddress"`
	TokenID         *bigint.BigInt `json:"tokenId"`
	Amount          *bigint.BigInt `json:"amount"`
	OwnerAddress    common.Address `json:"ownerOf"`
}

type CollectibleContractOwnership struct {
	Owners  []CollectibleOwner `json:"owners"`
	Network string             `json:"network"`
	Cursor  string             `json:"cursor"`
}

func infuraOwnershipToCommon(contractAddress common.Address, ownersMap map[common.Address][]CollectibleOwner) (*thirdparty.CollectibleContractOwnership, error) {
	owners := make([]thirdparty.CollectibleOwner, 0, len(ownersMap))

	for ownerAddress, ownerTokens := range ownersMap {
		tokenBalances := make([]thirdparty.TokenBalance, 0, len(ownerTokens))

		for _, token := range ownerTokens {
			tokenBalances = append(tokenBalances, thirdparty.TokenBalance{
				TokenID: token.TokenID,
				Balance: token.Amount,
			})
		}

		owners = append(owners, thirdparty.CollectibleOwner{
			OwnerAddress:  ownerAddress,
			TokenBalances: tokenBalances,
		})
	}

	ownership := thirdparty.CollectibleContractOwnership{
		ContractAddress: contractAddress,
		Owners:          owners,
	}

	return &ownership, nil
}

type AttributeValue string

func (st *AttributeValue) UnmarshalJSON(b []byte) error {
	var item interface{}
	if err := json.Unmarshal(b, &item); err != nil {
		return err
	}

	switch v := item.(type) {
	case float64:
		*st = AttributeValue(strconv.FormatFloat(v, 'f', 2, 64))
	case int:
		*st = AttributeValue(strconv.Itoa(v))
	case string:
		*st = AttributeValue(v)
	}
	return nil
}

type Attribute struct {
	TraitType string         `json:"trait_type"`
	Value     AttributeValue `json:"value"`
}

type AssetMetadata struct {
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Permalink    string      `json:"permalink"`
	ImageURL     string      `json:"image"`
	AnimationURL string      `json:"animation_url"`
	Attributes   []Attribute `json:"attributes"`
}

type ContractMetadata struct {
	ContractAddress string `json:"contract"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	TokenType       string `json:"tokenType"`
}

type Asset struct {
	ContractAddress common.Address `json:"contract"`
	TokenID         *bigint.BigInt `json:"tokenId"`
	Metadata        AssetMetadata  `json:"metadata"`
}

type NFTList struct {
	Total      *bigint.BigInt `json:"total"`
	PageNumber int            `json:"pageNumber"`
	PageSize   int            `json:"pageSize"`
	Network    string         `json:"network"`
	Account    string         `json:"account"`
	Cursor     string         `json:"cursor"`
	Assets     []Asset        `json:"assets"`
}

func (c *Asset) toCollectiblesData(id thirdparty.CollectibleUniqueID) thirdparty.CollectibleData {
	return thirdparty.CollectibleData{
		ID:           id,
		Name:         c.Metadata.Name,
		Description:  c.Metadata.Description,
		Permalink:    c.Metadata.Permalink,
		ImageURL:     c.Metadata.ImageURL,
		AnimationURL: c.Metadata.AnimationURL,
		Traits:       infuraToCollectibleTraits(c.Metadata.Attributes),
	}
}

func (c *Asset) toCommon(id thirdparty.CollectibleUniqueID) thirdparty.FullCollectibleData {
	return thirdparty.FullCollectibleData{
		CollectibleData: c.toCollectiblesData(id),
		CollectionData:  nil,
	}
}

func (l *NFTList) toCommon() []thirdparty.FullCollectibleData {
	ret := make([]thirdparty.FullCollectibleData, 0, len(l.Assets))
	for _, asset := range l.Assets {
		id := thirdparty.CollectibleUniqueID{
			ContractID: thirdparty.ContractID{
				ChainID: chainStringToChainID(l.Network),
				Address: asset.ContractAddress,
			},
			TokenID: asset.TokenID,
		}
		item := asset.toCommon(id)
		ret = append(ret, item)
	}
	return ret
}

func infuraToCollectibleTraits(attributes []Attribute) []thirdparty.CollectibleTrait {
	ret := make([]thirdparty.CollectibleTrait, 0, len(attributes))
	caser := cases.Title(language.Und, cases.NoLower)
	for _, orig := range attributes {
		dest := thirdparty.CollectibleTrait{
			TraitType: strings.Replace(orig.TraitType, "_", " ", 1),
			Value:     caser.String(string(orig.Value)),
		}

		ret = append(ret, dest)
	}
	return ret
}

func (c *ContractMetadata) toCommon(id thirdparty.ContractID) thirdparty.CollectionData {
	return thirdparty.CollectionData{
		ID:   id,
		Name: c.Name,
	}
}
