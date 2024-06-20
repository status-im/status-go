package connector

func NewAPI(s *Service) *API {
	return &API{
		s: s,
	}
}

type API struct {
	s *Service
}

func (api *API) CallRPC(inputJSON string) (string, error) {
	return api.s.rpcClient.CallRaw(inputJSON), nil
}
