package node

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/status-im/status-go/geth/log"
)

// SubscribeServerEvents subscribes to server and listens to
// PeerEventTypeAdd and PeerEventTypeDrop events.
func SubscribeServerEvents(ctx context.Context, node *node.Node) error {
	server := node.Server()
	if server == nil {
		return errors.New("server is unavailable")
	}

	ch := make(chan *p2p.PeerEvent)
	subscription := server.SubscribeEvents(ch)

	log.Debug("Subscribed to server events")

	for {
		select {
		case event := <-ch:
			if isAddDropPeerEvent(event.Type) {
				updateNodeInfo(node)
				updateNodePeers(node)
			}
		case err := <-subscription.Err():
			if err != nil {
				log.Warn("Subscription failed", "err", err.Error())
			}
			subscription.Unsubscribe()
			return nil
		case <-ctx.Done():
			subscription.Unsubscribe()
			return nil
		}
	}
}

func isAddDropPeerEvent(eventType p2p.PeerEventType) bool {
	return eventType == p2p.PeerEventTypeAdd || eventType == p2p.PeerEventTypeDrop
}
