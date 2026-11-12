package unfurlers

import (
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/images"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/contacts"
	"github.com/status-im/status-go/pkg/multiformat"
	"github.com/status-im/status-go/pkg/services/sharedurls"
)

//go:generate go tool mockgen -package=mock_unfurlers -source=unfurler_status.go -destination=./mock/unfurler_status.go

type StatusDataProvider interface {
	GetContactByID(pubKey string) (*contacts.Contact, error)
	FetchContact(contactID string, waitForResponse bool) (*contacts.Contact, error)
	FetchCommunity(communityID string) (*communities.Community, error)
	GetThreadPreviewData(URL string) (*ThreadPreviewData, error)
}

type ThreadPreviewData struct {
	Name         string
	ParentName   string
	Community    *communities.Community
	ImageDataURI string
}

type StatusUnfurler struct {
	provider StatusDataProvider
	logger   *zap.Logger
	url      string
}

func NewStatusUnfurler(URL string, provider StatusDataProvider, logger *zap.Logger) *StatusUnfurler {
	return &StatusUnfurler{
		provider: provider,
		logger:   logger.With(zap.String("url", URL)),
		url:      URL,
	}
}

func updateThumbnail(image *images.IdentityImage, thumbnail *common.LinkPreviewThumbnail) error {
	if image.IsEmpty() {
		return nil
	}

	width, height, err := images.GetImageDimensions(image.Payload)
	if err != nil {
		return fmt.Errorf("failed to get image dimensions: %w", err)
	}

	dataURI, err := image.GetDataURI()
	if err != nil {
		return fmt.Errorf("failed to get data uri: %w", err)
	}

	thumbnail.Width = width
	thumbnail.Height = height
	thumbnail.DataURI = dataURI

	return nil
}

func (u *StatusUnfurler) buildContactData(publicKey string) (*common.StatusContactLinkPreview, error) {
	// contactID == "0x" + secp251k1 compressed public key as hex-encoded string
	contactID, err := multiformat.DeserializeCompressedKey(publicKey)
	if err != nil {
		return nil, err
	}

	contact, err := u.provider.GetContactByID(contactID)
	if err != nil {
		return nil, err
	}

	// If no contact found locally, fetch it from waku
	if contact == nil {
		contact, err = u.provider.FetchContact(contactID, true)
		if err != nil {
			return nil, fmt.Errorf("failed to request contact info from mailserver for public key '%s': %w", logutils.TruncateWithDot(publicKey), err)
		}
		if contact == nil {
			return nil, fmt.Errorf("contact wasn't found at the store node %s", logutils.TruncateWithDot(publicKey))
		}
	}

	c := &common.StatusContactLinkPreview{
		PublicKey:   contactID,
		DisplayName: contact.DisplayName,
		Description: contact.Bio,
	}

	if image, ok := contact.Images[images.SmallDimName]; ok {
		if err = updateThumbnail(&image, &c.Icon); err != nil {
			u.logger.Warn("unfurling status link: failed to set contact thumbnail", zap.Error(err))
		}
	}

	return c, nil
}

func (u *StatusUnfurler) buildCommunityData(communityID string) (*communities.Community, *common.StatusCommunityLinkPreview, error) {
	// This automatically checks the database
	community, err := u.provider.FetchCommunity(communityID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get community info for communityID '%s': %w", logutils.TruncateWithDot(communityID), err)
	}

	if community == nil {
		return community, nil, fmt.Errorf("community info fetched, but it is empty")
	}

	statusCommunityLinkPreviews, err := community.ToStatusLinkPreview()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get status community link preview for communityID '%s': %w", logutils.TruncateWithDot(communityID), err)
	}

	return community, statusCommunityLinkPreviews, nil
}

func (u *StatusUnfurler) buildChannelData(channelUUID string, communityID string) (*common.StatusCommunityChannelLinkPreview, error) {
	community, communityData, err := u.buildCommunityData(communityID)
	if err != nil {
		return nil, fmt.Errorf("failed to build channel community data: %w", err)
	}

	channel, ok := community.Chats()[channelUUID]
	if !ok {
		return nil, fmt.Errorf("channel with channelID '%s' not found in community '%s'", logutils.TruncateWithDot(channelUUID), logutils.TruncateWithDot(communityID))
	}

	return &common.StatusCommunityChannelLinkPreview{
		ChannelUUID: channelUUID,
		Emoji:       channel.Identity.Emoji,
		DisplayName: channel.Identity.DisplayName,
		Description: channel.Identity.Description,
		Color:       channel.Identity.Color,
		Community:   communityData,
	}, nil
}

// UnfurlThread resolves metadata locally. Message serialization redacts thread
// metadata so each recipient resolves it using their own access.
func (u *StatusUnfurler) UnfurlThread() (*common.LinkPreview, error) {
	if u.provider == nil {
		return nil, fmt.Errorf("thread data provider unavailable")
	}
	data, err := u.provider.GetThreadPreviewData(u.url)
	if err != nil {
		return nil, err
	}
	if data == nil || strings.TrimSpace(data.Name) == "" || strings.TrimSpace(data.ParentName) == "" {
		return nil, fmt.Errorf("thread preview metadata unavailable")
	}
	preview := &common.LinkPreview{
		URL: u.url, Hostname: "status.app", Title: data.Name,
		Description: data.ParentName, ThreadParentName: data.ParentName,
		ThreadAccessible: true,
	}
	if data.Community != nil {
		community, err := data.Community.ToStatusLinkPreview()
		if err != nil {
			return nil, err
		}
		preview.Thumbnail = community.Banner
		preview.Description = community.DisplayName
		preview.ThreadParentName = "#" + strings.TrimPrefix(data.ParentName, "#")
	} else if data.ImageDataURI != "" {
		if payload, err := images.GetPayloadFromURI(data.ImageDataURI); err == nil {
			var thumbnail common.LinkPreviewThumbnail
			if err := updateThumbnail(&images.IdentityImage{Payload: payload}, &thumbnail); err == nil {
				preview.Thumbnail = thumbnail
			}
		}
	}
	return preview, nil
}

func (u *StatusUnfurler) Unfurl() (*common.StatusLinkPreview, error) {
	preview := new(common.StatusLinkPreview)
	preview.URL = u.url

	resp, err := sharedurls.ParseSharedURL(u.url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse shared url: %w", err)
	}

	// If a URL has been successfully parsed,
	// any further errors should not be returned, only logged.

	if resp.Contact != nil {
		preview.Contact, err = u.buildContactData(resp.Contact.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("error when building contact data: %w", err)
		}
		return preview, nil
	}

	// NOTE: Currently channel data comes together with community data,
	//		 both `Community` and `Channel` fields will be present.
	//		 So we check for Channel first, then Community.

	if resp.Channel != nil {
		if resp.Community == nil {
			return preview, fmt.Errorf("channel community can't be empty")
		}
		preview.Channel, err = u.buildChannelData(resp.Channel.ChannelUUID, resp.Community.CommunityID)
		if err != nil {
			return nil, fmt.Errorf("error when building channel data: %w", err)
		}
		return preview, nil
	}

	if resp.Community != nil {
		_, preview.Community, err = u.buildCommunityData(resp.Community.CommunityID)
		if err != nil {
			return nil, fmt.Errorf("error when building community data: %w", err)
		}
		return preview, nil
	}

	return nil, fmt.Errorf("shared url does not contain contact, community or channel data")
}
