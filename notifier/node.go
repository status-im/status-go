package notifier

import (
	"encoding/json"
	stdlog "log"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/params"
)

// NewStatusBackend : setup a status nodebackend with an active whisper service
func NewStatusBackend(dataDir string, clusterConfigFile string, networkID uint64) *api.StatusBackend {
	config, err := makeNodeConfig(dataDir, clusterConfigFile, networkID)
	if err != nil {
		stdlog.Fatalf("Making config failed %s", err)
		return nil
	}

	var logger = log.New("package", "status-go/cmd/notifier")

	backend := api.NewStatusBackend()
	cfg, err := loadNodeConfig(config)
	if err != nil {
		logger.Error("Node start failed", "error", err)
		return nil
	}
	err = backend.StartNode(cfg)
	if err != nil {
		logger.Error("Node start failed", "error", err)
		return nil
	}
	if backend == nil {
		logger.Error("Node start failed", "error", "Nil backend")
		return nil
	}

	return backend
}

// makeNodeConfig : generates the config for a whisper based node
func makeNodeConfig(dataDir string, clusterConfigFile string, networkID uint64) (*params.NodeConfig, error) {
	nodeConfig, err := params.NewNodeConfig(dataDir, clusterConfigFile, networkID)
	if err != nil {
		return nil, err
	}

	nodeConfig.LightEthConfig.Enabled = false

	whisperConfig := nodeConfig.WhisperConfig
	whisperConfig.Enabled = true
	whisperConfig.EnableMailServer = false
	whisperConfig.LightClient = false
	whisperConfig.MinimumPoW = params.WhisperMinimumPoW
	whisperConfig.TTL = params.WhisperTTL

	// TODO(adriacidre) remove this as shouldn't be needed since introduction of loadNodeConfig
	// nodeConfig.UpstreamConfig.Enabled = true
	// nodeConfig.UpstreamConfig.URL = "https://ropsten.infura.io/z6GCTmjdP3FETEJmMBI4"

	return nodeConfig, nil
}

func loadNodeConfig(config *params.NodeConfig) (*params.NodeConfig, error) {
	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	return params.LoadNodeConfig(string(cfg))
}
