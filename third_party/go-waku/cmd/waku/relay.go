package main

import (
	"context"
	"sync"
	"time"

	"github.com/waku-org/go-waku/waku/v2/node"
	wprotocol "github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/rendezvous"
)

func handleRelayTopics(ctx context.Context, wg *sync.WaitGroup, wakuNode *node.WakuNode, pubSubTopicMap map[string][]string) error {
	for nodeTopic, cTopics := range pubSubTopicMap {
		nodeTopic := nodeTopic
		_, err := wakuNode.Relay().Subscribe(ctx, wprotocol.NewContentFilter(nodeTopic, cTopics...), relay.WithoutConsumer())
		if err != nil {
			return err
		}

		if len(options.Rendezvous.Nodes) != 0 {
			// Register the node in rendezvous point
			iter := rendezvous.NewRendezvousPointIterator(options.Rendezvous.Nodes)

			wg.Add(1)
			go func(nodeTopic string) {
				t := time.NewTicker(rendezvous.RegisterDefaultTTL)
				defer t.Stop()
				defer wg.Done()

				for {
					select {
					case <-ctx.Done():
						return
					case <-t.C:
						// Register in rendezvous points periodically
						wakuNode.Rendezvous().RegisterWithNamespace(ctx, nodeTopic, iter.RendezvousPoints())
					}
				}
			}(nodeTopic)

			wg.Add(1)
			go func(nodeTopic string) {
				defer wg.Done()
				desiredOutDegree := wakuNode.Relay().Params().D
				t := time.NewTicker(7 * time.Second)
				defer t.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-t.C:
						peerCnt := len(wakuNode.Relay().PubSub().ListPeers(nodeTopic))
						peersToFind := desiredOutDegree - peerCnt
						if peersToFind <= 0 {
							continue
						}

						rp := <-iter.Next(ctx)
						if rp == nil {
							continue
						}
						ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
						wakuNode.Rendezvous().DiscoverWithNamespace(ctx, nodeTopic, rp, peersToFind)
						cancel()
					}
				}
			}(nodeTopic)

		}
	}

	// Protected topics
	for _, protectedTopic := range options.Relay.ProtectedTopics {
		if err := wakuNode.Relay().AddSignedTopicValidator(protectedTopic.Topic, protectedTopic.PublicKey); err != nil {
			return nonRecoverErrorMsg("could not add signed topic validator: %w", err)
		}
	}

	return nil
}
