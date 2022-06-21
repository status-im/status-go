package requests

import (
	"errors"

	"github.com/ethereum/go-ethereum/log"
	userimages "github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/images"
	"github.com/status-im/status-go/protocol/protobuf"
)

var (
	ErrCreateCommunityInvalidName         = errors.New("create-community: invalid name")
	ErrCreateCommunityInvalidColor        = errors.New("create-community: invalid color")
	ErrCreateCommunityInvalidDescription  = errors.New("create-community: invalid description")
	ErrCreateCommunityInvalidIntroMessage = errors.New("create-community: invalid intro message")
	ErrCreateCommunityInvalidOutroMessage = errors.New("create-community: invalid outro message")
	ErrCreateCommunityInvalidMembership   = errors.New("create-community: invalid membership")
)

const (
	maxNameLength         = 30
	maxDescriptionLength  = 140
	maxIntroMessageLength = 1400
	maxOutroMessageLength = 80
)

type CreateCommunity struct {
	Name                         string                               `json:"name"`
	Description                  string                               `json:"description"`
	IntroMessage                 string                               `json:"introMessage,omitempty"`
	OutroMessage                 string                               `json:"outroMessage,omitempty"`
	Color                        string                               `json:"color"`
	Emoji                        string                               `json:"emoji"`
	Membership                   protobuf.CommunityPermissions_Access `json:"membership"`
	EnsOnly                      bool                                 `json:"ensOnly"`
	EnsSubdomain                 string                               `json:"ensSubdomain"`
	Image                        string                               `json:"image"`
	ImageAx                      int                                  `json:"imageAx"`
	ImageAy                      int                                  `json:"imageAy"`
	ImageBx                      int                                  `json:"imageBx"`
	ImageBy                      int                                  `json:"imageBy"`
	Banner                       userimages.CroppedImage              `json:"banner"`
	HistoryArchiveSupportEnabled bool                                 `json:"historyArchiveSupportEnabled,omitempty"`
	PinMessageAllMembersEnabled  bool                                 `json:"pinMessageAllMembersEnabled,omitempty"`
	Encrypted                    bool                                 `json:"encrypted,omitempty"`
}

func adaptIdentityImageToProtobuf(img *userimages.IdentityImage) *protobuf.IdentityImage {
	return &protobuf.IdentityImage{
		Payload:    img.Payload,
		SourceType: protobuf.IdentityImage_RAW_PAYLOAD,
		ImageType:  images.ImageType(img.Payload),
	}
}

func (c *CreateCommunity) Validate() error {
	if c.Name == "" || len(c.Name) > maxNameLength {
		return ErrCreateCommunityInvalidName
	}

	if c.Description == "" || len(c.Description) > maxDescriptionLength {
		return ErrCreateCommunityInvalidDescription
	}

	if len(c.IntroMessage) > maxIntroMessageLength {
		return ErrCreateCommunityInvalidIntroMessage
	}

	if len(c.OutroMessage) > maxOutroMessageLength {
		return ErrCreateCommunityInvalidOutroMessage
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

	if c.Image != "" || c.Banner.ImagePath != "" {
		ciis := make(map[string]*protobuf.IdentityImage)
		if c.Image != "" {
			log.Info("has-image", "image", c.Image)
			imgs, err := userimages.GenerateIdentityImages(c.Image, c.ImageAx, c.ImageAy, c.ImageBx, c.ImageBy)
			if err != nil {
				return nil, err
			}
			for _, img := range imgs {
				ciis[img.Name] = adaptIdentityImageToProtobuf(img)
			}
		}
		if c.Banner.ImagePath != "" {
			log.Info("has-banner", "image", c.Banner.ImagePath)
			img, err := userimages.GenerateBannerImage(c.Banner.ImagePath, c.Banner.X, c.Banner.Y, c.Banner.X+c.Banner.Width, c.Banner.Y+c.Banner.Height)
			if err != nil {
				return nil, err
			}
			ciis[img.Name] = adaptIdentityImageToProtobuf(img)
		}
		ci.Images = ciis
		log.Info("set images", "images", ci)
	}

	description := &protobuf.CommunityDescription{
		Identity: ci,
		Permissions: &protobuf.CommunityPermissions{
			Access:       c.Membership,
			EnsOnly:      c.EnsOnly,
			EnsSubdomain: c.EnsSubdomain,
		},
		AdminSettings: &protobuf.CommunityAdminSettings{
			PinMessageAllMembersEnabled: c.PinMessageAllMembersEnabled,
		},
		IntroMessage: c.IntroMessage,
		OutroMessage: c.OutroMessage,
		Encrypted:    c.Encrypted,
	}
	return description, nil
}
