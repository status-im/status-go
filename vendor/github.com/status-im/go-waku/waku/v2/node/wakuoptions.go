package node

import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/config"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/multiformats/go-multiaddr"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	rendezvous "github.com/status-im/go-waku-rendezvous"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	"github.com/status-im/go-waku/waku/v2/protocol/store"
	"github.com/status-im/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

// Default clientId
const clientId string = "Go Waku v2 node"

// Default minRelayPeersToPublish
const defaultMinRelayPeersToPublish = 0

type WakuNodeParameters struct {
	hostAddr       *net.TCPAddr
	advertiseAddr  *net.IP
	multiAddr      []ma.Multiaddr
	addressFactory basichost.AddrsFactory
	privKey        *ecdsa.PrivateKey
	libP2POpts     []libp2p.Option

	logger *zap.SugaredLogger

	enableRelay      bool
	enableFilter     bool
	isFilterFullNode bool
	filterOpts       []filter.Option
	wOpts            []pubsub.Option

	minRelayPeersToPublish int

	enableStore     bool
	shouldResume    bool
	storeMsgs       bool
	messageProvider store.MessageProvider
	maxMessages     int
	maxDuration     time.Duration

	swapMode                int
	swapDisconnectThreshold int
	swapPaymentThreshold    int

	enableRendezvous       bool
	enableRendezvousServer bool
	rendevousStorage       rendezvous.Storage
	rendezvousOpts         []pubsub.DiscoverOpt

	enableDiscV5     bool
	udpPort          int
	discV5bootnodes  []*enode.Node
	discV5Opts       []pubsub.DiscoverOpt
	discV5autoUpdate bool

	keepAliveInterval time.Duration

	enableLightPush bool

	connStatusC chan ConnStatus

	storeFactory storeFactory
}

type WakuNodeOption func(*WakuNodeParameters) error

// Default options used in the libp2p node
var DefaultWakuNodeOptions = []WakuNodeOption{
	WithLogger(utils.Logger()),
	WithWakuRelay(),
}

// MultiAddresses return the list of multiaddresses configured in the node
func (w WakuNodeParameters) MultiAddresses() []ma.Multiaddr {
	return w.multiAddr
}

// Identity returns a libp2p option containing the identity used by the node
func (w WakuNodeParameters) Identity() config.Option {
	return libp2p.Identity(*w.GetPrivKey())
}

func (w WakuNodeParameters) AddressFactory() basichost.AddrsFactory {
	return w.addressFactory
}

// WithLogger is a WakuNodeOption that adds a custom logger
func WithLogger(l *zap.Logger) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.logger = l.Sugar()
		return nil
	}
}

// WithHostAddress is a WakuNodeOption that configures libp2p to listen on a specific address
func WithHostAddress(hostAddr *net.TCPAddr) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.hostAddr = hostAddr
		hostAddrMA, err := manet.FromNetAddr(hostAddr)
		if err != nil {
			return err
		}
		params.multiAddr = append(params.multiAddr, hostAddrMA)

		return nil
	}
}

// WithAdvertiseAddress is a WakuNodeOption that allows overriding the address used in the waku node with custom value
func WithAdvertiseAddress(address *net.TCPAddr, enableWS bool, wsPort int) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.advertiseAddr = &address.IP

		advertiseAddress, err := manet.FromNetAddr(address)
		if err != nil {
			return err
		}

		params.addressFactory = func([]ma.Multiaddr) []ma.Multiaddr {
			var result []multiaddr.Multiaddr
			result = append(result, advertiseAddress)
			if enableWS {
				wsMa, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/ws", address, wsPort))
				result = append(result, wsMa)
			}
			return result
		}
		return nil
	}
}

// WithMultiaddress is a WakuNodeOption that configures libp2p to listen on a list of multiaddresses
func WithMultiaddress(addresses []ma.Multiaddr) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.multiAddr = append(params.multiAddr, addresses...)
		return nil
	}
}

// WithPrivateKey is used to set an ECDSA private key in a libp2p node
func WithPrivateKey(privKey *ecdsa.PrivateKey) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.privKey = privKey
		return nil
	}
}

func (w *WakuNodeParameters) GetPrivKey() *crypto.PrivKey {
	privKey := crypto.PrivKey((*crypto.Secp256k1PrivateKey)(w.privKey))
	return &privKey
}

// WithLibP2POptions is a WakuNodeOption used to configure the libp2p node.
// This can potentially override any libp2p config that was set with other
// WakuNodeOption
func WithLibP2POptions(opts ...libp2p.Option) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.libP2POpts = opts
		return nil
	}
}

