package collectibles

import (
	"github.com/status-im/status-go/protocol/communities/token"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

// Combined Collection+Collectible info, used to display a detailed view of a collectible
type Collectible struct {
	DataType        CollectibleDataType            `json:"data_type"`
	ID              thirdparty.CollectibleUniqueID `json:"id"`
	ContractType    w_common.ContractType          `json:"contract_type"`
	CollectibleData *CollectibleData               `json:"collectible_data,omitempty"`
	CollectionData  *CollectionData                `json:"collection_data,omitempty"`
	CommunityData   *CommunityData                 `json:"community_data,omitempty"`
	Ownership       []thirdparty.AccountBalance    `json:"ownership,omitempty"`
	IsFirst         bool                           `json:"is_first,omitempty"`
	LatestTxHash    string                         `json:"latest_tx_hash,omitempty"`
	ReceivedAmount  float64                        `json:"received_amount,omitempty"`
}

type CollectibleData struct {
	Name               string                         `json:"name"`
	Description        *string                        `json:"description,omitempty"`
	ImageURL           *string                        `json:"image_url,omitempty"`
	ThumbnailURL       *string                        `json:"thumbnail_url,omitempty"`
	AnimationURL       *string                        `json:"animation_url,omitempty"`
	AnimationMediaType *string                        `json:"animation_media_type,omitempty"`
	Traits             *[]thirdparty.CollectibleTrait `json:"traits,omitempty"`
	BackgroundColor    *string                        `json:"background_color,omitempty"`
	Soulbound          *bool                          `json:"soulbound,omitempty"`
}

type CollectionData struct {
	Name     string             `json:"name"`
	Slug     string             `json:"slug"`
	ImageURL string             `json:"image_url"`
	Socials  *CollectionSocials `json:"socials"`
}

type CollectionSocials struct {
	Website       string `json:"website"`
	TwitterHandle string `json:"twitter_handle"`
}

type CommunityData struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	Color           string                `json:"color"`
	PrivilegesLevel token.PrivilegesLevel `json:"privileges_level"`
	ImageURL        *string               `json:"image_url,omitempty"`
}

func idToCollectible(id thirdparty.CollectibleUniqueID) Collectible {
	ret := Collectible{
		DataType: CollectibleDataTypeUniqueID,
		ID:       id,
	}
	return ret
}

func idsToCollectibles(ids []thirdparty.CollectibleUniqueID) []Collectible {
	res := make([]Collectible, 0, len(ids))

	for _, id := range ids {
		c := idToCollectible(id)
		res = append(res, c)
	}

	return res
}

func thirdpartyCollectionDataToCollectionData(collectionData *thirdparty.CollectionData) CollectionData {
	ret := CollectionData{}
	if collectionData != nil {
		ret = CollectionData{
			Name:     collectionData.Name,
			Slug:     collectionData.Slug,
			ImageURL: collectionData.ImageURL,
		}
		if collectionData.Socials != nil {
			ret.Socials = &CollectionSocials{
				Website:       collectionData.Socials.Website,
				TwitterHandle: collectionData.Socials.TwitterHandle,
			}
		}
	}
	return ret
}

func getContractType(c *thirdparty.FullCollectibleData) w_common.ContractType {
	if c.CollectibleData.ContractType != w_common.ContractTypeUnknown {
		return c.CollectibleData.ContractType
	}
	if c.CollectionData != nil && c.CollectionData.ContractType != w_common.ContractTypeUnknown {
		return c.CollectionData.ContractType
	}
	return w_common.ContractTypeUnknown
}

// applyAssetSizeLimit drops the URL of every asset a provider reported as
// larger than maxSize, in place, before the data reaches a client.
//
// Dropping rather than flagging is what keeps this free of client work: an
// empty animation URL already means "this collectible is still", and an empty
// image URL already means "fall back to the thumbnail", so an oversized asset
// degrades down the chain consumers already walk and ends at the placeholder
// they already show. The cost is that there is no way to ask for it anyway —
// a client never learns the URL existed.
//
// A size of zero means the provider did not report one, which is the common
// case rather than a suspicious one: Alchemy sizes the asset it cached but not
// the renders it derives on delivery. Those pass. So does everything when
// maxSize is not set, which is how a client that has not asked for a cap gets
// today's behaviour.
func applyAssetSizeLimit(data []thirdparty.FullCollectibleData, maxSize int64) {
	if maxSize <= 0 {
		return
	}

	for i := range data {
		c := &data[i].CollectibleData

		if c.AnimationSize > maxSize {
			c.AnimationURL = ""
			c.AnimationMediaType = ""
			c.AnimationSize = 0
		}
		if c.ImageSize > maxSize {
			c.ImageURL = ""
			c.ImageSize = 0
		}
		if c.ThumbnailSize > maxSize {
			c.ThumbnailURL = ""
			c.ThumbnailSize = 0
		}
	}
}

