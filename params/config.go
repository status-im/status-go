package params

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"gopkg.in/go-playground/validator.v9"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/pkg/security"
	wakuv2common "github.com/status-im/status-go/wakuv2/common"
)

// ----------
// WakuV2Config
// ----------

// WakuV2Config provides a configuration for Waku service.
type WakuV2Config struct {
	// Enabled set to true enables Waku subprotocol.
	Enabled bool

	// Host interface in which to start libp2p protocol
	Host string

	// Port number in which to start libp2p protocol (0 for random)
	Port int

	// LightClient should be true if the node will not relay messages and only rely on lightpush/filter nodes
	LightClient bool

	// DiscoveryLimit indicates the maximum number of peers to discover
	DiscoveryLimit int

	// MaxMessageSize is a maximum size of a devp2p packet handled by the Waku protocol,
	// not only the size of envelopes sent in that packet.
	MaxMessageSize uint32

	// Nameserver determines which nameserver will be used for dns discovery
	Nameserver string

	// UDPPort number to start discovery v5
	UDPPort int

	// AutoUpdate instructs the node to update their own ip address and port with the values seen by other nodes
	AutoUpdate bool

	// EnableMissingMessageVerification indicates whether the storenodes must be queried periodically to retrieve any missing message
	EnableMissingMessageVerification bool

	// EnableMissingMessageVerification indicates whether storenodes must be queried periodically to confirm if messages sent are actually propagated in the network
	EnableStoreConfirmationForMessagesSent bool
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

	// Fleet is a name of a selected fleet.
	// An according fleet configuration is loaded from hard-coded lists.
	Fleet string

	// WakuNodes is a list of waku2 multiaddresses
	WakuNodes []string

	// DiscV5Nodes is a list of enr to be used for ambient discovery
	DiscV5BootstrapNodes []string

	//Waku network identifier
	ClusterID uint16
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

// ----------
// NodeConfig
// ----------

// NodeConfig stores configuration options for a node
type NodeConfig struct {
	// NetworkID sets network to use for selecting peers to connect to
	NetworkID uint64 `json:"NetworkId" validate:"required"`

	RootDataDir string `json:",omitempty"`

	// DataDir is the file system folder the node should use for any data storage needs.
	DataDir string `validate:"required"`

	// KeycardPairingDataFile is the file where we keep keycard pairings data.
	// It's specified by clients (and not in status-go) when creating a new account,
	// because this file is initialized by status-keycard-go and we need to use it before initializing the node.
	// I guess proper way would be to ask status-go for the file path, or just duplicate the file path in both backend and client.
	// note: this field won't be saved into db, it's local to the device.
	KeycardPairingDataFile string

	// NodeKey is the hex-encoded node ID (private key). Should be a valid secp256k1 private key that will be used for both
	// remote peer identification as well as network traffic encryption.
	NodeKey string

	// APIModules is a comma-separated list of API modules exposed via *any* (HTTP/WS/IPC) RPC interface.
	APIModules string `validate:"required"`

	// HTTPEnabled specifies whether the http RPC server is to be enabled by default.
	HTTPEnabled bool

	// HTTPHost is the host interface on which to start the HTTP RPC server.
	// Pass empty string if no HTTP RPC interface needs to be started.
	HTTPHost string

	// HTTPPort is the TCP port number on which to start the Geth's HTTP RPC server.
	HTTPPort int

	// WSEnabled specifies whether the Websocket RPC server is to be enabled by default.
	WSEnabled bool

	// WSHost is the host interface on which to start Geth's Websocket RPC server.
	WSHost string

	// WSPort is the TCP port number on which to start the Geth's Websocket RPC server.
	WSPort int

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

	// LogEnabled enables the logger
	LogEnabled bool `json:"LogEnabled"`

	// LogFile is a folder which contains LogFile
	LogDir string

	// LogFile is filename where exposed logs get written to
	LogFile string

	// RuntimeLoglevel defines minimum log level for this session only, not affecting the db-stored node configuration
	RuntimeLogLevel string `validate:"omitempty,eq=ERROR|eq=WARN|eq=INFO|eq=DEBUG|eq=TRACE"`

	// LogLevel defines minimum log level. Valid names are "ERROR", "WARN", "INFO", "DEBUG", and "TRACE".
	LogLevel string `validate:"eq=ERROR|eq=WARN|eq=INFO|eq=DEBUG|eq=TRACE"`

	// LogNamespaces defines log level per namespace. Example: "namespace1:debug,namespace2.namespace3:error"
	// It doesn't affect LogLevel for unmentioned namespaces.
	LogNamespaces string

	// LogMaxBackups defines number of rotated log files that will be stored.
	LogMaxBackups int

	// LogMaxSize in megabytes after current size is reached log file will be rotated.
	LogMaxSize int

	// LogCompressRotated if true all rotated files will be gzipped.
	LogCompressRotated bool

	// LogToStderr defines whether logged info should also be output to os.Stderr
	LogToStderr bool

	// Initial networks to load
	Networks []Network

	// ClusterConfig extra configuration for supporting cluster peers.
	ClusterConfig ClusterConfig `json:"ClusterConfig," validate:"structonly"`

	// WakuV2Config provides a configuration for WakuV2 protocol.
	WakuV2Config WakuV2Config `json:"WakuV2Config" validate:"structonly"`

	// ShhextConfig extra configuration for service running under shhext namespace.
	ShhextConfig ShhextConfig `json:"ShhextConfig," validate:"structonly"`

	// WalletConfig extra configuration for wallet.Service.
	WalletConfig WalletConfig

	// BrowsersConfig extra configuration for browsers.Service.
	BrowsersConfig BrowsersConfig

	// PermissionsConfig extra configuration for permissions.Service.
	PermissionsConfig PermissionsConfig

	// ConnectorConfig extra configuration for connector.Service
	ConnectorConfig ConnectorConfig

	TorrentConfig TorrentConfig

	OutputMessageCSVEnabled bool

	// ProcessBackedupMessages should be set to true when user follows recovery (using seed phrase or keycard) onboarding flow
	ProcessBackedupMessages bool
}

// WalletConfig extra configuration for wallet.Service.
type WalletConfig struct {
	Enabled                   bool
	OpenseaAPIKey             security.SensitiveString            `json:"OpenseaAPIKey"`
	RaribleMainnetAPIKey      security.SensitiveString            `json:"RaribleMainnetAPIKey"`
	RaribleTestnetAPIKey      security.SensitiveString            `json:"RaribleTestnetAPIKey"`
	AlchemyAPIKeys            map[uint64]security.SensitiveString `json:"AlchemyAPIKeys"`
	InfuraAPIKey              security.SensitiveString            `json:"InfuraAPIKey"`
	InfuraAPIKeySecret        security.SensitiveString            `json:"InfuraAPIKeySecret"`
	StatusProxyMarketUser     security.SensitiveString            `json:"StatusProxyMarketUser"`
	StatusProxyMarketPassword security.SensitiveString            `json:"StatusProxyMarketPassword"`
	MarketDataProxyConfig     MarketDataProxyConfig               `json:"MarketDataProxyConfig"`
	// FIXME: remove when EthRpcProxy* is integrated
	StatusProxyBlockchainUser     security.SensitiveString `json:"StatusProxyBlockchainUser"`
	StatusProxyBlockchainPassword security.SensitiveString `json:"StatusProxyBlockchainPassword"`

	StatusProxyStageName   string                   `json:"StatusProxyStageName"`
	EnableCelerBridge      bool                     `json:"EnableCelerBridge"`
	EnableMercuryoProvider bool                     `json:"EnableMercuryoProvider"`
	EthRpcProxyUrl         security.SensitiveString `json:"EthRpcProxyUrl"`
	EthRpcProxyUser        security.SensitiveString `json:"EthRpcProxyUser"`
	EthRpcProxyPassword    security.SensitiveString `json:"EthRpcProxyPassword"`

	TokensListsAutoRefreshInterval      int `json:"TokensListsAutoRefreshInterval"`      // in seconds
	TokensListsAutoRefreshCheckInterval int `json:"TokensListsAutoRefreshCheckInterval"` // in seconds
}

type MarketDataProxyConfig struct {
	UrlOverride             security.SensitiveString `json:"UrlOverride"`
	StageName               string                   `json:"StageName"`
	User                    security.SensitiveString `json:"User"`
	Password                security.SensitiveString `json:"Password"`
	FullDataRefreshInterval int                      `json:"FullDataRefreshInterval"`
	PriceRefreshInterval    int                      `json:"PriceRefreshInterval"`
}

// MarshalJSON custom marshalling to avoid exposing sensitive data in log,
// there's a function called `startNode` will log NodeConfig which include WalletConfig
func (wc WalletConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Enabled                             bool `json:"Enabled"`
		EnableCelerBridge                   bool `json:"EnableCelerBridge"`
		EnableMercuryoProvider              bool `json:"EnableMercuryoProvider"`
		TokensListsAutoRefreshInterval      int  `json:"TokensListsAutoRefreshInterval"`
		TokensListsAutoRefreshCheckInterval int  `json:"TokensListsAutoRefreshCheckInterval"`
	}{
		Enabled:                             wc.Enabled,
		EnableCelerBridge:                   wc.EnableCelerBridge,
		EnableMercuryoProvider:              wc.EnableMercuryoProvider,
		TokensListsAutoRefreshInterval:      wc.TokensListsAutoRefreshInterval,
		TokensListsAutoRefreshCheckInterval: wc.TokensListsAutoRefreshCheckInterval,
	})
}

