package collectibles

import (
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

// Combined Collection+Collectible info, used to display a detailed view of a collectible
type CollectibleDetails struct {
	ID                 thirdparty.CollectibleUniqueID        `json:"id"`
	Name               string                                `json:"name"`
	Description        string                                `json:"description"`
	ImageURL           string                                `json:"image_url"`
	AnimationURL       string                                `json:"animation_url"`
	AnimationMediaType string                                `json:"animation_media_type"`
	Traits             []thirdparty.CollectibleTrait         `json:"traits"`
	BackgroundColor    string                                `json:"background_color"`
	CollectionName     string                                `json:"collection_name"`
	CollectionSlug     string                                `json:"collection_slug"`
	CollectionImageURL string                                `json:"collection_image_url"`
	CommunityInfo      *thirdparty.CollectiblesCommunityInfo `json:"community_info,omitempty"`
}

// Combined Collection+Collectible info, used to display a basic view of a collectible in a list
type CollectibleHeader struct {
	ID                 thirdparty.CollectibleUniqueID `json:"id"`
	Name               string                         `json:"name"`
	ImageURL           string                         `json:"image_url"`
	AnimationURL       string                         `json:"animation_url"`
	AnimationMediaType string                         `json:"animation_media_type"`
	BackgroundColor    string                         `json:"background_color"`
	CollectionName     string                         `json:"collection_name"`
	CollectionSlug     string                         `json:"collection_slug"`
	CollectionImageURL string                         `json:"collection_image_url"`
	CommunityHeader    *CommunityHeader               `json:"community_header,omitempty"`
}

type CommunityHeader struct {
	CommunityID     string                `json:"community_id"`
	CommunityName   string                `json:"community_name"`
	CommunityColor  string                `json:"community_color"`
	PrivilegesLevel token.PrivilegesLevel `json:"privileges_level"`
}

type CommunityCollectibleHeader struct {
	ID              thirdparty.CollectibleUniqueID `json:"id"`
	Name            string                         `json:"name"`
	CommunityHeader CommunityHeader                `json:"community_header"`
}

func fullCollectibleDataToHeader(c thirdparty.FullCollectibleData) CollectibleHeader {
	ret := CollectibleHeader{
		ID:                 c.CollectibleData.ID,
		Name:               c.CollectibleData.Name,
		ImageURL:           c.CollectibleData.ImageURL,
		AnimationURL:       c.CollectibleData.AnimationURL,
		AnimationMediaType: c.CollectibleData.AnimationMediaType,
		BackgroundColor:    c.CollectibleData.BackgroundColor,
	}
	if c.CollectionData != nil {
		ret.CollectionName = c.CollectionData.Name
		ret.CollectionSlug = c.CollectionData.Slug
		ret.CollectionImageURL = c.CollectionData.ImageURL
	}
	return ret
}

func fullCollectibleDataToDetails(c thirdparty.FullCollectibleData) CollectibleDetails {
	ret := CollectibleDetails{
		ID:                 c.CollectibleData.ID,
		Name:               c.CollectibleData.Name,
		Description:        c.CollectibleData.Description,
		ImageURL:           c.CollectibleData.ImageURL,
		AnimationURL:       c.CollectibleData.AnimationURL,
		AnimationMediaType: c.CollectibleData.AnimationMediaType,
		BackgroundColor:    c.CollectibleData.BackgroundColor,
		Traits:             c.CollectibleData.Traits,
	}
	if c.CollectionData != nil {
		ret.CollectionName = c.CollectionData.Name
		ret.CollectionSlug = c.CollectionData.Slug
		ret.CollectionImageURL = c.CollectionData.ImageURL
	}
	return ret
}

func communityInfoToHeader(c thirdparty.CollectiblesCommunityInfo) CommunityHeader {
	return CommunityHeader{
		CommunityID:     c.CommunityID,
		CommunityName:   c.CommunityName,
		CommunityColor:  c.CommunityColor,
		PrivilegesLevel: c.PrivilegesLevel,
	}
}
