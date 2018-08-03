package helpers

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

var (
	// ErrNoRunningNode node is not running.
	ErrNoRunningNode = errors.New("there is no running node")
	// ErrEmptyPeerURL provided peer URL is empty
	ErrEmptyPeerURL = errors.New("empty peer url")
)

// waitForPeer waits for a peer to be added
func waitForPeer(p *p2p.Server, u string, e p2p.PeerEventType, t time.Duration, subscribed chan struct{}) error {
	if p == nil {
		return ErrNoRunningNode
	}
	if u == "" {
		return ErrEmptyPeerURL
	}
	parsedPeer, err := discover.ParseNode(u)
	if err != nil {
		return err
	}

	ch := make(chan *p2p.PeerEvent)
	subscription := p.SubscribeEvents(ch)
	defer subscription.Unsubscribe()
	close(subscribed)

	for {
		select {
		case ev := <-ch:
			if ev.Type == e && ev.Peer == parsedPeer.ID {
				return nil
			}
		case err := <-subscription.Err():
			if err != nil {
				return err
			}
		case <-time.After(t):
			return errors.New("wait for peer: timeout")
		}
	}
}

// WaitForPeerAsync waits for a peer to be added asynchronously
func WaitForPeerAsync(p *p2p.Server, u string, e p2p.PeerEventType, t time.Duration) <-chan error {
	subscribed := make(chan struct{})
	errCh := make(chan error)
	go func() {
		errCh <- waitForPeer(p, u, e, t, subscribed)
	}()
	<-subscribed
	return errCh
}

// PeerFromEvent returns a NodeID of the first peer that match the criteria.
func PeerFromEvent(events <-chan *p2p.PeerEvent, etype p2p.PeerEventType) (nodeID discover.NodeID, err error) {
	select {
	case ev := <-events:
		if ev.Type == etype {
			return ev.Peer, nil
		}
		return nodeID, fmt.Errorf("invalid event '%s' when expected '%s'", ev.Type, etype)
	case <-time.After(10 * time.Second):
		return nodeID, fmt.Errorf("timed out")
	}
}
