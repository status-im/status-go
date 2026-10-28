package messaging

import (
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	bindings "github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"

	"github.com/status-im/status-go/pkg/messaging/delivery"
	"github.com/status-im/status-go/pkg/messaging/waku"
	"github.com/status-im/status-go/pkg/messaging/waku/fleets"
)

// deliveryPresets maps the fleets logos-delivery has a network preset for.
var deliveryPresets = map[string]string{
	fleets.StatusProd:    "status.prod",
	fleets.StatusStaging: "status.staging",
}

func deliveryConfig(params CoreParams) (bindings.Config, error) {
	preset, ok := deliveryPresets[params.Fleet]
	if !ok {
		return bindings.Config{}, fmt.Errorf("fleet %q has no logos-delivery preset", params.Fleet)
	}
	mode := bindings.ModeCore
	if params.Mode == waku.ModeEdge {
		mode = bindings.ModeEdge
	}
	overrides := bindings.Overrides{}
	if params.NodeKey != nil {
		overrides["nodekey"] = hex.EncodeToString(crypto.FromECDSA(params.NodeKey))
	}
	if params.WakuConfig.Port != 0 {
		overrides["tcp-port"] = params.WakuConfig.Port
	}
	if params.WakuConfig.UDPPort != 0 {
		overrides["discv5-udp-port"] = params.WakuConfig.UDPPort
	}
	if params.DataDir != "" {
		overrides["local-storage-path"] = params.DataDir
	}
	if len(overrides) == 0 {
		overrides = nil
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
