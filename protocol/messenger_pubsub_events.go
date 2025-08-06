package protocol

import (
	messagingtypes "github.com/status-im/status-go/messaging/types"
)

type RetrievedMessagesEvent struct {
	Filter     messagingtypes.ChatFilter
	SHHMessage *messagingtypes.ReceivedMessage
	Messages   []*messagingtypes.Message
}
