package params

import (
	"encoding/json"
	"fmt"
	"go/build"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/params"
	"github.com/status-im/status-go/static"
	validator "gopkg.in/go-playground/validator.v9"
)

// ----------
// LightEthConfig
// ----------

// LightEthConfig holds LES-related configuration
// Status nodes are always lightweight clients (due to mobile platform constraints)
type LightEthConfig struct {
	// Enabled flag specifies whether protocol is enabled
	Enabled bool

	// DatabaseCache is memory (in MBs) allocated to internal caching (min 16MB / database forced)
	DatabaseCache int
}

// ----------
// WhisperConfig
// ----------

// WhisperConfig holds SHH-related configuration
type WhisperConfig struct {
	// Enabled flag specifies whether protocol is enabled
	Enabled bool

	// LightClient should be true if the node should start with an empty bloom filter and not forward messages from other nodes
	LightClient bool

	// EnableMailServer is mode when node is capable of delivering expired messages on demand
	EnableMailServer bool

	// DataDir is the file system folder Whisper should use for any data storage needs.
	// For instance, MailServer will use this directory to store its data.
	DataDir string

	// MinimumPoW minimum PoW for Whisper messages
	MinimumPoW float64

	// MailServerPassword for symmetric encryption with MailServer.
	// (if no account file selected, then this password is used for symmetric encryption).
	MailServerPassword string

	// MailServerAsymKey is an hex-encoded asymmetric key to decrypt messages sent to MailServer.
	MailServerAsymKey string

	// RateLimit minimum time between queries to mail server per peer
	MailServerRateLimit int

	// MailServerCleanupPeriod time in seconds to wait to run mail server cleanup
	MailServerCleanupPeriod int

	// TTL time to live for messages, in seconds
	TTL int

	// EnableNTPSync enables NTP synchronizations
	EnableNTPSync bool
}

// String dumps config object as nicely indented JSON
func (c *WhisperConfig) String() string {
	data, _ := json.MarshalIndent(c, "", "    ") // nolint: gas
	return string(data)
}

// ----------
// SwarmConfig
// ----------

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

// ----------
// ClusterConfig
// ----------

// ClusterConfig holds configuration for supporting cluster peers, which is a temporary
// means for mobile devices to get connected to Ethereum network (UDP-based discovery
// may not be available, so we need means to discover the network manually).
type ClusterConfig struct {
	// Enabled flag specifies whether feature is enabled
	Enabled bool

	// Fleet is a type of selected fleet.
	Fleet string

	// StaticNodes is a list of static nodes for this fleet.
	StaticNodes []string

	// BootNodes is a list of cluster peer nodes for this fleet.
	BootNodes []string

	// TrustedMailServers is a list of verified Mail Servers for this fleet.
	TrustedMailServers []string

	// RendezvousNodes is a list rendezvous discovery nodes.
	RendezvousNodes []string
}

// String dumps config object as nicely indented JSON
func (c *ClusterConfig) String() string {
	data, _ := json.MarshalIndent(c, "", "    ") // nolint: gas
	return string(data)
}

// Limits represent min and max amount of peers
type Limits struct {
	Min, Max int
}

// NewLimits creates new Limits config with given min and max values.
func NewLimits(min, max int) Limits {
	return Limits{
		Min: min,
		Max: max,
	}
}

// ----------
// UpstreamRPCConfig
// ----------

// UpstreamRPCConfig stores configuration for upstream rpc connection.
type UpstreamRPCConfig struct {
	// Enabled flag specifies whether feature is enabled
	Enabled bool

	// URL sets the rpc upstream host address for communication with
	// a non-local infura endpoint.
	URL string
}

// ----------
// NodeConfig
// ----------

