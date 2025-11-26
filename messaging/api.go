package messaging

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/waku-org/go-waku/waku/v2/api/history"
	"github.com/waku-org/sds-go-bindings/sds"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/status-im/status-go/connection"
	"github.com/status-im/status-go/messaging/adapters"
	"github.com/status-im/status-go/messaging/layers/encryption"
	"github.com/status-im/status-go/messaging/layers/transport"
	"github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/pkg/pubsub"
)

type API struct {
	core *Core
}

func NewAPI(core *Core) *API {
	return &API{
		core: core,
	}
}

func (a *API) Start() error {
	return a.core.start()
}

func (a *API) Stop() error {
	return a.core.stop()
}

func (a *API) Publisher() *pubsub.Publisher {
	return a.core.publisher
}

func (a *API) SDSManager() *sds.ReliabilityManager {
	return a.core.stack.SDSManager
}

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

func (a *API) GetStats() types.TransportStats {
	return adapters.FromWakuTransportStats(a.core.stack.Transport.GetStats())
}

func (a *API) RetrieveRawAll() (map[types.ChatFilter][]*types.ReceivedMessage, error) {
	filters, err := a.core.stack.Transport.RetrieveRawAll()
	if err != nil {
		return nil, err
	}
	chatFilters := make(map[types.ChatFilter][]*types.ReceivedMessage)
	for k, v := range filters {
		chatFilters[*adapters.FromTransportFilter(&k)] = adapters.FromWakuMessages(v)
	}
	return chatFilters, nil
}

func (a *API) HandleReceivedMessages(msg *types.ReceivedMessage) (*types.HandleMessageResponse, error) {
	return a.core.controller.Processor().ProcessMessage(msg)
}

func (a *API) GetKeysForGroup(groupID []byte) ([]*encryption.HashRatchetKeyCompatibility, error) {
	return a.core.stack.Encryption.GetKeysForGroup(groupID)
}

func (a *API) GetCurrentKeyForGroup(groupID []byte) (*encryption.HashRatchetKeyCompatibility, error) {
	return a.core.stack.Encryption.GetCurrentKeyForGroup(groupID)
}

func (a *API) SaveHashRatchetMessage(groupID []byte, keyID []byte, m *types.ReceivedMessage) error {
	return a.core.controller.SaveHashRatchetMessage(groupID, keyID, m)
}

func (a *API) GetHashRatchetMessagesCountForGroup(groupID []byte) (int, error) {
	return a.core.controller.GetHashRatchetMessagesCountForGroup(groupID)
}

func (a *API) GetEphemeralKey() (*ecdsa.PrivateKey, error) {
	return a.core.controller.Processor().GetEphemeralKey()
}

func (a *API) PersonalTopicFilter() *types.ChatFilter {
	return adapters.FromTransportFilter(a.core.stack.Transport.PersonalTopicFilter())
}

>>>>>>> 43d1dfa3e (feat: enable sds for wrap message)
func (a *API) GetCurrentTime() uint64 {
	return a.core.stack.Transport.GetCurrentTime()
}
