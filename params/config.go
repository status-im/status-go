package params

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	validator "gopkg.in/go-playground/validator.v9"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/params"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/static"
	"github.com/status-im/status-go/whisper/v6"
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

	// TrustedNodes is a list of trusted servers
	TrustedNodes []string

	//MinTrustedFraction is minimum percentage of connected trusted servers to validate header(1-100)
	MinTrustedFraction int
}

// ----------
// DatabaseConfig
// ----------

type DatabaseConfig struct {
	PGConfig PGConfig
}

// ----------
// PGConfig
// ----------

type PGConfig struct {
	// Enabled whether we should use a Postgres instance
	Enabled bool
	// The URI of the server
	URI string
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

	// MailServerPassword for symmetric encryption of whisper message history requests.
	// (if no account file selected, then this password is used for symmetric encryption).
	MailServerPassword string

	// MailServerAsymKey is an hex-encoded asymmetric key to decrypt messages sent to MailServer.
	MailServerAsymKey string

	// MailServerRateLimit minimum time between queries to mail server per peer.
	MailServerRateLimit int

	// MailServerDataRetention is a number of days data should be stored by MailServer.
	MailServerDataRetention int

	// TTL time to live for messages, in seconds
	TTL int

	// MaxMessageSize is a maximum size of a devp2p packet handled by the Whisper protocol,
	// not only the size of envelopes sent in that packet.
	MaxMessageSize uint32

	// DatabaseConfig is configuration for which datastore we use
	DatabaseConfig DatabaseConfig

	// EnableRateLimiter set to true enables IP and peer ID rate limiting.
	EnableRateLimiter bool

	// RateLimitIP sets the limit on the number of messages per second
	// from a given IP.
	RateLimitIP int64

	// RateLimitPeerID sets the limit on the number of messages per second
	// from a given peer ID.
	RateLimitPeerID int64

	// RateLimitTolerance is a number of how many a limit must be exceeded
	// in order to drop a peer.
	// If equal to 0, the peers are never dropped.
	RateLimitTolerance int64
}

// String dumps config object as nicely indented JSON
func (c *WhisperConfig) String() string {
	data, _ := json.MarshalIndent(c, "", "    ") // nolint: gas
	return string(data)
}

// ----------
// WakuConfig
// ----------

// WakuConfig provides a configuration for Waku service.
type WakuConfig struct {
	// Enabled set to true enables Waku subprotocol.
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

	// MailServerPassword for symmetric encryption of whisper message history requests.
	// (if no account file selected, then this password is used for symmetric encryption).
	MailServerPassword string

	// MailServerRateLimit minimum time between queries to mail server per peer.
	MailServerRateLimit int

	// MailServerDataRetention is a number of days data should be stored by MailServer.
	MailServerDataRetention int

	// TTL time to live for messages, in seconds
	TTL int

	// MaxMessageSize is a maximum size of a devp2p packet handled by the Whisper protocol,
	// not only the size of envelopes sent in that packet.
	MaxMessageSize uint32

	// DatabaseConfig is configuration for which data store we use.
	DatabaseConfig DatabaseConfig

	// EnableRateLimiter set to true enables IP and peer ID rate limiting.
	EnableRateLimiter bool

	// RateLimitIP sets the limit on the number of messages per second
	// from a given IP.
	RateLimitIP int64

	// RateLimitPeerID sets the limit on the number of messages per second
	// from a given peer ID.
	RateLimitPeerID int64

	// RateLimitTolerance is a number of how many a limit must be exceeded
	// in order to drop a peer.
	// If equal to 0, the peers are never dropped.
	RateLimitTolerance int64
}

// IncentivisationConfig holds incentivisation-related configuration
type IncentivisationConfig struct {
	// Enabled flag specifies whether protocol is enabled
	Enabled bool `validate:"required"`
	// Endpoint for the RPC calls
	RPCEndpoint string `validate:"required"`
	// Contract address
	ContractAddress string `validate:"required"`
	// IP address that is used
	IP string `validate:"required"`
	// Port
	Port uint16 `validate:"required"`
}

