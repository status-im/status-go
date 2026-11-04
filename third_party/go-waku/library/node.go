package library

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
	"unsafe"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ethereum/go-ethereum/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/waku-org/go-waku/waku/persistence"
	dbutils "github.com/waku-org/go-waku/waku/persistence/utils"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

// WakuInstance represents the state of the waku node
type WakuInstance struct {
	ctx    context.Context
	cancel context.CancelFunc
	ID     uint

	node                *node.WakuNode
	cb                  unsafe.Pointer
	mobileSignalHandler MobileSignalHandler

	relayTopics []string
}

var wakuInstances map[uint]*WakuInstance
var wakuInstancesMutex sync.RWMutex

var errWakuNodeNotReady = errors.New("not initialized")
var errWakuNodeAlreadyConfigured = errors.New("already configured")
var errWakuNodeNotConfigured = errors.New("not configured")
var errWakuAlreadyStarted = errors.New("already started")
var errWakuNodeNotStarted = errors.New("not started")

func init() {
	wakuInstances = make(map[uint]*WakuInstance)
}

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func Init() *WakuInstance {
	wakuInstancesMutex.Lock()
	defer wakuInstancesMutex.Unlock()

	id := uint(len(wakuInstances))
	instance := &WakuInstance{
		ID: id,
	}
	wakuInstances[id] = instance
	return instance
}

func GetInstance(id uint) (*WakuInstance, error) {
	wakuInstancesMutex.RLock()
	defer wakuInstancesMutex.RUnlock()

	instance, ok := wakuInstances[id]
	if !ok {
		return nil, errors.New("instance not found")
	}

	return instance, nil
}

type ValidationType int64

const (
	None          ValidationType = iota
	MustBeStarted ValidationType = iota
	MustBeStopped
	NotConfigured
)

func validateInstance(instance *WakuInstance, validationType ValidationType) error {
	if instance == nil {
		return errWakuNodeNotReady
	}

	switch validationType {
	case NotConfigured:
		if instance.node != nil {
			return errWakuNodeAlreadyConfigured
		}
	case MustBeStarted:
		if instance.node == nil {
			return errWakuNodeNotConfigured
		}
		if instance.ctx == nil {
			return errWakuNodeNotStarted
		}
	case MustBeStopped:
		if instance.node == nil {
			return errWakuNodeNotConfigured
		}
		if instance.ctx != nil {
			return errWakuAlreadyStarted
		}
	}

	return nil
}

