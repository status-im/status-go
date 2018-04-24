package main

import (
	stdlog "log"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
)

// statusNode : setup a status node with an active whisper service
func statusNode() *node.StatusNode {
	config, err := makeNodeConfig()
	if err != nil {
		stdlog.Fatalf("Making config failed %s", err)
		return nil
	}

	var logger = log.New("package", "status-go/cmd/statusd")

	backend := api.NewStatusBackend()
	err = backend.StartNode(config)
	if err != nil {
		logger.Error("Node start failed", "error", err)
		return nil
	}
	if backend == nil {
		logger.Error("Node start failed", "error", "Nil backend")
		return nil
	}

	return backend.StatusNode()
}

// makeNodeConfig : generates the config for a whisper based node
func makeNodeConfig() (*params.NodeConfig, error) {
	devMode := !*prodMode
	nodeConfig, err := params.NewNodeConfig(*dataDir, *clusterConfigFile, uint64(*networkID), devMode)
	if err != nil {
		return nil, err
	}

	whisperConfig := nodeConfig.WhisperConfig
	whisperConfig.Enabled = true
	whisperConfig.EnableMailServer = false
	whisperConfig.LightClient = false
	whisperConfig.MinimumPoW = params.WhisperMinimumPoW
	whisperConfig.TTL = params.WhisperTTL

	return nodeConfig, nil
}
