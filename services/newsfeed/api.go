package newsfeed

import (
	"github.com/status-im/status-go/protocol"
)

type API struct {
	service *Service
}

// FetchNewsMessages fetches news messages from the News Feed
// and returns a MessengerResponse containing the AC notifications
func (api *API) FetchNewsMessages() (*protocol.MessengerResponse, error) {
	return api.service.FetchNewsMessages()
}

func (api *API) ToggleNewsFeedEnabled(value bool) error {
	return api.service.ToggleNewsFeedEnabled(value)
}

func (api *API) ToggleNewsRSSEnabled(value bool) error {
	return api.service.ToggleNewsRSSEnabled(value)
}
