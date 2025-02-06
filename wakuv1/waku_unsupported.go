package wakuv1

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/status-im/status-go/waku/types"
	"github.com/waku-org/go-waku/waku/v2/api/history"
)

var notAvailableStr = "not available in WakuV1"
var notAvailableError = errors.New(notAvailableStr)

func (w *Waku) PeerCount() int {
	return -1
}

func (w *Waku) StartDiscV5() error {
	return notAvailableError
}

func (w *Waku) StopDiscV5() error {
	return notAvailableError
}

func (w *Waku) AddStorePeer(address multiaddr.Multiaddr) (peer.ID, error) {
	return "", notAvailableError
}

func (w *Waku) SubscribeToPubsubTopic(topic string, optPublicKey *ecdsa.PublicKey) error {
	// not available in WakuV1
	return notAvailableError
}

func (w *Waku) UnsubscribeFromPubsubTopic(topic string) error {
	// not available in WakuV1
	return notAvailableError
}

func (w *Waku) RetrievePubsubTopicKey(topic string) (*ecdsa.PrivateKey, error) {
	// not available in WakuV1
	return nil, notAvailableError
}

func (w *Waku) StorePubsubTopicKey(topic string, privKey *ecdsa.PrivateKey) error {
	// not available in WakuV1
	return notAvailableError
}

func (w *Waku) RemovePubsubTopicKey(topic string) error {
	// not available in WakuV1
	return notAvailableError
}

func (w *Waku) AddRelayPeer(address multiaddr.Multiaddr) (peer.ID, error) {
	return "", notAvailableError
}

func (w *Waku) DialPeer(address multiaddr.Multiaddr) error {
	return notAvailableError
}

func (w *Waku) DialPeerByID(peerID peer.ID) error {
	return notAvailableError
}

func (w *Waku) ListenAddresses() ([]multiaddr.Multiaddr, error) {
	return nil, notAvailableError
}

func (w *Waku) RelayPeersByTopic(topic string) (*types.PeerList, error) {
	return nil, notAvailableError
}

func (w *Waku) ENR() (*enode.Node, error) {
	return nil, notAvailableError
}

func (w *Waku) DropPeer(peerID peer.ID) error {
	return notAvailableError
}

func (w *Waku) SubscribeToConnStatusChanges() (*types.ConnStatusSubscription, error) {
	return nil, notAvailableError
}

func (w *Waku) SetCriteriaForMissingMessageVerification(peerID peer.ID, pubsubTopic string, contentTopics []types.TopicType) error {
	return notAvailableError
}

func (w *Waku) Peers() types.PeerStats {
	return types.PeerStats{}
}

func (w *Waku) ConfirmMessageDelivered(hashes []common.Hash) {
}

func (w *Waku) PeerID() peer.ID {
	panic(notAvailableStr)
}

func (w *Waku) GetActiveStorenode() peer.ID {
	panic(notAvailableStr)
}

func (w *Waku) OnStorenodeChanged() <-chan peer.ID {
	panic(notAvailableStr)
}

func (w *Waku) OnStorenodeNotWorking() <-chan struct{} {
	panic(notAvailableStr)
}

func (w *Waku) OnStorenodeAvailable() <-chan peer.ID {
	panic(notAvailableStr)
}

func (w *Waku) WaitForAvailableStoreNode(ctx context.Context) bool {
	return false
}

func (w *Waku) SetStorenodeConfigProvider(c history.StorenodeConfigProvider) {
	panic(notAvailableStr)
}

func (w *Waku) ProcessMailserverBatch(
	ctx context.Context,
	batch types.MailserverBatch,
	storenodeID peer.ID,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	return notAvailableError
}

func (w *Waku) IsStorenodeAvailable(peerID peer.ID) bool {
	panic(notAvailableStr)
}

func (w *Waku) PerformStorenodeTask(fn func() error, opts ...history.StorenodeTaskOption) error {
	panic(notAvailableStr)
}

func (w *Waku) DisconnectActiveStorenode(ctx context.Context, backoff time.Duration, shouldCycle bool) {
	panic(notAvailableStr)
}
