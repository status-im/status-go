package params

const (
	// ClientIdentifier is client identifier to advertise over the network
	ClientIdentifier = "StatusIM"

	// DataDir is default data directory used by statusd executable
	DataDir = "statusd-data"

	// KeyStoreDir is default directory where private keys are stored, relative to DataDir
	KeyStoreDir = "keystore"

	// IPCFile is filename of exposed IPC RPC Server
	IPCFile = "geth.ipc"

	// RPCEnabledDefault is the default state of whether the http rpc server is supposed
	// to be started along with a node.
	RPCEnabledDefault = false

	// HTTPHost is host interface for the HTTP RPC server
	HTTPHost = "localhost"

	// HTTPPort is HTTP-RPC port (replaced in unit tests)
	HTTPPort = 8545

	// APIModules is a list of modules to expose via any type of RPC (HTTP, IPC, in-proc)
	APIModules = "db,eth,net,web3,shh,personal,admin"

	// WSHost is a host interface for the websocket RPC server
	WSHost = "localhost"

	// WSPort is a WS-RPC port (replaced in unit tests)
	WSPort = 8546

	// MaxPeers is the maximum number of global peers
	MaxPeers = 25

	// MaxPendingPeers is the maximum number of peers that can be pending in the
	// handshake phase, counted separately for inbound and outbound connections.
	MaxPendingPeers = 0

	// DefaultGas default amount of gas used for transactions
	DefaultGas = 180000

	// DefaultFileDescriptorLimit is fd limit that database can use
	DefaultFileDescriptorLimit = uint64(2048)

	// DatabaseCache is memory (in MBs) allocated to internal caching (min 16MB / database forced)
	DatabaseCache = 16

	// CHTRootConfigURL defines URL to file containing hard-coded CHT roots
	// TODO remove this hack, once CHT sync is implemented on LES side
	CHTRootConfigURL = "https://gist.githubusercontent.com/tiabc/83ed515fbb0c0e9d39700a6279072b6a/raw/a8c7b08488fab3c1d9139b18af33da3df823e3ff/cht.json"

	// BootClusterConfigURL defines URL to file containing hard-coded CHT roots and boot nodes
	// TODO remove this hack, once CHT sync is implemented on LES side
	BootClusterConfigURL = "https://gist.githubusercontent.com/tiabc/83ed515fbb0c0e9d39700a6279072b6a/raw/a8c7b08488fab3c1d9139b18af33da3df823e3ff/cht.json"

	// LogFile defines where to write logs to
	LogFile = "geth.log"

	// LogLevel defines the minimum log level to report
	LogLevel = "INFO"

	// LogLevelSuccinct defines the log level when only errors are reported.
	// Useful when the default INFO level becomes too verbose.
	LogLevelSuccinct = "ERROR"

	// LogToStderr defines whether logged info should also be output to os.Stderr
	LogToStderr = true

	// WhisperDataDir is directory where Whisper data is stored, relative to DataDir
	WhisperDataDir = "wnode"

	// WhisperPort is Whisper node listening port
	WhisperPort = 30379

	// WhisperMinimumPoW amount of work for Whisper message to be added to sending queue
	WhisperMinimumPoW = 0.001

	// WhisperTTL is time to live for messages, in seconds
	WhisperTTL = 120

	// FirebaseNotificationTriggerURL is URL where FCM notification requests are sent to
	FirebaseNotificationTriggerURL = "https://fcm.googleapis.com/fcm/send"

	// MainNetworkID is id of the main network
	MainNetworkID = 1

	// RopstenNetworkID is id of a test network (on PoW)
	RopstenNetworkID = 3

	// RinkebyNetworkID is id of a test network (on PoA)
	RinkebyNetworkID = 4
)

// NetworkIDs is a list of currently supported networks.
var NetworkIDs = map[uint64]bool{
	MainNetworkID:    true,
	RopstenNetworkID: true,
	RinkebyNetworkID: true,
}
