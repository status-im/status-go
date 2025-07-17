package protocol

import (
	messagingtypes "github.com/status-im/status-go/messaging/types"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/waku/types"
)

type RetrievedMessagesEvent struct {
	Filter     messagingtypes.ChatFilter
	SHHMessage *messagingtypes.ReceivedMessage
	Messages   []*v1protocol.StatusMessage
}

type DatasyncMessagesSentEvent struct {
	Messages []*types.NewMessage
}
