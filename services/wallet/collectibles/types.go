package collectibles

import (
	"errors"

	"github.com/status-im/status-go/protocol/communities/token"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

type CollectibleDataType byte

const (
	CollectibleDataTypeUniqueID CollectibleDataType = iota
	CollectibleDataTypeHeader
	CollectibleDataTypeDetails
	CollectibleDataTypeCommunityHeader
)

type CollectionDataType byte

const (
	CollectionDataTypeContractID CollectionDataType = iota
	CollectionDataTypeDetails
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

type Collection struct {
	DataType       CollectionDataType    `json:"data_type"`
	ID             thirdparty.ContractID `json:"id"`
	CommunityID    string                `json:"community_id"`
	ContractType   w_common.ContractType `json:"contract_type"`
	CollectionData *CollectionData       `json:"collection_data,omitempty"`
}

type CollectibleData struct {
	Name               string                         `json:"name"`
	Description        *string                        `json:"description,omitempty"`
	ImageURL           *string                        `json:"image_url,omitempty"`
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

func getContractType(c thirdparty.FullCollectibleData) w_common.ContractType {
	if c.CollectibleData.ContractType != w_common.ContractTypeUnknown {
		return c.CollectibleData.ContractType
	}
	if c.CollectionData != nil && c.CollectionData.ContractType != w_common.ContractTypeUnknown {
		return c.CollectionData.ContractType
	}
	return w_common.ContractTypeUnknown
}

func fullCollectibleDataToID(c thirdparty.FullCollectibleData) Collectible {
	ret := Collectible{
		DataType:     CollectibleDataTypeUniqueID,
		ID:           c.CollectibleData.ID,
		ContractType: getContractType(c),
	}
	return ret
}

func fullCollectiblesDataToID(data []thirdparty.FullCollectibleData) []Collectible {
	res := make([]Collectible, 0, len(data))

	for _, c := range data {
		id := fullCollectibleDataToID(c)
		res = append(res, id)
	}

	return res
}

func fullCollectibleDataToHeader(c thirdparty.FullCollectibleData) Collectible {
	ret := Collectible{
		DataType:     CollectibleDataTypeHeader,
		ID:           c.CollectibleData.ID,
		ContractType: getContractType(c),
		CollectibleData: &CollectibleData{
			Name:               c.CollectibleData.Name,
			ImageURL:           &c.CollectibleData.ImageURL,
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
		header := fullCollectibleDataToHeader(c)
		res = append(res, header)
	}

	return res
}

func fullCollectibleDataToDetails(c thirdparty.FullCollectibleData) Collectible {
	ret := Collectible{
		DataType:     CollectibleDataTypeDetails,
		ID:           c.CollectibleData.ID,
		ContractType: getContractType(c),
		CollectibleData: &CollectibleData{
			Name:               c.CollectibleData.Name,
			Description:        &c.CollectibleData.Description,
			ImageURL:           &c.CollectibleData.ImageURL,
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
		details := fullCollectibleDataToDetails(c)
		res = append(res, details)
	}

	return res
}

func fullCollectiblesDataToCommunityHeader(data []thirdparty.FullCollectibleData) []Collectible {
	res := make([]Collectible, 0, len(data))

	for _, localCollectibleData := range data {
		// to satisfy gosec: C601 checks
		c := localCollectibleData
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

func fullCollectiblesDataToDataType(collectibles []thirdparty.FullCollectibleData, dataType CollectibleDataType) ([]Collectible, error) {
	switch dataType {
	case CollectibleDataTypeUniqueID:
		return fullCollectiblesDataToID(collectibles), nil
	case CollectibleDataTypeHeader:
		return fullCollectiblesDataToHeaders(collectibles), nil
	case CollectibleDataTypeDetails:
		return fullCollectiblesDataToDetails(collectibles), nil
	case CollectibleDataTypeCommunityHeader:
		return fullCollectiblesDataToCommunityHeader(collectibles), nil
	}
	return nil, errors.New("unknown data type")
}

func collectionDataToID(c thirdparty.CollectionData) Collection {
	ret := Collection{
		DataType:     CollectionDataTypeContractID,
		ID:           c.ID,
		CommunityID:  c.CommunityID,
		ContractType: c.ContractType,
	}
	return ret
}

func collectionsDataToID(data []thirdparty.CollectionData) []Collection {
	res := make([]Collection, 0, len(data))

	for _, c := range data {
		id := collectionDataToID(c)
		res = append(res, id)
	}

	return res
}

func collectionDataToDetails(c thirdparty.CollectionData) Collection {
	collectionData := thirdpartyCollectionDataToCollectionData(&c)
	return Collection{
		DataType:       CollectionDataTypeDetails,
		ID:             c.ID,
		CommunityID:    c.CommunityID,
		ContractType:   c.ContractType,
		CollectionData: &collectionData,
	}
}

func collectionsDataToDetails(data []thirdparty.CollectionData) []Collection {
	res := make([]Collection, 0, len(data))

	for _, c := range data {
		details := collectionDataToDetails(c)
		res = append(res, details)
	}

	return res
}

func collectionsDataToDataType(collections []thirdparty.CollectionData, dataType CollectionDataType) ([]Collection, error) {
	switch dataType {
	case CollectionDataTypeContractID:
		return collectionsDataToID(collections), nil
	case CollectionDataTypeDetails:
		return collectionsDataToDetails(collections), nil
	}
	return nil, errors.New("unknown data type")
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
