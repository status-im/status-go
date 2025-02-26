package common

const (
	MainnetChainID              uint64 = 1
	SepoliaChainID              uint64 = 11155111
	OptimismChainID             uint64 = 10
	OptimismSepoliaChainID      uint64 = 11155420
	ArbitrumChainID             uint64 = 42161
	ArbitrumSepoliaChainID      uint64 = 421614
	BaseChainID                 uint64 = 8453
	BaseSepoliaChainID          uint64 = 84532
	StatusNetworkSepoliaChainID uint64 = 1660990954
	BNBSmartChainID             uint64 = 56
	BNBSmartChainTestnetChainID uint64 = 97
)

// ProviderID represents the internal ID of a blockchain provider
type ProviderID = string

// Provider IDs
const (
	StatusSmartProxy  = "status-smart-proxy"
	ProxyNodefleet    = "proxy-nodefleet"
	ProxyInfura       = "proxy-infura"
	ProxyGrove        = "proxy-grove"
	SmartProxyAlchemy = "smart-proxy-alchemy"
	Nodefleet         = "nodefleet"
	Infura            = "infura"
	Grove             = "grove"
	Alchemy           = "alchemy"
	DirectInfura      = "direct-infura"
	DirectGrove       = "direct-grove"
	DirectStatus      = "direct-status"
)
