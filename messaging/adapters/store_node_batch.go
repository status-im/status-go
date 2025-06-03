package adapters

import (
	"github.com/status-im/status-go/messaging/types"
	wakutypes "github.com/status-im/status-go/waku/types"
)

func FromWakuBatch(b *wakutypes.MailserverBatch) *types.StoreNodeBatch {
	if b == nil {
		return nil
	}
	return &types.StoreNodeBatch{
		From:        b.From,
		To:          b.To,
		PubsubTopic: b.PubsubTopic,
		Topics:      FromWakuTopics(b.Topics),
		ChatIDs:     b.ChatIDs,
	}
}

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
