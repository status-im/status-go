package adapters

import (
	"github.com/status-im/status-go/pkg/messaging/types"
	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
)

func ToWakuTopic(t types.ContentTopic) wakutypes.TopicType {
	return wakutypes.TopicType(t)
}

func ToWakuTopics(topics []types.ContentTopic) []wakutypes.TopicType {
	if topics == nil {
		return nil
	}
	wakuTopics := make([]wakutypes.TopicType, len(topics))
	for i, t := range topics {
		wakuTopics[i] = ToWakuTopic(t)
	}
	return wakuTopics
}

func FromWakuTopic(t wakutypes.TopicType) types.ContentTopic {
	return types.ContentTopic(t)
}