// BrowsersConfig extra configuration for browsers.Service.
type BrowsersConfig struct {
	Enabled bool
}

// PermissionsConfig extra configuration for permissions.Service.
type PermissionsConfig struct {
	Enabled bool
}

// ConnectorConfig extra configuration for provider.Service
type ConnectorConfig struct {
	Enabled bool
}

type PushNotificationServer struct {
	*ecdsa.PublicKey
}

func (p *PushNotificationServer) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(crypto.FromECDSAPub(p.PublicKey))), nil
}

func (p *PushNotificationServer) UnmarshalText(data []byte) error {
	pubKeyBytes, err := hex.DecodeString(string(data))
	if err != nil {
		return err
	}

	pk, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		return err
	}

	p.PublicKey = pk
	return nil
}

// ShhextConfig defines options used by shhext service.
type ShhextConfig struct {
	PFSEnabled bool
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
	VerifyTransactionURL string

	// VerifyENSURL is the URL for verifying ens names.
	// IMPORTANT: It should always be mainnet unless used for testing
	VerifyENSURL string

	// VerifyENSContractAddress is the address of the contract used to verify ENS
	// No default is provided and if not set ENS resolution is disabled
	VerifyENSContractAddress string

	VerifyTransactionChainID int64

	// DefaultPushNotificationsServers is the default-status run push notification servers
	DefaultPushNotificationsServers []*PushNotificationServer

	// AnonMetricsSendID is the public key used by a metrics node to decrypt metrics protobufs
	AnonMetricsSendID string

	// AnonMetricsServerEnabled indicates whether or not the
	AnonMetricsServerEnabled bool

	// AnonMetricsServerPostgresURI is the uri used to connect to a postgres db
	AnonMetricsServerPostgresURI string

	// BandwidthStatsEnabled indicates if a signal is going to be emitted to indicate the upload and download rate
	BandwidthStatsEnabled bool
}

