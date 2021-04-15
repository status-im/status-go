package wakuv2ext

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p-core/peer"
	store "github.com/status-im/go-waku/waku/v2/protocol/waku_store"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/services/ext"
)

const (
	// defaultWorkTime is a work time reported in messages sent to MailServer nodes.
	defaultWorkTime = 5
)

// PublicAPI extends waku public API.
type PublicAPI struct {
	*ext.PublicAPI
	service   *Service
	publicAPI types.PublicWakuAPI
	log       log.Logger
}

// NewPublicAPI returns instance of the public API.
func NewPublicAPI(s *Service) *PublicAPI {
	return &PublicAPI{
		PublicAPI: ext.NewPublicAPI(s.Service, s.w),
		service:   s,
		publicAPI: s.w.PublicWakuAPI(),
		log:       log.New("package", "status-go/services/wakuext.PublicAPI"),
	}
}

// RequestMessages sends a request for historic messages to a MailServer.
func (api *PublicAPI) RequestMessages(_ context.Context, r ext.StoreRequest) (types.HexBytes, error) {
	api.log.Info("RequestMessages", "request", r)

	now := api.service.w.GetCurrentTime()
	r.SetDefaults(now)

	if r.From > r.To {
		return nil, fmt.Errorf("Query range is invalid: from > to (%d > %d)", r.From, r.To)
	}

	h := store.GenerateRequestId()

	mailserver, err := peer.IDB58Decode(r.MailServerPeer)
	if err != nil {
		return nil, err
	}

	options := []store.HistoryRequestOption{
		store.WithRequestId(h),
		store.WithPeer(mailserver),
		store.WithTimeout(r.Timeout * time.Second),
		store.WithPaging(r.Asc, r.PageSize),
	}

	// TODO: handle cursor

	var hash types.Hash
	copy(hash[:], h[:types.HashLength])

	if !r.Force {
		err := api.service.RequestsRegistry().Register(hash, r.Topics)
		if err != nil {
			return nil, err
		}
	}

	if err := api.service.w.RequestStoreMessages(r.Topics, r.From, r.To, options); err != nil {
		if !r.Force {
			api.service.RequestsRegistry().Unregister(hash)
		}
		return nil, err
	}

	return hash[:], nil
}
