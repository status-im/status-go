package pairing

import (
	"context"

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
