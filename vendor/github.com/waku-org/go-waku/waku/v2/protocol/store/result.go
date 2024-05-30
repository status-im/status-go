package store

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
)

// Result represents a valid response from a store node
type Result struct {
	started      bool
	messages     []*pb.WakuMessageKeyValue
	store        *WakuStore
	storeRequest *pb.StoreQueryRequest
	cursor       []byte
	peerID       peer.ID
}

func (r *Result) Cursor() []byte {
	return r.cursor
}

func (r *Result) IsComplete() bool {
	return r.cursor == nil
}

func (r *Result) PeerID() peer.ID {
	return r.peerID
}

func (r *Result) Query() *pb.StoreQueryRequest {
	return r.storeRequest
}

func (r *Result) Next(ctx context.Context) (bool, error) {
	if !r.started {
		r.started = true
		return len(r.messages) != 0, nil
	}

	if r.IsComplete() {
		r.cursor = nil
		r.messages = nil
		return false, nil
	}

	newResult, err := r.store.next(ctx, r)
	if err != nil {
		return false, err
	}

	r.cursor = newResult.cursor
	r.messages = newResult.messages

	return !r.IsComplete(), nil
}

func (r *Result) Messages() []*pb.WakuMessageKeyValue {
	if !r.started {
		return nil
	}
	return r.messages
}