// NewNode initializes a waku node. Receives a JSON string containing the configuration, and use default values for those config items not specified
func NewNode(instance *WakuInstance, configJSON string) error {
	if err := validateInstance(instance, NotConfigured); err != nil {
		return err
	}

	config, err := getConfig(configJSON)
	if err != nil {
		return err
	}

	hostAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", *config.Host, *config.Port))
	if err != nil {
		return err
	}

	var prvKey *ecdsa.PrivateKey
	if config.NodeKey != nil {
		prvKey, err = crypto.HexToECDSA(*config.NodeKey)
		if err != nil {
			return err
		}
	} else {
		key, err := randomHex(32)
		if err != nil {
			return err
		}
		prvKey, err = crypto.HexToECDSA(key)
		if err != nil {
			return err
		}
	}

	opts := []node.WakuNodeOption{
		node.WithPrivateKey(prvKey),
		node.WithHostAddress(hostAddr),
		node.WithKeepAlive(10*time.Second, time.Duration(*config.KeepAliveInterval)*time.Second),
		node.WithClusterID(uint16(config.ClusterID)),
		node.WithShards(config.Shards),
	}

	if *config.EnableRelay {
		var pubsubOpt []pubsub.Option
		if config.GossipSubParams != nil {
			params := getGossipSubParams(config.GossipSubParams)
			pubsubOpt = append(pubsubOpt, pubsub.WithGossipSubParams(params))
		}

		pubsubOpt = append(pubsubOpt, pubsub.WithSeenMessagesTTL(getSeenTTL(config)))

		opts = append(opts, node.WithWakuRelayAndMinPeers(*config.MinPeersToPublish, pubsubOpt...))
	}

	if config.DNS4DomainName != "" {
		opts = append(opts, node.WithDNS4Domain(config.DNS4DomainName))
	}

	if config.Websockets.Enabled {
		if config.Websockets.Secure {
			if config.DNS4DomainName == "" {
				utils.Logger().Warn("using secure websockets without a dns4 domain name might indicate a misconfiguration")
			}
			opts = append(opts, node.WithSecureWebsockets(config.Websockets.Host, *config.Websockets.Port, config.Websockets.CertPath, config.Websockets.KeyPath))
		} else {
			opts = append(opts, node.WithWebsockets(config.Websockets.Host, *config.Websockets.Port))

		}
	}

	opts = append(opts, node.WithWakuFilterLightNode())

	if *config.EnableStore {
		var db *sql.DB
		var migrationFn func(*sql.DB, *zap.Logger) error
		db, migrationFn, err = dbutils.ParseURL(*config.DatabaseURL, dbutils.DBSettings{}, utils.Logger())
		if err != nil {
			return err
		}
		opts = append(opts, node.WithWakuStore())
		dbStore, err := persistence.NewDBStore(prometheus.DefaultRegisterer, utils.Logger(),
			persistence.WithDB(db),
			persistence.WithMigrations(migrationFn),
			persistence.WithRetentionPolicy(*config.RetentionMaxMessages, time.Duration(*config.RetentionTimeSeconds)*time.Second),
		)
		if err != nil {
			return err
		}
		opts = append(opts, node.WithMessageProvider(dbStore))
	}

	if *config.EnableDiscV5 {
		discoveredNodes := node.GetNodesFromDNSDiscovery(utils.Logger(), context.TODO(), config.DNSDiscoveryNameServer, config.DNSDiscoveryURLs)
		discv5Opts, err := node.GetDiscv5Option(discoveredNodes, config.DiscV5BootstrapNodes, *config.DiscV5UDPPort, true)
		if err != nil {
			return err
		}
		opts = append(opts, discv5Opts)
	}

	instance.relayTopics = config.RelayTopics

	lvl, err := zapcore.ParseLevel(*config.LogLevel)
	if err != nil {
		return err
	}

	opts = append(opts, node.WithLogLevel(lvl))
	opts = append(opts, node.WithPrometheusRegisterer(prometheus.DefaultRegisterer))

	w, err := node.New(opts...)
	if err != nil {
		return err
	}

	instance.node = w

	return nil
}

func stop(instance *WakuInstance) {
	if instance.cancel != nil {
		instance.node.Stop()
		instance.cancel()
		instance.cancel = nil
		instance.ctx = nil
	}
}

// Start starts the waku node
func Start(instance *WakuInstance) error {
	if err := validateInstance(instance, MustBeStopped); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())

	if err := instance.node.Start(ctx); err != nil {
		cancel()
		return err
	}

	// Record started state only after node.Start() succeeds, so a failed Start()
	// leaves IsStarted() false and Free() will not Stop() a node that never ran.
	instance.ctx, instance.cancel = ctx, cancel

	if instance.node.DiscV5() != nil {
		if err := instance.node.DiscV5().Start(context.Background()); err != nil {
			stop(instance)
			return err
		}
	}

	for _, topic := range instance.relayTopics {
		err := relaySubscribe(instance, topic)
		if err != nil {
			stop(instance)
			return err
		}
	}

	return nil
}

// Stop stops a waku node
func Stop(instance *WakuInstance) error {
	if err := validateInstance(instance, None); err != nil {
		return err
	}

	stop(instance)

	return nil
}

// Free stops a waku instance and frees the resources allocated to a waku node
func Free(instance *WakuInstance) error {
	if err := validateInstance(instance, None); err != nil {
		return err
	}

	if instance.cancel != nil {
		stop(instance)
	}

	wakuInstancesMutex.Lock()
	defer wakuInstancesMutex.Unlock()
	delete(wakuInstances, instance.ID)

	return nil
}

// IsStarted is used to determine is a node is started or not
func IsStarted(instance *WakuInstance) bool {
	return instance != nil && instance.ctx != nil
}

// PeerID is used to obtain the peer ID of the waku node
func PeerID(instance *WakuInstance) (string, error) {
	if err := validateInstance(instance, MustBeStarted); err != nil {
		return "", err
	}

	return instance.node.ID(), nil
}

