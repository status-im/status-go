package params

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/core"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/static"
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

// errors
var (
	ErrMissingDataDir             = errors.New("missing required 'DataDir' parameter")
	ErrMissingNetworkID           = errors.New("missing required 'NetworkID' parameter")
	ErrEmptyPasswordFile          = errors.New("password file cannot be empty")
	ErrNoPasswordFileValueSet     = errors.New("password file path not set")
	ErrEmptyAuthorizationKeyFile  = errors.New("authorization key file cannot be empty")
	ErrAuthorizationKeyFileNotSet = errors.New("authorization key file is not set")
)

// LightEthConfig holds LES-related configuration
// Status nodes are always lightweight clients (due to mobile platform constraints)
type LightEthConfig struct {
	// Enabled flag specifies whether protocol is enabled
	Enabled bool

	// Genesis is JSON to seed the chain database with
	Genesis string

	// DatabaseCache is memory (in MBs) allocated to internal caching (min 16MB / database forced)
	DatabaseCache int
}

// FirebaseConfig holds FCM-related configuration
type FirebaseConfig struct {
	// AuthorizationKeyFile file path that contains FCM authorization key
	AuthorizationKeyFile string

	// NotificationTriggerURL URL used to send push notification requests to
	NotificationTriggerURL string
}

// ReadAuthorizationKeyFile reads and loads FCM authorization key
func (c *FirebaseConfig) ReadAuthorizationKeyFile() ([]byte, error) {
	if len(c.AuthorizationKeyFile) == 0 {
		return nil, ErrAuthorizationKeyFileNotSet
	}

	key, err := ioutil.ReadFile(c.AuthorizationKeyFile)
	if err != nil {
		return nil, err
	}

	key = bytes.TrimRight(key, "\n")

	if len(key) == 0 {
		return nil, ErrEmptyAuthorizationKeyFile
	}

	return key, nil
}

// WhisperConfig holds SHH-related configuration
type WhisperConfig struct {
	// Enabled flag specifies whether protocol is enabled
	Enabled bool

	// PasswordFile contains a password for symmetric encryption with MailServer.
	PasswordFile string

	// Password for symmetric encryption with MailServer.
	// (if no account file selected, then this password is used for symmetric encryption).
	Password string

	// EnableMailServer is mode when node is capable of delivering expired messages on demand
	EnableMailServer bool

	// DataDir is the file system folder Whisper should use for any data storage needs.
	// For instance, MailServer will use this directory to store its data.
	DataDir string

	// MinimumPoW minimum PoW for Whisper messages
	MinimumPoW float64

	// TTL time to live for messages, in seconds
	TTL int

	// FirebaseConfig extra configuration for Firebase Cloud Messaging
	FirebaseConfig *FirebaseConfig `json:"FirebaseConfig,"`
}

// ReadPasswordFile reads and returns content of the password file
func (c *WhisperConfig) ReadPasswordFile() error {
	if len(c.PasswordFile) == 0 {
		return ErrNoPasswordFileValueSet
	}

	password, err := ioutil.ReadFile(c.PasswordFile)
	if err != nil {
		return err
	}
	password = bytes.TrimRight(password, "\n")

	if len(password) == 0 {
		return ErrEmptyPasswordFile
	}

	c.Password = string(password)

	return nil
}

// String dumps config object as nicely indented JSON
func (c *WhisperConfig) String() string {
	data, _ := json.MarshalIndent(c, "", "    ") // nolint: gas
	return string(data)
}

// SwarmConfig holds Swarm-related configuration
type SwarmConfig struct {
	// Enabled flag specifies whether protocol is enabled
	Enabled bool
}

// String dumps config object as nicely indented JSON
func (c *SwarmConfig) String() string {
	data, _ := json.MarshalIndent(c, "", "    ") // nolint: gas
	return string(data)
}

