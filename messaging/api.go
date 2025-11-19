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

func (a *API) GetCurrentTime() uint64 {
	return a.core.stack.Transport.GetCurrentTime()
}

func (a *API) PeerCount() int {
	return a.core.stack.Transport.PeerCount()
}

func (a *API) Peers() types.PeerStats {
	return adapters.FromWakuPeerStats(a.core.stack.Transport.Peers())
}

func (a *API) PeerID() peer.ID {
	return a.core.waku.PeerID()
}

func (a *API) ConfirmMessagesProcessed(ids []string, timestamp uint64) error {
	return a.core.stack.Transport.ConfirmMessagesProcessed(ids, timestamp)
}

func (a *API) CleanMessagesProcessed(timestamp uint64) error {
	return a.core.stack.Transport.CleanMessagesProcessed(timestamp)
}

func (a *API) SetEnvelopeEventsHandler(handler types.EnvelopeEventsHandler) error {
	return a.core.stack.Transport.SetEnvelopeEventsHandler(handler)
}

func (a *API) ClearProcessedMessageIDsCache() error {
	return a.core.stack.Transport.ClearProcessedMessageIDsCache()
}

func (a *API) ListenAddresses() ([]multiaddr.Multiaddr, error) {
	return a.core.stack.Transport.ListenAddresses()
}

func (a *API) ENR() (*enode.Node, error) {
	return a.core.stack.Transport.ENR()
}

func (a *API) AddRelayPeer(address multiaddr.Multiaddr) (peer.ID, error) {
	return a.core.stack.Transport.AddRelayPeer(address)
}

func (a *API) DialPeer(address multiaddr.Multiaddr) error {
	return a.core.stack.Transport.DialPeer(address)
}

func (a *API) DialPeerByID(peerID peer.ID) error {
	return a.core.stack.Transport.DialPeerByID(peerID)
}

func (a *API) DropPeer(peerID peer.ID) error {
	return a.core.stack.Transport.DropPeer(peerID)
}

func (a *API) MarkP2PMessageAsProcessed(hash ethcommon.Hash) {
	a.core.stack.Transport.MarkP2PMessageAsProcessed(hash)
}

func (a *API) ConnectionChanged(state connection.State) {
	a.core.connectionChanged(state)
}

func (a *API) SubscribeToPubsubTopic(topic string) error {
	return a.core.stack.Transport.SubscribeToPubsubTopic(topic)
}

func (a *API) SetCriteriaForMissingMessageVerification(peerInfo peer.AddrInfo, filters types.ChatFilters) {
	a.core.stack.Transport.SetCriteriaForMissingMessageVerification(peerInfo, adapters.ToTransportFilters(filters))
}

func (a *API) GetActiveStorenode() peer.AddrInfo {
	return a.core.stack.Transport.GetActiveStorenode()
}

func (a *API) DisconnectActiveStorenode(ctx context.Context, backoffReason time.Duration, shouldCycle bool) {
	a.core.stack.Transport.DisconnectActiveStorenode(ctx, backoffReason, shouldCycle)
}

func (a *API) OnStorenodeChanged() <-chan peer.ID {
	return a.core.stack.Transport.OnStorenodeChanged()
}

func (a *API) OnStorenodeNotWorking() <-chan struct{} {
	return a.core.stack.Transport.OnStorenodeNotWorking()
}

func (a *API) OnStorenodeAvailable() <-chan peer.ID {
	return a.core.stack.Transport.OnStorenodeAvailable()
}

func (a *API) WaitForAvailableStoreNode(ctx context.Context) bool {
	return a.core.stack.Transport.WaitForAvailableStoreNode(ctx)
}

func (a *API) PerformStorenodeTask(fn func() error, opts ...history.StorenodeTaskOption) error {
	return a.core.stack.Transport.PerformStorenodeTask(fn, opts...)
}

func (a *API) ProcessMailserverBatch(
	ctx context.Context,
	batch types.StoreNodeBatch,
	storenode peer.AddrInfo,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	return a.core.stack.Transport.ProcessMailserverBatch(ctx, *adapters.ToWakuBatch(&batch), storenode, pageLimit, shouldProcessNextPage, processEnvelopes)
}

func (a *API) SetStorenodeConfigProvider(c history.StorenodeConfigProvider) {
	a.core.stack.Transport.SetStorenodeConfigProvider(c)
}

func (a *API) ReportUserOnline(publicKey *ecdsa.PublicKey, eventTime uint64) {
	a.core.stack.Reliability.ReportPeerOnline(publicKey, eventTime)
}

func (a *API) MetricsPushReceivedMessages(receivedMessages types.ReceivedMessages) {
	if a.core.wakumetrics != nil {
		a.core.wakumetrics.PushReceivedMessages(adapters.ToWakumetricsReceivedMessages(receivedMessages))
	}
}

func (a *API) MetricsPushSentMessage(pubsubTopic string, contentTopic string, messageType string, messageSize uint32) {
	if a.core.wakumetrics != nil {
		a.core.wakumetrics.PushRawMessageByType(pubsubTopic, contentTopic, messageType, messageSize)
	}
}

func (a *API) GenerateHashRatchetKey(groupID []byte) error {
	return a.core.generateHashRatchetKey(groupID)
}

