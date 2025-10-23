package newsfeed

import (
	"time"

	"github.com/status-im/status-go/protocol"
)

//go:generate go tool mockgen -package=mock_newsfeed -source=interfaces.go -destination=mock/interfaces.go

type ActivityCenter interface {
	AddNotification(response *protocol.MessengerResponse, notification *protocol.ActivityCenterNotification) error
}

type Persistence interface {
	NewsFeedEnabled() (bool, error)
	SaveNewsFeedEnabled(bool) error
	NewsRSSEnabled() (bool, error)
	SaveNewsRSSEnabled(bool) error
	NewsFeedLastFetchedTimestamp() (time.Time, error)
	SaveNewsFeedLastFetchedTimestamp(time.Time) error
}
