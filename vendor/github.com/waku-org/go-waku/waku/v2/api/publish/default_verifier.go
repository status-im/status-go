package publish

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
)

func NewDefaultStorenodeMessageVerifier(store *store.WakuStore) StorenodeMessageVerifier {
	return &defaultStorenodeMessageVerifier{
		store: store,
	}
}

type defaultStorenodeMessageVerifier struct {
	store *store.WakuStore
}

func (d *defaultStorenodeMessageVerifier) MessageHashesExist(ctx context.Context, requestID []byte, peerID peer.ID, pageSize uint64, messageHashes []pb.MessageHash) ([]pb.MessageHash, error) {
	var opts []store.RequestOption
	opts = append(opts, store.WithRequestID(requestID))
	opts = append(opts, store.WithPeer(peerID))
	opts = append(opts, store.WithPaging(false, pageSize))
	opts = append(opts, store.IncludeData(false))

	response, err := d.store.QueryByHash(ctx, messageHashes, opts...)
	if err != nil {
		return nil, err
	}

	result := make([]pb.MessageHash, len(response.Messages()))
	for i, msg := range response.Messages() {
		result[i] = msg.WakuMessageHash()
	}

	return result, nil
}
