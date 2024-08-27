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
	commonapi "github.com/waku-org/go-waku/waku/v2/api/common"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
	"go.uber.org/zap"
)

type storenodeRequestor struct {
	node   *WakuNode
	logger *zap.Logger
}

func newStorenodeRequestor(node *WakuNode, logger *zap.Logger) commonapi.StorenodeRequestor {
	return &storenodeRequestor{
		node:   node,
		logger: logger.Named("storenodeRequestor"),
	}
}

func (s *storenodeRequestor) GetMessagesByHash(ctx context.Context, peerInfo peer.AddrInfo, pageSize uint64, messageHashes []pb.MessageHash) (commonapi.StoreRequestResult, error) {
	requestIDStr := hex.EncodeToString(protocol.GenerateRequestID())

	logger := s.logger.With(zap.Stringer("peerID", peerInfo.ID), zap.String("requestID", requestIDStr))

	logger.Debug("sending store request")

	storeRequest := &storepb.StoreQueryRequest{
		RequestId:         requestIDStr,
		MessageHashes:     make([][]byte, len(messageHashes)),
		IncludeData:       true,
		PaginationCursor:  nil,
		PaginationForward: false,
		PaginationLimit:   proto.Uint64(pageSize),
	}

	for i, mhash := range messageHashes {
		storeRequest.MessageHashes[i] = mhash.Bytes()
	}

	storeResponse, err := s.node.StoreQuery(ctx, storeRequest, peerInfo)
	if err != nil {
		return nil, err
	}

	if storeResponse.GetStatusCode() != http.StatusOK {
		return nil, fmt.Errorf("could not query storenode: %s %d %s", requestIDStr, storeResponse.GetStatusCode(), storeResponse.GetStatusDesc())
	}

	return newStoreResultImpl(s.node, peerInfo, storeRequest, storeResponse), nil
}

func (s *storenodeRequestor) Query(ctx context.Context, peerInfo peer.AddrInfo, storeRequest *storepb.StoreQueryRequest) (commonapi.StoreRequestResult, error) {
	storeResponse, err := s.node.StoreQuery(ctx, storeRequest, peerInfo)
	if err != nil {
		return nil, err
	}

	if storeResponse.GetStatusCode() != http.StatusOK {
		return nil, fmt.Errorf("could not query storenode: %s %d %s", storeRequest.RequestId, storeResponse.GetStatusCode(), storeResponse.GetStatusDesc())
	}

	return newStoreResultImpl(s.node, peerInfo, storeRequest, storeResponse), nil
}
