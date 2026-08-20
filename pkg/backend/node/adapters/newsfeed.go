package adapters

import (
	"github.com/status-im/status-go/internal/protocol"
)

type NewsFeedActivityCenter struct {
	messenger *protocol.Messenger
}

func NewNewsFeedActivityCenterAdapter(messenger *protocol.Messenger) *NewsFeedActivityCenter {
	return &NewsFeedActivityCenter{
		messenger: messenger,
	}
}

func (p *NewsFeedActivityCenter) AddNotification(response *protocol.MessengerResponse, notification *protocol.ActivityCenterNotification) error {
	if p.messenger == nil {
		return ErrMessengerNotReady
	}
	return p.messenger.AddActivityCenterNotification(response, notification, nil)
}
