package messaging

import (
	"context"
	"crypto/ecdsa"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/connection"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/messaging/transport"
	wakutypes "github.com/status-im/status-go/waku/types"
	"github.com/waku-org/go-waku/waku/v2/api/history"
)

type API struct {
	transport *transport.Transport
}

func NewAPI(transport *transport.Transport) *API {
	return &API{
		transport: transport,
	}
}

func (a *API) InitChats(chats ChatsToInitialize, publicKeys []*ecdsa.PublicKey) error {
	_, err := a.transport.InitFilters(chats.toFilters(), publicKeys)
	return err
}

func (a *API) InitPublicChats(chats ChatsToInitialize) (ChatFilters, error) {
	filters, err := a.transport.InitPublicFilters(chats.toFilters())
	return fromTransportFilters(filters), err
}

func (a *API) InitCommunities(communities CommunitiesToInitialize) (ChatFilters, error) {
	filters, err := a.transport.InitCommunityFilters(communities.toFilters())
	return fromTransportFilters(filters), err
}

func (a *API) ChatFilters() ChatFilters {
	return fromTransportFilters(a.transport.Filters())
}

func (a *API) ChatFilterByChatID(chatID string) *ChatFilter {
	return fromTransportFilter(a.transport.FilterByChatID(chatID))
}

func (a *API) ChatFilterByTopic(topic []byte) *ChatFilter {
	return fromTransportFilter(a.transport.FilterByTopic(topic))
}

func (a *API) ChatFiltersByIdentities(identities []string) ChatFilters {
	return fromTransportFilters(a.transport.FiltersByIdentities(identities))
}

func (a *API) RemoveFilters(filters ChatFilters) error {
	return a.transport.RemoveFilters(filters.toTransportFilters())
}

func (a *API) RemoveFilterByChatID(chatID string) (*ChatFilter, error) {
	filter, err := a.transport.RemoveFilterByChatID(chatID)
	if err != nil {
		return nil, err
	}
	return fromTransportFilter(filter), nil
}

func (a *API) ResetChatFilters(ctx context.Context) error {
	return a.transport.ResetFilters(ctx)
}

func (a *API) ProcessNegotiatedSecret(secret types.NegotiatedSecret) (*ChatFilter, error) {
	filter, err := a.transport.ProcessNegotiatedSecret(secret)
	if err != nil {
		return nil, err
	}
	return fromTransportFilter(filter), nil
}

func (a *API) JoinPublicChat(chatID string) (*ChatFilter, error) {
	filter, err := a.transport.JoinPublic(chatID)
	if err != nil {
		return nil, err
	}
	return fromTransportFilter(filter), nil
}

func (a *API) JoinPrivateChat(publicKey *ecdsa.PublicKey) (*ChatFilter, error) {
	filter, err := a.transport.JoinPrivate(publicKey)
	if err != nil {
		return nil, err
	}
	return fromTransportFilter(filter), nil
}

func (a *API) JoinGroupChat(publicKeys []*ecdsa.PublicKey) (ChatFilters, error) {
	filters, err := a.transport.JoinGroup(publicKeys)
	if err != nil {
		return nil, err
	}
	return fromTransportFilters(filters), nil
}

func (a *API) GetStats() wakutypes.StatsSummary {
	return a.transport.GetStats()
}

func (a *API) RetrieveRawAll() (map[ChatFilter][]*wakutypes.Message, error) {
	filters, err := a.transport.RetrieveRawAll()
	if err != nil {
		return nil, err
	}
	chatFilters := make(map[ChatFilter][]*wakutypes.Message)
	for k, v := range filters {
		chatFilters[*fromTransportFilter(&k)] = v
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

func (a *API) PersonalTopicFilter() *ChatFilter {
	return fromTransportFilter(a.transport.PersonalTopicFilter())
}

func (a *API) LoadKeyFilters(key *ecdsa.PrivateKey) (*ChatFilter, error) {
	filter, err := a.transport.LoadKeyFilters(key)
	if err != nil {
		return nil, err
	}
	return fromTransportFilter(filter), nil
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

func (a *API) SetEnvelopeEventsHandler(handler EnvelopeEventsHandler) error {
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

func (a *API) SetCriteriaForMissingMessageVerification(peerInfo peer.AddrInfo, filters ChatFilters) {
	a.transport.SetCriteriaForMissingMessageVerification(peerInfo, filters.toTransportFilters())
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
	batch wakutypes.MailserverBatch,
	storenode peer.AddrInfo,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	return a.transport.ProcessMailserverBatch(ctx, batch, storenode, pageLimit, shouldProcessNextPage, processEnvelopes)
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