// BootClusterConfig holds configuration for supporting boot cluster, which is a temporary
// means for mobile devices to get connected to Ethereum network (UDP-based discovery
// may not be available, so we need means to discover the network manually).
type BootClusterConfig struct {
	// Enabled flag specifies whether feature is enabled
	Enabled bool

	// BootNodes list of bootstrap nodes for a given network (Ropsten, Rinkeby, Homestead),
	// for a given mode (production vs development)
	BootNodes []string
}

// String dumps config object as nicely indented JSON
func (c *BootClusterConfig) String() string {
	data, _ := json.MarshalIndent(c, "", "    ") // nolint: gas
	return string(data)
}

//=====================================================================================

// UpstreamRPCConfig stores configuration for upstream rpc connection.
type UpstreamRPCConfig struct {
	// Enabled flag specifies whether feature is enabled
	Enabled bool

	// URL sets the rpc upstream host address for communication with
	// a non-local infura endpoint.
	URL string
}

//=====================================================================================

// NodeConfig stores configuration options for a node
type NodeConfig struct {
	// DevMode is true when given configuration is to be used during development.
	// For production, this flag should be turned off, so that more strict requirements
	// are applied to node's configuration
	DevMode bool

	// NetworkID sets network to use for selecting peers to connect to
	NetworkID uint64 `json:"NetworkId" validate:"required"`

	// DataDir is the file system folder the node should use for any data storage needs.
	DataDir string `validate:"required"`

	// KeyStoreDir is the file system folder that contains private keys.
	// If KeyStoreDir is empty, the default location is the "keystore" subdirectory of DataDir.
	KeyStoreDir string

	// PrivateKeyFile is a filename with node ID (private key)
	// This file should contain a valid secp256k1 private key that will be used for both
	// remote peer identification as well as network traffic encryption.
	NodeKeyFile string

	// Discovery set to true will enabled discovery protocol.
	Discovery bool

	// ListenAddr is an IP address and port of this node (e.g. 127.0.0.1:30303).
	ListenAddr string

	// Name sets the instance name of the node. It must not contain the / character.
	Name string `validate:"excludes=/"`

	// Version exposes program's version. It is used in the devp2p node identifier.
	Version string

	// APIModules is a comma-separated list of API modules exposed via *any* (HTTP/WS/IPC) RPC interface.
	APIModules string

	// HTTPHost is the host interface on which to start the HTTP RPC server.
	// Pass empty string if no HTTP RPC interface needs to be started.
	HTTPHost string

	// RPCEnabled specifies whether the http RPC server is to be enabled by default.
	RPCEnabled bool

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

	// LogFile is filename where exposed logs get written to
	LogFile string

	// LogLevel defines minimum log level. Valid names are "ERROR", "WARN", "INFO", "DEBUG", and "TRACE".
	LogLevel string `validate:"eq=ERROR|eq=WARN|eq=INFO|eq=DEBUG|eq=TRACE"`

	// LogToStderr defines whether logged info should also be output to os.Stderr
	LogToStderr bool

	// UpstreamConfig extra config for providing upstream infura server.
	UpstreamConfig UpstreamRPCConfig `json:"UpstreamConfig"`

	// BootClusterConfig extra configuration for supporting cluster
	BootClusterConfig *BootClusterConfig `json:"BootClusterConfig," validate:"structonly"`

	// LightEthConfig extra configuration for LES
	LightEthConfig *LightEthConfig `json:"LightEthConfig," validate:"structonly"`

	// WhisperConfig extra configuration for SHH
	WhisperConfig *WhisperConfig `json:"WhisperConfig," validate:"structonly"`

	// SwarmConfig extra configuration for Swarm and ENS
	SwarmConfig *SwarmConfig `json:"SwarmConfig," validate:"structonly"`
}

