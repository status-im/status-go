package waku

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
)

// storeRequestor performs one raw store-query request against a single storenode.
// It is the only seam that talks to the store transport, which keeps the pager
// unit-testable and lets Phase 2 swap go-waku's store client for the Logos
// Delivery store FFI here without touching the pager.
type storeRequestor interface {
	query(ctx context.Context, peerInfo peer.AddrInfo, request *storepb.StoreQueryRequest) (messages []*storepb.WakuMessageKeyValue, cursor []byte, err error)
}

// wakuStoreRequestor is the go-waku-backed storeRequestor: the request rides the
// go-waku node's libp2p host (this is the remaining go-waku/libp2p coupling that
// Phase 2 removes).
type wakuStoreRequestor struct {
	store *store.WakuStore
}

func (r wakuStoreRequestor) query(ctx context.Context, peerInfo peer.AddrInfo, request *storepb.StoreQueryRequest) ([]*storepb.WakuMessageKeyValue, []byte, error) {
	result, err := r.store.RequestRaw(ctx, peerInfo, request)
	if err != nil {
		return nil, nil, err
	}
	return result.Messages(), result.Cursor(), nil
}
