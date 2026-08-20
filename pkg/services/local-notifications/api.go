package localnotifications

import (
	"context"
)

func NewAPI(s *Service) *API {
	return &API{s}
}

type API struct {
	s *Service
}

func (api *API) NotificationPreferences(ctx context.Context) ([]NotificationPreference, error) {
	return api.s.db.GetPreferences()
}
