package protocol

import (
	"fmt"
	"github.com/status-im/status-go/api/multiformat"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/communities"
	"go.uber.org/zap"

	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/common"
)

type StatusUnfurler struct {
	m      *Messenger
	logger *zap.Logger
	url    string
}

func NewStatusUnfurler(URL string, messenger *Messenger, logger *zap.Logger) *StatusUnfurler {
	return &StatusUnfurler{
		m:      messenger,
		logger: logger,
		url:    URL,
	}
}

func buildThumbnail(image *images.IdentityImage, thumbnail *common.LinkPreviewThumbnail) error {
	if image.IsEmpty() {
		return nil
	}

	var err error

	thumbnail.Width, thumbnail.Height, err = images.GetImageDimensions(image.Payload)
	if err != nil {
		return fmt.Errorf("failed to get image dimensions: %w", err)
	}

	thumbnail.DataURI, err = image.GetDataURI()
	if err != nil {
		return fmt.Errorf("failed to get data uri: %w", err)
	}

	return nil
}

func (u *StatusUnfurler) buildContactData(contactData *ContactURLData) (*common.StatusContactLinkPreview, error) {
	c := new(common.StatusContactLinkPreview)
	c.PublicKey = contactData.PublicKey
	c.DisplayName = contactData.DisplayName
	c.Description = contactData.Description

	contactID, err := multiformat.DeserializeCompressedKey(contactData.PublicKey)
	if err != nil {
		return nil, err
	}

	contact := u.m.GetContactByID(contactID)
	if contact == nil {
		return c, nil
	}

	// TODO: Should we try to fetch from waku?
	//if contact == nil {
	//	if contact, err = u.m.RequestContactInfoFromMailserver(contactData.PublicKey, true); err != nil {
	//		u.logger.Warn("StatusUnfurler: failed to request contact info from mailserver")
	//		return c, err
	//	}
	//}

	if image, ok := contact.Images[images.SmallDimName]; ok {
		if err = buildThumbnail(&image, &c.Icon); err != nil {
			return c, fmt.Errorf("failed to set thumbnail: %w", err)
		}
	}

	return c, nil
}

func (u *StatusUnfurler) fillCommunityImages(community *communities.Community, icon *common.LinkPreviewThumbnail, banner *common.LinkPreviewThumbnail) error {
	if image, ok := community.Images()[images.SmallDimName]; ok {
		if err := buildThumbnail(&images.IdentityImage{Payload: image.Payload}, icon); err != nil {
			u.logger.Warn("unfurling status link: failed to set community thumbnail", zap.Error(err))
		}
	}

	if image, ok := community.Images()[images.BannerIdentityName]; ok {
		if err := buildThumbnail(&images.IdentityImage{Payload: image.Payload}, banner); err != nil {
			u.logger.Warn("unfurling status link: failed to set community banner", zap.Error(err))
		}
	}

	return nil
}

func (u *StatusUnfurler) buildCommunityData(data *CommunityURLData) (*common.StatusCommunityLinkPreview, error) {
	c := new(common.StatusCommunityLinkPreview)

	// First, fill the output with the data from URL
	c.CommunityID = data.CommunityID
	c.DisplayName = data.DisplayName
	c.Description = data.Description
	c.MembersCount = data.MembersCount
	c.Color = data.Color
	c.TagIndices = data.TagIndices

	// Now check if there's newer information in the database
	communityID, err := types.DecodeHex(data.CommunityID)
	if err != nil {
		return c, fmt.Errorf("failed to decode community id: %w", err)
	}

	community, err := u.m.GetCommunityByID(communityID)
	if err != nil {
		return c, nil
	}

	err = u.fillCommunityImages(community, &c.Icon, &c.Banner)
	if err != nil {
		return c, err
	}

	return c, nil
}

func (u *StatusUnfurler) buildChannelData(data *CommunityChannelURLData, communityData *CommunityURLData) (*common.StatusCommunityChannelLinkPreview, error) {
	c := new(common.StatusCommunityChannelLinkPreview)

	c.ChannelUUID = data.ChannelUUID
	c.Emoji = data.Emoji
	c.DisplayName = data.DisplayName
	c.Description = data.Description
	c.Color = data.Color

	if communityData == nil {
		return nil, fmt.Errorf("channel communtiy can't be empty")
	}

	community, err := u.buildCommunityData(communityData)
	if err != nil {
		return nil, fmt.Errorf("failed to build channel community data: %w", err)
	}

	c.Community = community

	return c, nil
}

func (u *StatusUnfurler) Unfurl() (common.StatusLinkPreview, error) {

	var preview common.StatusLinkPreview
	preview.URL = u.url

	resp, err := u.m.ParseSharedURL(u.url)
	if err != nil {
		return preview, err
	}

	// If a URL has been successfully parsed,
	// any further errors should not be returned, only logged.

	if resp.Contact != nil {
		preview.Contact, err = u.buildContactData(resp.Contact)
		u.logger.Warn("error when building contact data: ", zap.Error(err))
		return preview, nil
	}

	// NOTE: Currently channel data comes together with community data,
	//		 both `Community` and `Channel` fields will be present.
	//		 So we check for Channel first, then Community.

	if resp.Channel != nil {
		preview.Channel, err = u.buildChannelData(resp.Channel, resp.Community)
		if err != nil {
			u.logger.Warn("error when building channel data: ", zap.Error(err))
		}
		return preview, nil
	}

	if resp.Community != nil {
		preview.Community, err = u.buildCommunityData(resp.Community)
		if err != nil {
			u.logger.Warn("error when building community data: ", zap.Error(err))
		}
		return preview, nil
	}

	u.logger.Info("<<< StatusUnfurler::Unfurl",
		zap.Any("contact", preview.Contact),
		zap.Any("community", preview.Community),
		zap.Any("channel", preview.Channel))

	return preview, nil
}
