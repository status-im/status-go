package devtests

import (
	"crypto/ecdsa"
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/params"
	statusrpc "github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/t/devtests/miner"
	"github.com/stretchr/testify/suite"
)

// DevNodeSuite provides convenient wrapper for starting node with clique backend for mining.
type DevNodeSuite struct {
	suite.Suite

	Remote            *rpc.Client
	Eth               *ethclient.Client
	Local             *statusrpc.Client
	DevAccount        *ecdsa.PrivateKey
	DevAccountAddress common.Address

	dir     string
	backend *api.StatusBackend
	miner   *node.Node
}

// SetupTest creates clique node and status node with an rpc connection to a clique node.
func (s *DevNodeSuite) SetupTest() {
	account, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.DevAccount = account
	s.DevAccountAddress = crypto.PubkeyToAddress(account.PublicKey)
	s.miner, err = miner.NewDevNode(s.DevAccountAddress)
	s.Require().NoError(err)
	s.Require().NoError(miner.StartWithMiner(s.miner))

	s.dir, err = ioutil.TempDir("", "devtests-")
	s.Require().NoError(err)
	config, err := params.NewNodeConfig(
		s.dir,
		1337,
	)
	s.Require().NoError(err)
	config.WhisperConfig.Enabled = false
	config.LightEthConfig.Enabled = false
	config.UpstreamConfig.Enabled = true
	config.UpstreamConfig.URL = s.miner.IPCEndpoint()
	s.backend = api.NewStatusBackend()
	s.Require().NoError(s.backend.StartNode(config))

	s.Remote, err = s.miner.Attach()
	s.Require().NoError(err)
	s.Eth = ethclient.NewClient(s.Remote)
	s.Local = s.backend.StatusNode().RPCClient()
}

// TearDownTest stops status node and clique node.
func (s *DevNodeSuite) TearDownTest() {
	if s.miner != nil {
		s.Require().NoError(s.miner.Stop())
		s.miner = nil
	}
	if s.backend != nil {
		s.Require().NoError(s.backend.StopNode())
		s.backend = nil
	}
	if len(s.dir) != 0 {
		os.RemoveAll(s.dir)
		s.dir = ""
	}
}
