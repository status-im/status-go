package e2e

import (
	"encoding/json"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/t/utils"
	"github.com/status-im/status-go/transactions"
	"github.com/status-im/status-go/waku"
)

// StatusNodeTestSuite defines a test suite with StatusNode.
type StatusNodeTestSuite struct {
	suite.Suite
	StatusNode *node.StatusNode
}

var (
	// All general log messages in this package should be routed through this logger.
	logger = log.New("package", "status-go/t/e2e")

	// Settings for testing
	networks = json.RawMessage("{}")
	settings = accounts.Settings{
		Address:           types.HexToAddress("0xaC540f3745Ff2964AFC1171a5A0DD726d1F6B472"),
		CurrentNetwork:    "mainnet_rpc",
		DappsAddress:      types.HexToAddress("0xa1300f99fDF7346986CbC766903245087394ecd0"),
		EIP1581Address:    types.HexToAddress("0xa1DDDE9235a541d1344550d969715CF43982de9f"),
		InstallationID:    "d3efcff6-cffa-560e-a547-21d3858cbc51",
		KeyUID:            "0x4e8129f3edfc004875be17bf468a784098a9f69b53c095be1f52deff286935ab",
		LatestDerivedPath: 0,
		Name:              "Jittery Cornflowerblue Kingbird",
		Networks:          &networks,
		PhotoPath:         "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAjklEQVR4nOzXwQmFMBAAUZXUYh32ZB32ZB02sxYQQSZGsod55/91WFgSS0RM+SyjA56ZRZhFmEWYRRT6h+M6G16zrxv6fdJpmUWYRbxsYr13dKfanpN0WmYRZhGzXz6AWYRZRIfbaX26fT9Jk07LLMIsosPt9I/dTDotswizCG+nhFmEWYRZhFnEHQAA///z1CFkYamgfQAAAABJRU5ErkJggg==",
		PreviewPrivacy:    false,
		PublicKey:         "0x04211fe0f69772ecf7eb0b5bfc7678672508a9fb01f2d699096f0d59ef7fe1a0cb1e648a80190db1c0f5f088872444d846f2956d0bd84069f3f9f69335af852ac0",
		SigningPhrase:     "yurt joey vibe",
		WalletRootAddress: types.HexToAddress("0xaB591fd819F86D0A6a2EF2Bcb94f77807a7De1a6")}
)

func Init() {
	utils.Init()
	for id := range utils.TestNetworkNames {
		nodeConfig, err := utils.MakeTestNodeConfig(id)
		if err != nil {
			panic(err)
		}

		err = importTestAccounts(nodeConfig.KeyStoreDir)
		if err != nil {
			panic(err)
		}
	}
}

// StartTestNode initiazes a StatusNode instances with configuration retrieved
// from the test config.
func (s *StatusNodeTestSuite) StartTestNode(opts ...TestNodeOption) {
	nodeConfig, err := utils.MakeTestNodeConfig(utils.GetNetworkID())
	s.NoError(err)

	// Apply any options altering node config.
	for i := range opts {
		opts[i](nodeConfig)
	}

	// import account keys
	s.NoError(importTestAccounts(nodeConfig.KeyStoreDir))

	s.False(s.StatusNode.IsRunning())
	s.NoError(s.StatusNode.Start(nodeConfig, nil))
	s.True(s.StatusNode.IsRunning())
}

// StopTestNode attempts to stop initialized StatusNode.
func (s *StatusNodeTestSuite) StopTestNode() {
	s.NotNil(s.StatusNode)
	s.True(s.StatusNode.IsRunning())
	s.NoError(s.StatusNode.Stop())
	s.False(s.StatusNode.IsRunning())
}

// BackendTestSuite is a test suite with api.GethStatusBackend initialized
// and a few utility methods to start and stop node or get various services.
type BackendTestSuite struct {
	suite.Suite
	Backend *api.GethStatusBackend
}

// SetupTest initializes Backend.
func (s *BackendTestSuite) SetupTest() {
	s.Backend = api.NewGethStatusBackend()
	s.NotNil(s.Backend)
}

// TearDownTest cleans up the packages state.
func (s *BackendTestSuite) TearDownTest() {
	signal.ResetDefaultNodeNotificationHandler()
}

// StartTestBackend imports some keys and starts a node.
func (s *BackendTestSuite) StartTestBackend(opts ...TestNodeOption) {
	nodeConfig, err := utils.MakeTestNodeConfig(utils.GetNetworkID())
	s.Require().NoError(err)

	// Apply any options altering node config.
	for i := range opts {
		opts[i](nodeConfig)
	}
	s.NoError(s.Backend.AccountManager().InitKeystore(nodeConfig.KeyStoreDir))
	// import account keys
	s.NoError(importTestAccounts(nodeConfig.KeyStoreDir))

	// start node
	s.False(s.Backend.IsNodeRunning())
	s.Require().NoError(s.Backend.StartNode(nodeConfig))
	s.True(s.Backend.IsNodeRunning())
}

func (s *BackendTestSuite) StartTestBackendWithAccount(account multiaccounts.Account, password string, subaccs []accounts.Account, opts ...TestNodeOption) {
	nodeConfig, err := utils.MakeTestNodeConfig(utils.GetNetworkID())
	s.Require().NoError(err)

	// Apply any options altering node config.
	for i := range opts {
		opts[i](nodeConfig)
	}

	keystoreDir := nodeConfig.KeyStoreDir
	dataDir := nodeConfig.DataDir
	nodeConfig.KeyStoreDir = "keystore"
	nodeConfig.DataDir = "/"
	// accounts must be imported before keystore is initialized
	s.NoError(importTestAccounts(keystoreDir))
	s.Backend.UpdateRootDataDir(dataDir)
	s.NoError(s.Backend.OpenAccounts())
	s.NoError(s.Backend.AccountManager().InitKeystore(keystoreDir))

	s.Require().NoError(s.Backend.StartNodeWithAccountAndConfig(account, password, settings, nodeConfig, subaccs))
}

func (s *BackendTestSuite) LogoutAndStop() {
	s.NoError(s.Backend.Logout())
	s.StopTestBackend()
}

// StopTestBackend stops the node.
func (s *BackendTestSuite) StopTestBackend() {
	s.True(s.Backend.IsNodeRunning())
	s.NoError(s.Backend.StopNode())
	s.False(s.Backend.IsNodeRunning())
}

// RestartTestNode restarts a currently running node.
func (s *BackendTestSuite) RestartTestNode() {
	s.True(s.Backend.IsNodeRunning())
	s.Require().NoError(s.Backend.RestartNode())
	s.True(s.Backend.IsNodeRunning())
}

// WakuService returns a reference to the Waku service.
func (s *BackendTestSuite) WakuService() *waku.Waku {
	wakuService, err := s.Backend.StatusNode().WakuService()
	s.NoError(err)
	s.NotNil(wakuService)

	return wakuService
}

// Transactor returns a reference to the Transactor.
func (s *BackendTestSuite) Transactor() *transactions.Transactor {
	return s.Backend.Transactor()
}

func importTestAccounts(keyStoreDir string) (err error) {
	logger.Debug("Import accounts to", "dir", keyStoreDir)

	err = utils.ImportTestAccount(keyStoreDir, utils.GetAccount1PKFile())
	if err != nil {
		return
	}

	return utils.ImportTestAccount(keyStoreDir, utils.GetAccount2PKFile())
}
