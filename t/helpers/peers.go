package helpers

import (
	"errors"
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

type peer interface {
	Server() *p2p.Server
	IsRunning() bool
}

// waitForPeer waits for a peer to be added
func waitForPeer(p peer, u string, e p2p.PeerEventType, t time.Duration, subscribed chan struct{}) error {
	if !p.IsRunning() {
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
	subscription := p.Server().SubscribeEvents(ch)
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
func WaitForPeerAsync(p peer, u string, e p2p.PeerEventType, t time.Duration) <-chan error {
	subscribed := make(chan struct{})
	errCh := make(chan error)
	go func() {
		errCh <- waitForPeer(p, u, e, t, subscribed)
	}()
	<-subscribed
	return errCh
}
