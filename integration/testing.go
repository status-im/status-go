package integration

import (
	"context"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/testing"
)

// TestNodeOption is a callback passed to StartTestNode which alters its config.
type TestNodeOption func(config *params.NodeConfig)

// WithUpstream returns TestNodeOption which enabled UpstreamConfig.
func WithUpstream(url string) TestNodeOption {
	return func(config *params.NodeConfig) {
		config.UpstreamConfig.Enabled = true
		config.UpstreamConfig.URL = url
	}
}

// MakeTestNodeConfig defines a function to return a giving params.NodeConfig
// where specific network addresses are assigned based on provieded network id.
func MakeTestNodeConfig(networkID int) (*params.NodeConfig, error) {
	testDir := filepath.Join(TestDataDir, TestNetworkNames[networkID])

	if runtime.GOOS == "windows" {
		testDir = filepath.ToSlash(testDir)
	}

	// run tests with "INFO" log level only
	// when `go test` invoked with `-v` flag
	errorLevel := "ERROR"
	if testing.Verbose() {
		errorLevel = "INFO"
	}

	configJSON := `{
		"NetworkId": ` + strconv.Itoa(networkID) + `,
		"DataDir": "` + testDir + `",
		"HTTPPort": ` + strconv.Itoa(TestConfig.Node.HTTPPort) + `,
		"WSPort": ` + strconv.Itoa(TestConfig.Node.WSPort) + `,
		"LogLevel": "` + errorLevel + `"
	}`

	nodeConfig, err := params.LoadNodeConfig(configJSON)
	if err != nil {
		return nil, err
	}
	return nodeConfig, nil
}

// FirstBlockHash validates Attach operation for the NodeManager.
func FirstBlockHash(nodeManager common.NodeManager) (string, error) {
	// obtain RPC client for running node
	runningNode, err := nodeManager.Node()
	if err != nil {
		return "", err
	}

	rpcClient, err := runningNode.Attach()
	if err != nil {
		return "", err
	}

	// get first block
	var firstBlock struct {
		Hash gethcommon.Hash `json:"hash"`
	}

	err = rpcClient.CallContext(context.Background(), &firstBlock, "eth_getBlockByNumber", "0x0", true)
	if err != nil {
		return "", err
	}

	return firstBlock.Hash.Hex(), nil
}
