package params

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
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
	ErrMissingDataDir            = errors.New("missing required 'DataDir' parameter")
	ErrMissingNetworkId          = errors.New("missing required 'NetworkId' parameter")
	ErrEmptyPasswordFile         = errors.New("password file cannot be empty")
	ErrEmptyIdentityFile         = errors.New("identity file cannot be empty")
	ErrEmptyAuthorizationKeyFile = errors.New("authorization key file cannot be empty")
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
	// Enabled flag specifies whether  protocol is enabled
	Enabled bool

	// Genesis is JSON to seed the chain database with
	Genesis string

	// DatabaseCache is memory (in MBs) allocated to internal caching (min 16MB / database forced)
	DatabaseCache int
}

type FirebaseConfig struct {
	// AuthorizationKeyFile file path that contains FCM authorization key
	AuthorizationKeyFile string

	// NotificationTriggerURL URL used to send push notification requests to
	NotificationTriggerURL string
}

// WhisperConfig holds SHH-related configuration
type WhisperConfig struct {
	// Enabled flag specifies whether  protocol is enabled
	Enabled bool

	// IdentityFile path to private key, that will be loaded as identity into Whisper
	IdentityFile string

	// PasswordFile path to password file, for non-interactive password entry
	// (if no account file selected, then this password is used for symmetric encryption)
	PasswordFile string

	// EchoMode if mode is on, prints some arguments for diagnostics
	EchoMode bool

	// BootstrapNode whether node doesn't actively connect to peers, and waits for incoming connections
	BootstrapNode bool

	// ForwarderNode is mode when node only forwards messages, neither sends nor decrypts messages
	ForwarderNode bool

	// MailServerNode is mode when node is capable of delivering expired messages on demand
	MailServerNode bool

	// NotificationServerNode is mode when node is capable of sending Push (and probably other kinds) Notifications
	NotificationServerNode bool

	// DataDir is the file system folder Whisper should use for any data storage needs.
	DataDir string

	// Port Whisper node's listening port
	Port int

	// MinimumPoW minimum PoW for Whisper messages
	MinimumPoW float64

	// TTL time to live for messages, in seconds
	TTL int

	// FirebaseConfig extra configuration for Firebase Cloud Messaging
	FirebaseConfig *FirebaseConfig `json:"FirebaseConfig,"`
}

// SwarmConfig holds Swarm-related configuration
type SwarmConfig struct {
	// Enabled flag specifies whether  protocol is enabled
	Enabled bool
}

// NodeConfig stores configuration options for a node
type NodeConfig struct {
	// TestNet flag whether given configuration describes a test or mainnet
	TestNet bool

	// NetworkId sets network to use for selecting peers to connect to
	NetworkId int

	// DataDir is the file system folder the node should use for any data storage needs.
	DataDir string

	// KeyStoreDir is the file system folder that contains private keys.
	// If KeyStoreDir is empty, the default location is the "keystore" subdirectory of DataDir.
	KeyStoreDir string

	// PrivateKeyFile is a filename with node ID (private key)
	// This file should contain a valid secp256k1 private key that will be used for both
	// remote peer identification as well as network traffic encryption.
	NodeKeyFile string

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

	// LogToFile specified whether logs should be saved into file
	LogEnabled bool

	// LogFile is filename where exposed logs get written to
	LogFile string

	// LogLevel defines minimum log level. Valid names are "ERROR", "WARNING", "INFO", "DEBUG", and "DETAIL".
	LogLevel string

	// ChainConfig extra configuration for blockchain
	*ChainConfig `json:"ChainConfig,"`

	// LightEthConfig extra configuration for LES
	LightEthConfig *LightEthConfig `json:"LightEthConfig,"`

	// WhisperConfig extra configuration for SHH
	WhisperConfig *WhisperConfig `json:"WhisperConfig,"`

	// SwarmConfig extra configuration for Swarm and ENS
	SwarmConfig *SwarmConfig `json:"SwarmConfig,"`
}

