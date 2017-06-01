package testing

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
	assertions "github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	TestConfig *common.TestConfig

	// RootDir is the main application directory
	RootDir string

	// TestDataDir is data directory used for tests
	TestDataDir string

	// TestNetworkNames network ID to name mapping
	TestNetworkNames = map[int]string{
		params.MainNetworkID:    "Mainnet",
		params.RopstenNetworkID: "Ropsten",
		params.RinkebyNetworkID: "Rinkeby",
	}
)

func init() {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// setup root directory
	RootDir = filepath.Dir(pwd)
	if strings.HasSuffix(RootDir, "geth") || strings.HasSuffix(RootDir, "cmd") { // we need to hop one more level
		RootDir = filepath.Join(RootDir, "..")
	}

	// setup auxiliary directories
	TestDataDir = filepath.Join(RootDir, ".ethereumtest")

	TestConfig, err = common.LoadTestConfig()
	if err != nil {
		panic(err)
	}
}

type BaseTestSuite struct {
	suite.Suite
	NodeManager common.NodeManager
}

func (s *BaseTestSuite) StartTestNode(networkID int) {
	require := s.Require()
	require.NotNil(s.NodeManager)

	nodeConfig, err := MakeTestNodeConfig(networkID)
	require.NoError(err)

	keyStoreDir := filepath.Join(TestDataDir, TestNetworkNames[networkID], "keystore")
	require.NoError(common.ImportTestAccount(keyStoreDir, "test-account1.pk"))
	require.NoError(common.ImportTestAccount(keyStoreDir, "test-account2.pk"))

	require.False(s.NodeManager.IsNodeRunning())
	nodeStarted, err := s.NodeManager.StartNode(nodeConfig)
	require.NoError(err)
	require.NotNil(nodeStarted)
	<-nodeStarted
	require.True(s.NodeManager.IsNodeRunning())
}

func (s *BaseTestSuite) StopTestNode() {
	require := s.Require()
	require.NotNil(s.NodeManager)
	require.True(s.NodeManager.IsNodeRunning())
	nodeStopped, err := s.NodeManager.StopNode()
	require.NoError(err)
	<-nodeStopped
	require.False(s.NodeManager.IsNodeRunning())
}

func FirstBlockHash(require *assertions.Assertions, nodeManager common.NodeManager, expectedHash string) {
	require.NotNil(nodeManager)

	var firstBlock struct {
		Hash gethcommon.Hash `json:"hash"`
	}

	// obtain RPC client for running node
	runningNode, err := nodeManager.Node()
	require.NoError(err)
	require.NotNil(runningNode)

	rpcClient, err := runningNode.Attach()
	require.NoError(err)

	// get first block
	err = rpcClient.CallContext(context.Background(), &firstBlock, "eth_getBlockByNumber", "0x0", true)
	require.NoError(err)

	require.Equal(expectedHash, firstBlock.Hash.Hex())
}

func MakeTestNodeConfig(networkID int) (*params.NodeConfig, error) {
	configJSON := `{
		"NetworkId": ` + strconv.Itoa(networkID) + `,
		"DataDir": "` + filepath.Join(TestDataDir, TestNetworkNames[networkID]) + `",
		"HTTPPort": ` + strconv.Itoa(TestConfig.Node.HTTPPort) + `,
		"WSPort": ` + strconv.Itoa(TestConfig.Node.WSPort) + `,
		"LoggerConfig": {
			"Enabled": true,
			"LogLevel": "ERROR",
			"LogToFile": false,
			"LogToRemote": true,
			"RemoteAPIKey": "` + params.LoggerRemoteAPIKey + `"
		}
	}`
	nodeConfig, err := params.LoadNodeConfig(configJSON)
	if err != nil {
		return nil, err
	}
	return nodeConfig, nil
}

// AttachLogger creates and attaches test logger
func AttachLogger(hostName, logLevel string) error {
	loggerConfig := &params.LoggerConfig{
		Enabled:             true,
		RemoteHostName:      hostName,
		Level:               logLevel,
		RemoteAPIKey:        params.LoggerRemoteAPIKey,
		RemoteFlushInterval: 1,
		RemoteBufferSize:    25,
		LogToRemote:         false,
		LogToStderr:         true,
		LogToFile:           false,
	}

	nodeLogger, err := common.NewLogger(loggerConfig)
	if err != nil {
		return err
	}
	nodeLogger.Attach()

	return nil
}

// LoadFromFile is useful for loading test data, from testdata/filename into a variable
// nolint: errcheck
func LoadFromFile(filename string) string {
	f, err := os.Open(filename)
	if err != nil {
		return ""
	}

	buf := bytes.NewBuffer(nil)
	io.Copy(buf, f)
	f.Close()

	return string(buf.Bytes())
}
