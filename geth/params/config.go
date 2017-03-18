package params

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"math/big"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
)

// default node configuration options
var (
	UseMainnetFlag = "false" // to be overridden via -ldflags '-X geth/params.UseMainnetFlag'
	UseMainnet     = false
)

func init() {
	if UseMainnetFlag == "true" { // set at compile time, here we make sure to set corresponding boolean flag
		UseMainnet = true
	}
}

var (
	ErrMissingDataDir   = errors.New("missing required 'DataDir' parameter")
	ErrMissingNetworkId = errors.New("missing required 'NetworkId' parameter")
)

// ChainConfig holds core blockchain settings. It is stored in the database on a per block basis.
type ChainConfig struct {
	// ChainId identifies the current chain and is used for replay protection
	ChainId *big.Int

	// HomesteadBlock is Homestead switch block (nil = no fork, 0 = already homestead)
	HomesteadBlock *big.Int

	// DAOForkBlock TheDAO hard-fork switch block (nil = no fork)
	DAOForkBlock *big.Int

	// DAOForkSupport Whether the nodes supports or opposes the DAO hard-fork
	DAOForkSupport bool

	// EIP150Block is EIP150 HF block (nil = no fork)
	EIP150Block *big.Int

	// EIP150Hash is EIP150 HF hash (fast sync aid)
	EIP150Hash common.Hash

	// EIP155Block is EIP155 HF block
	EIP155Block *big.Int

	// EIP158Block is EIP158 HF block
	EIP158Block *big.Int
}

// LightEthConfig holds LES-related configuration
// Status nodes are always lightweight clients (due to mobile platform constraints)
type LightEthConfig struct {
	// Genesis is JSON to seed the chain database with
	Genesis string

	// DatabaseCache is memory (in MBs) allocated to internal caching (min 16MB / database forced)
	DatabaseCache int
}

// WhisperConfig holds SHH-related configuration
type WhisperConfig struct{}

// SwarmConfig holds Swarm-related configuration
type SwarmConfig struct{}

// NodeConfig stores configuration options for a node
type NodeConfig struct {
	// TestNet flag whether given configuration describes a test or mainnet
	TestNet bool

	// NetworkId sets network to use for selecting peers to connect to
	NetworkId int

	// DataDir is the file system folder the node should use for any data storage needs.
	DataDir string

	// Name sets the instance name of the node. It must not contain the / character.
	Name string

	// Version exposes program's version. It is used in the devp2p node identifier.
	Version string

	// APIModules is a comma-separated list of API modules exposed via *any* (HTTP/WS/IPC) RPC interface.
	APIModules string

	// HTTPHost is the host interface on which to start the HTTP RPC server.
	// Pass empty string if no HTTP RPC interface needs to be started.
	HTTPHost string

	// HTTPPort is the TCP port number on which to start the Geth's HTTP RPC server.
	HTTPPort int

	// WSHost is a host interface for the WebSocket RPC server
	WSHost string

	// WSPort is the TCP port number on which to start the Geth's WebSocket RPC server.
	WSPort int

	// WSEnabled specifies whether WS-RPC Server is enabled or not
	WSEnabled bool

	// IPCFile is filename of exposed IPC RPC Server
	IPCFile string

	// IPCEnabled specifies whether IPC-RPC Server is enabled or not
	IPCEnabled bool

	// TLSEnabled specifies whether TLS support should be enabled on node or not
	// TLS support is only planned in go-ethereum, so we are using our own patch.
	TLSEnabled bool

	// MaxPeers is the maximum number of (global) peers that can be connected.
	// Set to zero, if only static or trusted peers are allowed to connect.
	MaxPeers int

	// MaxPendingPeers is the maximum number of peers that can be pending in the
	// handshake phase, counted separately for inbound and outbound connections.
	MaxPendingPeers int

	// ChainConfig extra configuration for blockchain
	*ChainConfig `json:"ChainConfig,"`

	// LightEthConfig extra configuration for LES
	*LightEthConfig `json:"LightEthConfig,"`

	// WhisperConfig extra configuration for SHH
	*WhisperConfig `json:"WhisperConfig,"`

	// SwarmConfig extra configuration for Swarm and ENS
	*SwarmConfig `json:"SwarmConfig,"`
}

