package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/params"
)

func main() {
	flag.Parse()

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

	// start debug server and collecting metrics
	if *statsEnabled {
		log.Info("Stats enabled", "addr", *statsAddr)

		go func() {
			metrics := newMetrics(backend)
			if err := startDebugServer(*statsAddr, metrics); err != nil {
				log.Error("Failed to start metrics server", "err", err.Error())
			}
		}()
	}

	ethNode, err := backend.NodeManager().Node()
	if err != nil {
		log.Crit("Getting node failed", "err", err.Error())
		os.Exit(1)
	}

	// wait till node has been stopped
	ethNode.Wait()
}

// printHeader prints command header
func printHeader(config *params.NodeConfig) {
	fmt.Println("Starting Whisper V5 node...")
	fmt.Printf("Config: %s\n", config.WhisperConfig)
}
