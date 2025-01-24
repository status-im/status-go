//go:build use_nwaku
// +build use_nwaku

package wakuv2

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/waku-org/go-waku/waku/v2/api/publish"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
	"github.com/waku-org/waku-go-bindings/waku"
)

type storenodeMessageVerifier struct {
	node *waku.WakuNode
}

func newStorenodeMessageVerifier(node *waku.WakuNode) publish.StorenodeMessageVerifier {
	return &storenodeMessageVerifier{
		node: node,
	}
}

func (d *storenodeMessageVerifier) MessageHashesExist(ctx context.Context, requestID []byte, peerInfo peer.AddrInfo, pageSize uint64, messageHashes []pb.MessageHash) ([]pb.MessageHash, error) {
	requestIDStr := hex.EncodeToString(requestID)
	storeRequest := &storepb.StoreQueryRequest{
		RequestId:         requestIDStr,
		MessageHashes:     make([][]byte, len(messageHashes)),
		IncludeData:       false,
		PaginationCursor:  nil,
		PaginationForward: false,
		PaginationLimit:   proto.Uint64(pageSize),
	}

	for i, mhash := range messageHashes {
		storeRequest.MessageHashes[i] = mhash.Bytes()
	}

	response, err := d.node.StoreQuery(ctx, storeRequest, peerInfo)
	if err != nil {
		return nil, err
	}

	if response.GetStatusCode() != http.StatusOK {
		return nil, fmt.Errorf("could not query storenode: %s %d %s", requestIDStr, response.GetStatusCode(), response.GetStatusDesc())
	}

	result := make([]pb.MessageHash, len(response.Messages))
	for i, msg := range response.Messages {
		result[i] = pb.ToMessageHash(msg.GetMessageHash())
	}

	return result, nil
}
