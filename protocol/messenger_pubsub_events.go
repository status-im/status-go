package protocol

import (
	"github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/protocol/common"
)

type RetrievedMessagesEvent struct {
	Filter     types.ChatFilter
	SHHMessage *types.ReceivedMessage
	Messages   []*common.StatusMessage
}
