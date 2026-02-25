package params

type LogosStorageNodeConfig struct {
	// Default: INFO
	LogLevel string

	// Specifies what kind of logs should be written to stdout
	// Default: auto
	LogFormat string

	// Enable the metrics server
	// Default: false
	MetricsEnabled bool

	// Listening address of the metrics server
	// Default: 127.0.0.1
	MetricsAddress string

	// Listening HTTP port of the metrics server
	// Default: 8008
	MetricsPort int

	// The directory where Logos Storage will store configuration and data
	// Default:
	// $HOME\AppData\Roaming\Storage on Windows
	// $HOME/Library/Application Support/Storage on macOS
	// $HOME/.cache/storage on Linux
	DataDir string

	// Multi Addresses to listen on
	// Default: ["/ip4/0.0.0.0/tcp/0"]
	ListenAddrs []string

	// Specify method to use for determining public address.
	// Must be one of: any, none, upnp, pmp, extip:<IP>
	// Default: any
	Nat string

	// Discovery (UDP) port
	// Default: 8090
	DiscoveryPort int

	// Source of network (secp256k1) private key file path or name
	// Default: "key"
	NetPrivKeyFile string

	// Specifies one or more bootstrap nodes to use when connecting to the network.
	BootstrapNodes []string

	// The maximum number of peers to connect to.
	// Default: 160
	MaxPeers int

	// Number of worker threads (\"0\" = use as many threads as there are CPU cores available)
	// Default: 0
	NumThreads int

	// Node agent string which is used as identifier in network
	// Default: "Logos Storage"
	AgentString string

	// Backend for main repo store (fs, sqlite, leveldb)
	// Default: fs
	RepoKind string

	// The size of the total storage quota dedicated to the node
	// Default: 20 GiBs
	StorageQuota int

	// Default block timeout in seconds - 0 disables the ttl
	// Default: 30 days
	BlockTtl string

	// Time interval in seconds - determines frequency of block
	// maintenance cycle: how often blocks are checked for expiration and cleanup
	// Default: 10 minutes
	BlockMaintenanceInterval string

	// Number of blocks to check every maintenance cycle
	// Default: 1000
	BlockMaintenanceNumberOfBlocks int

	// Number of times to retry fetching a block before giving up
	// Default: 3000
	BlockRetries int

	// The size of the block cache, 0 disables the cache -
	// might help on slow hardrives
	// Default: 0
	CacheSize int

	// Default: "" (no log file)
	LogFile string
}

type LogosStorageConfig struct {
	Enabled    bool
	NodeConfig LogosStorageNodeConfig
}
