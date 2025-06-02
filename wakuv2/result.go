//go:build use_nwaku
// +build use_nwaku

package wakuv2

import (
	"context"
	"encoding/hex"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/waku-org/waku-go-bindings/waku"

	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
)

type storeResultImpl struct {
	done bool

	node          *waku.WakuNode
	storeRequest  *storepb.StoreQueryRequest
	storeResponse *storepb.StoreQueryResponse
	peerInfo      peer.AddrInfo
}

func newStoreResultImpl(node *waku.WakuNode, peerInfo peer.AddrInfo, storeRequest *storepb.StoreQueryRequest, storeResponse *storepb.StoreQueryResponse) *storeResultImpl {
	return &storeResultImpl{
		node:          node,
		storeRequest:  storeRequest,
		storeResponse: storeResponse,
		peerInfo:      peerInfo,
	}
}

func (r *storeResultImpl) Cursor() []byte {
	return r.storeResponse.GetPaginationCursor()
}

func (r *storeResultImpl) IsComplete() bool {
	return r.done
}

func (r *storeResultImpl) PeerID() peer.ID {
	return r.peerInfo.ID
}

func (r *storeResultImpl) Query() *storepb.StoreQueryRequest {
	return r.storeRequest
}

func (r *storeResultImpl) Response() *storepb.StoreQueryResponse {
	return r.storeResponse
}

func (r *storeResultImpl) Next(ctx context.Context, opts ...store.RequestOption) error {
	// TODO: opts is being ignored. Will require some changes in go-waku. For now using this
	// is not necessary

	if r.storeResponse.GetPaginationCursor() == nil {
		r.done = true
		return nil
	}

	r.storeRequest.RequestId = hex.EncodeToString(protocol.GenerateRequestID())
	r.storeRequest.PaginationCursor = r.storeResponse.PaginationCursor

	bindingsStoreRequest, err := PbToBindingsStoreRequest(r.storeRequest)

	if err != nil {
		return err
	}

	bindingsStoreResponse, err := r.node.StoreQuery(ctx, bindingsStoreRequest, r.peerInfo)
	if err != nil {
		return err
	}

	storeResponse := storepb.StoreQueryResponse{RequestId: bindingsStoreResponse.RequestId}

	r.storeResponse = &storeResponse
	return nil
}

func (r *storeResultImpl) Messages() []*storepb.WakuMessageKeyValue {
	return r.storeResponse.GetMessages()
}