// NodeConfig stores configuration options for a node
type NodeConfig struct {
	// NetworkID sets network to use for selecting peers to connect to
	NetworkID uint64 `json:"NetworkId" validate:"required"`

	// DataDir is the file system folder the node should use for any data storage needs.
	DataDir string `validate:"required"`

	// KeyStoreDir is the file system folder that contains private keys.
	KeyStoreDir string `validate:"required"`

	// NodeKey is the hex-encoded node ID (private key). Should be a valid secp256k1 private key that will be used for both
	// remote peer identification as well as network traffic encryption.
	NodeKey string

	// NoDiscovery set to true will disable discovery protocol.
	NoDiscovery bool

	// Rendezvous enables discovery protocol.
	Rendezvous bool

	// ListenAddr is an IP address and port of this node (e.g. 127.0.0.1:30303).
	ListenAddr string

	// AdvertiseAddr is a public IP address the node wants to be found with.
	// It is especially useful when using floating IPs attached to a server.
	AdvertiseAddr string

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

	log log.Logger

	// LogEnabled enables the logger
	LogEnabled bool `json:"LogEnabled"`

	// LogFile is filename where exposed logs get written to
	LogFile string

	// LogLevel defines minimum log level. Valid names are "ERROR", "WARN", "INFO", "DEBUG", and "TRACE".
	LogLevel string `validate:"eq=ERROR|eq=WARN|eq=INFO|eq=DEBUG|eq=TRACE"`

	// LogToStderr defines whether logged info should also be output to os.Stderr
	LogToStderr bool

	// UpstreamConfig extra config for providing upstream infura server.
	UpstreamConfig UpstreamRPCConfig `json:"UpstreamConfig"`

	// ClusterConfig extra configuration for supporting cluster peers.
	ClusterConfig ClusterConfig `json:"ClusterConfig," validate:"structonly"`

	// LightEthConfig extra configuration for LES
	LightEthConfig LightEthConfig `json:"LightEthConfig," validate:"structonly"`

	// WhisperConfig extra configuration for SHH
	WhisperConfig WhisperConfig `json:"WhisperConfig," validate:"structonly"`

	// SwarmConfig extra configuration for Swarm and ENS
	SwarmConfig SwarmConfig `json:"SwarmConfig," validate:"structonly"`

	// RegisterTopics a list of specific topics where the peer wants to be
	// discoverable.
	RegisterTopics []discv5.Topic `json:"RegisterTopics"`

	// RequiredTopics list of topics where a client wants to search for
	// discoverable peers with the discovery limits.
	RequireTopics map[discv5.Topic]Limits `json:"RequireTopics"`

	// StatusServiceEnabled enables status service api
	StatusServiceEnabled bool

	// DebugAPIEnabled enables debug api
	DebugAPIEnabled bool

	// MailServerRegistryAddress is the MailServerRegistry contract address
	MailServerRegistryAddress string
}

// NewNodeConfigWithDefaults creates new node configuration object
// with some defaults suitable for adhoc use.
func NewNodeConfigWithDefaults(dataDir, fleet string, networkID uint64) (*NodeConfig, error) {
	nodeConfig, err := NewNodeConfig(dataDir, fleet, networkID)
	if err != nil {
		return nil, err
	}

	if dataDir != "" {
		nodeConfig.KeyStoreDir = path.Join(dataDir, "keystore")
		nodeConfig.WhisperConfig.DataDir = path.Join(dataDir, "wnode")
	}

	if fleet != FleetUndefined {
		statusConfigJSON, err := static.Asset(fmt.Sprintf("../config/cli/fleet-%s.json", fleet))
		if err == nil {
			err = LoadConfigFromJSON(string(statusConfigJSON), nodeConfig)
		}
		if err != nil {
			return nil, fmt.Errorf("default config could not be loaded: %s", err)
		}
	}

	nodeConfig.HTTPHost = ""
	nodeConfig.ListenAddr = ":30303"
	nodeConfig.LogEnabled = true
	nodeConfig.LogLevel = "INFO"
	nodeConfig.LogToStderr = true
	nodeConfig.WhisperConfig.Enabled = true
	nodeConfig.WhisperConfig.EnableNTPSync = true

	nodeConfig.updatePeerLimits()

	return nodeConfig, nil
}

// NewNodeConfigWithDefaultsAndFiles creates new node configuration object
// with some defaults suitable for adhoc use and applies config files on top.
func NewNodeConfigWithDefaultsAndFiles(
	dataDir, fleet string, networkID uint64, files ...string,
) (*NodeConfig, error) {
	c, err := NewNodeConfigWithDefaults(dataDir, fleet, networkID)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if err := loadConfigConfigFromFile(file, c); err != nil {
			return nil, err
		}
	}

	c.updatePeerLimits()

	return c, nil
}

// updatePeerLimits will set default peer limits expectations based on enabled services.
func (c *NodeConfig) updatePeerLimits() {
	if c.NoDiscovery && !c.Rendezvous {
		return
	}
	if c.WhisperConfig.Enabled {
		c.RequireTopics[WhisperDiscv5Topic] = WhisperDiscv5Limits
		// TODO(dshulyak) register mailserver limits when we will change how they are handled.
	}
	if c.LightEthConfig.Enabled {
		c.RequireTopics[discv5.Topic(LesTopic(int(c.NetworkID)))] = LesDiscoveryLimits
	}
}

