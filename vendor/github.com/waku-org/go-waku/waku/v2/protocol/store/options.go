package store

import (
	"errors"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"github.com/waku-org/go-waku/waku/v2/protocol"
)

type Parameters struct {
	SelectedPeer      peer.ID
	PeerAddr          multiaddr.Multiaddr
	PeerSelectionType peermanager.PeerSelection
	PreferredPeers    peer.IDSlice
	RequestID         []byte
	Cursor            []byte
	PageLimit         uint64
	Forward           bool
	IncludeData       bool
}

type RequestOption func(*Parameters) error

// WithPeer is an option used to specify the peerID to request the message history.
// Note that this option is mutually exclusive to WithPeerAddr, only one of them can be used.
func WithPeer(p peer.ID) RequestOption {
	return func(params *Parameters) error {
		params.SelectedPeer = p
		if params.PeerAddr != nil {
			return errors.New("WithPeer and WithPeerAddr options are mutually exclusive")
		}
		return nil
	}
}

// WithPeerAddr is an option used to specify a peerAddress to request the message history.
// This new peer will be added to peerStore.
// Note that this option is mutually exclusive to WithPeerAddr, only one of them can be used.
func WithPeerAddr(pAddr multiaddr.Multiaddr) RequestOption {
	return func(params *Parameters) error {
		params.PeerAddr = pAddr
		if params.SelectedPeer != "" {
			return errors.New("WithPeerAddr and WithPeer options are mutually exclusive")
		}
		return nil
	}
}

// WithAutomaticPeerSelection is an option used to randomly select a peer from the peer store
// to request the message history. If a list of specific peers is passed, the peer will be chosen
// from that list assuming it supports the chosen protocol, otherwise it will chose a peer
// from the node peerstore
// Note: This option is avaiable only with peerManager
func WithAutomaticPeerSelection(fromThesePeers ...peer.ID) RequestOption {
	return func(params *Parameters) error {
		params.PeerSelectionType = peermanager.Automatic
		params.PreferredPeers = fromThesePeers
		return nil
	}
}

// WithFastestPeerSelection is an option used to select a peer from the peer store
// with the lowest ping. If a list of specific peers is passed, the peer will be chosen
// from that list assuming it supports the chosen protocol, otherwise it will chose a peer
// from the node peerstore
// Note: This option is avaiable only with peerManager
func WithFastestPeerSelection(fromThesePeers ...peer.ID) RequestOption {
	return func(params *Parameters) error {
		params.PeerSelectionType = peermanager.LowestRTT
		return nil
	}
}

// WithRequestID is an option to set a specific request ID to be used when
// creating a store request
func WithRequestID(requestID []byte) RequestOption {
	return func(params *Parameters) error {
		params.RequestID = requestID
		return nil
	}
}

// WithAutomaticRequestID is an option to automatically generate a request ID
// when creating a store request
func WithAutomaticRequestID() RequestOption {
	return func(params *Parameters) error {
		params.RequestID = protocol.GenerateRequestID()
		return nil
	}
}

func WithCursor(cursor []byte) RequestOption {
	return func(params *Parameters) error {
		params.Cursor = cursor
		return nil
	}
}

// WithPaging is an option used to specify the order and maximum number of records to return
func WithPaging(forward bool, limit uint64) RequestOption {
	return func(params *Parameters) error {
		params.Forward = forward
		params.PageLimit = limit
		return nil
	}
}

// IncludeData is an option used to indicate whether you want to return the message content or not
func IncludeData(v bool) RequestOption {
	return func(params *Parameters) error {
		params.IncludeData = v
		return nil
	}
}

// Default options to be used when querying a store node for results
func DefaultOptions() []RequestOption {
	return []RequestOption{
		WithAutomaticRequestID(),
		WithAutomaticPeerSelection(),
		WithPaging(true, DefaultPageSize),
		IncludeData(true),
	}
}
