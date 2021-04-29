package wakuv2ext

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/store"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/services/ext"
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

	h := protocol.GenerateRequestId()

	mailserver, err := peer.Decode(r.MailServerPeer)
	if err != nil {
		return nil, err
	}

	options := []store.HistoryRequestOption{
		store.WithRequestId(h),
		store.WithPeer(mailserver),
		store.WithPaging(r.Asc, r.PageSize),
	}

	// TODO: timeout

	if r.Cursor != nil {
		options = append(options, store.WithCursor(&pb.Index{
			Digest:       r.Cursor.Digest,
			ReceivedTime: r.Cursor.ReceivedTime,
		}))
	}

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
