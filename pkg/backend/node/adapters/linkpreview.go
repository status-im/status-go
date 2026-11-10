package adapters

import (
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/images"
	"github.com/status-im/status-go/internal/protocol"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/contacts"
	"github.com/status-im/status-go/pkg/services/linkpreview/unfurlers"
	"github.com/status-im/status-go/pkg/services/sharedurls"
)

type LinkPreviewSettingsAdapter struct {
	db *accounts.Database
}

func NewLinkPreviewSettingsAdapter(db *accounts.Database) *LinkPreviewSettingsAdapter {
	return &LinkPreviewSettingsAdapter{
		db: db,
	}
}

func (a *LinkPreviewSettingsAdapter) GetUnfurlingMode() (settings.URLUnfurlingModeType, error) {
	mode, err := a.db.URLUnfurlingMode()
	return settings.URLUnfurlingModeType(mode), err
}

type LinkPreviewMessengerAdapter struct {
	messenger *protocol.Messenger
}

func NewLinkPreviewMessengerAdapter(m *protocol.Messenger) *LinkPreviewMessengerAdapter {
	return &LinkPreviewMessengerAdapter{
		messenger: m,
	}
}

func (a *LinkPreviewMessengerAdapter) ResolveThreadPreview(URL string) (*common.LinkPreview, error) {
	return unfurlers.NewStatusUnfurler(URL, a, zap.NewNop()).UnfurlThread()
}

func (a *LinkPreviewMessengerAdapter) GetContactByID(pubKey string) (*contacts.Contact, error) {
	if a.messenger == nil {
		return nil, ErrMessengerNotReady
	}
	return a.messenger.GetContactByID(pubKey), nil
}

func (a *LinkPreviewMessengerAdapter) FetchContact(contactID string, waitForResponse bool) (*contacts.Contact, error) {
	if a.messenger == nil {
		return nil, ErrMessengerNotReady
	}
	return a.messenger.FetchContact(contactID, waitForResponse)
}

func (a *LinkPreviewMessengerAdapter) FetchCommunity(communityID string) (*communities.Community, error) {
	if a.messenger == nil {
		return nil, ErrMessengerNotReady
	}
	return a.messenger.FetchCommunity(&protocol.FetchCommunityRequest{
		CommunityKey:    communityID,
		TryDatabase:     true,
		WaitForResponse: true,
	})
}

func (a *LinkPreviewMessengerAdapter) GetThreadPreviewData(URL string) (*unfurlers.ThreadPreviewData, error) {
	if a.messenger == nil {
		return nil, ErrMessengerNotReady
	}
	data, err := sharedurls.NewService(NewSharedUrlsMessengerAdapter(a.messenger)).ParseThreadURL(URL)
	if err != nil {
		return nil, err
	}
	chat := a.messenger.Chat(data.ChatID)
	if chat == nil || !chat.Active {
		return nil, fmt.Errorf("thread parent chat unavailable")
	}
	identity := a.messenger.IdentityPublicKey()
	if identity == nil {
		return nil, ErrMessengerNotReady
	}
	if chat.PrivateGroupChat() {
		joined := false
		for _, member := range chat.Members {
			if member.ID == contacts.ContactIDFromPublicKey(identity) {
				joined = true
				break
			}
		}
		if !joined {
			return nil, fmt.Errorf("thread parent group inaccessible")
		}
	}
	var community *communities.Community
	if chat.CommunityID != "" {
		if !strings.HasPrefix(chat.ID, chat.CommunityID) || len(chat.ID) <= len(chat.CommunityID) {
			return nil, fmt.Errorf("invalid parent channel identity")
		}
		communityKey, err := hexutil.Decode(chat.CommunityID)
		if err != nil {
			return nil, err
		}
		community, err = a.messenger.GetCommunityByID(communityKey)
		if err != nil || community == nil || !community.CanView(identity, chat.ID[len(chat.CommunityID):]) {
			return nil, fmt.Errorf("thread parent channel inaccessible")
		}
	}
	threads, err := a.messenger.ThreadsByChatID(chat.ID)
	if err != nil {
		return nil, err
	}
	parentName := chat.Name
	var contact *contacts.Contact
	if chat.OneToOne() {
		parentName = ""
		contact = a.messenger.GetContactByIDWithImages(chat.ID)
		if contact != nil {
			// Apply the usual name priority without sharing a local nickname.
			publicContact := *contact
			publicContact.LocalNickname = ""
			parentName = publicContact.PrimaryName()
		}
	}
	for _, thread := range threads {
		if thread != nil && thread.ChatID == chat.ID && thread.ThreadID == data.ThreadID {
			return &unfurlers.ThreadPreviewData{
				Name: thread.Name, ParentName: parentName, Community: community,
				ImageDataURI: threadPreviewImage(chat, contact),
			}, nil
		}
	}
	return nil, fmt.Errorf("thread unavailable")
}

// threadPreviewImage uses an existing photo only; identicons are not banners.
func threadPreviewImage(chat *protocol.Chat, contact *contacts.Contact) string {
	if chat.PrivateGroupChat() {
		return chat.Base64Image
	}
	if chat.OneToOne() && contact != nil {
		for _, name := range []string{images.LargeDimName, images.SmallDimName} {
			if image, ok := contact.Images[name]; ok && len(image.Payload) > 0 {
				if _, _, err := images.GetImageDimensions(image.Payload); err != nil {
					continue
				}
				if dataURI, err := image.GetDataURI(); err == nil {
					return dataURI
				}
			}
		}
	}
	return ""
}