// NewNodeConfig creates new node configuration object
func NewNodeConfig(dataDir string, networkId int) (*NodeConfig, error) {
	nodeConfig := &NodeConfig{
		NetworkId:       networkId,
		DataDir:         dataDir,
		Name:            DefaultClientIdentifier,
		Version:         Version,
		HTTPHost:        DefaultHTTPHost,
		HTTPPort:        DefaultHTTPPort,
		APIModules:      DefaultAPIModules,
		WSHost:          DefaultWSHost,
		WSPort:          DefaultWSPort,
		MaxPeers:        DefaultMaxPeers,
		MaxPendingPeers: DefaultMaxPendingPeers,
		IPCFile:         DefaultIPCFile,
		ChainConfig:     &ChainConfig{},
		LightEthConfig: &LightEthConfig{
			DatabaseCache: DefaultDatabaseCache,
		},
		WhisperConfig: &WhisperConfig{},
		SwarmConfig:   &SwarmConfig{},
	}

	nodeConfig.populateChainConfig()

	return nodeConfig, nil
}

// populateChainConfig does necessary adjustments to config object (depending on network node will be runnin on)
func (c *NodeConfig) populateChainConfig() {
	c.TestNet = false
	if c.NetworkId == TestNetworkId {
		c.TestNet = true
	}

	if c.TestNet {
		// Homestead fork
		c.ChainConfig.HomesteadBlock = params.TestnetChainConfig.HomesteadBlock

		// DAO fork
		c.ChainConfig.DAOForkBlock = params.TestnetChainConfig.DAOForkBlock
		c.ChainConfig.DAOForkSupport = params.TestnetChainConfig.DAOForkSupport

		// DoS reprice fork
		c.ChainConfig.EIP150Block = params.TestnetChainConfig.EIP150Block
		c.ChainConfig.EIP150Hash = params.TestnetChainConfig.EIP150Hash

		// DoS state cleanup fork
		c.ChainConfig.EIP155Block = params.TestnetChainConfig.EIP155Block
		c.ChainConfig.EIP158Block = params.TestnetChainConfig.EIP158Block
		c.ChainConfig.ChainId = params.TestnetChainConfig.ChainId

		if len(c.DataDir) > 0 {
			c.DataDir = filepath.Join(c.DataDir, "testnet")
		}
		c.Genesis = core.DefaultTestnetGenesisBlock()
	} else {
		// Homestead fork
		c.ChainConfig.HomesteadBlock = params.MainNetHomesteadBlock
		// DAO fork
		c.ChainConfig.DAOForkBlock = params.MainNetDAOForkBlock
		c.ChainConfig.DAOForkSupport = true

		// DoS reprice fork
		c.ChainConfig.EIP150Block = params.MainNetHomesteadGasRepriceBlock
		c.ChainConfig.EIP150Hash = params.MainNetHomesteadGasRepriceHash

		// DoS state cleanup fork
		c.ChainConfig.EIP155Block = params.MainNetSpuriousDragon
		c.ChainConfig.EIP158Block = params.MainNetSpuriousDragon
		c.ChainConfig.ChainId = params.MainNetChainID

		c.Genesis = core.DefaultGenesisBlock()
	}
}

// LoadNodeConfig parses incoming JSON and returned it as Config
func LoadNodeConfig(configJSON string) (*NodeConfig, error) {
	nodeConfig, err := NewNodeConfig("", 0)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(strings.NewReader(configJSON))
	//decoder.UseNumber()

	// override default configuration with values by JSON input
	if err := decoder.Decode(&nodeConfig); err != nil {
		return nil, err
	}

	// repopulate
	nodeConfig.populateChainConfig()

	if len(nodeConfig.DataDir) == 0 {
		return nil, ErrMissingDataDir
	}

	if nodeConfig.NetworkId <= 0 {
		return nil, ErrMissingNetworkId
	}

	return nodeConfig, nil
}

// Save dumps configuration to the disk
func (c *NodeConfig) Save() error {
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(c.DataDir, os.ModePerm); err != nil {
		return err
	}

	configFilePath := filepath.Join(c.DataDir, "config.json")
	if err := ioutil.WriteFile(configFilePath, data, os.ModePerm); err != nil {
		return err
	}

	glog.V(logger.Info).Infof("config file saved: %v", configFilePath)
	return nil
}

// String dumps config object as nicely indented JSON
func (c *NodeConfig) String() string {
	data, _ := json.MarshalIndent(c, "", "    ")
	return string(data)
}
