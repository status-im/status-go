package waku

// Config represents the configuration state of a waku node.
type Config struct {
	MaxMessageSize           uint32  `toml:",omitempty"`
	MinimumAcceptedPoW       float64 `toml:",omitempty"`
	LightClient              bool    `toml:",omitempty"` // when true, it does not forward messages
	FullNode                 bool    `toml:",omitempty"` // when true, it forwards all messages
	RestrictLightClientsConn bool    `toml:",omitempty"` // when true, do not accept light client as peers if it is a light client itself
	EnableConfirmations      bool    `toml:",omitempty"` // when true, sends message confirmations
}

var DefaultConfig = Config{
	MaxMessageSize:           DefaultMaxMessageSize,
	MinimumAcceptedPoW:       DefaultMinimumPoW,
	RestrictLightClientsConn: true,
}
