package waku

// Config represents the configuration state of a waku node.
type Config struct {
	MaxMessageSize                        uint32  `toml:",omitempty"`
	MinimumAcceptedPOW                    float64 `toml:",omitempty"`
	RestrictConnectionBetweenLightClients bool    `toml:",omitempty"`
	EnableConfirmations                   bool    `toml:",omitempty"`
}

var DefaultConfig = Config{
	MaxMessageSize:                        DefaultMaxMessageSize,
	MinimumAcceptedPOW:                    DefaultMinimumPoW,
	RestrictConnectionBetweenLightClients: true,
}
