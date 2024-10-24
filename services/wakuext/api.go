package wakuext

import (
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/services/ext"
)

// PublicAPI extends waku public API.
type PublicAPI struct {
	*ext.PublicAPI
	service   *Service
	publicAPI types.PublicWakuAPI
}

// NewPublicAPI returns instance of the public API.
func NewPublicAPI(s *Service) *PublicAPI {
	return &PublicAPI{
		PublicAPI: ext.NewPublicAPI(s.Service, s.w),
		service:   s,
		publicAPI: s.w.PublicWakuAPI(),
	}
}
