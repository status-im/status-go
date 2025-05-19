package messaging

import (
	"context"
	"crypto/ecdsa"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/waku-org/go-waku/waku/v2/api/history"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/connection"
	ethtypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/messaging/adapters"
	"github.com/status-im/status-go/messaging/layers/transport"
	"github.com/status-im/status-go/messaging/types"
	wakutypes "github.com/status-im/status-go/waku/types"
)

type API struct {
	waku      wakutypes.Waku
	transport *transport.Transport
}

func NewAPI(waku wakutypes.Waku, transport *transport.Transport) *API {
	return &API{
		waku:      waku,
		transport: transport,
	}
}

func (a *API) InitChats(chats types.ChatsToInitialize, publicKeys []*ecdsa.PublicKey) error {
	_, err := a.transport.InitFilters(adapters.ChatsToInitializeToTransport(chats), publicKeys)
	return err
}

func (a *API) InitPublicChats(chats types.ChatsToInitialize) (types.ChatFilters, error) {
	filters, err := a.transport.InitPublicFilters(adapters.ChatsToInitializeToTransport(chats))
	return adapters.FromTransportFilters(filters), err
}

func (a *API) InitCommunities(communities types.CommunitiesToInitialize) (types.ChatFilters, error) {
	filters, err := a.transport.InitCommunityFilters(adapters.CommunitiesToInitializeToTransport(communities))
	return adapters.FromTransportFilters(filters), err
}

func (a *API) ChatFilters() types.ChatFilters {
	return adapters.FromTransportFilters(a.transport.Filters())
}

func (a *API) ChatFilterByChatID(chatID string) *types.ChatFilter {
	return adapters.FromTransportFilter(a.transport.FilterByChatID(chatID))
}

func (a *API) ChatFilterByTopic(topic []byte) *types.ChatFilter {
	return adapters.FromTransportFilter(a.transport.FilterByTopic(topic))
}

func (a *API) ChatFiltersByIdentities(identities []string) types.ChatFilters {
	return adapters.FromTransportFilters(a.transport.FiltersByIdentities(identities))
}

func (a *API) RemoveFilters(filters types.ChatFilters) error {
	return a.transport.RemoveFilters(adapters.ToTransportFilters(filters))
}

func (a *API) RemoveFilterByChatID(chatID string) (*types.ChatFilter, error) {
	filter, err := a.transport.RemoveFilterByChatID(chatID)
	if err != nil {
		return nil, err
	}
	return adapters.FromTransportFilter(filter), nil
}

func (a *API) ResetChatFilters(ctx context.Context) error {
	return a.transport.ResetFilters(ctx)
}

func (a *API) ProcessNegotiatedSecret(secret ethtypes.NegotiatedSecret) (*types.ChatFilter, error) {
	filter, err := a.transport.ProcessNegotiatedSecret(secret)
	if err != nil {
		return nil, err
	}
	return adapters.FromTransportFilter(filter), nil
}

func (a *API) JoinPublicChat(chatID string) (*types.ChatFilter, error) {
	filter, err := a.transport.JoinPublic(chatID)
	if err != nil {
		return nil, err
	}
	return adapters.FromTransportFilter(filter), nil
}

func (a *API) JoinPrivateChat(publicKey *ecdsa.PublicKey) (*types.ChatFilter, error) {
	filter, err := a.transport.JoinPrivate(publicKey)
	if err != nil {
		return nil, err
	}
	return adapters.FromTransportFilter(filter), nil
}

func (a *API) JoinGroupChat(publicKeys []*ecdsa.PublicKey) (types.ChatFilters, error) {
	filters, err := a.transport.JoinGroup(publicKeys)
	if err != nil {
		return nil, err
	}
	return adapters.FromTransportFilters(filters), nil
}

func (a *API) GetStats() wakutypes.StatsSummary {
	return a.transport.GetStats()
}

