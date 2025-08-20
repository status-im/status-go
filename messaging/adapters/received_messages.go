package adapters

import (
	"github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/messaging/wakumetrics"
)

func ToWakumetricsReceivedMessages(receivedMessages types.ReceivedMessages) wakumetrics.ReceivedMessages {
	return wakumetrics.ReceivedMessages{
		Filter:     receivedMessages.Filter,
		SHHMessage: receivedMessages.SHHMessage,
		Messages:   receivedMessages.Messages,
	}
}
