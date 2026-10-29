package waku

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"

	"github.com/status-im/status-go/internal/panics"
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
	store rawStoreRequester
}

// rawStoreRequester is the part of go-waku's *store.WakuStore the requestor uses.
type rawStoreRequester interface {
	RequestRaw(ctx context.Context, peerInfo peer.AddrInfo, request *storepb.StoreQueryRequest) (store.Result, error)
}

// query returns as soon as ctx is done. RequestRaw only uses ctx to open the
// stream, so after cancellation the request keeps running in the background until
// the storenode replies or the connection drops.
func (r wakuStoreRequestor) query(ctx context.Context, peerInfo peer.AddrInfo, request *storepb.StoreQueryRequest) ([]*storepb.WakuMessageKeyValue, []byte, error) {
	type reply struct {
		result store.Result
		err    error
	}
	replies := make(chan reply, 1)
	go func() {
		defer panics.LogOnPanic()
		result, err := r.store.RequestRaw(ctx, peerInfo, request)
		replies <- reply{result, err}
	}()

	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case rep := <-replies:
		if rep.err != nil {
			return nil, nil, rep.err
		}
		return rep.result.Messages(), rep.result.Cursor(), nil
	}
}