// NewNodeConfig creates new node configuration object
func NewNodeConfig(dataDir string, networkID uint64, devMode bool) (*NodeConfig, error) {
	nodeConfig := &NodeConfig{
		DevMode:         devMode,
		NetworkID:       networkID,
		DataDir:         dataDir,
		Name:            ClientIdentifier,
		Version:         Version,
		RPCEnabled:      RPCEnabledDefault,
		HTTPHost:        HTTPHost,
		HTTPPort:        HTTPPort,
		ListenAddr:      ListenAddr,
		APIModules:      APIModules,
		WSHost:          WSHost,
		WSPort:          WSPort,
		MaxPeers:        MaxPeers,
		MaxPendingPeers: MaxPendingPeers,
		IPCFile:         IPCFile,
		LogFile:         LogFile,
		LogLevel:        LogLevel,
		LogToStderr:     LogToStderr,
		BootClusterConfig: &BootClusterConfig{
			Enabled:   true,
			BootNodes: []string{},
		},
		LightEthConfig: &LightEthConfig{
			Enabled:       true,
			DatabaseCache: DatabaseCache,
		},
		WhisperConfig: &WhisperConfig{
			Enabled:    true,
			MinimumPoW: WhisperMinimumPoW,
			TTL:        WhisperTTL,
			FirebaseConfig: &FirebaseConfig{
				NotificationTriggerURL: FirebaseNotificationTriggerURL,
			},
		},
		SwarmConfig: &SwarmConfig{},
	}

	// adjust dependent values
	if err := nodeConfig.updateConfig(); err != nil {
		return nil, err
	}

	return nodeConfig, nil
}

// LoadNodeConfig parses incoming JSON and returned it as Config
func LoadNodeConfig(configJSON string) (*NodeConfig, error) {
	nodeConfig, err := loadNodeConfig(configJSON)
	if err != nil {
		return nil, err
	}

	if err := nodeConfig.Validate(); err != nil {
		return nil, err
	}

	return nodeConfig, nil
}

func loadNodeConfig(configJSON string) (*NodeConfig, error) {
	nodeConfig, err := NewNodeConfig("", 0, true)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(strings.NewReader(configJSON))

	// override default configuration with values by JSON input
	if err := decoder.Decode(&nodeConfig); err != nil {
		return nil, err
	}

	// repopulate
	if err := nodeConfig.updateConfig(); err != nil {
		return nil, err
	}

	return nodeConfig, nil
}

// Validate checks if NodeConfig fields have valid values.
//
// It returns nil if there are no errors, otherwise one or more errors
// can be returned. Multiple errors are joined with a new line.
//
// A single error for a struct:
//
//   type TestStruct struct {
//       TestField string `validate:"required"`
//   }
//
// has the following format:
//
//   Key: 'TestStruct.TestField' Error:Field validation for 'TestField' failed on the 'required' tag
//
func (c *NodeConfig) Validate() error {
	validate := NewValidator()

	if err := validate.Struct(c); err != nil {
		return err
	}

	if c.BootClusterConfig.Enabled {
		if err := validate.Struct(c.BootClusterConfig); err != nil {
			return err
		}
	}

	if c.LightEthConfig.Enabled {
		if err := validate.Struct(c.LightEthConfig); err != nil {
			return err
		}
	}

	if c.WhisperConfig.Enabled {
		if err := validate.Struct(c.WhisperConfig); err != nil {
			return err
		}
	}

	if c.SwarmConfig.Enabled {
		if err := validate.Struct(c.SwarmConfig); err != nil {
			return err
		}
	}

	return nil
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

	log.Info(fmt.Sprintf("config file saved: %v", configFilePath))
	return nil
}

// updateConfig traverses configuration and adjusts dependent fields
// (we have a development/production and mobile/full node dependent configurations)
func (c *NodeConfig) updateConfig() error {
	if err := c.updateGenesisConfig(); err != nil {
		return err
	}

	if err := c.updateUpstreamConfig(); err != nil {
		return err
	}

	if err := c.updateBootClusterConfig(); err != nil {
		return err
	}

	if err := c.updateRelativeDirsConfig(); err != nil {
		return err
	}

	return nil
}

