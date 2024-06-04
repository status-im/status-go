package store

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
)

// Result represents a valid response from a store node
type Result struct {
	done bool

	messages      []*pb.WakuMessageKeyValue
	store         *WakuStore
	storeRequest  *pb.StoreQueryRequest
	storeResponse *pb.StoreQueryResponse
	cursor        []byte
	peerID        peer.ID
}

func (r *Result) Cursor() []byte {
	return r.cursor
}

func (r *Result) IsComplete() bool {
	return r.done
}

func (r *Result) PeerID() peer.ID {
	return r.peerID
}

func (r *Result) Query() *pb.StoreQueryRequest {
	return r.storeRequest
}

func (r *Result) Response() *pb.StoreQueryResponse {
	return r.storeResponse
}

func (r *Result) Next(ctx context.Context) error {
	if r.cursor == nil {
		r.done = true
		r.messages = nil
		return nil
	}

	newResult, err := r.store.next(ctx, r)
	if err != nil {
		return err
	}

	r.cursor = newResult.cursor
	r.messages = newResult.messages

	return nil
}

func (r *Result) Messages() []*pb.WakuMessageKeyValue {
	return r.messages
}
