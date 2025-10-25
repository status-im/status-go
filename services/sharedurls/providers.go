package sharedurls

import (
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/contacts"
)

type DataProvider interface {
	GetCommunityByID(communityID types.HexBytes) (*communities.Community, error)
	GetContactByID(pubKey string) *contacts.Contact
}
