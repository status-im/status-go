package adapters

import (
	messagingtypes "github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/contacts"
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

func (a *LinkPreviewMessengerAdapter) FetchCommunity(communityID string, shard *messagingtypes.Shard) (*communities.Community, error) {
	if a.messenger == nil {
		return nil, ErrMessengerNotReady
	}
	return a.messenger.FetchCommunity(&protocol.FetchCommunityRequest{
		CommunityKey:    communityID,
		Shard:           shard,
		TryDatabase:     true,
		WaitForResponse: true,
	})
}