func fullCollectibleDataToHeader(c *thirdparty.FullCollectibleData) Collectible {
	ret := Collectible{
		DataType:     CollectibleDataTypeHeader,
		ID:           c.CollectibleData.ID,
		ContractType: getContractType(c),
		CollectibleData: &CollectibleData{
			Name:               c.CollectibleData.Name,
			ImageURL:           &c.CollectibleData.ImageURL,
			ThumbnailURL:       &c.CollectibleData.ThumbnailURL,
			AnimationURL:       &c.CollectibleData.AnimationURL,
			AnimationMediaType: &c.CollectibleData.AnimationMediaType,
			BackgroundColor:    &c.CollectibleData.BackgroundColor,
			Soulbound:          &c.CollectibleData.Soulbound,
		},
	}
	collectionData := thirdpartyCollectionDataToCollectionData(c.CollectionData)
	ret.CollectionData = &collectionData
	if c.CollectibleData.CommunityID != "" {
		communityData := communityInfoToData(c.CollectibleData.CommunityID, c.CommunityInfo, c.CollectibleCommunityInfo)
		ret.CommunityData = &communityData
	}
	ret.Ownership = c.Ownership
	return ret
}

func fullCollectiblesDataToHeaders(data []thirdparty.FullCollectibleData) []Collectible {
	res := make([]Collectible, 0, len(data))

	for _, c := range data {
		header := fullCollectibleDataToHeader(&c)
		res = append(res, header)
	}

	return res
}

func fullCollectibleDataToDetails(c *thirdparty.FullCollectibleData) Collectible {
	ret := Collectible{
		DataType:     CollectibleDataTypeDetails,
		ID:           c.CollectibleData.ID,
		ContractType: getContractType(c),
		CollectibleData: &CollectibleData{
			Name:               c.CollectibleData.Name,
			Description:        &c.CollectibleData.Description,
			ImageURL:           &c.CollectibleData.ImageURL,
			ThumbnailURL:       &c.CollectibleData.ThumbnailURL,
			AnimationURL:       &c.CollectibleData.AnimationURL,
			AnimationMediaType: &c.CollectibleData.AnimationMediaType,
			BackgroundColor:    &c.CollectibleData.BackgroundColor,
			Traits:             &c.CollectibleData.Traits,
			Soulbound:          &c.CollectibleData.Soulbound,
		},
	}
	collectionData := thirdpartyCollectionDataToCollectionData(c.CollectionData)
	ret.CollectionData = &collectionData
	if c.CollectibleData.CommunityID != "" {
		communityData := communityInfoToData(c.CollectibleData.CommunityID, c.CommunityInfo, c.CollectibleCommunityInfo)
		ret.CommunityData = &communityData
	}
	ret.Ownership = c.Ownership
	return ret
}

func fullCollectiblesDataToDetails(data []thirdparty.FullCollectibleData) []Collectible {
	res := make([]Collectible, 0, len(data))

	for _, c := range data {
		details := fullCollectibleDataToDetails(&c)
		res = append(res, details)
	}

	return res
}

func fullCollectiblesDataToCommunityHeader(data []thirdparty.FullCollectibleData) []Collectible {
	res := make([]Collectible, 0, len(data))

	for _, localCollectibleData := range data {
		// to satisfy gosec: C601 checks
		c := &localCollectibleData
		collectibleID := c.CollectibleData.ID
		communityID := c.CollectibleData.CommunityID

		if communityID == "" {
			continue
		}

		communityData := communityInfoToData(communityID, c.CommunityInfo, c.CollectibleCommunityInfo)

		header := Collectible{
			ID:           collectibleID,
			ContractType: getContractType(c),
			CollectibleData: &CollectibleData{
				Name:     c.CollectibleData.Name,
				ImageURL: &c.CollectibleData.ImageURL,
			},
			CommunityData: &communityData,
			Ownership:     c.Ownership,
			IsFirst:       c.CollectibleData.IsFirst,
		}

		res = append(res, header)
	}

	return res
}

func communityInfoToData(communityID string, community *thirdparty.CommunityInfo, communityCollectible *thirdparty.CollectibleCommunityInfo) CommunityData {
	ret := CommunityData{
		ID: communityID,
	}

	if community != nil {
		ret.Name = community.CommunityName
		ret.Color = community.CommunityColor
		ret.ImageURL = &community.CommunityImage
	}

	if communityCollectible != nil {
		ret.PrivilegesLevel = communityCollectible.PrivilegesLevel
	}

	return ret
}

func IDsFromAssets(assets []*thirdparty.FullCollectibleData) []thirdparty.CollectibleUniqueID {
	result := make([]thirdparty.CollectibleUniqueID, len(assets))
	for i, asset := range assets {
		result[i] = asset.CollectibleData.ID
	}
	return result
}
