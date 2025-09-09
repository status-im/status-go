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
	ethtypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/messaging/adapters"
	"github.com/status-im/status-go/messaging/common"
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
	_, err := a.core.transport.InitFilters(adapters.ChatsToInitializeToTransport(chats), publicKeys)
	return err
}

func (a *API) InitPublicChats(chats types.ChatsToInitialize) (types.ChatFilters, error) {
	filters, err := a.core.transport.InitPublicFilters(adapters.ChatsToInitializeToTransport(chats))
	return adapters.FromTransportFilters(filters), err
}

func (a *API) InitCommunities(communities types.CommunitiesToInitialize) (types.ChatFilters, error) {
	filters, err := a.core.transport.InitCommunityFilters(adapters.CommunitiesToInitializeToTransport(communities))
	return adapters.FromTransportFilters(filters), err
}

func (a *API) ChatFilters() types.ChatFilters {
	return adapters.FromTransportFilters(a.core.transport.Filters())
}

func (a *API) ChatFilterByChatID(chatID string) *types.ChatFilter {
	return adapters.FromTransportFilter(a.core.transport.FilterByChatID(chatID))
}

func (a *API) ChatFilterByTopic(topic []byte) *types.ChatFilter {
	return adapters.FromTransportFilter(a.core.transport.FilterByTopic(topic))
}

func (a *API) ChatFiltersByIdentities(identities []string) types.ChatFilters {
	return adapters.FromTransportFilters(a.core.transport.FiltersByIdentities(identities))
}

func (a *API) RemoveFilters(filters types.ChatFilters) error {
	return a.core.transport.RemoveFilters(adapters.ToTransportFilters(filters))
}

func (a *API) RemoveFilterByChatID(chatID string) (*types.ChatFilter, error) {
	filter, err := a.core.transport.RemoveFilterByChatID(chatID)
	if err != nil {
		return nil, err
	}
	return adapters.FromTransportFilter(filter), nil
}

func (a *API) UpdateFilterPriority(chatID string, priority uint64) error {
	transportFilter := a.core.transport.FilterByChatID(chatID)
	if transportFilter == nil {
		return errors.New("filter not found")
	}

	transportFilter.Priority = priority

	return nil
}

func (a *API) UpdateFilterEphemerality(chatID string, ephemeral bool) error {
	transportFilter := a.core.transport.FilterByChatID(chatID)
	if transportFilter == nil {
		return errors.New("filter not found")
	}

	transportFilter.Ephemeral = ephemeral

	return nil
}