// String dumps config object as nicely indented JSON
func (c *IncentivisationConfig) String() string {
	data, _ := json.MarshalIndent(c, "", "    ") // nolint: gas
	return string(data)
}

// Validate validates the IncentivisationConfig struct and returns an error if inconsistent values are found
func (c *IncentivisationConfig) Validate(validate *validator.Validate) error {
	if err := validate.Struct(c); err != nil {
		return err
	}
	return nil
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
	// Enabled flag specifies that nodes in this configuration are taken into account.
	Enabled bool

	// Fleet is a name of a selected fleet. If it has a value, nodes are loaded
	// from a file, namely `fleet-*.{{ .Fleet }}.json`. Nodes can be added to any list
	// in `ClusterConfig`.
	Fleet string

	// StaticNodes is a list of static nodes.
	StaticNodes []string

	// BootNodes is a list of bootnodes.
	BootNodes []string

	// TrustedMailServers is a list of verified and trusted Mail Server nodes.
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

	// HTTPEnabled specifies whether the http RPC server is to be enabled by default.
	HTTPEnabled bool

	// HTTPHost is the host interface on which to start the HTTP RPC server.
	// Pass empty string if no HTTP RPC interface needs to be started.
	HTTPHost string

	// HTTPPort is the TCP port number on which to start the Geth's HTTP RPC server.
	HTTPPort int

	// HTTPVirtualHosts is the list of virtual hostnames which are allowed on incoming requests.
	// This is by default {'localhost'}. Using this prevents attacks like
	// DNS rebinding, which bypasses SOP by simply masquerading as being within the same
	// origin. These attacks do not utilize CORS, since they are not cross-domain.
	// By explicitly checking the Host-header, the server will not allow requests
	// made against the server with a malicious host domain.
	// Requests using an IP address directly are not affected.
	HTTPVirtualHosts []string

	// HTTPCors is the Cross-Origin Resource Sharing header to send to requesting
	// clients. Please be aware that CORS is a browser enforced security, it's fully
	// useless for custom HTTP clients.
	HTTPCors []string

	// IPCEnabled specifies whether IPC-RPC Server is enabled or not
	IPCEnabled bool

	// IPCFile is filename of exposed IPC RPC Server
	IPCFile string

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

	// LogMobileSystem enables log redirection to android/ios system logger.
	LogMobileSystem bool

	// LogFile is filename where exposed logs get written to
	LogFile string

	// LogLevel defines minimum log level. Valid names are "ERROR", "WARN", "INFO", "DEBUG", and "TRACE".
	LogLevel string `validate:"eq=ERROR|eq=WARN|eq=INFO|eq=DEBUG|eq=TRACE"`

	// LogMaxBackups defines number of rotated log files that will be stored.
	LogMaxBackups int

	// LogMaxSize in megabytes after current size is reached log file will be rotated.
	LogMaxSize int

	// LogCompressRotated if true all rotated files will be gzipped.
	LogCompressRotated bool

	// LogToStderr defines whether logged info should also be output to os.Stderr
	LogToStderr bool

	// EnableStatusService should be true to enable methods under status namespace.
	EnableStatusService bool

	// EnableNTPSync enables NTP synchronizations
	EnableNTPSync bool

	// UpstreamConfig extra config for providing upstream infura server.
	UpstreamConfig UpstreamRPCConfig `json:"UpstreamConfig"`

	// ClusterConfig extra configuration for supporting cluster peers.
	ClusterConfig ClusterConfig `json:"ClusterConfig," validate:"structonly"`

	// LightEthConfig extra configuration for LES
	LightEthConfig LightEthConfig `json:"LightEthConfig," validate:"structonly"`

	// WhisperConfig extra configuration for SHH
	WhisperConfig WhisperConfig `json:"WhisperConfig," validate:"structonly"`

	// WakuConfig provides a configuration for Waku subprotocol.
	WakuConfig WakuConfig `json:"WakuConfig" validate:"structonly"`

	// IncentivisationConfig extra configuration for incentivisation service
	IncentivisationConfig IncentivisationConfig `json:"IncentivisationConfig," validate:"structonly"`

	// ShhextConfig extra configuration for service running under shhext namespace.
	ShhextConfig ShhextConfig `json:"ShhextConfig," validate:"structonly"`

	// WalletConfig extra configuration for wallet.Service.
	WalletConfig WalletConfig

	// BrowsersConfig extra configuration for browsers.Service.
	BrowsersConfig BrowsersConfig

	// PermissionsConfig extra configuration for permissions.Service.
	PermissionsConfig PermissionsConfig

	// MailserversConfig extra configuration for mailservers.Service
	// (persistent storage of user's mailserver records).
	MailserversConfig MailserversConfig

	// SwarmConfig extra configuration for Swarm and ENS
	SwarmConfig SwarmConfig `json:"SwarmConfig," validate:"structonly"`

	// RegisterTopics a list of specific topics where the peer wants to be
	// discoverable.
	RegisterTopics []discv5.Topic `json:"RegisterTopics"`

	// RequiredTopics list of topics where a client wants to search for
	// discoverable peers with the discovery limits.
	RequireTopics map[discv5.Topic]Limits `json:"RequireTopics"`

	// MailServerRegistryAddress is the MailServerRegistry contract address
	MailServerRegistryAddress string
}

