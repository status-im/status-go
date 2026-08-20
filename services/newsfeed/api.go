package newsfeed

import (
	"github.com/status-im/status-go/internal/protocol"
)

type API struct {
	service *Service
}

// FetchNewsMessages fetches news messages from the News Feed
// and returns a MessengerResponse containing the AC notifications
func (api *API) FetchNewsMessages() (*protocol.MessengerResponse, error) {
	return api.service.FetchNewsMessages()
}

func (api *API) Enabled() (bool, error) {
	return api.service.storage.GetEnabled()
}

func (api *API) SetEnabled(value bool) error {
	return api.service.ToggleEnabled(value)
}

func (api *API) NotificationsEnabled() (bool, error) {
	return api.service.storage.GetNotificationsEnabled()
}

func (api *API) SetNotificationsEnabled(value bool) error {
	return api.service.storage.SaveNotificationsEnabled(value)
}

func (api *API) RSSEnabled() (bool, error) {
	return api.service.storage.GetRSSEnabled()
}

func (api *API) SetRSSEnabled(value bool) error {
	return api.service.ToggleRSSEnabled(value)
}
