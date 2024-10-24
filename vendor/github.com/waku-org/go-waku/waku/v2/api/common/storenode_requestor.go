package common

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
)

type StorenodeRequestor interface {
	Query(ctx context.Context, peerID peer.ID, query *pb.StoreQueryRequest) (StoreRequestResult, error)
}
