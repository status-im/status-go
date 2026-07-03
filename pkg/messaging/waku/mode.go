package wakuv2

import "fmt"

// Mode selects how the node participates in the network, mirroring the
// logos-delivery Messaging API's "mode" (Core vs Edge).
//
//   - Core: a full relay node — runs discv5 and serves peer exchange. It is the
//     default (zero value).
//   - Edge: a light node — uses lightpush/filter and the peer-exchange client,
//     with discv5 disabled.
type Mode int

const (
	// ModeCore is a full relay node. It is the default (zero value).
	ModeCore Mode = iota
	// ModeEdge is a light node.
	ModeEdge
)

// ModeFromLightClient maps the app-facing LightClient boolean onto a Mode. It is
// the bridge for callers that still carry WakuV2Config.LightClient (kept for
// backwards compatibility at the API surface): Edge when true, Core otherwise.
func ModeFromLightClient(lightClient bool) Mode {
	if lightClient {
		return ModeEdge
	}
	return ModeCore
}

// IsLightClient reports whether the mode is a light (Edge) node. It is the
// single internal replacement for the former Config.LightClient flag.
func (m Mode) IsLightClient() bool {
	return m == ModeEdge
}

// Validate rejects unknown mode values.
func (m Mode) Validate() error {
	switch m {
	case ModeCore, ModeEdge:
		return nil
	default:
		return fmt.Errorf("invalid waku mode: %d", int(m))
	}
}

// applyTo derives the peer-exchange and discv5 flags on cfg from the mode. Kept
// here (in the waku layer) so the Core/Edge peer-management policy lives with the
// node it configures. Unknown values default to Core (relay); Validate rejects
// them separately.
func (m Mode) applyTo(cfg *Config) {
	if m == ModeEdge {
		cfg.EnablePeerExchangeServer = false
		cfg.EnablePeerExchangeClient = true
		cfg.EnableDiscV5 = false
		return
	}
	// ModeCore (default / relay).
	cfg.EnablePeerExchangeServer = true
	cfg.EnablePeerExchangeClient = false
	cfg.EnableDiscV5 = true
}
