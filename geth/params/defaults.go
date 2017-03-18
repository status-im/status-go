package params

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const (
	// DefaultClientIdentifier is client identifier to advertise over the network
	DefaultClientIdentifier = "status"

	// DefaultIPCFile is filename of exposed IPC RPC Server
	DefaultIPCFile = "geth.ipc"

	// DefaultHTTPHost is host interface for the HTTP RPC server
	DefaultHTTPHost = "localhost"

	// DefaultHTTPPort is HTTP-RPC port (replaced in unit tests)
	DefaultHTTPPort = 8545

	// DefaultAPIModules is a list of modules to expose vie HTTP RPC
	// TODO remove "admin" on main net
	DefaultAPIModules = "db,eth,net,web3,shh,personal,admin"

	// DefaultWSHost is a host interface for the websocket RPC server
	DefaultWSHost = "localhost"

	// DefaultWSPort is a WS-RPC port (replaced in unit tests)
	DefaultWSPort = 8546

	// DefaultMaxPeers is the maximum number of global peers
	DefaultMaxPeers = 25

	// DefaultMaxPendingPeers is the maximum number of peers that can be pending in the
	// handshake phase, counted separately for inbound and outbound connections.
	DefaultMaxPendingPeers = 0

	// DefaultGas default amount of gas used for transactions
	DefaultGas = 180000

	// DefaultFileDescriptorLimit is fd limit that database can use
	DefaultFileDescriptorLimit = uint64(2048)

	// DefaultDatabaseCache is memory (in MBs) allocated to internal caching (min 16MB / database forced)
	DefaultDatabaseCache = 128

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
