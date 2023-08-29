package protocol

import (
	"net/url"

	"go.uber.org/zap"

	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

type StatusUnfurler struct {
	m      *Messenger
	logger *zap.Logger
	url    *url.URL
}

func NewStatusUnfurler(URL *url.URL, messenger *Messenger, logger *zap.Logger) *StatusUnfurler {
	return &StatusUnfurler{
		m:      messenger,
		logger: logger,
		url:    URL,
	}
}

func (u *StatusUnfurler) fillContactData(contactData *ContactURLData, preview *common.LinkPreview) {
	preview.Title = contactData.DisplayName
	preview.Description = contactData.Description

	var err error
	contact := u.m.GetContactByID(contactData.PublicKey)

	if contact == nil {
		if contact, err = u.m.RequestContactInfoFromMailserver(contactData.PublicKey, true); err != nil {
			u.logger.Warn("StatusUnfurler: failed to request contact info from mailserver")
			return
		}
	}

	if thumbImage, ok := contact.Images[images.SmallDimName]; ok {
		if imageBase64, err := thumbImage.GetDataURI(); err == nil {
			preview.Thumbnail.Width = thumbImage.Width
			preview.Thumbnail.Height = thumbImage.Height
			preview.Thumbnail.DataURI = imageBase64
		}
	}
}

func (u *StatusUnfurler) Unfurl() (common.LinkPreview, error) {
	preview := newDefaultLinkPreview(u.url)
	preview.Type = protobuf.UnfurledLink_STATUS_SHARED_URL

	resp, err := u.m.ParseSharedURL(u.url.String())
	if err != nil {
		return preview, err
	}

	if resp.Contact != nil {
		u.fillContactData(resp.Contact, &preview)
	}

	if resp.Community != nil {
		preview.Title = resp.Community.DisplayName
		preview.Description = resp.Community.Description
	}

	if resp.Channel != nil {
		preview.Title = resp.Channel.DisplayName
		preview.Description = resp.Channel.Description
	}

	return preview, nil
}
