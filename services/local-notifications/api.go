package localnotifications

func NewAPI(s *Service) *API {
	return &API{s}
}

type API struct {
	s *Service
}
