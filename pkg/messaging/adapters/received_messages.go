package adapters

import (
	"github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/pkg/messaging/wakumetrics"
)

func ToWakumetricsReceivedMessages(receivedMessages types.ReceivedMessages) wakumetrics.ReceivedMessages {
	return wakumetrics.ReceivedMessages{
		Filter:     receivedMessages.Filter,
		SHHMessage: receivedMessages.SHHMessage,
		Messages:   receivedMessages.Messages,
	}
}
