package devtests

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	statusrpc "github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/t/devtests/miner"
)

// DevNodeSuite provides convenient wrapper for starting node with clique backend for mining.
type DevNodeSuite struct {
	suite.Suite

	Remote            *rpc.Client
	Eth               *ethclient.Client
	Local             *statusrpc.Client
	DevAccount        *ecdsa.PrivateKey
	DevAccountAddress types.Address

	dir     string
	backend *api.GethStatusBackend
	miner   *node.Node
}

// SetupTest creates clique node and status node with an rpc connection to a clique node.
func (s *DevNodeSuite) SetupTest() {
	account, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.DevAccount = account
	s.DevAccountAddress = crypto.PubkeyToAddress(account.PublicKey)
	s.miner, err = miner.NewDevNode(common.Address(s.DevAccountAddress))
	s.Require().NoError(err)
	s.Require().NoError(miner.StartWithMiner(s.miner))

	s.dir, err = ioutil.TempDir("", "devtests-")
	s.Require().NoError(err)
	config, err := params.NewNodeConfig(
		s.dir,
		1337,
	)
	networks := json.RawMessage("{}")
	settings := accounts.Settings{Networks: &networks}
	s.Require().NoError(err)
	config.WakuConfig.Enabled = false
	config.LightEthConfig.Enabled = false
	config.UpstreamConfig.Enabled = true
	config.WalletConfig.Enabled = true
	config.UpstreamConfig.URL = s.miner.IPCEndpoint()
	s.backend = api.NewGethStatusBackend()
	s.Require().NoError(s.backend.AccountManager().InitKeystore(config.KeyStoreDir))
	_, err = s.backend.AccountManager().ImportAccount(s.DevAccount, "test")
	s.Require().NoError(err)
	s.backend.UpdateRootDataDir(s.dir)
	s.Require().NoError(s.backend.OpenAccounts())
	keyUIDHex := sha256.Sum256(crypto.FromECDSAPub(&account.PublicKey))
	keyUID := types.EncodeHex(keyUIDHex[:])
	s.Require().NoError(s.backend.StartNodeWithAccountAndConfig(multiaccounts.Account{
		Name:   "main",
		KeyUID: keyUID,
	}, "test", settings, config, []accounts.Account{{Address: s.DevAccountAddress, Wallet: true, Chat: true}}))
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
		s.Require().NoError(s.backend.Logout())
		s.backend = nil
	}
	if len(s.dir) != 0 {
		os.RemoveAll(s.dir)
		s.dir = ""
	}
}
