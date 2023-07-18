package thirdparty

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/bigint"
	w_common "github.com/status-im/status-go/services/wallet/common"
)

type CollectibleUniqueID struct {
	ChainID         w_common.ChainID `json:"chainID"`
	ContractAddress common.Address   `json:"contractAddress"`
	TokenID         *bigint.BigInt   `json:"tokenID"`
}

func (k *CollectibleUniqueID) HashKey() string {
	return fmt.Sprintf("%d+%s+%s", k.ChainID, k.ContractAddress.String(), k.TokenID.String())
}

func GroupCollectibleUIDsByChainID(uids []CollectibleUniqueID) map[w_common.ChainID][]CollectibleUniqueID {
	ret := make(map[w_common.ChainID][]CollectibleUniqueID)

	for _, uid := range uids {
		if _, ok := ret[uid.ChainID]; !ok {
			ret[uid.ChainID] = make([]CollectibleUniqueID, 0, len(uids))
		}
		ret[uid.ChainID] = append(ret[uid.ChainID], uid)
	}

	return ret
}

type CollectionTrait struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type CollectionData struct {
	Name     string                     `json:"name"`
	Slug     string                     `json:"slug"`
	ImageURL string                     `json:"image_url"`
	Traits   map[string]CollectionTrait `json:"traits"`
}

type CollectibleTrait struct {
	TraitType   string `json:"trait_type"`
	Value       string `json:"value"`
	DisplayType string `json:"display_type"`
	MaxValue    string `json:"max_value"`
}

type CollectibleData struct {
	ID                 CollectibleUniqueID `json:"id"`
	Name               string              `json:"name"`
	Description        string              `json:"description"`
	Permalink          string              `json:"permalink"`
	ImageURL           string              `json:"image_url"`
	AnimationURL       string              `json:"animation_url"`
	AnimationMediaType string              `json:"animation_media_type"`
	Traits             []CollectibleTrait  `json:"traits"`
	BackgroundColor    string              `json:"background_color"`
	TokenURI           string              `json:"token_uri"`
	CollectionData     CollectionData      `json:"collection_data"`
}

type CollectibleHeader struct {
	ID                 CollectibleUniqueID `json:"id"`
	Name               string              `json:"name"`
	ImageURL           string              `json:"image_url"`
	AnimationURL       string              `json:"animation_url"`
	AnimationMediaType string              `json:"animation_media_type"`
	BackgroundColor    string              `json:"background_color"`
	CollectionName     string              `json:"collection_name"`
}

type CollectibleDataContainer struct {
	Collectibles   []CollectibleData
	NextCursor     string
	PreviousCursor string
}

func (c *CollectibleData) toHeader() CollectibleHeader {
	return CollectibleHeader{
		ID:                 c.ID,
		Name:               c.Name,
		ImageURL:           c.ImageURL,
		AnimationURL:       c.AnimationURL,
		AnimationMediaType: c.AnimationMediaType,
		BackgroundColor:    c.BackgroundColor,
		CollectionName:     c.CollectionData.Name,
	}
}

func CollectiblesToHeaders(collectibles []CollectibleData) []CollectibleHeader {
	res := make([]CollectibleHeader, 0, len(collectibles))

	for _, c := range collectibles {
		res = append(res, c.toHeader())
	}

	return res
}

type CollectibleOwnershipProvider interface {
	CanProvideAccountOwnership(chainID uint64) (bool, error)
	FetchAccountOwnership(chainID uint64, address common.Address) (*CollectibleData, error)
}

type CollectibleMetadataProvider interface {
	CanProvideCollectibleMetadata(id CollectibleUniqueID, tokenURI string) (bool, error)
	FetchCollectibleMetadata(id CollectibleUniqueID, tokenURI string) (*CollectibleData, error)
}

type TokenBalance struct {
	TokenID *bigint.BigInt `json:"tokenId"`
	Balance *bigint.BigInt `json:"balance"`
}

type TokenBalancesPerContractAddress = map[common.Address][]TokenBalance

type CollectibleOwner struct {
	OwnerAddress  common.Address `json:"ownerAddress"`
	TokenBalances []TokenBalance `json:"tokenBalances"`
}

type CollectibleContractOwnership struct {
	ContractAddress common.Address     `json:"contractAddress"`
	Owners          []CollectibleOwner `json:"owners"`
}

type CollectibleContractOwnershipProvider interface {
	FetchCollectibleOwnersByContractAddress(chainID w_common.ChainID, contractAddress common.Address) (*CollectibleContractOwnership, error)
	IsChainSupported(chainID w_common.ChainID) bool
}