// TorrentConfig provides configuration for the BitTorrent client used for message history archives.
type TorrentConfig struct {
	// Enabled set to true enables Community History Archive protocol
	Enabled bool
	// Port number which the BitTorrent client will listen to for conntections
	Port int
	// DataDir is the file system folder Status should use for message archive torrent data.
	DataDir string
	// TorrentDir is the file system folder Status should use for storing torrent metadata files.
	TorrentDir string
}

// Validate validates the ShhextConfig struct and returns an error if inconsistent values are found
func (c *ShhextConfig) Validate(validate *validator.Validate) error {
	if err := validate.Struct(c); err != nil {
		return err
	}
	return nil
}

func (c *NodeConfig) setDefaultPushNotificationsServers() error {
	if len(c.ShhextConfig.DefaultPushNotificationsServers) > 0 {
		return nil
	}

	servers := DefaultPushNotificationServers()

	// If empty set the default servers
	logutils.ZapLogger().Debug("setting default push notification servers",
		zap.Strings("servers", servers))

	for _, pk := range servers {
		keyBytes, err := hex.DecodeString("04" + pk)
		if err != nil {
			return err
		}

		key, err := crypto.UnmarshalPubkey(keyBytes)
		if err != nil {
			return err
		}

		c.ShhextConfig.DefaultPushNotificationsServers = append(
			c.ShhextConfig.DefaultPushNotificationsServers,
			&PushNotificationServer{PublicKey: key},
		)
	}

	return nil
}

