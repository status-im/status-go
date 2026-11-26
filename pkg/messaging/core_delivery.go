package messaging

import (
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	bindings "github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"

	"github.com/status-im/status-go/pkg/messaging/delivery"
	"github.com/status-im/status-go/pkg/messaging/waku"
)

// deliveryConfig selects the logos-delivery preset named after the fleet: a
// built-in one, or one registered from a fleets file by
// delivery.RegisterFleetPresets.
func deliveryConfig(params CoreParams) (bindings.Config, error) {
	if params.Fleet == "" {
		return bindings.Config{}, fmt.Errorf("no fleet to select a logos-delivery preset by")
	}
	preset := params.Fleet
	mode := bindings.ModeCore
	if params.Mode == waku.ModeEdge {
		mode = bindings.ModeEdge
	}
	overrides := bindings.Overrides{
		"reliability":                 false,
		"backfill-enabled":            false,
		"configured-store-nodes-only": true,
	}
	if params.NodeKey != nil {
		overrides["nodekey"] = hex.EncodeToString(crypto.FromECDSA(params.NodeKey))
	}
	if params.WakuConfig.Port != 0 {
		overrides["tcp-port"] = params.WakuConfig.Port
	}
	if params.WakuConfig.UDPPort != 0 {
		overrides["discv5-udp-port"] = params.WakuConfig.UDPPort
		// QUIC listens on the UDP port of the TCP one.
		if params.WakuConfig.UDPPort == params.WakuConfig.Port {
			overrides["quic-support"] = false
		}
	}
	if params.DataDir != "" {
		overrides["local-storage-path"] = params.DataDir
	}
	return bindings.Config{Mode: mode, Preset: preset, MessagingOverrides: overrides}, nil
}

func newDeliveryBackend(params CoreParams, logger *zap.Logger) (*delivery.Adapter, error) {
	cfg, err := deliveryConfig(params)
	if err != nil {
		return nil, err
	}
	return delivery.NewAdapter(cfg, logger)
}
