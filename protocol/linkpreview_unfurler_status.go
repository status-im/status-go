package protocol

import (
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

func (u *StatusUnfurler) createContactData(contactData *ContactURLData) *common.StatusContactLinkPreview {
	c := new(common.StatusContactLinkPreview)
	c.PublicKey = contactData.PublicKey
	c.DisplayName = contactData.DisplayName
	c.Description = contactData.Description

	var err error
	contact := u.m.GetContactByID(contactData.PublicKey)

	if contact == nil {
		if contact, err = u.m.RequestContactInfoFromMailserver(contactData.PublicKey, true); err != nil {
			u.logger.Warn("StatusUnfurler: failed to request contact info from mailserver")
			return c
		}
	}

	if thumbImage, ok := contact.Images[images.SmallDimName]; ok {
		if imageBase64, err := thumbImage.GetDataURI(); err == nil {
			c.Icon.Width = thumbImage.Width
			c.Icon.Height = thumbImage.Height
			c.Icon.DataURI = imageBase64
		}
	}

	return c
}

func (u *StatusUnfurler) Unfurl() (common.StatusLinkPreview, error) {

	var preview common.StatusLinkPreview

	resp, err := u.m.ParseSharedURL(u.url)
	if err != nil {
		return preview, err
	}

	if resp.Contact != nil {
		preview.Contact = u.createContactData(resp.Contact)
	}

	if resp.Community != nil {
		preview.Community = new(common.StatusCommunityLinkPreview)
		preview.Community.DisplayName = resp.Community.DisplayName
		preview.Community.Description = resp.Community.Description
	}

	if resp.Channel != nil {
		preview.Channel = new(common.StatusCommunityChannelLinkPreview)
		preview.Channel.DisplayName = resp.Channel.DisplayName
		preview.Channel.Description = resp.Channel.Description
	}

	return preview, nil
}
