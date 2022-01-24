package requests

import (
	"errors"

	"github.com/ethereum/go-ethereum/log"
	userimages "github.com/status-im/status-go/images"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/images"
	"github.com/status-im/status-go/protocol/protobuf"
)

var (
	ErrCreateCommunityInvalidName        = errors.New("create-community: invalid name")
	ErrCreateCommunityInvalidColor       = errors.New("create-community: invalid color")
	ErrCreateCommunityInvalidDescription = errors.New("create-community: invalid description")
	ErrCreateCommunityInvalidMembership  = errors.New("create-community: invalid membership")
)

type CreateCommunity struct {
	Name                          string                               `json:"name"`
	Description                   string                               `json:"description"`
	Color                         string                               `json:"color"`
	Emoji                         string                               `json:"emoji"`
	Membership                    protobuf.CommunityPermissions_Access `json:"membership"`
	EnsOnly                       bool                                 `json:"ensOnly"`
	Image                         string                               `json:"image"`
	ImageAx                       int                                  `json:"imageAx"`
	ImageAy                       int                                  `json:"imageAy"`
	ImageBx                       int                                  `json:"imageBx"`
	ImageBy                       int                                  `json:"imageBy"`
	MessageArchiveSeedingEnabled  bool                                 `json:"messageArchiveSeedingEnabled,omitempty"`
	MessageArchiveFetchingEnabled bool                                 `json:"messageArchiveFetchingEnabled,omitempty"`
}

func adaptIdentityImageToProtobuf(img *userimages.IdentityImage) *protobuf.IdentityImage {
	return &protobuf.IdentityImage{
		Payload:    img.Payload,
		SourceType: protobuf.IdentityImage_RAW_PAYLOAD,
		ImageType:  images.ImageType(img.Payload),
	}
}

func (c *CreateCommunity) Validate() error {
	if c.Name == "" {
		return ErrCreateCommunityInvalidName
	}

	if c.Description == "" {
		return ErrCreateCommunityInvalidDescription
	}

	if c.Membership == protobuf.CommunityPermissions_UNKNOWN_ACCESS {
		return ErrCreateCommunityInvalidMembership
	}

	if c.Color == "" {
		return ErrCreateCommunityInvalidColor
	}

	return nil
}

func (c *CreateCommunity) ToCommunityDescription() (*protobuf.CommunityDescription, error) {
	ci := &protobuf.ChatIdentity{
		DisplayName: c.Name,
		Color:       c.Color,
		Emoji:       c.Emoji,
		Description: c.Description,
	}

	if c.Image != "" {
		log.Info("has-image", "image", c.Image)
		ciis := make(map[string]*protobuf.IdentityImage)
		imgs, err := userimages.GenerateIdentityImages(c.Image, c.ImageAx, c.ImageAy, c.ImageBx, c.ImageBy)
		if err != nil {
			return nil, err
		}
		for _, img := range imgs {
			ciis[img.Name] = adaptIdentityImageToProtobuf(img)
		}
		ci.Images = ciis
		log.Info("set images", "images", ci)
	}

	description := &protobuf.CommunityDescription{
		Identity: ci,
		Permissions: &protobuf.CommunityPermissions{
			Access:  c.Membership,
			EnsOnly: c.EnsOnly,
		},
	}
	return description, nil
}

func (c *CreateCommunity) ToCommunitySettings(publicKey string) (params.CommunitySettings, error) {

	settings := params.CommunitySettings{
		CommunityID:                   publicKey,
		MessageArchiveFetchingEnabled: c.MessageArchiveFetchingEnabled,
		MessageArchiveSeedingEnabled:  c.MessageArchiveSeedingEnabled,
	}

	return settings, nil
}
