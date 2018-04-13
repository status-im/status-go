package e2e

import (
	"context"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
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

// WithDataDir returns TestNodeOption that allows to set another data dir.
func WithDataDir(path string) TestNodeOption {
	return func(config *params.NodeConfig) {
		config.DataDir = path
	}
}

// FirstBlockHash validates Attach operation for the StatusNode.
func FirstBlockHash(statusNode *node.StatusNode) (string, error) {
	// obtain RPC client for running node
	runningNode, err := statusNode.GethNode()
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
