package sharedurls

import (
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/pkg/contacts"
	"github.com/status-im/status-go/protocol/communities"
)

type DataProvider interface {
	GetCommunityByID(communityID types.HexBytes) (*communities.Community, error)
	GetContactByID(pubKey string) *contacts.Contact
}
