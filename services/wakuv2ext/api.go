package wakuv2ext

import (
	"github.com/status-im/status-go/services/ext"
)

// PublicAPI extends waku public API.
type PublicAPI struct {
	*ext.PublicAPI
	service *Service
}

// NewPublicAPI returns instance of the public API.
func NewPublicAPI(s *Service) *PublicAPI {
	return &PublicAPI{
		PublicAPI: ext.NewPublicAPI(s.Service),
		service:   s,
	}
}
