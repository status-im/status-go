package types

import (
	"crypto/ecdsa"

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
	chatID       string
	filterID     string
	identity     string
	pubsubTopic  string
	contentTopic ContentTopic
	discovery    bool
	negotiated   bool
	listen       bool
	ephemeral    bool
	priority     uint64
}

type ChatFilterConfig struct {
	ChatID       string
	FilterID     string
	Identity     string
	PubsubTopic  string
	ContentTopic ContentTopic
	Discovery    bool
	Negotiated   bool
	Listen       bool
	Ephemeral    bool
	Priority     uint64
}

func NewChatFilter(config *ChatFilterConfig) *ChatFilter {
	return &ChatFilter{
		chatID:       config.ChatID,
		filterID:     config.FilterID,
		identity:     config.Identity,
		pubsubTopic:  config.PubsubTopic,
		contentTopic: config.ContentTopic,
		discovery:    config.Discovery,
		negotiated:   config.Negotiated,
		listen:       config.Listen,
		ephemeral:    config.Ephemeral,
		priority:     config.Priority,
	}
}

func (c *ChatFilter) ChatID() string {
	return c.chatID
}

func (c *ChatFilter) FilterID() string {
	return c.filterID
}

func (c *ChatFilter) Identity() string {
	return c.identity
}

func (c *ChatFilter) PubsubTopic() string {
	return c.pubsubTopic
}

func (c *ChatFilter) ContentTopic() ContentTopic {
	return c.contentTopic
}

func (c *ChatFilter) IsDiscovery() bool {
	return c.discovery
}

func (c *ChatFilter) IsNegotiated() bool {
	return c.negotiated
}

func (c *ChatFilter) IsListening() bool {
	return c.listen
}

func (c *ChatFilter) IsEphemeral() bool {
	return c.ephemeral
}

func (c *ChatFilter) Priority() uint64 {
	return c.priority
}

type ChatFilters []*ChatFilter
