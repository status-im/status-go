package messaging

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/messaging/transport"
	wakutypes "github.com/status-im/status-go/waku/types"
	"github.com/status-im/status-go/wakuv2"
)

type ChatToInitialize struct {
	ChatID      string
	PubsubTopic string
}

type ChatsToInitialize []*ChatToInitialize

func (c ChatsToInitialize) toFilters() []transport.FiltersToInitialize {
	filters := make([]transport.FiltersToInitialize, len(c))
	for i, chat := range c {
		filters[i] = transport.FiltersToInitialize{
			ChatID:      chat.ChatID,
			PubsubTopic: chat.PubsubTopic,
		}
	}
	return filters
}

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

func fromTransportFilter(filter *transport.Filter) *ChatFilter {
	if filter == nil {
		return nil
	}
	return &ChatFilter{
		ChatID:       filter.ChatID,
		FilterID:     filter.FilterID,
		Identity:     filter.Identity,
		PubsubTopic:  filter.PubsubTopic,
		ContentTopic: filter.ContentTopic,
		Discovery:    filter.Discovery,
		Negotiated:   filter.Negotiated,
		Listen:       filter.Listen,
		Ephemeral:    filter.Ephemeral,
		Priority:     filter.Priority,
	}
}

func fromTransportFilters(filters []*transport.Filter) ChatFilters {
	chatFilters := make([]*ChatFilter, len(filters))
	for i, filter := range filters {
		chatFilters[i] = fromTransportFilter(filter)
	}
	return chatFilters
}

func (c *ChatFilter) toTransportFilter() *transport.Filter {
	return &transport.Filter{
		ChatID:       c.ChatID,
		FilterID:     c.FilterID,
		Identity:     c.Identity,
		PubsubTopic:  c.PubsubTopic,
		ContentTopic: c.ContentTopic,
		Discovery:    c.Discovery,
		Negotiated:   c.Negotiated,
		Listen:       c.Listen,
		Ephemeral:    c.Ephemeral,
		Priority:     c.Priority,
	}
}

func (c ChatFilters) toTransportFilters() []*transport.Filter {
	transportFilters := make([]*transport.Filter, len(c))
	for i, filter := range c {
		transportFilters[i] = filter.toTransportFilter()
	}
	return transportFilters
}

type CommunityToInitialize struct {
	Shard   *wakuv2.Shard
	PrivKey *ecdsa.PrivateKey
}

type CommunitiesToInitialize []*CommunityToInitialize

func (c *CommunityToInitialize) toFilter() *transport.CommunityFilterToInitialize {
	return &transport.CommunityFilterToInitialize{
		Shard:   c.Shard,
		PrivKey: c.PrivKey,
	}
}

func (c CommunitiesToInitialize) toFilters() []transport.CommunityFilterToInitialize {
	communityFilters := make([]transport.CommunityFilterToInitialize, len(c))
	for i, filter := range c {
		communityFilters[i] = *filter.toFilter()
	}
	return communityFilters
}

type EnvelopeEventsHandler interface {
	EnvelopeSent([][]byte)
	EnvelopeExpired([][]byte, error)
	MailServerRequestCompleted(types.Hash, types.Hash, []byte, error)
	MailServerRequestExpired(types.Hash)
}

type EnvelopeEventsConfig struct {
	EnvelopeEventsHandler      EnvelopeEventsHandler
	MaxMessageDeliveryAttempts int
	MailServerConfirmations    bool
}