// WithWakuRelay enables the Waku V2 Relay protocol. This WakuNodeOption
// accepts a list of WakuRelay gossipsub option to setup the protocol
func WithWakuRelay(opts ...pubsub.Option) WakuNodeOption {
	return WithWakuRelayAndMinPeers(defaultMinRelayPeersToPublish, opts...)
}

// WithWakuRelayAndMinPeers enables the Waku V2 Relay protocol. This WakuNodeOption
// accepts a min peers require to publish and a list of WakuRelay gossipsub option to setup the protocol
func WithWakuRelayAndMinPeers(minRelayPeersToPublish int, opts ...pubsub.Option) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableRelay = true
		params.wOpts = opts
		params.minRelayPeersToPublish = minRelayPeersToPublish
		return nil
	}
}

// WithDiscoveryV5 is a WakuOption used to enable DiscV5 peer discovery
func WithDiscoveryV5(udpPort int, bootnodes []*enode.Node, autoUpdate bool, discoverOpts ...pubsub.DiscoverOpt) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableDiscV5 = true
		params.udpPort = udpPort
		params.discV5bootnodes = bootnodes
		params.discV5Opts = discoverOpts
		params.discV5autoUpdate = autoUpdate
		return nil
	}
}

// WithRendezvous is a WakuOption used to enable go-waku-rendezvous discovery.
// It accepts an optional list of DiscoveryOpt options
func WithRendezvous(discoverOpts ...pubsub.DiscoverOpt) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableRendezvous = true
		params.rendezvousOpts = discoverOpts
		return nil
	}
}

// WithRendezvousServer is a WakuOption used to set the node as a rendezvous
// point, using an specific storage for the peer information
func WithRendezvousServer(storage rendezvous.Storage) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableRendezvousServer = true
		params.rendevousStorage = storage
		return nil
	}
}

// WithWakuFilter enables the Waku V2 Filter protocol. This WakuNodeOption
// accepts a list of WakuFilter gossipsub options to setup the protocol
func WithWakuFilter(fullNode bool, filterOpts ...filter.Option) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableFilter = true
		params.isFilterFullNode = fullNode
		params.filterOpts = filterOpts
		return nil
	}
}

// WithWakuStore enables the Waku V2 Store protocol and if the messages should
// be stored or not in a message provider
func WithWakuStore(shouldStoreMessages bool, shouldResume bool) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableStore = true
		params.storeMsgs = shouldStoreMessages
		params.shouldResume = shouldResume
		return nil
	}
}

func WithWakuStoreFactory(factory storeFactory) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.storeFactory = factory

		return nil
	}
}

// WithWakuSwap set the option of the Waku V2 Swap protocol
func WithWakuSwap(mode int, disconnectThreshold, paymentThreshold int) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.swapMode = mode
		params.swapDisconnectThreshold = disconnectThreshold
		params.swapPaymentThreshold = paymentThreshold
		return nil
	}
}

// WithWakuStoreAndRetentionPolicy enables the Waku V2 Store protocol, storing them in an optional message provider
// applying an specific retention policy
func WithWakuStoreAndRetentionPolicy(shouldResume bool, maxDuration time.Duration, maxMessages int) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableStore = true
		params.storeMsgs = true
		params.shouldResume = shouldResume
		params.maxDuration = maxDuration
		params.maxMessages = maxMessages
		return nil
	}
}

// WithMessageProvider is a WakuNodeOption that sets the MessageProvider
// used to store and retrieve persisted messages
func WithMessageProvider(s store.MessageProvider) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.messageProvider = s
		return nil
	}
}

// WithLightPush is a WakuNodeOption that enables the lightpush protocol
func WithLightPush() WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableLightPush = true
		return nil
	}
}

// WithKeepAlive is a WakuNodeOption used to set the interval of time when
// each peer will be ping to keep the TCP connection alive
func WithKeepAlive(t time.Duration) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.keepAliveInterval = t
		return nil
	}
}

// WithConnectionStatusChannel is a WakuNodeOption used to set a channel where the
// connection status changes will be pushed to. It's useful to identify when peer
// connections and disconnections occur
func WithConnectionStatusChannel(connStatus chan ConnStatus) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.connStatusC = connStatus
		return nil
	}
}

// Default options used in the libp2p node
var DefaultLibP2POptions = []libp2p.Option{
	libp2p.DefaultTransports,
	libp2p.UserAgent(clientId),
	libp2p.EnableNATService(), // TODO: is this needed?)
	libp2p.ConnectionManager(connmgr.NewConnManager(200, 300, 0)),
}