// WalletConfig extra configuration for wallet.Service.
type WalletConfig struct {
	Enabled bool
}

// BrowsersConfig extra configuration for browsers.Service.
type BrowsersConfig struct {
	Enabled bool
}

// PermissionsConfig extra configuration for permissions.Service.
type PermissionsConfig struct {
	Enabled bool
}

// MailserversConfig extra configuration for mailservers.Service.
type MailserversConfig struct {
	Enabled bool
}

// ShhextConfig defines options used by shhext service.
type ShhextConfig struct {
	PFSEnabled bool
	// BackupDisabledDataDir is the file system folder the node should use for any data storage needs that it doesn't want backed up.
	BackupDisabledDataDir string
	// InstallationId id of the current installation
	InstallationID string
	// MailServerConfirmations should be true if client wants to receive confirmatons only from a selected mail servers.
	MailServerConfirmations bool
	// EnableConnectionManager turns on management of the mail server connections if true.
	EnableConnectionManager bool
	// EnableLastUsedMonitor guarantees that last used mail server will be tracked and persisted into the storage.
	EnableLastUsedMonitor bool
	// ConnectionTarget will be used by connection manager. It will ensure that we connected with configured number of servers.
	ConnectionTarget int
	// RequestsDelay used to ensure that no similar requests are sent within short periods of time.
	RequestsDelay time.Duration

	// MaxServerFailures defines maximum allowed expired requests before server will be swapped to another one.
	MaxServerFailures int

	// MaxMessageDeliveryAttempts defines how many times we will try to deliver not-acknowledged envelopes.
	MaxMessageDeliveryAttempts int

	// WhisperCacheDir is a folder where whisper filters may persist messages before delivering them
	// to a client.
	WhisperCacheDir string

	// DisableGenericDiscoveryTopic indicates whether we should be listening on the old discovery
	DisableGenericDiscoveryTopic bool

	// SendV1Messages indicates whether we should be sending v1-compatible only messages
	SendV1Messages bool

	// DatasyncEnabled indicates whether we should enable dataasync
	DataSyncEnabled bool

	// VerifyTransactionURL is the URL for verifying transactions.
	// IMPORTANT: It should always be mainnet unless used for testing
	VerifyTransactionURL     string
	VerifyTransactionChainID int64
}

