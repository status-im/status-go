package common

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