// updateGenesisConfig does necessary adjustments to config object (depending on network node will be running on)
func (c *NodeConfig) updateGenesisConfig() error {
	var genesis *core.Genesis

	switch c.NetworkID {
	case MainNetworkID:
		genesis = core.DefaultGenesisBlock()
	case RopstenNetworkID:
		genesis = core.DefaultTestnetGenesisBlock()
	case RinkebyNetworkID:
		genesis = core.DefaultRinkebyGenesisBlock()
	case StatusChainNetworkID:
		var err error
		genesis, err = c.DefaultStatusChainGenesisBlock()
		if err != nil {
			return err
		}
	default:
		return nil
	}

	// encode the genesis into JSON
	enc, err := json.Marshal(genesis)
	if err != nil {
		return err
	}
	c.LightEthConfig.Genesis = string(enc)

	return nil
}

// DefaultStatusChainGenesisBlock returns the StatusChain network genesis block.
func (c *NodeConfig) DefaultStatusChainGenesisBlock() (*core.Genesis, error) {
	genesisJSON, err := static.Asset("config/status-chain-genesis.json")
	if err != nil {
		return nil, fmt.Errorf("status-chain-genesis.json could not be loaded: %s", err)
	}

	var genesis *core.Genesis
	err = json.Unmarshal(genesisJSON, &genesis)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal status-chain-genesis.json: %s", err)
	}
	return genesis, nil
}

// updateUpstreamConfig sets the proper UpstreamConfig.URL for the network id being used.
func (c *NodeConfig) updateUpstreamConfig() error {

	// If we have a URL already set then keep URL incase
	// of custom server.
	if c.UpstreamConfig.URL != "" {
		return nil
	}

	switch c.NetworkID {
	case MainNetworkID:
		c.UpstreamConfig.URL = MainnetEthereumNetworkURL
	case RopstenNetworkID:
		c.UpstreamConfig.URL = RopstenEthereumNetworkURL
	case RinkebyNetworkID:
		c.UpstreamConfig.URL = RinkebyEthereumNetworkURL
	}

	return nil
}

// updateBootClusterConfig loads boot nodes and CHT for a given network and mode.
// This is necessary until we have LES protocol support CHT sync, and better node
// discovery on mobile devices)
func (c *NodeConfig) updateBootClusterConfig() error {
	if !c.BootClusterConfig.Enabled {
		return nil
	}

	// TODO: Remove this thing as this is an ugly hack.
	// Once CHT sync sub-protocol is working in LES, we will rely on it, as it provides
	// decentralized solution. For now, in order to avoid forcing users to long sync times
	// we use central static resource
	type subClusterConfig struct {
		Number    int      `json:"number"`
		Hash      string   `json:"hash"`
		BootNodes []string `json:"bootnodes"`
	}
	type clusterConfig struct {
		NetworkID   int              `json:"networkID"`
		GenesisHash string           `json:"genesisHash"`
		Prod        subClusterConfig `json:"prod"`
		Dev         subClusterConfig `json:"dev"`
	}

	chtFile, err := static.Asset("config/staticpeers.json")
	if err != nil {
		return fmt.Errorf("staticpeers.json could not be loaded: %s", err)
	}

	var clusters []clusterConfig
	err = json.Unmarshal(chtFile, &clusters)
	if err != nil {
		return fmt.Errorf("failed to unmarshal staticpeers.json: %s", err)
	}

	for _, cluster := range clusters {
		if cluster.NetworkID == int(c.NetworkID) {
			c.BootClusterConfig.BootNodes = cluster.Prod.BootNodes
			if c.DevMode {
				c.BootClusterConfig.BootNodes = cluster.Dev.BootNodes
			}
			break
		}
	}

	return nil
}

// updateRelativeDirsConfig updates directories that should be wrt to DataDir
func (c *NodeConfig) updateRelativeDirsConfig() error {
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

	return nil
}

// String dumps config object as nicely indented JSON
func (c *NodeConfig) String() string {
	data, _ := json.MarshalIndent(c, "", "    ")
	return string(data)
}
