package node

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
)

// All general log messages in this package should be routed through this logger.
var logger = logutils.ZapLogger().Named("metrics.node")

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
				// We start a goroutine here because updateNodeMetrics
				// is calling a method on the p2p server, which
				// blocks until the server is available:
				// https://github.com/status-im/status-go/blob/e60f425b45d00d3880b42fdd77b460ec465a9f55/vendor/github.com/ethereum/go-ethereum/p2p/server.go#L301
				// https://github.com/status-im/status-go/blob/e60f425b45d00d3880b42fdd77b460ec465a9f55/vendor/github.com/ethereum/go-ethereum/p2p/server.go#L746
				// If there's back-pressure on the peer event feed
				// https://github.com/status-im/status-go/blob/e60f425b45d00d3880b42fdd77b460ec465a9f55/vendor/github.com/ethereum/go-ethereum/p2p/server.go#L783
				// The event channel above might become full while updateNodeMetrics
				// is called, which means is never consumed, the server blocks on publishing on
				// it, and the two will deadlock (server waits for the channel above to be consumed,
				// this code waits for the server to respond to peerCount, which is in the same
				// event loop).
				// Calling it in a different go-routine will allow this code to keep
				// processing peer added events, therefore the server will not lock and
				// keep processing requests.
				go func() {
					defer common.LogOnPanic()
					if err := updateNodeMetrics(node, event.Type); err != nil {
						logger.Error("failed to update node metrics", zap.Error(err))
					}
				}()
			}
		case err := <-subscription.Err():
			if err != nil {
				logger.Error("Subscription failed", zap.Error(err))
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
