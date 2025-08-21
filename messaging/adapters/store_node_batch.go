package adapters

import (
	"github.com/status-im/status-go/messaging/types"
	wakutypes "github.com/status-im/status-go/messaging/waku/types"
)

func ToWakuBatch(b *types.StoreNodeBatch) *wakutypes.MailserverBatch {
	if b == nil {
		return nil
	}
	return &wakutypes.MailserverBatch{
		From:        b.From,
		To:          b.To,
		PubsubTopic: b.PubsubTopic,
		Topics:      ToWakuTopics(b.Topics),
		ChatIDs:     b.ChatIDs,
	}
}
