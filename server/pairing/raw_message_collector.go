package pairing

import (
	"context"

	messagingtypes "github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

type RawMessageCollector struct {
	rawMessages []*messagingtypes.RawMessage
}

func (r *RawMessageCollector) dispatchMessage(_ context.Context, rawMessage messagingtypes.RawMessage) (messagingtypes.RawMessage, error) {
	r.rawMessages = append(r.rawMessages, &rawMessage)
	return rawMessage, nil
}

func (r *RawMessageCollector) getRawMessages() []*messagingtypes.RawMessage {
	return r.rawMessages
}

func (r *RawMessageCollector) convertToSyncRawMessage() *protobuf.SyncRawMessage {
	syncRawMessage := new(protobuf.SyncRawMessage)
	for _, m := range r.getRawMessages() {
		rawMessage := new(protobuf.RawMessage)
		rawMessage.Payload = m.Payload
		rawMessage.MessageType = m.MessageType
		syncRawMessage.RawMessages = append(syncRawMessage.RawMessages, rawMessage)
	}
	return syncRawMessage
}