// NewNodeConfig creates new node configuration object with bare-minimum defaults
func NewNodeConfig(dataDir, fleet string, networkID uint64) (*NodeConfig, error) {
	nodeConfig := &NodeConfig{
		NetworkID:       networkID,
		DataDir:         dataDir,
		Version:         Version,
		RPCEnabled:      false,
		HTTPHost:        "localhost",
		HTTPPort:        8545,
		ListenAddr:      ":0",
		APIModules:      "eth,net,web3,peer",
		MaxPeers:        25,
		MaxPendingPeers: 0,
		IPCFile:         "geth.ipc",
		log:             log.New("package", "status-go/params.NodeConfig"),
		LogFile:         "",
		LogLevel:        "ERROR",
		UpstreamConfig: UpstreamRPCConfig{
			URL: getUpstreamURL(networkID),
		},
		ClusterConfig: ClusterConfig{
			Enabled:     fleet != FleetUndefined,
			Fleet:       fleet,
			StaticNodes: []string{},
			BootNodes:   []string{},
		},
		LightEthConfig: LightEthConfig{
			Enabled:       false,
			DatabaseCache: 16,
		},
		WhisperConfig: WhisperConfig{
			Enabled:       false,
			MinimumPoW:    WhisperMinimumPoW,
			TTL:           WhisperTTL,
			EnableNTPSync: false,
		},
		SwarmConfig:    SwarmConfig{},
		RegisterTopics: []discv5.Topic{},
		RequireTopics:  map[discv5.Topic]Limits{},
	}

	return nodeConfig, nil
}

// NewConfigFromJSON parses incoming JSON and returned it as Config
func NewConfigFromJSON(configJSON string) (*NodeConfig, error) {
	nodeConfig, err := NewNodeConfig("", FleetUndefined, 0)
	if err != nil {
		return nil, err
	}

	if err := LoadConfigFromJSON(configJSON, nodeConfig); err != nil {
		return nil, err
	}

	return nodeConfig, nil
}

// LoadConfigFromJSON parses incoming JSON and returned it as Config
func LoadConfigFromJSON(configJSON string, nodeConfig *NodeConfig) error {
	if err := loadNodeConfig(configJSON, nodeConfig); err != nil {
		return err
	}

	if err := nodeConfig.Validate(); err != nil {
		return err
	}

	return nil
}

func loadNodeConfig(configJSON string, nodeConfig *NodeConfig) error {
	decoder := json.NewDecoder(strings.NewReader(configJSON))

	// override default configuration with values by JSON input
	if err := decoder.Decode(&nodeConfig); err != nil {
		return err
	}

	return nil
}

func loadConfigConfigFromFile(path string, config *NodeConfig) error {
	jsonConfig, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	if err = loadNodeConfig(string(jsonConfig), config); err != nil {
		return err
	}

	return nil
}

// LoadConfigFromFiles reads the configuration files specified in configFilePaths,
// merging the values in order in the config argument
func LoadConfigFromFiles(configFilePaths []string, config *NodeConfig) error {
	for _, path := range configFilePaths {
		if err := loadConfigConfigFromFile(path, config); err != nil {
			return err
		}
	}

	return nil
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

	if c.NodeKey != "" {
		if _, err := crypto.HexToECDSA(c.NodeKey); err != nil {
			return fmt.Errorf("NodeKey is invalid (%s): %v", c.NodeKey, err)
		}
	}

	if c.UpstreamConfig.Enabled && c.LightEthConfig.Enabled {
		return fmt.Errorf("both UpstreamConfig and LightEthConfig are enabled, but they are mutually exclusive")
	}

	if err := c.validateChildStructs(validate); err != nil {
		return err
	}

	if !c.NoDiscovery && len(c.ClusterConfig.BootNodes) == 0 {
		// No point in running discovery if we don't have bootnodes.
		// In case we do have bootnodes, NoDiscovery should be true.
		return fmt.Errorf("NoDiscovery is false, but ClusterConfig.BootNodes is empty")
	}

	if len(c.ClusterConfig.RendezvousNodes) == 0 {
		if c.Rendezvous {
			return fmt.Errorf("Rendezvous is enabled, but ClusterConfig.RendezvousNodes is empty")
		}
	} else if !c.Rendezvous {
		return fmt.Errorf("Rendezvous is disabled, but ClusterConfig.RendezvousNodes is not empty")
	}

	return nil
}