func (a *API) RetrieveRawAll() (map[types.ChatFilter][]*types.ReceivedMessage, error) {
	filters, err := a.transport.RetrieveRawAll()
	if err != nil {
		return nil, err
	}
	chatFilters := make(map[types.ChatFilter][]*types.ReceivedMessage)
	for k, v := range filters {
		chatFilters[*adapters.FromTransportFilter(&k)] = adapters.FromWakuMessages(v)
	}
	return chatFilters, nil
}

func (a *API) SendPublic(ctx context.Context, newMessage *wakutypes.NewMessage, chatName string) ([]byte, error) {
	return a.transport.SendPublic(ctx, newMessage, chatName)
}

func (a *API) SendPrivateWithSharedSecret(ctx context.Context, newMessage *wakutypes.NewMessage, publicKey *ecdsa.PublicKey, secret []byte) ([]byte, error) {
	return a.transport.SendPrivateWithSharedSecret(ctx, newMessage, publicKey, secret)
}

func (a *API) SendPrivateWithPartitioned(ctx context.Context, newMessage *wakutypes.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
	return a.transport.SendPrivateWithPartitioned(ctx, newMessage, publicKey)
}

func (a *API) SendPrivateOnPersonalTopic(ctx context.Context, newMessage *wakutypes.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
	return a.transport.SendPrivateOnPersonalTopic(ctx, newMessage, publicKey)
}

func (a *API) PersonalTopicFilter() *types.ChatFilter {
	return adapters.FromTransportFilter(a.transport.PersonalTopicFilter())
}

func (a *API) LoadKeyFilters(key *ecdsa.PrivateKey) (*types.ChatFilter, error) {
	filter, err := a.transport.LoadKeyFilters(key)
	if err != nil {
		return nil, err
	}
	return adapters.FromTransportFilter(filter), nil
}

func (a *API) SendCommunityMessage(ctx context.Context, newMessage *wakutypes.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
	return a.transport.SendCommunityMessage(ctx, newMessage, publicKey)
}

func (a *API) Track(identifier []byte, hashes [][]byte, newMessages []*wakutypes.NewMessage) {
	a.transport.Track(identifier, hashes, newMessages)
}

func (a *API) TrackMany(identifiers [][]byte, hashes [][]byte, newMessages []*wakutypes.NewMessage) {
	a.transport.TrackMany(identifiers, hashes, newMessages)
}

func (a *API) GetCurrentTime() uint64 {
	return a.transport.GetCurrentTime()
}

func (a *API) MaxMessageSize() uint32 {
	return a.transport.MaxMessageSize()
}

func (a *API) Stop() error {
	return a.transport.Stop()
}

func (a *API) PeerCount() int {
	return a.transport.PeerCount()
}

func (a *API) Peers() wakutypes.PeerStats {
	return a.transport.Peers()
}

func (a *API) ConfirmMessagesProcessed(ids []string, timestamp uint64) error {
	return a.transport.ConfirmMessagesProcessed(ids, timestamp)
}

func (a *API) CleanMessagesProcessed(timestamp uint64) error {
	return a.transport.CleanMessagesProcessed(timestamp)
}

func (a *API) SetEnvelopeEventsHandler(handler types.EnvelopeEventsHandler) error {
	return a.transport.SetEnvelopeEventsHandler(handler)
}

func (a *API) ClearProcessedMessageIDsCache() error {
	return a.transport.ClearProcessedMessageIDsCache()
}

func (a *API) ListenAddresses() ([]multiaddr.Multiaddr, error) {
	return a.transport.ListenAddresses()
}

func (a *API) RelayPeersByTopic(topic string) (*wakutypes.PeerList, error) {
	return a.transport.RelayPeersByTopic(topic)
}

func (a *API) ENR() (*enode.Node, error) {
	return a.transport.ENR()
}

func (a *API) AddRelayPeer(address multiaddr.Multiaddr) (peer.ID, error) {
	return a.transport.AddRelayPeer(address)
}

func (a *API) DialPeer(address multiaddr.Multiaddr) error {
	return a.transport.DialPeer(address)
}

