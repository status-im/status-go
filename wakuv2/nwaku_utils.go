//go:build use_nwaku
// +build use_nwaku

package wakuv2

import (
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
	"github.com/waku-org/waku-go-bindings/waku/common"
)

func HexToPbHash(hexHash common.MessageHash) (pb.MessageHash, error) {
	bytesHash, err := hexHash.Bytes()
	if err != nil {
		return pb.MessageHash{}, err
	}

	pbHash := pb.ToMessageHash(bytesHash)
	return pbHash, nil
}

func PbToHexHash(pbHash pb.MessageHash) (common.MessageHash, error) {
	return common.ToMessageHash(pbHash.String())
}

func PbToBindingsStoreRequest(pbStoreRequest *storepb.StoreQueryRequest) (*common.StoreQueryRequest, error) {

	var messageHashes []common.MessageHash
	var paginationCursor common.MessageHash

	for _, hash := range pbStoreRequest.MessageHashes {
		hexHash, err := PbToHexHash(pb.ToMessageHash(hash))
		if err != nil {
			return nil, err
		}
		messageHashes = append(messageHashes, hexHash)
	}

	paginationCursor, err := PbToHexHash(pb.ToMessageHash(pbStoreRequest.PaginationCursor))

	if err != nil {
		return nil, err
	}

	bindingsQueryRequest := common.StoreQueryRequest{
		RequestId:         pbStoreRequest.RequestId,
		IncludeData:       pbStoreRequest.IncludeData,
		PubsubTopic:       *pbStoreRequest.PubsubTopic,
		ContentTopics:     &pbStoreRequest.ContentTopics,
		TimeStart:         pbStoreRequest.TimeStart,
		MessageHashes:     &messageHashes,
		TimeEnd:           pbStoreRequest.TimeEnd,
		PaginationCursor:  &paginationCursor,
		PaginationForward: pbStoreRequest.PaginationForward,
		PaginationLimit:   pbStoreRequest.PaginationLimit,
	}
	return &bindingsQueryRequest, nil
}
