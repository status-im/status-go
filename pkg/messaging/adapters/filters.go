package adapters

import (
	transport2 "github.com/status-im/status-go/pkg/messaging/layers/transport"
	"github.com/status-im/status-go/pkg/messaging/types"
)

func ChatsToInitializeToTransport(c types.ChatsToInitialize) []transport2.FiltersToInitialize {
	filters := make([]transport2.FiltersToInitialize, len(c))
	for i, chat := range c {
		filters[i] = transport2.FiltersToInitialize{
			ChatID:      chat.ChatID,
			PubsubTopic: chat.PubsubTopic,
			// TODO (#6384) temporary flag while migrating community shards
			DistinctByPubsub: chat.IsCommunity,
		}
	}
	return filters
}

func CommunityToInitializeToTransport(c *types.CommunityToInitialize) *transport2.CommunityFilterToInitialize {
	return &transport2.CommunityFilterToInitialize{
		PrivKey: c.PrivKey,
	}
}

func CommunitiesToInitializeToTransport(c types.CommunitiesToInitialize) []transport2.CommunityFilterToInitialize {
	communityFilters := make([]transport2.CommunityFilterToInitialize, len(c))
	for i, filter := range c {
		communityFilters[i] = *CommunityToInitializeToTransport(filter)
	}
	return communityFilters
}

func FromTransportFilter(filter *transport2.Filter) *types.ChatFilter {
	if filter == nil {
		return nil
	}

	return types.NewChatFilter(
		&types.ChatFilterConfig{
			ChatID:       filter.ChatID,
			FilterID:     filter.FilterID,
			Identity:     filter.Identity,
			PubsubTopic:  filter.PubsubTopic,
			ContentTopic: FromWakuTopic(filter.ContentTopic),
			Discovery:    filter.Discovery,
			Negotiated:   filter.Negotiated,
			Listen:       filter.Listen,
			Ephemeral:    filter.Ephemeral,
			Priority:     filter.Priority,
		},
	)
}

func FromTransportFilters(filters []*transport2.Filter) types.ChatFilters {
	chatFilters := make([]*types.ChatFilter, len(filters))
	for i, filter := range filters {
		chatFilters[i] = FromTransportFilter(filter)
	}
	return chatFilters
}

func ToTransportFilter(c *types.ChatFilter) *transport2.Filter {
	return &transport2.Filter{
		ChatID:       c.ChatID(),
		FilterID:     c.FilterID(),
		Identity:     c.Identity(),
		PubsubTopic:  c.PubsubTopic(),
		ContentTopic: ToWakuTopic(c.ContentTopic()),
		Discovery:    c.IsDiscovery(),
		Negotiated:   c.IsNegotiated(),
		Listen:       c.IsListening(),
		Ephemeral:    c.IsEphemeral(),
		Priority:     c.Priority(),
	}
}

func ToTransportFilters(c types.ChatFilters) []*transport2.Filter {
	transportFilters := make([]*transport2.Filter, len(c))
	for i, filter := range c {
		transportFilters[i] = ToTransportFilter(filter)
	}
	return transportFilters
}
