package adapters

import (
	"github.com/status-im/status-go/protocol"
)

type NewsFeedActivityCenterAdapter struct {
	messenger *protocol.Messenger
}

func NewNewsFeedActivityCenterAdapter(messenger *protocol.Messenger) *NewsFeedActivityCenterAdapter {
	return &NewsFeedActivityCenterAdapter{
		messenger: messenger,
	}
}

func (p *NewsFeedActivityCenterAdapter) AddNotification(response *protocol.MessengerResponse, notification *protocol.ActivityCenterNotification) error {
	if p.messenger == nil {
		return ErrMessengerNotReady
	}
	return p.messenger.AddActivityCenterNotification(response, notification, nil)
}
