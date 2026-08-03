package alchemy

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/collectibles/ownership"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const AlchemyID = "alchemy"
const AlchemyProxyID = "alchemy-proxy"

type TokenBalance struct {
	TokenID *bigint.BigInt `json:"tokenId"`
	Balance *bigint.BigInt `json:"balance"`
}

type CollectibleOwner struct {
	OwnerAddress  common.Address `json:"ownerAddress"`
	TokenBalances []TokenBalance `json:"tokenBalances"`
}

type CollectibleContractOwnership struct {
	Owners  []CollectibleOwner `json:"owners"`
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

type RawFull struct {
	RawMetadata RawMetadata `json:"metadata"`
}

type Raw struct {
	RawMetadata interface{} `json:"metadata"`
}

func (r *Raw) UnmarshalJSON(b []byte) error {
	raw := RawFull{
		RawMetadata{
			Attributes: make([]Attribute, 0),
		},
	}

	// Field structure is not known in advance
	_ = json.Unmarshal(b, &raw)

	r.RawMetadata = raw.RawMetadata
	return nil
}

type OpenSeaMetadata struct {
	ImageURL        string `json:"imageUrl"`
	TwitterUsername string `json:"twitterUsername"`
	ExternalURL     string `json:"externalUrl"`
}

type Contract struct {
	Address         common.Address  `json:"address"`
	Name            string          `json:"name"`
	Symbol          string          `json:"symbol"`
	TokenType       string          `json:"tokenType"`
	OpenSeaMetadata OpenSeaMetadata `json:"openseaMetadata"`
}

type ContractList struct {
	Contracts []Contract `json:"contracts"`
}

type Image struct {
	ImageURL string `json:"pngUrl"`
	// ContentType describes the media behind CachedAnimationURL, which the
	// provider populates for still assets too.
	ContentType string `json:"contentType"`
	// ThumbnailURL is Alchemy's own resized preview, small enough for a list
	// element. Absent for some assets, in which case consumers fall back to
	// ImageURL.
	ThumbnailURL         string `json:"thumbnailUrl"`
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
	Balance     *bigint.BigInt `json:"balance,omitempty"`
	AcquiredAt  *AcquiredAt    `json:"acquiredAt,omitempty"`
}

type AcquiredAt struct {
	BlockTimestamp *time.Time `json:"blockTimestamp"`
}

type OwnedNFTList struct {
	OwnedNFTs  []Asset        `json:"ownedNfts"`
	TotalCount *bigint.BigInt `json:"totalCount"`
	PageKey    string         `json:"pageKey"`
}

type NFTList struct {
	NFTs []Asset `json:"nfts"`
}

type BatchContractAddresses struct {
	Addresses []common.Address `json:"contractAddresses"`
}

type BatchTokenIDs struct {
	IDs []TokenID `json:"tokens"`
}

type TokenID struct {
	ContractAddress common.Address `json:"contractAddress"`
	TokenID         *bigint.BigInt `json:"tokenId"`
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

func alchemyToContractType(tokenType string) walletCommon.ContractType {
	switch tokenType {
	case "ERC721":
		return walletCommon.ContractTypeERC721
	case "ERC1155":
		return walletCommon.ContractTypeERC1155
	default:
		return walletCommon.ContractTypeUnknown
	}
}

func (c *Contract) toCollectionSocials() *thirdparty.CollectionSocials {
	return &thirdparty.CollectionSocials{
		Website:       c.OpenSeaMetadata.ExternalURL,
		TwitterHandle: c.OpenSeaMetadata.TwitterUsername,
		Provider:      AlchemyID,
	}
}

func (c *Contract) toCollectionData(id thirdparty.ContractID) thirdparty.CollectionData {
	ret := thirdparty.CollectionData{
		ID:           id,
		ContractType: alchemyToContractType(c.TokenType),
		Provider:     AlchemyID,
		Name:         c.Name,
		ImageURL:     c.OpenSeaMetadata.ImageURL,
		Traits:       make(map[string]thirdparty.CollectionTrait, 0),
		Socials:      c.toCollectionSocials(),
	}
	return ret
}

// animationURL returns the cached asset only when it can actually move.
// Alchemy fills cachedUrl for still collectibles as well, so passing it on
// unconditionally makes every still NFT look animated and costs consumers a
// full-size download for nothing.
func (c *Asset) animationURL() string {
	if !thirdparty.IsAnimatedMediaType(c.Image.ContentType) {
		return ""
	}

	return c.Image.CachedAnimationURL
}

func (c *Asset) toCollectiblesData(id thirdparty.CollectibleUniqueID) thirdparty.CollectibleData {
	rawMetadata := c.Raw.RawMetadata.(RawMetadata)

	return thirdparty.CollectibleData{
		ID:           id,
		ContractType: alchemyToContractType(c.Contract.TokenType),
		Provider:     AlchemyID,
		Name:         c.Name,
		Description:  c.Description,
		ImageURL:     c.Image.ImageURL,
		ThumbnailURL: c.Image.ThumbnailURL,
		AnimationURL: c.animationURL(),
		Traits:       alchemyToCollectibleTraits(rawMetadata.Attributes),
		TokenURI:     c.TokenURI,
	}
}

func (c *Asset) toCommon(id thirdparty.CollectibleUniqueID, owner *common.Address) thirdparty.FullCollectibleData {
	contractData := c.Contract.toCollectionData(id.ContractID)
	txTimestamp := ownership.InvalidTimestamp
	if c.AcquiredAt != nil && c.AcquiredAt.BlockTimestamp != nil && !c.AcquiredAt.BlockTimestamp.IsZero() {
		txTimestamp = c.AcquiredAt.BlockTimestamp.Unix()
	}
	var ownership []thirdparty.AccountBalance
	if owner != nil {
		ownership = []thirdparty.AccountBalance{
			{
				Address:     *owner,
				Balance:     c.Balance,
				TxTimestamp: txTimestamp,
			},
		}
	}
	return thirdparty.FullCollectibleData{
		CollectibleData: c.toCollectiblesData(id),
		CollectionData:  &contractData,
		Ownership:       ownership,
	}
}

func alchemyToCollectiblesData(chainID walletCommon.ChainID, l []Asset, owner *common.Address) []thirdparty.FullCollectibleData {
	ret := make([]thirdparty.FullCollectibleData, 0, len(l))
	for _, asset := range l {
		id := thirdparty.CollectibleUniqueID{
			ContractID: thirdparty.ContractID{
				ChainID: chainID,
				Address: asset.Contract.Address,
			},
			TokenID: asset.TokenID,
		}
		item := asset.toCommon(id, owner)
		ret = append(ret, item)
	}
	return ret
}

func alchemyToCollectionsData(chainID walletCommon.ChainID, l []Contract) []thirdparty.CollectionData {
	ret := make([]thirdparty.CollectionData, 0, len(l))
	for _, contract := range l {
		id := thirdparty.ContractID{
			ChainID: chainID,
			Address: contract.Address,
		}
		item := contract.toCollectionData(id)
		ret = append(ret, item)
	}
	return ret
}