func (a *API) EncryptionSubscriptions() *types.EncryptionSubscriptions {
	return adapters.FromEncryptionSubscriptions(a.core.stack.Encryption.Subscriptions())
}

func (a *API) GetAllHRKeysMarshaledV1(groupID []byte) ([]byte, error) {
	return a.core.stack.Encryption.GetAllHRKeysMarshaledV1(groupID)
}

func (a *API) GetAllHRKeysMarshaledV2(groupID []byte) ([]byte, error) {
	return a.core.stack.Encryption.GetAllHRKeysMarshaledV2(groupID)
}

func (a *API) EncryptWithHashRatchet(groupID []byte, payload []byte) ([]byte, []byte, uint32, error) {
	return a.core.encryptWithHashRatchet(groupID, payload)
}

func (a *API) DecryptWithHashRatchet(keyID []byte, seqNo uint32, payload []byte) ([]byte, error) {
	data, err := a.core.stack.Encryption.DecryptWithHashRatchet(keyID, seqNo, payload)
	if err == encryption.ErrNoRatchetKey {
		return nil, types.ErrNoRatchetKey
	}
	return data, err
}

func (a *API) BuildHashRatchetMessage(groupID []byte, payload []byte) ([]byte, error) {
	return a.core.buildHashRatchetMessage(groupID, payload)
}

func (a *API) DecryptMessage(myIdentityKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, data []byte) ([]byte, error) {
	data, err := a.core.decryptMessage(myIdentityKey, theirPublicKey, data)
	if err == encryption.ErrHashRatchetGroupIDNotFound {
		return nil, types.ErrHashRatchetGroupIDNotFound
	}
	return data, err
}

func (a *API) EncryptCommunityGrants(privateKey *ecdsa.PrivateKey, recipientGrants map[*ecdsa.PublicKey][]byte) (map[uint32][]byte, error) {
	return a.core.stack.Encryption.EncryptCommunityGrants(privateKey, recipientGrants)
}

func (a *API) DecryptCommunityGrant(myIdentityKey *ecdsa.PrivateKey, senderKey *ecdsa.PublicKey, grants map[uint32][]byte) ([]byte, error) {
	return a.core.stack.Encryption.DecryptCommunityGrant(myIdentityKey, senderKey, grants)
}

func (a *API) HandleHashRatchetKeysPayload(groupID, encodedKeys []byte, myIdentityKey *ecdsa.PrivateKey, theirIdentityKey *ecdsa.PublicKey) error {
	_, err := a.core.stack.Encryption.HandleHashRatchetKeysPayload(groupID, encodedKeys, myIdentityKey, theirIdentityKey)
	return err
}

func (a *API) HandleHashRatchetHeadersPayload(encodedHeaders [][]byte) error {
	return a.core.stack.Encryption.HandleHashRatchetHeadersPayload(encodedHeaders)
}

func (a *API) AddInstallation(identity []byte, timestamp int64, installation *types.Installation, enabled bool) ([]*types.Installation, error) {
	allInstallations, err := a.core.stack.Encryption.AddInstallation(identity, timestamp, adapters.ToEncryptionInstallation(installation), enabled)
	if err != nil {
		return nil, err
	}
	return adapters.FromEncryptionInstallations(allInstallations), nil
}

func (a *API) AddInstallations(identity []byte, timestamp int64, installations []*types.Installation, enabled bool) ([]*types.Installation, error) {
	allInstallations, err := a.core.stack.Encryption.AddInstallations(identity, timestamp, adapters.ToEncryptionInstallations(installations), enabled)
	if err != nil {
		return nil, err
	}
	return adapters.FromEncryptionInstallations(allInstallations), nil
}

func (a *API) GetOurInstallations(myIdentityKey *ecdsa.PublicKey) ([]*types.Installation, error) {
	installations, err := a.core.stack.Encryption.GetOurInstallations(myIdentityKey)
	if err != nil {
		return nil, err
	}
	return adapters.FromEncryptionInstallations(installations), nil
}

func (a *API) GetOurActiveInstallations(myIdentityKey *ecdsa.PublicKey) ([]*types.Installation, error) {
	installations, err := a.core.stack.Encryption.GetOurActiveInstallations(myIdentityKey)
	if err != nil {
		return nil, err
	}
	return adapters.FromEncryptionInstallations(installations), nil
}

func (a *API) SetInstallationMetadata(myIdentityKey *ecdsa.PublicKey, installationID string, data *types.InstallationMetadata) error {
	return a.core.stack.Encryption.SetInstallationMetadata(myIdentityKey, installationID, adapters.ToEncryptionInstallationMetadata(data))
}

func (a *API) SetInstallationName(myIdentityKey *ecdsa.PublicKey, installationID string, name string) error {
	return a.core.stack.Encryption.SetInstallationName(myIdentityKey, installationID, name)
}

func (a *API) EnableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	return a.core.stack.Encryption.EnableInstallation(myIdentityKey, installationID)
}

func (a *API) DisableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	return a.core.stack.Encryption.DisableInstallation(myIdentityKey, installationID)
}

func (a *API) Metrics() string {
	return a.core.metrics()
}

func ToContentTopic(s string) []byte {
	return transport.ToTopic(s)
}

func ContactCodeTopic(publicKey *ecdsa.PublicKey) string {
	return transport.ContactCodeTopic(publicKey)
}

func GenerateInstallationID() string {
	return uuid.New().String()
}