// UpdateWithDefaults updates config with missing default values, as
// the config is only generated once and is thereafter pulled from the database.
// The way it is stored in the database makes this step necessary as it's stored as a blob and can't be easily migrated.
func (c *NodeConfig) UpdateWithDefaults() error {
	// Empty APIModules will fallback to services' APIs definition.
	// If any API is defined as public, it will be exposed.
	// We disallow empty APIModules to avoid confusion
	// when some APIs suddenly become available for Dapps.
	// More: https://github.com/status-im/status-go/issues/1870.
	if c.APIModules == "" {
		c.APIModules = "net,web3,eth"
	}

	// Ensure TorrentConfig is valid
	if c.TorrentConfig.Enabled {
		if c.TorrentConfig.DataDir == "" {
			c.TorrentConfig.DataDir = filepath.Join(c.RootDataDir, ArchivesRelativePath)
		}
		if c.TorrentConfig.TorrentDir == "" {
			c.TorrentConfig.TorrentDir = filepath.Join(c.RootDataDir, TorrentTorrentsRelativePath)
		}
	}

	return c.setDefaultPushNotificationsServers()
}

// NewNodeConfig creates new node configuration object with bare-minimum defaults.
// Important: the returned config is not validated.
func NewNodeConfig(dataDir string, networkID uint64) (*NodeConfig, error) {
	var keycardPairingDataFile string

	if dataDir != "" {
		keycardPairingDataFile = filepath.Join(dataDir, "keycard", "pairings.json")
	}

	config := &NodeConfig{
		NetworkID:              networkID,
		RootDataDir:            dataDir,
		DataDir:                dataDir,
		KeycardPairingDataFile: keycardPairingDataFile,
		HTTPHost:               "localhost",
		HTTPPort:               8545,
		HTTPVirtualHosts:       []string{"localhost"},
		APIModules:             "eth,net,web3,peer,wallet",
		LogFile:                "",
		LogLevel:               "ERROR",
		WakuV2Config: WakuV2Config{
			Host:           "0.0.0.0",
			Port:           0,
			MaxMessageSize: wakuv2common.DefaultMaxMessageSize,
		},
		ShhextConfig: ShhextConfig{},
		TorrentConfig: TorrentConfig{
			Enabled:    false,
			Port:       9025,
			DataDir:    dataDir + "/archivedata",
			TorrentDir: dataDir + "/torrents",
		},
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

// Validate checks if NodeConfig fields have valid values.
//
// It returns nil if there are no errors, otherwise one or more errors
// can be returned. Multiple errors are joined with a new line.
//
// A single error for a struct:
//
//	type TestStruct struct {
//	    TestField string `validate:"required"`
//	}
//
// has the following format:
//
//	Key: 'TestStruct.TestField' Error:Field validation for 'TestField' failed on the 'required' tag
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

	if err := c.validateChildStructs(validate); err != nil {
		return err
	}

	if c.ShhextConfig.PFSEnabled && len(c.ShhextConfig.InstallationID) == 0 {
		return fmt.Errorf("PFSEnabled is true, but InstallationID is empty")
	}

	return nil
}

func (c *NodeConfig) validateChildStructs(validate *validator.Validate) error {
	// Validate child structs
	if err := c.ClusterConfig.Validate(validate); err != nil {
		return err
	}
	if err := c.ShhextConfig.Validate(validate); err != nil {
		return err
	}
	if err := c.TorrentConfig.Validate(validate); err != nil {
		return err
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

func (c *TorrentConfig) Validate(validate *validator.Validate) error {
	if !c.Enabled {
		return nil
	}

	if err := validate.Struct(c); err != nil {
		return err
	}

	if c.Enabled && (c.DataDir == "" || c.TorrentDir == "") {
		return fmt.Errorf("TorrentConfig.DataDir and TorrentConfig.TorrentDir cannot be \"\"")
	}
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

func (c *NodeConfig) LogFilePath() string {
	if c.LogDir == "" {
		return filepath.Join(c.RootDataDir, c.LogFile)
	}
	return filepath.Join(c.LogDir, c.LogFile)
}

func (c *NodeConfig) ProfileLogSettings() logutils.LogSettings {
	return logutils.LogSettings{
		Enabled:         c.LogEnabled,
		Level:           c.LogLevel,
		Namespaces:      c.LogNamespaces,
		File:            c.LogFilePath(),
		MaxSize:         c.LogMaxSize,
		MaxBackups:      c.LogMaxBackups,
		CompressRotated: c.LogCompressRotated,
		LogToStderr:     c.LogToStderr,
	}
}
