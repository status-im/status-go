package params

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const (
	// ClientIdentifier is client identifier to advertise over the network
	ClientIdentifier = "StatusIM"

	// DataDir is default data directory used by statusd executable
	DataDir = "statusd-data"

	// KeyStoreDir is default directory where private keys are stored, relative to DataDir
	KeyStoreDir = "keystore"

	// IPCFile is filename of exposed IPC RPC Server
	IPCFile = "geth.ipc"

	// HTTPHost is host interface for the HTTP RPC server
	HTTPHost = "localhost"

	// HTTPPort is HTTP-RPC port (replaced in unit tests)
	HTTPPort = 8545

	// APIModules is a list of modules to expose vie HTTP RPC
	// TODO remove "admin" on main net
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
	DatabaseCache = 128

	// LogFile defines where to write logs to
	LogFile = "geth.log"

	// LogLevel defines the minimum log level to report
	LogLevel = "INFO"

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

	// TestNetworkId is id of a test network
	TestNetworkId = 3
)

// Gas price settings
var (
	GasPrice                = new(big.Int).Mul(big.NewInt(20), common.Shannon)  // Minimal gas price to accept for mining a transactions
	GpoMinGasPrice          = new(big.Int).Mul(big.NewInt(20), common.Shannon)  // Minimum suggested gas price
	GpoMaxGasPrice          = new(big.Int).Mul(big.NewInt(500), common.Shannon) // Maximum suggested gas price
	GpoFullBlockRatio       = 80                                                // Full block threshold for gas price calculation (%)
	GpobaseStepDown         = 10                                                // Suggested gas price base step down ratio (1/1000)
	GpobaseStepUp           = 100                                               // Suggested gas price base step up ratio (1/1000)
	GpobaseCorrectionFactor = 110                                               // Suggested gas price base correction factor (%)
)
