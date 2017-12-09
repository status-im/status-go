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

	// ListenAddr is an IP address and port of this node (e.g. 127.0.0.1:30303).
	ListenAddr = ":0"

	// APIModules is a list of modules to expose via any type of RPC (HTTP, IPC, in-proc)
	APIModules = "db,eth,net,web3,shh,personal,admin"

	// WSHost is a host interface for the websocket RPC server
	WSHost = "localhost"

	// SendTransactionMethodName defines the name for a giving transaction.
	SendTransactionMethodName = "eth_sendTransaction"

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

	// LogFile defines where to write logs to
	LogFile = ""

	// LogLevel defines the minimum log level to report
	LogLevel = "ERROR"

	// LogLevelSuccinct defines the log level when only errors are reported.
	// Useful when the default INFO level becomes too verbose.
	LogLevelSuccinct = "ERROR"

	// LogToStderr defines whether logged info should also be output to os.Stderr
	LogToStderr = true

	// WhisperDataDir is directory where Whisper data is stored, relative to DataDir
	WhisperDataDir = "wnode"

	// WhisperMinimumPoW amount of work for Whisper message to be added to sending queue
	WhisperMinimumPoW = 0.001

	// WhisperTTL is time to live for messages, in seconds
	WhisperTTL = 120

	// FirebaseNotificationTriggerURL is URL where FCM notification requests are sent to
	FirebaseNotificationTriggerURL = "https://fcm.googleapis.com/fcm/send"

	// MainnetEthereumNetworkURL is URL where the upstream ethereum network is loaded to
	// allow us avoid syncing node.
	MainnetEthereumNetworkURL = "https://mainnet.infura.io/nKmXgiFgc2KqtoQ8BCGJ"

	// RopstenEthereumNetworkURL is URL where the upstream ethereum network is loaded to
	// allow us avoid syncing node.
	RopstenEthereumNetworkURL = "https://ropsten.infura.io/nKmXgiFgc2KqtoQ8BCGJ"

	// RinkebyEthereumNetworkURL is URL where the upstream ethereum network is loaded to
	// allow us avoid syncing node.
	RinkebyEthereumNetworkURL = "https://rinkeby.infura.io/nKmXgiFgc2KqtoQ8BCGJ"

	// MainNetworkID is id of the main network
	MainNetworkID = 1

	// RopstenNetworkID is id of a test network (on PoW)
	RopstenNetworkID = 3

	// RinkebyNetworkID is id of a test network (on PoA)
	RinkebyNetworkID = 4

	// StatusChainNetworkID is id of a test network (private chain)
	StatusChainNetworkID = 777
)
