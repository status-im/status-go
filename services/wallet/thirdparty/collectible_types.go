package thirdparty

import (
	"database/sql"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/services/wallet/bigint"
	w_common "github.com/status-im/status-go/services/wallet/common"
)

var (
	ErrChainIDNotSupported = errors.New("chainID not supported")
)

const FetchNoLimit = 0
const FetchFromStartCursor = ""

type CollectibleProvider interface {
	ID() string
	IsChainSupported(chainID w_common.ChainID) bool
	IsConnected() bool
}

type ContractID struct {
	ChainID w_common.ChainID `json:"chainID"`
	Address common.Address   `json:"address"`
}

func (k *ContractID) HashKey() string {
	return fmt.Sprintf("%d+%s", k.ChainID, k.Address.String())
}

type CollectibleUniqueID struct {
	ContractID ContractID     `json:"contractID"`
	TokenID    *bigint.BigInt `json:"tokenID"`
}

func (k *CollectibleUniqueID) HashKey() string {
	return fmt.Sprintf("%s+%s", k.ContractID.HashKey(), k.TokenID.String())
}

func (k *CollectibleUniqueID) Same(other *CollectibleUniqueID) bool {
	return k.ContractID.ChainID == other.ContractID.ChainID && k.ContractID.Address == other.ContractID.Address && k.TokenID.Cmp(other.TokenID.Int) == 0
}

func RowsToCollectibles(rows *sql.Rows) ([]CollectibleUniqueID, error) {
	var ids []CollectibleUniqueID
	for rows.Next() {
		id := CollectibleUniqueID{
			TokenID: &bigint.BigInt{Int: big.NewInt(0)},
		}
		err := rows.Scan(
			&id.ContractID.ChainID,
			&id.ContractID.Address,
			(*bigint.SQLBigIntBytes)(id.TokenID.Int),
		)
		if err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	return ids, nil
}

func GroupCollectibleUIDsByChainID(uids []CollectibleUniqueID) map[w_common.ChainID][]CollectibleUniqueID {
	ret := make(map[w_common.ChainID][]CollectibleUniqueID)

	for _, uid := range uids {
		if _, ok := ret[uid.ContractID.ChainID]; !ok {
			ret[uid.ContractID.ChainID] = make([]CollectibleUniqueID, 0, len(uids))
		}
		ret[uid.ContractID.ChainID] = append(ret[uid.ContractID.ChainID], uid)
	}

	return ret
}

func GroupContractIDsByChainID(ids []ContractID) map[w_common.ChainID][]ContractID {
	ret := make(map[w_common.ChainID][]ContractID)

	for _, id := range ids {
		if _, ok := ret[id.ChainID]; !ok {
			ret[id.ChainID] = make([]ContractID, 0, len(ids))
		}
		ret[id.ChainID] = append(ret[id.ChainID], id)
	}

	return ret
}

type CollectionTrait struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// Collection info
type CollectionData struct {
	ID          ContractID                 `json:"id"`
	CommunityID string                     `json:"community_id"`
	Provider    string                     `json:"provider"`
	Name        string                     `json:"name"`
	Slug        string                     `json:"slug"`
	ImageURL    string                     `json:"image_url"`
	Traits      map[string]CollectionTrait `json:"traits"`
}

type CollectibleTrait struct {
	TraitType   string `json:"trait_type"`
	Value       string `json:"value"`
	DisplayType string `json:"display_type"`
	MaxValue    string `json:"max_value"`
}

// Collectible info
type CollectibleData struct {
	ID                 CollectibleUniqueID `json:"id"`
	CommunityID        string              `json:"community_id"`
	Provider           string              `json:"provider"`
	Name               string              `json:"name"`
	Description        string              `json:"description"`
	Permalink          string              `json:"permalink"`
	ImageURL           string              `json:"image_url"`
	AnimationURL       string              `json:"animation_url"`
	AnimationMediaType string              `json:"animation_media_type"`
	Traits             []CollectibleTrait  `json:"traits"`
	BackgroundColor    string              `json:"background_color"`
	TokenURI           string              `json:"token_uri"`
}

// Combined Collection+Collectible info returned by the CollectibleProvider
// Some providers may not return the CollectionData in the same API call, so it's optional
type FullCollectibleData struct {
	CollectibleData CollectibleData
	CollectionData  *CollectionData
}

type CollectiblesContainer[T any] struct {
	Items          []T
	NextCursor     string
	PreviousCursor string
}

type CollectibleOwnershipContainer CollectiblesContainer[CollectibleUniqueID]
type CollectionDataContainer CollectiblesContainer[CollectionData]
type CollectibleDataContainer CollectiblesContainer[CollectibleData]
type FullCollectibleDataContainer CollectiblesContainer[FullCollectibleData]

// Tried to find a way to make this generic, but couldn't, so the code below is duplicated somewhere else
func collectibleItemsToIDs(items []FullCollectibleData) []CollectibleUniqueID {
	ret := make([]CollectibleUniqueID, 0, len(items))
	for _, item := range items {
		ret = append(ret, item.CollectibleData.ID)
	}
	return ret
}

func (c *FullCollectibleDataContainer) ToOwnershipContainer() CollectibleOwnershipContainer {
	return CollectibleOwnershipContainer{
		Items:          collectibleItemsToIDs(c.Items),
		NextCursor:     c.NextCursor,
		PreviousCursor: c.PreviousCursor,
	}
}

// Community-related info. Present only for collectibles minted in a community.
// This info is directly fetched every time upon request since a change in community
// settings could affect it.

type CollectiblesCommunityInfo struct {
	CommunityID     string                `json:"community_id"`
	CommunityName   string                `json:"community_name"`
	CommunityColor  string                `json:"community_color"`
	CommunityImage  string                `json:"community_image"`
	PrivilegesLevel token.PrivilegesLevel `json:"privileges_level"`
}

type CollectibleMetadataProvider interface {
	CanProvideCollectibleMetadata(id CollectibleUniqueID, tokenURI string) (bool, error)
	FetchCollectibleMetadata(id CollectibleUniqueID, tokenURI string) (*FullCollectibleData, error)
}

type CollectibleCommunityInfoProvider interface {
	FetchCollectibleCommunityInfo(communityID string, id CollectibleUniqueID) (*CollectiblesCommunityInfo, error)
	FetchCollectibleCommunityTraits(communityID string, id CollectibleUniqueID) ([]CollectibleTrait, error)
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
	CollectibleProvider
	FetchCollectibleOwnersByContractAddress(chainID w_common.ChainID, contractAddress common.Address) (*CollectibleContractOwnership, error)
}

type CollectibleAccountOwnershipProvider interface {
	CollectibleProvider
	FetchAllAssetsByOwner(chainID w_common.ChainID, owner common.Address, cursor string, limit int) (*FullCollectibleDataContainer, error)
	FetchAllAssetsByOwnerAndContractAddress(chainID w_common.ChainID, owner common.Address, contractAddresses []common.Address, cursor string, limit int) (*FullCollectibleDataContainer, error)
}

type CollectibleDataProvider interface {
	CollectibleProvider
	FetchAssetsByCollectibleUniqueID(uniqueIDs []CollectibleUniqueID) ([]FullCollectibleData, error)
}

type CollectionDataProvider interface {
	CollectibleProvider
	FetchCollectionsDataByContractID(ids []ContractID) ([]CollectionData, error)
}
