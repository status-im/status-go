package node

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
)

// All general log messages in this package should be routed through this logger.
var logger = log.New("package", "status-go/metrics/node")

// SubscribeServerEvents subscribes to server and listens to
// PeerEventTypeAdd and PeerEventTypeDrop events.
func SubscribeServerEvents(ctx context.Context, node *node.Node) error {
	server := node.Server()

	if server == nil {
		return errors.New("server is unavailable")
	}

	ch := make(chan *p2p.PeerEvent, server.MaxPeers)
	subscription := server.SubscribeEvents(ch)
	defer subscription.Unsubscribe()

	logger.Debug("Subscribed to server events")

	for {
		select {
		case event := <-ch:
			if isAddDropPeerEvent(event.Type) {
				if err := updateNodeMetrics(node, event.Type); err != nil {
					log.Error("failed to update node metrics", "err", err)
				}
			}
		case err := <-subscription.Err():
			if err != nil {
				logger.Error("Subscription failed", "err", err)
			}
			return err
		case <-ctx.Done():
			return nil
		}
	}
}

func isAddDropPeerEvent(eventType p2p.PeerEventType) bool {
	return eventType == p2p.PeerEventTypeAdd || eventType == p2p.PeerEventTypeDrop
}
