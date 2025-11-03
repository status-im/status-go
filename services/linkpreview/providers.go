package linkpreview

import (
	messagingtypes "github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/contacts"
)

//go:generate go tool mockgen -package=mock_linkpreview -source=providers.go -destination=./mock/linkpreview_providers.go

type Persistence interface {
	GetUnfurlingMode() (settings.URLUnfurlingModeType, error)
}

type StatusDataProvider interface {
	GetContactByID(pubKey string) *contacts.Contact
	FetchContact(contactID string, waitForResponse bool) (*contacts.Contact, error)
	FetchCommunity(communityID string, shard *messagingtypes.Shard) (*communities.Community, error)
}