// ListenAddresses returns the multiaddresses the wakunode is listening to
func ListenAddresses(instance *WakuInstance) (string, error) {
	if err := validateInstance(instance, MustBeStarted); err != nil {
		return "", err
	}

	var addresses []string
	for _, addr := range instance.node.ListenAddresses() {
		addresses = append(addresses, addr.String())
	}

	return marshalJSON(addresses)
}

// AddPeer adds a node multiaddress and protocol to the wakunode peerstore
func AddPeer(instance *WakuInstance, address string, protocolID string) (string, error) {
	if err := validateInstance(instance, MustBeStarted); err != nil {
		return "", err
	}

	ma, err := multiaddr.NewMultiaddr(address)
	if err != nil {
		return "", err
	}

	peerID, err := instance.node.AddPeer([]multiaddr.Multiaddr{ma}, peerstore.Static, instance.relayTopics, libp2pProtocol.ID(protocolID))
	if err != nil {
		return "", err
	}

	return peerID.String(), nil
}

// Connect is used to connect to a peer at multiaddress. if ms > 0, cancel the function execution if it takes longer than N milliseconds
func Connect(instance *WakuInstance, address string, ms int) error {
	if err := validateInstance(instance, MustBeStarted); err != nil {
		return err
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(instance.ctx, time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = instance.ctx
	}

	return instance.node.DialPeer(ctx, address)
}

// ConnectPeerID is usedd to connect to a known peer by peerID. if ms > 0, cancel the function execution if it takes longer than N milliseconds
func ConnectPeerID(instance *WakuInstance, peerID string, ms int) error {
	if err := validateInstance(instance, MustBeStarted); err != nil {
		return err
	}

	var ctx context.Context
	var cancel context.CancelFunc

	pID, err := peer.Decode(peerID)
	if err != nil {
		return err
	}

	if ms > 0 {
		ctx, cancel = context.WithTimeout(instance.ctx, time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = instance.ctx
	}

	return instance.node.DialPeerByID(ctx, pID)
}

// Disconnect closes a connection to a known peer by peerID
func Disconnect(instance *WakuInstance, peerID string) error {
	if err := validateInstance(instance, MustBeStarted); err != nil {
		return err
	}

	pID, err := peer.Decode(peerID)
	if err != nil {
		return err
	}

	return instance.node.ClosePeerById(pID)
}

// PeerCnt returns the number of connected peers
func PeerCnt(instance *WakuInstance) (int, error) {
	if err := validateInstance(instance, MustBeStarted); err != nil {
		return 0, err
	}

	return instance.node.PeerCount(), nil
}

// ContentTopic creates a content topic string according to RFC 23
func ContentTopic(applicationName string, applicationVersion string, contentTopicName string, encoding string) string {
	contentTopic, _ := protocol.NewContentTopic(applicationName, applicationVersion, contentTopicName, encoding)
	return contentTopic.String()
}

// DefaultPubsubTopic returns the default pubsub topic used in waku2: /waku/2/default-waku/proto
func DefaultPubsubTopic() string {
	return protocol.DefaultPubsubTopic{}.String()
}

type subscriptionMsg struct {
	MessageID   string          `json:"messageId"`
	PubsubTopic string          `json:"pubsubTopic"`
	Message     *pb.WakuMessage `json:"wakuMessage"`
}

func toSubscriptionMessage(msg *protocol.Envelope) *subscriptionMsg {
	return &subscriptionMsg{
		MessageID:   msg.Hash().String(),
		PubsubTopic: msg.PubsubTopic(),
		Message:     msg.Message(),
	}
}

// Peers retrieves the list of peers known by the waku node
func Peers(instance *WakuInstance) (string, error) {
	if err := validateInstance(instance, MustBeStarted); err != nil {
		return "", err
	}

	peers, err := instance.node.Peers()
	if err != nil {
		return "", err
	}

	for _, p := range peers {
		addrs := []multiaddr.Multiaddr{}
		for i := range p.Addrs {
			// Filtering out SNI addresses due to https://github.com/waku-org/waku-rust-bindings/issues/66
			// TODO: once https://github.com/multiformats/rust-multiaddr/issues/88 is implemented, remove this
			_, err := p.Addrs[i].ValueForProtocol(multiaddr.P_SNI)
			if err != nil {
				addrs = append(addrs, p.Addrs[i])
			}
		}
		p.Addrs = addrs
	}

	return marshalJSON(peers)
}
