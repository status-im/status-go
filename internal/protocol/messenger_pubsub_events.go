package protocol

import (
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/pkg/messaging/types"
)

type RetrievedMessagesEvent struct {
	Filter     types.ChatFilter
	SHHMessage *types.ReceivedMessage
	Messages   []*common.StatusMessage
}
