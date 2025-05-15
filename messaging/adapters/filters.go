package adapters

import (
	"github.com/status-im/status-go/messaging/layers/transport"
	"github.com/status-im/status-go/messaging/types"
)

func ChatsToInitializeToTransport(c types.ChatsToInitialize) []transport.FiltersToInitialize {
	filters := make([]transport.FiltersToInitialize, len(c))
	for i, chat := range c {
		filters[i] = transport.FiltersToInitialize{
			ChatID:      chat.ChatID,
			PubsubTopic: chat.PubsubTopic,
		}
	}
	return filters
}

func CommunityToInitializeToTransport(c *types.CommunityToInitialize) *transport.CommunityFilterToInitialize {
	return &transport.CommunityFilterToInitialize{
		Shard:   c.Shard,
		PrivKey: c.PrivKey,
	}
}

func CommunitiesToInitializeToTransport(c types.CommunitiesToInitialize) []transport.CommunityFilterToInitialize {
	communityFilters := make([]transport.CommunityFilterToInitialize, len(c))
	for i, filter := range c {
		communityFilters[i] = *CommunityToInitializeToTransport(filter)
	}
	return communityFilters
}

func FromTransportFilter(filter *transport.Filter) *types.ChatFilter {
	if filter == nil {
		return nil
	}
	return &types.ChatFilter{
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

func FromTransportFilters(filters []*transport.Filter) types.ChatFilters {
	chatFilters := make([]*types.ChatFilter, len(filters))
	for i, filter := range filters {
		chatFilters[i] = FromTransportFilter(filter)
	}
	return chatFilters
}

func ToTransportFilter(c *types.ChatFilter) *transport.Filter {
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

func ToTransportFilters(c types.ChatFilters) []*transport.Filter {
	transportFilters := make([]*transport.Filter, len(c))
	for i, filter := range c {
		transportFilters[i] = ToTransportFilter(filter)
	}
	return transportFilters
}
