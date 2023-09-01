package collectibles

import "github.com/status-im/status-go/services/wallet/thirdparty"

// Combined Collection+Collectible info, used to display a detailed view of a collectible
type CollectibleDetails struct {
	ID                 thirdparty.CollectibleUniqueID `json:"id"`
	Name               string                         `json:"name"`
	Description        string                         `json:"description"`
	ImageURL           string                         `json:"image_url"`
	AnimationURL       string                         `json:"animation_url"`
	AnimationMediaType string                         `json:"animation_media_type"`
	Traits             []thirdparty.CollectibleTrait  `json:"traits"`
	BackgroundColor    string                         `json:"background_color"`
	CollectionName     string                         `json:"collection_name"`
	CollectionSlug     string                         `json:"collection_slug"`
	CollectionImageURL string                         `json:"collection_image_url"`
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

func fullCollectiblesDataToHeaders(data []thirdparty.FullCollectibleData) []CollectibleHeader {
	res := make([]CollectibleHeader, 0, len(data))

	for _, c := range data {
		res = append(res, fullCollectibleDataToHeader(c))
	}

	return res
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

func fullCollectiblesDataToDetails(data []thirdparty.FullCollectibleData) []CollectibleDetails {
	res := make([]CollectibleDetails, 0, len(data))

	for _, c := range data {
		res = append(res, fullCollectibleDataToDetails(c))
	}

	return res
}
