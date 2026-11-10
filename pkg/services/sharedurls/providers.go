package sharedurls

import (
	"crypto/ecdsa"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/contacts"
)

//go:generate go tool mockgen -package=mock_sharedurls -source=providers.go -destination=./mock/providers.go

type DataProvider interface {
	// IdentityPublicKey is used only to read legacy participant-bearing links.
	IdentityPublicKey() *ecdsa.PublicKey
	ResolveDirectThreadChatID(threadID string) (string, error)
	GetCommunityByID(communityID types.HexBytes) (*communities.Community, error)
	GetContactByID(pubKey string) (*contacts.Contact, error)
}

type NopDataProvider struct{}

func (*NopDataProvider) IdentityPublicKey() *ecdsa.PublicKey {
	return nil
}

func (*NopDataProvider) ResolveDirectThreadChatID(string) (string, error) {
	return "", errNoDataProvider
}

func (*NopDataProvider) GetCommunityByID(types.HexBytes) (*communities.Community, error) {
	return nil, errNoDataProvider
}

func (*NopDataProvider) GetContactByID(string) (*contacts.Contact, error) {
	return nil, errNoDataProvider
}
