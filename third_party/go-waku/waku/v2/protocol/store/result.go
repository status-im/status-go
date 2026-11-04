package store

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
)

// Result represents a valid response from a store node
type Result interface {
	Cursor() []byte
	IsComplete() bool
	PeerID() peer.ID
	Query() *pb.StoreQueryRequest
	Response() *pb.StoreQueryResponse
	Next(ctx context.Context, opts ...RequestOption) error
	Messages() []*pb.WakuMessageKeyValue
}

type resultImpl struct {
	done bool

	messages      []*pb.WakuMessageKeyValue
	store         *WakuStore
	storeRequest  *pb.StoreQueryRequest
	storeResponse *pb.StoreQueryResponse
	cursor        []byte
	peerID        peer.ID
}

func (r *resultImpl) Cursor() []byte {
	return r.cursor
}

func (r *resultImpl) IsComplete() bool {
	return r.done
}

func (r *resultImpl) PeerID() peer.ID {
	return r.peerID
}

func (r *resultImpl) Query() *pb.StoreQueryRequest {
	return r.storeRequest
}

func (r *resultImpl) Response() *pb.StoreQueryResponse {
	return r.storeResponse
}

func (r *resultImpl) Next(ctx context.Context, opts ...RequestOption) error {
	if r.cursor == nil {
		r.done = true
		r.messages = nil
		return nil
	}

	newResult, err := r.store.next(ctx, r, opts...)
	if err != nil {
		return err
	}

	r.cursor = newResult.cursor
	r.messages = newResult.messages

	return nil
}

func (r *resultImpl) Messages() []*pb.WakuMessageKeyValue {
	return r.messages
}
