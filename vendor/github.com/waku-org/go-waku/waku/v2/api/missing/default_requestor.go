package missing

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/api/common"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
)

func NewDefaultStorenodeRequestor(store *store.WakuStore) StorenodeRequestor {
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

func (d *defaultStorenodeRequestor) QueryWithCriteria(ctx context.Context, peerID peer.ID, pageSize uint64, pubsubTopic string, contentTopics []string, from *int64, to *int64) (common.StoreRequestResult, error) {
	return d.store.Query(ctx, store.FilterCriteria{
		ContentFilter: protocol.NewContentFilter(pubsubTopic, contentTopics...),
		TimeStart:     from,
		TimeEnd:       to,
	}, store.WithPeer(peerID), store.WithPaging(false, pageSize), store.IncludeData(false))
}
