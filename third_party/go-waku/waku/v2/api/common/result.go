package common

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
)

type StoreRequestResult interface {
	Cursor() []byte
	IsComplete() bool
	PeerID() peer.ID
	Next(ctx context.Context, opts ...store.RequestOption) error // TODO: see how to decouple store.RequestOption
	Messages() []*pb.WakuMessageKeyValue
}
