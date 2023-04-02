package pairing

import (
	"context"

	"github.com/status-im/status-go/protocol/protobuf"

	"github.com/status-im/status-go/protocol/common"
)

type RawMessageCollector struct {
	rawMessages []*common.RawMessage
}

func (r *RawMessageCollector) dispatchMessage(_ context.Context, rawMessage common.RawMessage) (common.RawMessage, error) {
	r.rawMessages = append(r.rawMessages, &rawMessage)
	return rawMessage, nil
}

func (r *RawMessageCollector) getRawMessages() []*common.RawMessage {
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