// NewNodeConfig creates new node configuration object
func NewNodeConfig(dataDir string, networkId int) (*NodeConfig, error) {
	nodeConfig := &NodeConfig{
		NetworkId:       networkId,
		DataDir:         dataDir,
		Name:            ClientIdentifier,
		Version:         Version,
		HTTPHost:        HTTPHost,
		HTTPPort:        HTTPPort,
		APIModules:      APIModules,
		WSHost:          WSHost,
		WSPort:          WSPort,
		MaxPeers:        MaxPeers,
		MaxPendingPeers: MaxPendingPeers,
		IPCFile:         IPCFile,
		LogFile:         LogFile,
		LogLevel:        LogLevel,
		ChainConfig:     &ChainConfig{},
		LightEthConfig: &LightEthConfig{
			Enabled:       true,
			DatabaseCache: DatabaseCache,
		},
		WhisperConfig: &WhisperConfig{
			Enabled:    true,
			Port:       WhisperPort,
			MinimumPoW: WhisperMinimumPoW,
			TTL:        WhisperTTL,
			FirebaseConfig: &FirebaseConfig{
				NotificationTriggerURL: FirebaseNotificationTriggerURL,
			},
		},
		SwarmConfig: &SwarmConfig{},
	}

	// auto-populate some dependent values
	nodeConfig.populateChainConfig()
	nodeConfig.populateDirs()

	return nodeConfig, nil
}

// populateDirs updates directories that should be wrt to DataDir
func (c *NodeConfig) populateDirs() {
	makeSubDirPath := func(baseDir, subDir string) string {
		if len(baseDir) == 0 {
			return ""
		}

		return filepath.Join(baseDir, subDir)
	}
	if len(c.KeyStoreDir) == 0 {
		c.KeyStoreDir = makeSubDirPath(c.DataDir, KeyStoreDir)
	}

	if len(c.WhisperConfig.DataDir) == 0 {
		c.WhisperConfig.DataDir = makeSubDirPath(c.DataDir, WhisperDataDir)
	}
}

// populateChainConfig does necessary adjustments to config object (depending on network node will be running on)
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

		c.LightEthConfig.Genesis = core.DefaultTestnetGenesisBlock()
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

		c.LightEthConfig.Genesis = core.DefaultGenesisBlock()
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
	nodeConfig.populateDirs()

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

// String dumps config object as nicely indented JSON
func (c *WhisperConfig) String() string {
	data, _ := json.MarshalIndent(c, "", "    ")
	return string(data)
}

// String dumps config object as nicely indented JSON
func (c *SwarmConfig) String() string {
	data, _ := json.MarshalIndent(c, "", "    ")
	return string(data)
}

// ReadPasswordFile reads and returns content of the password file
func (c *WhisperConfig) ReadPasswordFile() ([]byte, error) {
	if len(c.PasswordFile) <= 0 {
		return nil, ErrEmptyPasswordFile
	}

	password, err := ioutil.ReadFile(c.PasswordFile)
	if err != nil {
		return nil, err
	}
	password = bytes.TrimRight(password, "\n")

	if len(password) == 0 {
		return nil, ErrEmptyPasswordFile
	}

	return password, nil
}

// ReadIdentityFile reads and loads identity private key
func (c *WhisperConfig) ReadIdentityFile() (*ecdsa.PrivateKey, error) {
	if len(c.IdentityFile) <= 0 {
		return nil, ErrEmptyIdentityFile
	}

	identity, err := crypto.LoadECDSA(c.IdentityFile)
	if err != nil {
		return nil, err
	}

	if identity == nil {
		return nil, ErrEmptyIdentityFile
	}

	return identity, nil
}

// ReadAuthorizationKeyFile reads and loads FCM authorization key
func (c *FirebaseConfig) ReadAuthorizationKeyFile() ([]byte, error) {
	if len(c.AuthorizationKeyFile) <= 0 {
		return nil, ErrEmptyAuthorizationKeyFile
	}

	key, err := ioutil.ReadFile(c.AuthorizationKeyFile)
	if err != nil {
		return nil, err
	}
	key = bytes.TrimRight(key, "\n")

	if key == nil {
		return nil, ErrEmptyAuthorizationKeyFile
	}

	return key, nil
}
