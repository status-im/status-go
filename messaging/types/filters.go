package types

import (
	"crypto/ecdsa"

	wakutypes "github.com/status-im/status-go/waku/types"
	"github.com/status-im/status-go/wakuv2"
)

type ChatToInitialize struct {
	ChatID      string
	PubsubTopic string
}

type ChatsToInitialize []*ChatToInitialize

type CommunityToInitialize struct {
	Shard   *wakuv2.Shard
	PrivKey *ecdsa.PrivateKey
}

type CommunitiesToInitialize []*CommunityToInitialize

type ChatFilter struct {
	ChatID       string              `json:"chatId"`
	FilterID     string              `json:"filterId"`
	Identity     string              `json:"identity"`
	PubsubTopic  string              `json:"pubsubTopic"`
	ContentTopic wakutypes.TopicType `json:"topic"`
	Discovery    bool                `json:"discovery"`
	Negotiated   bool                `json:"negotiated"`
	Listen       bool                `json:"listen"`
	Ephemeral    bool                `json:"ephemeral"`
	Priority     uint64
}

type ChatFilters []*ChatFilter
