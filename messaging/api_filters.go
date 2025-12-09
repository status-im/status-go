package messaging

import (
	"crypto/ecdsa"

	"github.com/pkg/errors"

	"github.com/status-im/status-go/messaging/adapters"
	"github.com/status-im/status-go/messaging/types"
)

func (a *API) InitChats(chats types.ChatsToInitialize, publicKeys []*ecdsa.PublicKey) error {
	_, err := a.core.stack.Transport.InitFilters(adapters.ChatsToInitializeToTransport(chats), publicKeys)
	return err
}

func (a *API) InitPublicChats(chats types.ChatsToInitialize) (types.ChatFilters, error) {
	filters, err := a.core.stack.Transport.InitPublicFilters(adapters.ChatsToInitializeToTransport(chats))
	return adapters.FromTransportFilters(filters), err
}

func (a *API) InitCommunities(communities types.CommunitiesToInitialize) (types.ChatFilters, error) {
	filters, err := a.core.stack.Transport.InitCommunityFilters(adapters.CommunitiesToInitializeToTransport(communities))
	return adapters.FromTransportFilters(filters), err
}

func (a *API) ChatFilters() types.ChatFilters {
	return adapters.FromTransportFilters(a.core.stack.Transport.Filters())
}

func (a *API) ChatFilterByChatID(chatID string) *types.ChatFilter {
	return adapters.FromTransportFilter(a.core.stack.Transport.FilterByChatID(chatID))
}

func (a *API) ChatFilterByTopic(topic []byte) *types.ChatFilter {
	return adapters.FromTransportFilter(a.core.stack.Transport.FilterByTopic(topic))
}

func (a *API) ChatFiltersByIdentities(identities []string) types.ChatFilters {
	return adapters.FromTransportFilters(a.core.stack.Transport.FiltersByIdentities(identities))
}

func (a *API) RemoveFilters(filters types.ChatFilters) error {
	return a.core.stack.Transport.RemoveFilters(adapters.ToTransportFilters(filters))
}

func (a *API) RemoveFilterByChatID(chatID string) (*types.ChatFilter, error) {
	filter, err := a.core.stack.Transport.RemoveFilterByChatID(chatID)
	if err != nil {
		return nil, err
	}
	return adapters.FromTransportFilter(filter), nil
}

func (a *API) UpdateFilterPriority(chatID string, priority uint64) error {
	transportFilter := a.core.stack.Transport.FilterByChatID(chatID)
	if transportFilter == nil {
		return errors.New("filter not found")
	}

	transportFilter.Priority = priority

	return nil
}

func (a *API) UpdateFilterEphemerality(chatID string, ephemeral bool) error {
	transportFilter := a.core.stack.Transport.FilterByChatID(chatID)
	if transportFilter == nil {
		return errors.New("filter not found")
	}

	transportFilter.Ephemeral = ephemeral

	return nil
}

func (a *API) JoinPublicChat(chatID string) (*types.ChatFilter, error) {
	f, err := a.core.stack.Transport.JoinPublic(chatID)
	if err != nil {
		return nil, err
	}
	return adapters.FromTransportFilter(f), nil
}

func (a *API) JoinPrivateChat(publicKey *ecdsa.PublicKey) (*types.ChatFilter, error) {
	filter, err := a.core.stack.Transport.JoinPrivate(publicKey)
	if err != nil {
		return nil, err
	}
	return adapters.FromTransportFilter(filter), nil
}

func (a *API) JoinGroupChat(publicKeys []*ecdsa.PublicKey) (types.ChatFilters, error) {
	filters, err := a.core.stack.Transport.JoinGroup(publicKeys)
	if err != nil {
		return nil, err
	}
	return adapters.FromTransportFilters(filters), nil
}

func (a *API) PersonalTopicFilter() *types.ChatFilter {
	return adapters.FromTransportFilter(a.core.stack.Transport.PersonalTopicFilter())
}