func (a *API) DialPeerByID(peerID peer.ID) error {
	return a.transport.DialPeerByID(peerID)
}

func (a *API) DropPeer(peerID peer.ID) error {
	return a.transport.DropPeer(peerID)
}

func (a *API) MarkP2PMessageAsProcessed(hash common.Hash) {
	a.transport.MarkP2PMessageAsProcessed(hash)
}

func (a *API) ConnectionChanged(state connection.State) {
	a.transport.ConnectionChanged(state)
}

func (a *API) SubscribeToPubsubTopic(topic string, optPublicKey *ecdsa.PublicKey) error {
	return a.transport.SubscribeToPubsubTopic(topic, optPublicKey)
}

func (a *API) UnsubscribeFromPubsubTopic(topic string) error {
	return a.transport.UnsubscribeFromPubsubTopic(topic)
}

func (a *API) StorePubsubTopicKey(topic string, privKey *ecdsa.PrivateKey) error {
	return a.transport.StorePubsubTopicKey(topic, privKey)
}

func (a *API) RetrievePubsubTopicKey(topic string) (*ecdsa.PrivateKey, error) {
	return a.transport.RetrievePubsubTopicKey(topic)
}

func (a *API) RemovePubsubTopicKey(topic string) error {
	return a.transport.RemovePubsubTopicKey(topic)
}

func (a *API) ConfirmMessageDelivered(messageID string) {
	a.transport.ConfirmMessageDelivered(messageID)
}

func (a *API) SetCriteriaForMissingMessageVerification(peerInfo peer.AddrInfo, filters types.ChatFilters) {
	a.transport.SetCriteriaForMissingMessageVerification(peerInfo, adapters.ToTransportFilters(filters))
}

func (a *API) GetActiveStorenode() peer.AddrInfo {
	return a.transport.GetActiveStorenode()
}

func (a *API) DisconnectActiveStorenode(ctx context.Context, backoffReason time.Duration, shouldCycle bool) {
	a.transport.DisconnectActiveStorenode(ctx, backoffReason, shouldCycle)
}

func (a *API) OnStorenodeChanged() <-chan peer.ID {
	return a.transport.OnStorenodeChanged()
}

func (a *API) OnStorenodeNotWorking() <-chan struct{} {
	return a.transport.OnStorenodeNotWorking()
}

func (a *API) OnStorenodeAvailable() <-chan peer.ID {
	return a.transport.OnStorenodeAvailable()
}

func (a *API) WaitForAvailableStoreNode(ctx context.Context) bool {
	return a.transport.WaitForAvailableStoreNode(ctx)
}

func (a *API) IsStorenodeAvailable(peerID peer.ID) bool {
	return a.transport.IsStorenodeAvailable(peerID)
}

func (a *API) PerformStorenodeTask(fn func() error, opts ...history.StorenodeTaskOption) error {
	return a.transport.PerformStorenodeTask(fn, opts...)
}

func (a *API) ProcessMailserverBatch(
	ctx context.Context,
	batch types.StoreNodeBatch,
	storenode peer.AddrInfo,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	return a.transport.ProcessMailserverBatch(ctx, *adapters.ToWakuBatch(&batch), storenode, pageLimit, shouldProcessNextPage, processEnvelopes)
}

func (a *API) SetStorenodeConfigProvider(c history.StorenodeConfigProvider) {
	a.transport.SetStorenodeConfigProvider(c)
}

func ToContentTopic(s string) []byte {
	return transport.ToTopic(s)
}

func PartitionedTopic(publicKey *ecdsa.PublicKey) string {
	return transport.PartitionedTopic(publicKey)
}

func ContactCodeTopic(publicKey *ecdsa.PublicKey) string {
	return transport.ContactCodeTopic(publicKey)
}

func CommunityShardInfoTopic(communityID string) string {
	return transport.CommunityShardInfoTopic(communityID)
}

func CommunityShardInfoTopicPrefix() string {
	return transport.CommunityShardInfoTopicPrefix()
}
