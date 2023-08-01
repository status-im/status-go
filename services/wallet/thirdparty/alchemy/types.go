package alchemy

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

type TokenBalance struct {
	TokenID *bigint.HexBigInt `json:"tokenId"`
	Balance *bigint.BigInt    `json:"balance"`
}

type CollectibleOwner struct {
	OwnerAddress  common.Address `json:"ownerAddress"`
	TokenBalances []TokenBalance `json:"tokenBalances"`
}

type CollectibleContractOwnership struct {
	Owners  []CollectibleOwner `json:"ownerAddresses"`
	PageKey string             `json:"pageKey"`
}

func alchemyCollectibleOwnersToCommon(alchemyOwners []CollectibleOwner) []thirdparty.CollectibleOwner {
	owners := make([]thirdparty.CollectibleOwner, 0, len(alchemyOwners))
	for _, alchemyOwner := range alchemyOwners {
		balances := make([]thirdparty.TokenBalance, 0, len(alchemyOwner.TokenBalances))

		for _, alchemyBalance := range alchemyOwner.TokenBalances {
			balances = append(balances, thirdparty.TokenBalance{
				TokenID: &bigint.BigInt{Int: alchemyBalance.TokenID.Int},
				Balance: alchemyBalance.Balance,
			})
		}
		owner := thirdparty.CollectibleOwner{
			OwnerAddress:  alchemyOwner.OwnerAddress,
			TokenBalances: balances,
		}

		owners = append(owners, owner)
	}
	return owners
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

type RawMetadata struct {
	Attributes []Attribute `json:"attributes"`
}

type Raw struct {
	RawMetadata RawMetadata `json:"metadata"`
}

type OpenSeaMetadata struct {
	ImageURL string `json:"imageUrl"`
}

type Contract struct {
	Address         common.Address  `json:"address"`
	Name            string          `json:"name"`
	Symbol          string          `json:"symbol"`
	TokenType       string          `json:"tokenType"`
	OpenSeaMetadata OpenSeaMetadata `json:"openSeaMetadata"`
}

type Image struct {
	ImageURL             string `json:"pngUrl"`
	CachedAnimationURL   string `json:"cachedUrl"`
	OriginalAnimationURL string `json:"originalUrl"`
}

type Asset struct {
	Contract    Contract       `json:"contract"`
	TokenID     *bigint.BigInt `json:"tokenId"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Image       Image          `json:"image"`
	Raw         Raw            `json:"raw"`
	TokenURI    string         `json:"tokenUri"`
}

type NFTList struct {
	OwnedNFTs  []Asset        `json:"ownedNfts"`
	TotalCount *bigint.BigInt `json:"totalCount"`
	PageKey    string         `json:"pageKey"`
}

func alchemyToCollectibleTraits(attributes []Attribute) []thirdparty.CollectibleTrait {
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

func (c *Asset) toCollectionData(id thirdparty.ContractID) thirdparty.CollectionData {
	ret := thirdparty.CollectionData{
		ID:       id,
		Name:     c.Contract.Name,
		ImageURL: c.Contract.OpenSeaMetadata.ImageURL,
	}
	return ret
}

func (c *Asset) toCollectiblesData(id thirdparty.CollectibleUniqueID) thirdparty.CollectibleData {
	return thirdparty.CollectibleData{
		ID:           id,
		Name:         c.Name,
		Description:  c.Description,
		ImageURL:     c.Image.ImageURL,
		AnimationURL: c.Image.OriginalAnimationURL,
		Traits:       alchemyToCollectibleTraits(c.Raw.RawMetadata.Attributes),
	}
}

func (c *Asset) toCommon(id thirdparty.CollectibleUniqueID) thirdparty.FullCollectibleData {
	contractData := c.toCollectionData(id.ContractID)
	return thirdparty.FullCollectibleData{
		CollectibleData: c.toCollectiblesData(id),
		CollectionData:  &contractData,
	}
}

func (l *NFTList) toCommon(chainID walletCommon.ChainID) []thirdparty.FullCollectibleData {
	ret := make([]thirdparty.FullCollectibleData, 0, len(l.OwnedNFTs))
	for _, asset := range l.OwnedNFTs {
		id := thirdparty.CollectibleUniqueID{
			ContractID: thirdparty.ContractID{
				ChainID: chainID,
				Address: asset.Contract.Address,
			},
			TokenID: asset.TokenID,
		}
		item := asset.toCommon(id)
		ret = append(ret, item)
	}
	return ret
}
