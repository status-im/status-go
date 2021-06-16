package node

import (
	"crypto/ecdsa"
	"net"

	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"github.com/status-im/go-waku/waku/v2/protocol/store"
	wakurelay "github.com/status-im/go-wakurelay-pubsub"
)

type WakuNodeParameters struct {
	multiAddr  []ma.Multiaddr
	privKey    *crypto.PrivKey
	libP2POpts []libp2p.Option

	enableRelay bool
	wOpts       []wakurelay.Option

	enableStore bool
	storeMsgs   bool
	store       *store.WakuStore

	enableLightPush bool
}

type WakuNodeOption func(*WakuNodeParameters) error

// WithHostAddress is a WakuNodeOption that configures libp2p to listen on a list of net endpoint addresses
func WithHostAddress(hostAddr []net.Addr) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		var multiAddresses []ma.Multiaddr
		for _, addr := range hostAddr {
			hostAddrMA, err := manet.FromNetAddr(addr)
			if err != nil {
				return err
			}
			multiAddresses = append(multiAddresses, hostAddrMA)
		}

		params.multiAddr = append(params.multiAddr, multiAddresses...)

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
		privk := crypto.PrivKey((*crypto.Secp256k1PrivateKey)(privKey))
		params.privKey = &privk
		return nil
	}
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
func WithWakuRelay(opts ...wakurelay.Option) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableRelay = true
		params.wOpts = opts
		return nil
	}
}

// WithWakuStore enables the Waku V2 Store protocol and if the messages should
// be stored or not in a message provider
func WithWakuStore(shouldStoreMessages bool) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableStore = true
		params.storeMsgs = shouldStoreMessages
		params.store = store.NewWakuStore(shouldStoreMessages, nil)
		return nil
	}
}

// WithMessageProvider is a WakuNodeOption that sets the MessageProvider
// used to store and retrieve persisted messages
func WithMessageProvider(s store.MessageProvider) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		if params.store != nil {
			params.store.SetMsgProvider(s)
		} else {
			params.store = store.NewWakuStore(true, s)
		}
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

// Default options used in the libp2p node
var DefaultLibP2POptions = []libp2p.Option{
	libp2p.DefaultTransports,
	libp2p.NATPortMap(),       // Attempt to open ports using uPNP for NATed hosts.
	libp2p.EnableNATService(), // TODO: is this needed?)
	libp2p.ConnectionManager(connmgr.NewConnManager(200, 300, 0)),
}