// Validate validates the ShhextConfig struct and returns an error if inconsistent values are found
func (c *ShhextConfig) Validate(validate *validator.Validate) error {
	if err := validate.Struct(c); err != nil {
		return err
	}
	if c.PFSEnabled && len(c.BackupDisabledDataDir) == 0 {
		return errors.New("field BackupDisabledDataDir is required if PFSEnabled is true")
	}
	return nil
}

// Option is an additional setting when creating a NodeConfig
// using NewNodeConfigWithDefaults.
type Option func(*NodeConfig) error

// WithFleet loads one of the preconfigured Status fleets.
func WithFleet(fleet string) Option {
	return func(c *NodeConfig) error {
		if fleet == FleetUndefined {
			return nil
		}
		c.NoDiscovery = false
		c.ClusterConfig.Enabled = true
		return loadConfigFromAsset(fmt.Sprintf("../config/cli/fleet-%s.json", fleet), c)
	}
}

// WithLES enabled LES protocol.
func WithLES() Option {
	return func(c *NodeConfig) error {
		return loadConfigFromAsset("../config/cli/les-enabled.json", c)
	}
}

// WithMailserver enables MailServer.
func WithMailserver() Option {
	return func(c *NodeConfig) error {
		return loadConfigFromAsset("../config/cli/mailserver-enabled.json", c)
	}
}

// NewNodeConfigWithDefaults creates new node configuration object
// with some defaults suitable for adhoc use.
func NewNodeConfigWithDefaults(dataDir string, networkID uint64, opts ...Option) (*NodeConfig, error) {
	c, err := NewNodeConfig(dataDir, networkID)
	if err != nil {
		return nil, err
	}

	c.NoDiscovery = true
	c.HTTPHost = ""
	c.ListenAddr = ":30303"
	c.LogEnabled = true
	c.LogLevel = "INFO"
	c.LogMaxSize = 100
	c.LogCompressRotated = true
	c.LogMaxBackups = 3
	c.LogToStderr = true
	c.EnableNTPSync = true
	c.WhisperConfig.Enabled = true

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	c.updatePeerLimits()

	if err := c.Validate(); err != nil {
		return nil, err
	}

	return c, nil
}

// NewNodeConfigWithDefaultsAndFiles creates new node configuration object
// with some defaults suitable for adhoc use and applies config files on top.
func NewNodeConfigWithDefaultsAndFiles(
	dataDir string, networkID uint64, opts []Option, files []string,
) (*NodeConfig, error) {
	c, err := NewNodeConfigWithDefaults(dataDir, networkID, opts...)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if err := loadConfigConfigFromFile(file, c); err != nil {
			return nil, err
		}
	}

	c.updatePeerLimits()

	if err := c.Validate(); err != nil {
		return nil, err
	}

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

// NewNodeConfig creates new node configuration object with bare-minimum defaults.
// Important: the returned config is not validated.
func NewNodeConfig(dataDir string, networkID uint64) (*NodeConfig, error) {
	var keyStoreDir, wnodeDir string

	if dataDir != "" {
		keyStoreDir = filepath.Join(dataDir, "keystore")
	}
	if dataDir != "" {
		wnodeDir = filepath.Join(dataDir, "wnode")
	}

	config := &NodeConfig{
		NetworkID:        networkID,
		DataDir:          dataDir,
		KeyStoreDir:      keyStoreDir,
		Version:          Version,
		HTTPHost:         "localhost",
		HTTPPort:         8545,
		HTTPVirtualHosts: []string{"localhost"},
		ListenAddr:       ":0",
		APIModules:       "eth,net,web3,peer,wallet",
		MaxPeers:         25,
		MaxPendingPeers:  0,
		IPCFile:          "geth.ipc",
		log:              log.New("package", "status-go/params.NodeConfig"),
		LogFile:          "",
		LogLevel:         "ERROR",
		NoDiscovery:      true,
		EnableNTPSync:    true,
		UpstreamConfig: UpstreamRPCConfig{
			URL: getUpstreamURL(networkID),
		},
		LightEthConfig: LightEthConfig{
			DatabaseCache: 16,
		},
		WhisperConfig: WhisperConfig{
			DataDir:        wnodeDir,
			MinimumPoW:     WhisperMinimumPoW,
			TTL:            WhisperTTL,
			MaxMessageSize: whisper.DefaultMaxMessageSize,
		},
		ShhextConfig: ShhextConfig{
			BackupDisabledDataDir: dataDir,
		},
		SwarmConfig:    SwarmConfig{},
		RegisterTopics: []discv5.Topic{},
		RequireTopics:  map[discv5.Topic]Limits{},
	}

	return config, nil
}

