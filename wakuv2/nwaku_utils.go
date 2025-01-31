//go:build use_nwaku
// +build use_nwaku

package wakuv2

// TODO-nwaku remove this entire file once go-waku is removed from status-go
import (
	"github.com/status-im/status-go/wakuv2/common"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
	bindings "github.com/waku-org/waku-go-bindings/waku/common"
)

type envelopeImpl struct {
	msg   *pb.WakuMessage
	topic string
	hash  pb.MessageHash
}

func (e *envelopeImpl) Message() *pb.WakuMessage {
	return e.msg
}

func (e *envelopeImpl) PubsubTopic() string {
	return e.topic
}

func (e *envelopeImpl) Hash() pb.MessageHash {
	return e.hash
}

func HexToPbHash(hexHash bindings.MessageHash) (pb.MessageHash, error) {
	bytesHash, err := hexHash.Bytes()
	if err != nil {
		return pb.MessageHash{}, err
	}

	pbHash := pb.ToMessageHash(bytesHash)
	return pbHash, nil
}

func PbToHexHash(pbHash pb.MessageHash) (bindings.MessageHash, error) {
	return bindings.ToMessageHash(pbHash.String())
}

func PbToBindingsStoreRequest(pbStoreRequest *storepb.StoreQueryRequest) (*bindings.StoreQueryRequest, error) {

	var messageHashes []bindings.MessageHash
	var paginationCursor bindings.MessageHash

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

	bindingsQueryRequest := bindings.StoreQueryRequest{
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

func BindingsToPbStoreResponse(bindingsStoreResponse *bindings.StoreQueryResponse) (*storepb.StoreQueryResponse, error) {

	paginationCursor, err := bindingsStoreResponse.PaginationCursor.Bytes()
	if err != nil {
		return nil, err
	}

	pbQueryResponse := storepb.StoreQueryResponse{
		RequestId:        bindingsStoreResponse.RequestId,
		StatusCode:       bindingsStoreResponse.StatusCode,
		StatusDesc:       &bindingsStoreResponse.StatusDesc,
		PaginationCursor: paginationCursor,
	}

	if bindingsStoreResponse.Messages == nil {
		return &pbQueryResponse, nil
	}

	var messages []*storepb.WakuMessageKeyValue

	for _, message := range *bindingsStoreResponse.Messages {

		msgHash, err := message.MessageHash.Bytes()

		if err != nil {
			return nil, err
		}

		wakuMessage := pb.WakuMessage{
			Payload:        message.WakuMessage.Payload,
			ContentTopic:   message.WakuMessage.ContentTopic,
			Version:        message.WakuMessage.Version,
			Timestamp:      message.WakuMessage.Timestamp,
			Meta:           message.WakuMessage.Meta,
			Ephemeral:      message.WakuMessage.Ephemeral,
			RateLimitProof: message.WakuMessage.RateLimitProof,
		}

		pbMessage := storepb.WakuMessageKeyValue{
			MessageHash: msgHash,
			PubsubTopic: &message.PubsubTopic,
			Message:     &wakuMessage,
		}

		messages = append(messages, &pbMessage)
	}

	pbQueryResponse.Messages = messages

	return &pbQueryResponse, nil

}

func BindingsToCommonEnvelope(bindingsEnv bindings.Envelope) (common.Envelope, error) {

	hash, err := HexToPbHash(bindingsEnv.Hash())

	if err != nil {
		return nil, err
	}

	env := envelopeImpl{topic: bindingsEnv.PubsubTopic(), msg: bindingsEnv.Message(), hash: hash}

	return &env, nil
}
