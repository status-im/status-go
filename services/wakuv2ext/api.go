package wakuv2ext

import (
	"github.com/status-im/status-go/services/ext"
	wakutypes "github.com/status-im/status-go/waku/types"
)

// PublicAPI extends waku public API.
type PublicAPI struct {
	*ext.PublicAPI
	service   *Service
	publicAPI wakutypes.PublicWakuAPI
}

// NewPublicAPI returns instance of the public API.
func NewPublicAPI(s *Service) *PublicAPI {
	return &PublicAPI{
		PublicAPI: ext.NewPublicAPI(s.Service),
		service:   s,
		publicAPI: s.w.PublicWakuAPI(),
	}
}
