package devtests

import (
	"crypto/ecdsa"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/params"
	statusrpc "github.com/status-im/status-go/rpc"
	"github.com/stretchr/testify/suite"
)

// NewDevNode returns node with clieque engine and prefunded accounts.
func NewDevNode(faucet common.Address) (*node.Node, error) {
	cfg := node.DefaultConfig
	ipc, err := ioutil.TempFile("", "devnode-ipc-")
	if err != nil {
		return nil, err
	}
	cfg.IPCPath = ipc.Name()
	cfg.HTTPModules = []string{"eth"}
	cfg.DataDir = ""
	cfg.P2P.MaxPeers = 0
	cfg.P2P.ListenAddr = ":0"
	cfg.P2P.NoDiscovery = true
	cfg.P2P.DiscoveryV5 = false

	stack, err := node.New(&cfg)
	if err != nil {
		return nil, err
	}

	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	// ensure that etherbase is added to an account manager
	etherbase, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	acc, err := ks.ImportECDSA(etherbase, "")
	if err != nil {
		return nil, err
	}
	err = ks.Unlock(acc, "")
	if err != nil {
		return nil, err
	}

	ethcfg := eth.DefaultConfig
	ethcfg.NetworkId = 1337
	// 0 - mine only if transaction pending
	ethcfg.Genesis = core.DeveloperGenesisBlock(0, faucet)
	extra := make([]byte, 32) // extraVanity
	extra = append(extra, acc.Address[:]...)
	extra = append(extra, make([]byte, 65)...) // extraSeal
	ethcfg.Genesis.ExtraData = extra
	ethcfg.MinerGasPrice = big.NewInt(1)
	ethcfg.Etherbase = acc.Address

	return stack, stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return eth.New(ctx, &ethcfg)
	})

}

// StartWithMiner starts node with eth service and a miner.
func StartWithMiner(stack *node.Node) error {
	err := stack.Start()
	if err != nil {
		return err
	}
	var ethereum *eth.Ethereum
	err = stack.Service(&ethereum)
	if err != nil {
		return err
	}
	ethereum.TxPool().SetGasPrice(big.NewInt(1))
	return ethereum.StartMining(0)
}

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
	s.miner, err = NewDevNode(s.DevAccountAddress)
	s.Require().NoError(err)
	s.Require().NoError(StartWithMiner(s.miner))

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
