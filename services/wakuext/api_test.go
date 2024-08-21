package wakuext

import (
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/status-im/status-go/appdatabase"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/ext"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/waku"
	"github.com/status-im/status-go/walletdatabase"
)

func TestInitProtocol(t *testing.T) {
	config := params.NodeConfig{
		RootDataDir: t.TempDir(),
		ShhextConfig: params.ShhextConfig{
			InstallationID:          "2",
			PFSEnabled:              true,
			MailServerConfirmations: true,
			ConnectionTarget:        10,
		},
	}
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	require.NoError(t, err)

	waku := gethbridge.NewGethWakuWrapper(waku.New(nil, nil))
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	nodeWrapper := ext.NewTestNodeWrapper(nil, waku)
	service := New(config, nodeWrapper, nil, nil, db)

	appDB, cleanupDB, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "db.sql")
	defer func() { require.NoError(t, cleanupDB()) }()
	require.NoError(t, err)

	tmpfile, err := ioutil.TempFile("", "multi-accounts-tests-")
	require.NoError(t, err)
	multiAccounts, err := multiaccounts.InitializeDB(tmpfile.Name())
	require.NoError(t, err)

	acc := &multiaccounts.Account{KeyUID: "0xdeadbeef"}

	walletDB, cleanupWalletDB, err := helpers.SetupTestSQLDB(walletdatabase.DbInitializer{}, "db-wallet.sql")
	defer func() { require.NoError(t, cleanupWalletDB()) }()
	require.NoError(t, err)

	accountsFeed := &event.Feed{}

	err = service.InitProtocol("Test", privateKey, appDB, walletDB, nil, multiAccounts, acc, nil, nil, nil, nil, nil, zap.NewNop(), accountsFeed)
	require.NoError(t, err)
}

func TestShhExtSuite(t *testing.T) {
	suite.Run(t, new(ShhExtSuite))
}

type ShhExtSuite struct {
	suite.Suite

	dir      string
	nodes    []*node.Node
	wakus    []types.Waku
	services []*Service
}

func (s *ShhExtSuite) createAndAddNode() {
	idx := len(s.nodes)

	// create a node
	cfg := &node.Config{
		Name: strconv.Itoa(idx),
		P2P: p2p.Config{
			MaxPeers:    math.MaxInt32,
			NoDiscovery: true,
			ListenAddr:  ":0",
		},
		NoUSB: true,
	}
	stack, err := node.New(cfg)
	s.NoError(err)
	w := waku.New(nil, nil)
	stack.RegisterLifecycle(w)
	stack.RegisterAPIs(w.APIs())
	stack.RegisterProtocols(w.Protocols())
	s.NoError(err)

	// set up protocol
	config := params.NodeConfig{
		RootDataDir: s.dir,
		ShhextConfig: params.ShhextConfig{
			InstallationID:          "1",
			PFSEnabled:              true,
			MailServerConfirmations: true,
			ConnectionTarget:        10,
		},
	}
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	s.Require().NoError(err)
	nodeWrapper := ext.NewTestNodeWrapper(nil, gethbridge.NewGethWakuWrapper(w))
	service := New(config, nodeWrapper, nil, nil, db)

	appDB, cleanupDB, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, fmt.Sprintf("%d", idx))
	s.Require().NoError(err)
	defer func() { s.Require().NoError(cleanupDB()) }()

	tmpfile, err := ioutil.TempFile("", "multi-accounts-tests-")
	s.Require().NoError(err)

	multiAccounts, err := multiaccounts.InitializeDB(tmpfile.Name())
	s.Require().NoError(err)

	privateKey, err := crypto.GenerateKey()
	s.NoError(err)

	acc := &multiaccounts.Account{KeyUID: "0xdeadbeef"}

	walletDB, err := helpers.SetupTestMemorySQLDB(&walletdatabase.DbInitializer{})
	s.Require().NoError(err)

	accountsFeed := &event.Feed{}

	err = service.InitProtocol("Test", privateKey, appDB, walletDB, nil, multiAccounts, acc, nil, nil, nil, nil, nil, zap.NewNop(), accountsFeed)
	s.NoError(err)

	stack.RegisterLifecycle(service)
	stack.RegisterAPIs(service.APIs())
	stack.RegisterProtocols(service.Protocols())

	s.NoError(err)

	// start the node
	err = stack.Start()
	s.Require().NoError(err)

	// store references
	s.nodes = append(s.nodes, stack)
	s.wakus = append(s.wakus, gethbridge.NewGethWakuWrapper(w))
	s.services = append(s.services, service)
}

func (s *ShhExtSuite) SetupTest() {
	s.dir = s.T().TempDir()
}

func (s *ShhExtSuite) TearDownTest() {
	for _, n := range s.nodes {
		s.NoError(n.Close())
	}
	s.nodes = nil
	s.wakus = nil
	s.services = nil
}
