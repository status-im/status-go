package protocol

import (
	messagingtypes "github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/protocol/common"
)

type RetrievedMessagesEvent struct {
	Filter     messagingtypes.ChatFilter
	SHHMessage *messagingtypes.ReceivedMessage
	Messages   []*common.StatusMessage
}
