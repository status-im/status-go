package missing

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/api/common"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
)

func NewDefaultStorenodeRequestor(store *store.WakuStore) common.StorenodeRequestor {
	return &defaultStorenodeRequestor{
		store: store,
	}
}

type defaultStorenodeRequestor struct {
	store *store.WakuStore
}

func (d *defaultStorenodeRequestor) GetMessagesByHash(ctx context.Context, peerID peer.ID, pageSize uint64, messageHashes []pb.MessageHash) (common.StoreRequestResult, error) {
	return d.store.QueryByHash(ctx, messageHashes, store.WithPeer(peerID), store.WithPaging(false, pageSize))
}

func (d *defaultStorenodeRequestor) Query(ctx context.Context, peerID peer.ID, storeQueryRequest *storepb.StoreQueryRequest) (common.StoreRequestResult, error) {
	return d.store.RequestRaw(ctx, peerID, storeQueryRequest)
}
