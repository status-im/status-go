package main

import (
	"encoding/json"
	"expvar"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/params"
)

var (
	peersCount = expvar.NewInt("peers_count")
	peersInfo  = expvar.NewString("peers_info")
)

func main() {
	flag.Parse()

	if *statsEnabled {
		log.Info("Stats enabled", "addr", *statsAddr)

		go func() {
			if err := http.ListenAndServe(*statsAddr, nil); err != nil {
				log.Error("Failed to start metrics server", "err", err.Error())
			}
		}()
	}

	config, err := makeNodeConfig()
	if err != nil {
		log.Error("Making config failed", "err", err.Error())
		os.Exit(1)
	}

	printHeader(config)

	if *injectAccounts {
		if err := LoadTestAccounts(config.DataDir); err != nil {
			log.Crit("Failed to load test accounts", "err", err.Error())
			os.Exit(1)
		}
	}

	backend := api.NewStatusBackend()
	started, err := backend.StartNode(config)
	if err != nil {
		log.Crit("Node start failed", "err", err.Error())
		os.Exit(1)
	}

	// wait till node is started
	<-started

	if *injectAccounts {
		if err := InjectTestAccounts(backend.NodeManager()); err != nil {
			log.Crit("Failed to inject accounts", "err", err.Error())
			os.Exit(1)
		}
	}

	ethNode, err := backend.NodeManager().Node()
	if err != nil {
		log.Crit("Getting node failed", "err", err.Error())
		os.Exit(1)
	}

	if *statsEnabled {
		if err := monitorPeers(ethNode); err != nil {
			log.Crit("Failed to monitor peers", "err", err.Error())
			os.Exit(1)
		}
	}

	// wait till node has been stopped
	ethNode.Wait()
}

// printHeader prints command header
func printHeader(config *params.NodeConfig) {
	fmt.Println("Starting Whisper V5 node...")
	fmt.Printf("Config: %s\n", config.WhisperConfig)
}

func monitorPeers(n *node.Node) error {
	server := n.Server()
	if server == nil {
		return fmt.Errorf("failed to get server")
	}

	peerEvents := make(chan *p2p.PeerEvent)
	subscription := server.SubscribeEvents(peerEvents)
	go func() {
		for {
			select {
			case ev := <-peerEvents:
				log.Info("Received an event", "type", ev.Type, "peerID", ev.Peer.String())

				// Update expvar only if a peer is added or dropped.
				// Other events are: message sent and message received.
				if ev.Type == p2p.PeerEventTypeAdd || ev.Type == p2p.PeerEventTypeDrop {
					peersCount.Set(int64(server.PeerCount()))

					data, err := json.Marshal(server.PeersInfo())
					if err != nil {
						log.Warn("Failed to marshal peers info", "err", err.Error())
					}
					peersInfo.Set(string(data))
				}
			case err := <-subscription.Err():
				log.Error("Subscription failed", "err", err.Error())
				subscription.Unsubscribe()
				return
			}
		}
	}()

	return nil
}