// NewConfigFromJSON parses incoming JSON and returned it as Config
func NewConfigFromJSON(configJSON string) (*NodeConfig, error) {
	config, err := NewNodeConfig("", 0)
	if err != nil {
		return nil, err
	}

	if err := loadConfigFromJSON(configJSON, config); err != nil {
		return nil, err
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func loadConfigFromJSON(configJSON string, nodeConfig *NodeConfig) error {
	decoder := json.NewDecoder(strings.NewReader(configJSON))
	// override default configuration with values by JSON input
	return decoder.Decode(&nodeConfig)
}

func loadConfigConfigFromFile(path string, config *NodeConfig) error {
	jsonConfig, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return loadConfigFromJSON(string(jsonConfig), config)
}

func loadConfigFromAsset(name string, config *NodeConfig) error {
	data, err := static.Asset(name)
	if err != nil {
		return err
	}
	return loadConfigFromJSON(string(data), config)
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
// nolint: gocyclo
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

	if c.WhisperConfig.Enabled && c.WakuConfig.Enabled && c.WhisperConfig.DataDir == c.WakuConfig.DataDir {
		return fmt.Errorf("both Whisper and Waku are enabled and use the same data dir")
	}

	// Whisper's data directory must be relative to the main data directory
	// if EnableMailServer is true.
	if c.WhisperConfig.Enabled && c.WhisperConfig.EnableMailServer {
		if !strings.HasPrefix(c.WhisperConfig.DataDir, c.DataDir) {
			return fmt.Errorf("WhisperConfig.DataDir must start with DataDir fragment")
		}
	}

	if !c.NoDiscovery && len(c.ClusterConfig.BootNodes) == 0 {
		// No point in running discovery if we don't have bootnodes.
		// In case we do have bootnodes, NoDiscovery should be true.
		return fmt.Errorf("NoDiscovery is false, but ClusterConfig.BootNodes is empty")
	}

	if c.ShhextConfig.PFSEnabled && len(c.ShhextConfig.InstallationID) == 0 {
		return fmt.Errorf("PFSEnabled is true, but InstallationID is empty")
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
	if err := c.ShhextConfig.Validate(validate); err != nil {
		return err
	}
	if c.IncentivisationConfig.Enabled {
		if err := c.IncentivisationConfig.Validate(validate); err != nil {
			return err
		}
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
	case GoerliNetworkID:
		return GoerliEthereumNetworkURL
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

// LesTopic returns discovery v5 topic derived from genesis of the provided network.
// 1 - mainnet, 3 - ropsten, 4 - rinkeby
func LesTopic(netid int) string {
	switch netid {
	case 1:
		return LESDiscoveryIdentifier + types.Bytes2Hex(params.MainnetGenesisHash.Bytes()[:8])
	case 3:
		return LESDiscoveryIdentifier + types.Bytes2Hex(params.TestnetGenesisHash.Bytes()[:8])
	case 4:
		return LESDiscoveryIdentifier + types.Bytes2Hex(params.RinkebyGenesisHash.Bytes()[:8])
	default:
		return ""
	}
}
