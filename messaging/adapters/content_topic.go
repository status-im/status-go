package adapters

import (
	"github.com/status-im/status-go/messaging/types"
	wakutypes "github.com/status-im/status-go/waku/types"
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

func FromWakuTopics(topics []wakutypes.TopicType) []types.ContentTopic {
	if topics == nil {
		return nil
	}
	contentTopics := make([]types.ContentTopic, len(topics))
	for i, t := range topics {
		contentTopics[i] = FromWakuTopic(t)
	}
	return contentTopics
}
