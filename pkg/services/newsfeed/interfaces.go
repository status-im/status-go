package newsfeed

import (
	"time"

	"github.com/status-im/status-go/internal/protocol"
)

//go:generate go tool mockgen -package=mock_newsfeed -source=interfaces.go -destination=mock/interfaces.go

type ActivityCenter interface {
	AddNotification(response *protocol.MessengerResponse, notification *protocol.ActivityCenterNotification) error
}

type Persistence interface {
	GetEnabled() (bool, error)
	SaveEnabled(bool) error
	GetRSSEnabled() (bool, error)
	SaveRSSEnabled(bool) error
	GetNotificationsEnabled() (bool, error)
	SaveNotificationsEnabled(bool) error
	GetLastFetchedTimestamp() (time.Time, error)
	SaveLastFetchedTimestamp(time.Time) error
}