func (a *API) HandleSharedSecrets(secrets []*types.SharedSecret) error {
	for _, secret := range secrets {
		fSecret := ethtypes.NegotiatedSecret{
			PublicKey: secret.Identity,
			Key:       secret.Key,
		}
		_, err := a.core.transport.ProcessNegotiatedSecret(fSecret)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *API) JoinPublicChat(chatID string) (*types.ChatFilter, error) {
	return a.core.sender.JoinPublic(chatID)
}

func (a *API) JoinPrivateChat(publicKey *ecdsa.PublicKey) (*types.ChatFilter, error) {
	filter, err := a.core.transport.JoinPrivate(publicKey)
	if err != nil {
		return nil, err
	}
	return adapters.FromTransportFilter(filter), nil
}

func (a *API) JoinGroupChat(publicKeys []*ecdsa.PublicKey) (types.ChatFilters, error) {
	filters, err := a.core.transport.JoinGroup(publicKeys)
	if err != nil {
		return nil, err
	}
	return adapters.FromTransportFilters(filters), nil
}

func (a *API) GetStats() types.TransportStats {
	return adapters.FromWakuTransportStats(a.core.transport.GetStats())
}

func (a *API) RetrieveRawAll() (map[types.ChatFilter][]*types.ReceivedMessage, error) {
	filters, err := a.core.transport.RetrieveRawAll()
	if err != nil {
		return nil, err
	}
	chatFilters := make(map[types.ChatFilter][]*types.ReceivedMessage)
	for k, v := range filters {
		chatFilters[*adapters.FromTransportFilter(&k)] = adapters.FromWakuMessages(v)
	}
	return chatFilters, nil
}

func (a *API) SendPublic(ctx context.Context, chatName string, rawMessage types.RawMessage) ([]byte, error) {
	return a.core.sender.SendPublic(ctx, chatName, rawMessage)
}

func (a *API) SendPrivate(ctx context.Context, recipient *ecdsa.PublicKey, rawMessage *types.RawMessage) ([]byte, error) {
	return a.core.sender.SendPrivate(ctx, recipient, rawMessage)
}

func (a *API) SendGroup(ctx context.Context, recipients []*ecdsa.PublicKey, rawMessage types.RawMessage) ([]byte, error) {
	return a.core.sender.SendGroup(ctx, recipients, rawMessage)
}

func (a *API) SendCommunityMessage(ctx context.Context, rawMessage *types.RawMessage) ([]byte, error) {
	return a.core.sender.SendCommunityMessage(ctx, rawMessage)
}

func (a *API) HandleReceivedMessages(msg *types.ReceivedMessage) (*types.HandleMessageResponse, error) {
	return a.core.sender.HandleMessages(msg)
}

func (a *API) ValidateRawMessage(rawMessage *types.RawMessage) error {
	return a.core.sender.ValidateRawMessage(rawMessage)
}

func (a *API) GetKeysForGroup(groupID []byte) ([]*encryption.HashRatchetKeyCompatibility, error) {
	return a.core.sender.GetKeysForGroup(groupID)
}

func (a *API) GetCurrentKeyForGroup(groupID []byte) (*encryption.HashRatchetKeyCompatibility, error) {
	return a.core.sender.GetCurrentKeyForGroup(groupID)
}

func (a *API) SaveHashRatchetMessage(groupID []byte, keyID []byte, m *types.ReceivedMessage) error {
	return a.core.persistence.SaveHashRatchetMessage(groupID, keyID, m)
}

func (a *API) SendPubsubTopicKey(ctx context.Context, rawMessage *types.RawMessage) ([]byte, error) {
	return a.core.sender.SendPubsubTopicKey(ctx, rawMessage)
}

func (a *API) SendPairInstallation(ctx context.Context, recipient *ecdsa.PublicKey, rawMessage types.RawMessage) ([]byte, error) {
	return a.core.sender.SendPairInstallation(ctx, recipient, rawMessage)
}

func (a *API) GetEphemeralKey() (*ecdsa.PrivateKey, error) {
	return a.core.sender.GetEphemeralKey()
}

func (a *API) PersonalTopicFilter() *types.ChatFilter {
	return adapters.FromTransportFilter(a.core.transport.PersonalTopicFilter())
}

func (a *API) GetCurrentTime() uint64 {
	return a.core.transport.GetCurrentTime()
}

func (a *API) PeerCount() int {
	return a.core.transport.PeerCount()
}

func (a *API) Peers() types.PeerStats {
	return adapters.FromWakuPeerStats(a.core.transport.Peers())
}

func (a *API) PeerID() peer.ID {
	return a.core.waku.PeerID()
}

func (a *API) ConfirmMessagesProcessed(ids []string, timestamp uint64) error {
	return a.core.transport.ConfirmMessagesProcessed(ids, timestamp)
}

func (a *API) CleanMessagesProcessed(timestamp uint64) error {
	return a.core.transport.CleanMessagesProcessed(timestamp)
}

func (a *API) SetEnvelopeEventsHandler(handler types.EnvelopeEventsHandler) error {
	return a.core.transport.SetEnvelopeEventsHandler(handler)
}

func (a *API) ClearProcessedMessageIDsCache() error {
	return a.core.transport.ClearProcessedMessageIDsCache()
}

func (a *API) ListenAddresses() ([]multiaddr.Multiaddr, error) {
	return a.core.transport.ListenAddresses()
}

func (a *API) ENR() (*enode.Node, error) {
	return a.core.transport.ENR()
}

func (a *API) AddRelayPeer(address multiaddr.Multiaddr) (peer.ID, error) {
	return a.core.transport.AddRelayPeer(address)
}

func (a *API) DialPeer(address multiaddr.Multiaddr) error {
	return a.core.transport.DialPeer(address)
}

func (a *API) DialPeerByID(peerID peer.ID) error {
	return a.core.transport.DialPeerByID(peerID)
}

func (a *API) DropPeer(peerID peer.ID) error {
	return a.core.transport.DropPeer(peerID)
}

func (a *API) MarkP2PMessageAsProcessed(hash ethcommon.Hash) {
	a.core.transport.MarkP2PMessageAsProcessed(hash)
}

func (a *API) ConnectionChanged(state connection.State) {
	a.core.connectionChanged(state)
}

func (a *API) SubscribeToPubsubTopic(topic string, optPublicKey ...*ecdsa.PublicKey) error {
	return a.core.transport.SubscribeToPubsubTopic(topic, optPublicKey...)
}

func (a *API) UnsubscribeFromPubsubTopic(topic string) error {
	return a.core.transport.UnsubscribeFromPubsubTopic(topic)
}

func (a *API) StorePubsubTopicKey(topic string, privKey *ecdsa.PrivateKey) error {
	return a.core.transport.StorePubsubTopicKey(topic, privKey)
}

func (a *API) RetrievePubsubTopicKey(topic string) (*ecdsa.PrivateKey, error) {
	return a.core.transport.RetrievePubsubTopicKey(topic)
}

func (a *API) RemovePubsubTopicKey(topic string) error {
	return a.core.transport.RemovePubsubTopicKey(topic)
}

func (a *API) ConfirmMessageDelivered(messageID string) {
	a.core.transport.ConfirmMessageDelivered(messageID)
}

func (a *API) SetCriteriaForMissingMessageVerification(peerInfo peer.AddrInfo, filters types.ChatFilters) {
	a.core.transport.SetCriteriaForMissingMessageVerification(peerInfo, adapters.ToTransportFilters(filters))
}

func (a *API) GetActiveStorenode() peer.AddrInfo {
	return a.core.transport.GetActiveStorenode()
}

func (a *API) DisconnectActiveStorenode(ctx context.Context, backoffReason time.Duration, shouldCycle bool) {
	a.core.transport.DisconnectActiveStorenode(ctx, backoffReason, shouldCycle)
}

func (a *API) OnStorenodeChanged() <-chan peer.ID {
	return a.core.transport.OnStorenodeChanged()
}

func (a *API) OnStorenodeNotWorking() <-chan struct{} {
	return a.core.transport.OnStorenodeNotWorking()
}

func (a *API) OnStorenodeAvailable() <-chan peer.ID {
	return a.core.transport.OnStorenodeAvailable()
}

func (a *API) WaitForAvailableStoreNode(ctx context.Context) bool {
	return a.core.transport.WaitForAvailableStoreNode(ctx)
}

func (a *API) PerformStorenodeTask(fn func() error, opts ...history.StorenodeTaskOption) error {
	return a.core.transport.PerformStorenodeTask(fn, opts...)
}

func (a *API) ProcessMailserverBatch(
	ctx context.Context,
	batch types.StoreNodeBatch,
	storenode peer.AddrInfo,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	return a.core.transport.ProcessMailserverBatch(ctx, *adapters.ToWakuBatch(&batch), storenode, pageLimit, shouldProcessNextPage, processEnvelopes)
}

func (a *API) SetStorenodeConfigProvider(c history.StorenodeConfigProvider) {
	a.core.transport.SetStorenodeConfigProvider(c)
}

func (a *API) ResetDatasyncForPeer(publicKey *ecdsa.PublicKey, eventTime uint64) {
	a.core.resetDatasyncForPeer(publicKey, eventTime)
}

func (a *API) MetricsPushReceivedMessages(receivedMessages types.ReceivedMessages) {
	if a.core.wakumetrics != nil {
		a.core.wakumetrics.PushReceivedMessages(adapters.ToWakumetricsReceivedMessages(receivedMessages))
	}
}

func (a *API) GenerateHashRatchetKey(groupID []byte) error {
	return a.core.generateHashRatchetKey(groupID)
}

func (a *API) EncryptionSubscriptions() *types.EncryptionSubscriptions {
	return adapters.FromEncryptionSubscriptions(a.core.encryptor.Subscriptions())
}

func (a *API) GetAllHRKeysMarshaledV1(groupID []byte) ([]byte, error) {
	return a.core.encryptor.GetAllHRKeysMarshaledV1(groupID)
}

func (a *API) GetAllHRKeysMarshaledV2(groupID []byte) ([]byte, error) {
	return a.core.encryptor.GetAllHRKeysMarshaledV2(groupID)
}

func (a *API) EncryptWithHashRatchet(groupID []byte, payload []byte) ([]byte, []byte, uint32, error) {
	return a.core.encryptWithHashRatchet(groupID, payload)
}

func (a *API) DecryptWithHashRatchet(keyID []byte, seqNo uint32, payload []byte) ([]byte, error) {
	data, err := a.core.encryptor.DecryptWithHashRatchet(keyID, seqNo, payload)
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
	return a.core.encryptor.EncryptCommunityGrants(privateKey, recipientGrants)
}

func (a *API) DecryptCommunityGrant(myIdentityKey *ecdsa.PrivateKey, senderKey *ecdsa.PublicKey, grants map[uint32][]byte) ([]byte, error) {
	return a.core.encryptor.DecryptCommunityGrant(myIdentityKey, senderKey, grants)
}

func (a *API) HandleHashRatchetKeysPayload(groupID, encodedKeys []byte, myIdentityKey *ecdsa.PrivateKey, theirIdentityKey *ecdsa.PublicKey) error {
	_, err := a.core.encryptor.HandleHashRatchetKeysPayload(groupID, encodedKeys, myIdentityKey, theirIdentityKey)
	return err
}

func (a *API) HandleHashRatchetHeadersPayload(encodedHeaders [][]byte) error {
	return a.core.encryptor.HandleHashRatchetHeadersPayload(encodedHeaders)
}

func (a *API) AddInstallation(identity []byte, timestamp int64, installation *types.Installation, enabled bool) ([]*types.Installation, error) {
	allInstallations, err := a.core.encryptor.AddInstallation(identity, timestamp, adapters.ToEncryptionInstallation(installation), enabled)
	if err != nil {
		return nil, err
	}
	return adapters.FromEncryptionInstallations(allInstallations), nil
}

func (a *API) AddInstallations(identity []byte, timestamp int64, installations []*types.Installation, enabled bool) ([]*types.Installation, error) {
	allInstallations, err := a.core.encryptor.AddInstallations(identity, timestamp, adapters.ToEncryptionInstallations(installations), enabled)
	if err != nil {
		return nil, err
	}
	return adapters.FromEncryptionInstallations(allInstallations), nil
}

func (a *API) GetOurInstallations(myIdentityKey *ecdsa.PublicKey) ([]*types.Installation, error) {
	installations, err := a.core.encryptor.GetOurInstallations(myIdentityKey)
	if err != nil {
		return nil, err
	}
	return adapters.FromEncryptionInstallations(installations), nil
}

func (a *API) GetOurActiveInstallations(myIdentityKey *ecdsa.PublicKey) ([]*types.Installation, error) {
	installations, err := a.core.encryptor.GetOurActiveInstallations(myIdentityKey)
	if err != nil {
		return nil, err
	}
	return adapters.FromEncryptionInstallations(installations), nil
}

func (a *API) SetInstallationMetadata(myIdentityKey *ecdsa.PublicKey, installationID string, data *types.InstallationMetadata) error {
	return a.core.encryptor.SetInstallationMetadata(myIdentityKey, installationID, adapters.ToEncryptionInstallationMetadata(data))
}

func (a *API) SetInstallationName(myIdentityKey *ecdsa.PublicKey, installationID string, name string) error {
	return a.core.encryptor.SetInstallationName(myIdentityKey, installationID, name)
}

func (a *API) EnableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	return a.core.encryptor.EnableInstallation(myIdentityKey, installationID)
}

func (a *API) DisableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	return a.core.encryptor.DisableInstallation(myIdentityKey, installationID)
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

func CommunityShardInfoTopic(communityID string) string {
	return transport.CommunityShardInfoTopic(communityID)
}

func CommunityShardInfoTopicPrefix() string {
	return transport.CommunityShardInfoTopicPrefix()
}

func SetRekeyCompatibility(compatibility bool) {
	common.RekeyCompatibility = compatibility
}

func GenerateInstallationID() string {
	return uuid.New().String()
}
