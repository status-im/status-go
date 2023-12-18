package collectibles

import (
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

// Combined Collection+Collectible info, used to display a detailed view of a collectible
type Collectible struct {
	DataType        CollectibleDataType            `json:"data_type"`
	ID              thirdparty.CollectibleUniqueID `json:"id"`
	CollectibleData *CollectibleData               `json:"collectible_data,omitempty"`
	CollectionData  *CollectionData                `json:"collection_data,omitempty"`
	CommunityData   *CommunityData                 `json:"community_data,omitempty"`
	Ownership       []thirdparty.AccountBalance    `json:"ownership,omitempty"`
}

type CollectibleData struct {
	Name               string                         `json:"name"`
	Description        *string                        `json:"description,omitempty"`
	ImageURL           *string                        `json:"image_url,omitempty"`
	AnimationURL       *string                        `json:"animation_url,omitempty"`
	AnimationMediaType *string                        `json:"animation_media_type,omitempty"`
	Traits             *[]thirdparty.CollectibleTrait `json:"traits,omitempty"`
	BackgroundColor    *string                        `json:"background_color,omitempty"`
}

type CollectionData struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	ImageURL string `json:"image_url"`
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

func fullCollectibleDataToHeader(c thirdparty.FullCollectibleData) Collectible {
	ret := Collectible{
		DataType: CollectibleDataTypeHeader,
		ID:       c.CollectibleData.ID,
		CollectibleData: &CollectibleData{
			Name:               c.CollectibleData.Name,
			ImageURL:           &c.CollectibleData.ImageURL,
			AnimationURL:       &c.CollectibleData.AnimationURL,
			AnimationMediaType: &c.CollectibleData.AnimationMediaType,
			BackgroundColor:    &c.CollectibleData.BackgroundColor,
		},
	}
	if c.CollectionData != nil {
		ret.CollectionData = &CollectionData{
			Name:     c.CollectionData.Name,
			Slug:     c.CollectionData.Slug,
			ImageURL: c.CollectionData.ImageURL,
		}
	}

	return ret
}

func fullCollectibleDataToDetails(c thirdparty.FullCollectibleData) Collectible {
	ret := Collectible{
		DataType: CollectibleDataTypeHeader,
		ID:       c.CollectibleData.ID,
		CollectibleData: &CollectibleData{
			Name:               c.CollectibleData.Name,
			Description:        &c.CollectibleData.Description,
			ImageURL:           &c.CollectibleData.ImageURL,
			AnimationURL:       &c.CollectibleData.AnimationURL,
			AnimationMediaType: &c.CollectibleData.AnimationMediaType,
			BackgroundColor:    &c.CollectibleData.BackgroundColor,
			Traits:             &c.CollectibleData.Traits,
		},
	}
	if c.CollectionData != nil {
		ret.CollectionData = &CollectionData{
			Name:     c.CollectionData.Name,
			Slug:     c.CollectionData.Slug,
			ImageURL: c.CollectionData.ImageURL,
		}
	}
	return ret
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
