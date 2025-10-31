package node

import (
	"errors"

	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/contacts"
)

var (
	ErrMessengerNotReady = errors.New("messenger not ready")
)

// SharedUrlsMessengerAdapter is a wrapper around the messenger to make it compatible with the sharedurls service.
// This is to `return (contact, error)` in `GetContactByID`, which Messenger didn't do atm.
// The return error is needed to enable returning errNoDataProvider in NopDataProvider (nil object pattern).
type SharedUrlsMessengerAdapter struct {
	messenger *protocol.Messenger
}

func NewSharedUrlsMessengerAdapter(messenger *protocol.Messenger) *SharedUrlsMessengerAdapter {
	return &SharedUrlsMessengerAdapter{
		messenger: messenger,
	}
}

func (p *SharedUrlsMessengerAdapter) GetCommunityByID(communityID types.HexBytes) (*communities.Community, error) {
	if p.messenger == nil {
		return nil, ErrMessengerNotReady
	}
	return p.messenger.GetCommunityByID(communityID)
}

func (p *SharedUrlsMessengerAdapter) GetContactByID(pubKey string) (*contacts.Contact, error) {
	if p.messenger == nil {
		return nil, ErrMessengerNotReady
	}
	return p.messenger.GetContactByID(pubKey), nil
}
