package sharedurls

import (
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/contacts"
)

//go:generate go tool mockgen -package=mock_provider -source=providers.go -destination=./mock/providers.go


type DataProvider interface {
	GetCommunityByID(communityID types.HexBytes) (*communities.Community, error)
	GetContactByID(pubKey string) *contacts.Contact
}
