package protocol

import (
	"fmt"
	"github.com/status-im/status-go/api/multiformat"
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

	// TODO: Should we do this?
	//if contact == nil {
	//	if contact, err = u.m.RequestContactInfoFromMailserver(contactData.PublicKey, true); err != nil {
	//		u.logger.Warn("StatusUnfurler: failed to request contact info from mailserver")
	//		return c, err
	//	}
	//}

	if contact == nil {
		return c, nil
	}

	if image, ok := contact.Images[images.SmallDimName]; ok {
		if err = buildThumbnail(&image, &c.Icon); err != nil {
			u.logger.Warn("unfurling status link: failed to set thumbnail", zap.Error(err))
		}
	}

	return c, nil
}

func (u *StatusUnfurler) Unfurl() (common.StatusLinkPreview, error) {

	var preview common.StatusLinkPreview
	preview.URL = u.url

	resp, err := u.m.ParseSharedURL(u.url)
	if err != nil {
		return preview, err
	}

	if resp.Contact != nil {
		preview.Contact, err = u.buildContactData(resp.Contact)
	}

	if resp.Community != nil {
		// TODO: move to a separate func, finish
		preview.Community = new(common.StatusCommunityLinkPreview)
		preview.Community.DisplayName = resp.Community.DisplayName
		preview.Community.Description = resp.Community.Description
	}

	if resp.Channel != nil {
		// TODO: move to a separate func, finish
		preview.Channel = new(common.StatusCommunityChannelLinkPreview)
		preview.Channel.DisplayName = resp.Channel.DisplayName
		preview.Channel.Description = resp.Channel.Description
	}

	u.logger.Info("<<< StatusUnfurler::Unfurl",
		zap.Any("contact", preview.Contact),
		zap.Any("community", preview.Community),
		zap.Any("channel", preview.Channel))

	return preview, nil
}