func (c *NodeConfig) validateChildStructs(validate *validator.Validate) error {
	// Validate child structs
	if err := c.UpstreamConfig.Validate(validate); err != nil {
		return err
	}
	if err := c.ClusterConfig.Validate(validate); err != nil {
		return err
	}
	if err := c.LightEthConfig.Validate(validate); err != nil {
		return err
	}
	if err := c.WhisperConfig.Validate(validate); err != nil {
		return err
	}
	if err := c.SwarmConfig.Validate(validate); err != nil {
		return err
	}

	return nil
}

// Validate validates the UpstreamRPCConfig struct and returns an error if inconsistent values are found
func (c *UpstreamRPCConfig) Validate(validate *validator.Validate) error {
	if !c.Enabled {
		return nil
	}

	if err := validate.Struct(c); err != nil {
		return err
	}

	if _, err := url.ParseRequestURI(c.URL); err != nil {
		return fmt.Errorf("UpstreamRPCConfig.URL '%s' is invalid: %v", c.URL, err.Error())
	}

	return nil
}

// Validate validates the ClusterConfig struct and returns an error if inconsistent values are found
func (c *ClusterConfig) Validate(validate *validator.Validate) error {
	if !c.Enabled {
		return nil
	}

	if err := validate.Struct(c); err != nil {
		return err
	}

	if c.Fleet == "" {
		return fmt.Errorf("ClusterConfig.Fleet is empty")
	}

	return nil
}

// Validate validates the LightEthConfig struct and returns an error if inconsistent values are found
func (c *LightEthConfig) Validate(validate *validator.Validate) error {
	if !c.Enabled {
		return nil
	}

	if err := validate.Struct(c); err != nil {
		return err
	}

	return nil
}

// Validate validates the WhisperConfig struct and returns an error if inconsistent values are found
func (c *WhisperConfig) Validate(validate *validator.Validate) error {
	if !c.Enabled {
		return nil
	}

	if err := validate.Struct(c); err != nil {
		return err
	}

	if c.EnableMailServer {
		if c.DataDir == "" {
			return fmt.Errorf("WhisperConfig.DataDir must be specified when WhisperConfig.EnableMailServer is true")
		}
		if c.MailServerPassword == "" && c.MailServerAsymKey == "" {
			return fmt.Errorf("WhisperConfig.MailServerPassword or WhisperConfig.MailServerAsymKey must be specified when WhisperConfig.EnableMailServer is true")
		}

		if c.MailServerAsymKey != "" {
			if _, err := crypto.HexToECDSA(c.MailServerAsymKey); err != nil {
				return fmt.Errorf("WhisperConfig.MailServerAsymKey is invalid: %s", c.MailServerAsymKey)
			}
		}
	}

	return nil
}

// Validate validates the SwarmConfig struct and returns an error if inconsistent values are found
func (c *SwarmConfig) Validate(validate *validator.Validate) error {
	if !c.Enabled {
		return nil
	}

	if err := validate.Struct(c); err != nil {
		return err
	}

	return nil
}

func getUpstreamURL(networkID uint64) string {
	switch networkID {
	case MainNetworkID:
		return MainnetEthereumNetworkURL
	case RopstenNetworkID:
		return RopstenEthereumNetworkURL
	case RinkebyNetworkID:
		return RinkebyEthereumNetworkURL
	}

	return ""
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

	c.log.Info("config file saved", "path", configFilePath)
	return nil
}

// String dumps config object as nicely indented JSON
func (c *NodeConfig) String() string {
	data, _ := json.MarshalIndent(c, "", "    ")
	return string(data)
}

// FormatAPIModules returns a slice of APIModules.
func (c *NodeConfig) FormatAPIModules() []string {
	if len(c.APIModules) == 0 {
		return nil
	}

	return strings.Split(c.APIModules, ",")
}

// AddAPIModule adds a mobule to APIModules
func (c *NodeConfig) AddAPIModule(m string) {
	c.APIModules = fmt.Sprintf("%s,%s", c.APIModules, m)
}

// GetStatusHome gets home directory of status-go
func GetStatusHome() string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	return path.Join(gopath, "/src/github.com/status-im/status-go/")
}

// LesTopic returns discovery v5 topic derived from genesis of the provided network.
// 1 - mainnet, 3 - ropsten, 4 - rinkeby
func LesTopic(netid int) string {
	switch netid {
	case 1:
		return LESDiscoveryIdentifier + common.Bytes2Hex(params.MainnetGenesisHash.Bytes()[:8])
	case 3:
		return LESDiscoveryIdentifier + common.Bytes2Hex(params.TestnetGenesisHash.Bytes()[:8])
	case 4:
		return LESDiscoveryIdentifier + common.Bytes2Hex(params.RinkebyGenesisHash.Bytes()[:8])
	default:
		return ""
	}
}
