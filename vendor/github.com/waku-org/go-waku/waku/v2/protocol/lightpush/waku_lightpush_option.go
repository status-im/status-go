package lightpush

import (
	"errors"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type LightpushParameters struct {
	limitR rate.Limit
	limitB int
}

type Option func(*LightpushParameters)

// WithRateLimiter is an option used to specify a rate limiter for requests received in lightpush protocol
func WithRateLimiter(r rate.Limit, b int) Option {
	return func(params *LightpushParameters) {
		params.limitR = r
		params.limitB = b
	}
}

func DefaultLightpushOptions() []Option {
	return []Option{
		WithRateLimiter(1, 1),
	}
}

type lightPushRequestParameters struct {
	host              host.Host
	peerAddr          multiaddr.Multiaddr
	selectedPeers     peer.IDSlice
	maxPeers          int
	peerSelectionType peermanager.PeerSelection
	preferredPeers    peer.IDSlice
	requestID         []byte
	pm                *peermanager.PeerManager
	log               *zap.Logger
	pubsubTopic       string
}

// RequestOption is the type of options accepted when performing LightPush protocol requests
type RequestOption func(*lightPushRequestParameters) error

func WithMaxPeers(num int) RequestOption {
	return func(params *lightPushRequestParameters) error {
		params.maxPeers = num
		return nil
	}
}

// WithPeer is an option used to specify the peerID to push a waku message to
func WithPeer(p peer.ID) RequestOption {
	return func(params *lightPushRequestParameters) error {
		params.selectedPeers = append(params.selectedPeers, p)
		if params.peerAddr != nil {
			return errors.New("peerAddr and peerId options are mutually exclusive")
		}
		return nil
	}
}

// WithPeerAddr is an option used to specify a peerAddress
// This new peer will be added to peerStore.
// Note that this option is mutually exclusive to WithPeerAddr, only one of them can be used.
func WithPeerAddr(pAddr multiaddr.Multiaddr) RequestOption {
	return func(params *lightPushRequestParameters) error {
		params.peerAddr = pAddr
		if len(params.selectedPeers) != 0 {
			return errors.New("peerAddr and peerId options are mutually exclusive")
		}
		return nil
	}
}

// WithAutomaticPeerSelection is an option used to randomly select a peer from the peer store
// to push a waku message to. If a list of specific peers is passed, the peer will be chosen
// from that list assuming it supports the chosen protocol, otherwise it will chose a peer
// from the node peerstore
func WithAutomaticPeerSelection(fromThesePeers ...peer.ID) RequestOption {
	return func(params *lightPushRequestParameters) error {
		params.peerSelectionType = peermanager.Automatic
		params.preferredPeers = fromThesePeers
		return nil
	}
}

// WithFastestPeerSelection is an option used to select a peer from the peer store
// with the lowest ping. If a list of specific peers is passed, the peer will be chosen
// from that list assuming it supports the chosen protocol, otherwise it will chose a peer
// from the node peerstore
func WithFastestPeerSelection(fromThesePeers ...peer.ID) RequestOption {
	return func(params *lightPushRequestParameters) error {
		params.peerSelectionType = peermanager.LowestRTT
		return nil
	}
}

// WithPubSubTopic is used to specify the pubsub topic on which a WakuMessage will be broadcasted
func WithPubSubTopic(pubsubTopic string) RequestOption {
	return func(params *lightPushRequestParameters) error {
		params.pubsubTopic = pubsubTopic
		return nil
	}
}

// WithDefaultPubsubTopic is used to indicate that the message should be broadcasted in the default pubsub topic
func WithDefaultPubsubTopic() RequestOption {
	return func(params *lightPushRequestParameters) error {
		params.pubsubTopic = relay.DefaultWakuTopic
		return nil
	}
}

// WithRequestID is an option to set a specific request ID to be used when
// publishing a message
func WithRequestID(requestID []byte) RequestOption {
	return func(params *lightPushRequestParameters) error {
		params.requestID = requestID
		return nil
	}
}

// WithAutomaticRequestID is an option to automatically generate a request ID
// when publishing a message
func WithAutomaticRequestID() RequestOption {
	return func(params *lightPushRequestParameters) error {
		params.requestID = protocol.GenerateRequestID()
		return nil
	}
}

// DefaultOptions are the default options to be used when using the lightpush protocol
func DefaultOptions(host host.Host) []RequestOption {
	return []RequestOption{
		WithAutomaticPeerSelection(),
		WithMaxPeers(1), //keeping default as 2 for status use-case
	}
}
