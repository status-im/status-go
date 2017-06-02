package params

import (
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
	DatabaseCache = 128

	// BootClusterConfigURL defines URL to file containing hard-coded CHT roots and boot nodes
	// TODO remove this hack, once CHT sync is implemented on LES side
	BootClusterConfigURL = "https://gist.githubusercontent.com/farazdagi/a8d36e2818b3b2b6074d691da63a0c36/raw/"

	// LogFile defines where to write logs to
	LogFile = "geth.log"

	// LogLevel defines the minimum log level to report
	LogLevel = "INFO"

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

var (
	RopstenNetGenesisHash = common.HexToHash("0x41941023680923e0fe4d74a34bdac8141f2540e3ae90623718e47d66d1ca4a2d") // Testnet genesis hash to enforce below configs on
	MainNetGenesisHash    = common.HexToHash("0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3") // Mainnet genesis hash to enforce below configs on
)
