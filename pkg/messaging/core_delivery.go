package messaging

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/status-im/status-go/pkg/messaging/delivery"
	"github.com/status-im/status-go/pkg/messaging/waku"
	"github.com/status-im/status-go/pkg/messaging/waku/fleets"
)

// deliveryPresets maps the fleets logos-delivery has a network preset for.
var deliveryPresets = map[string]string{
	fleets.StatusProd:    "status.prod",
	fleets.StatusStaging: "status.staging",
}

func deliveryConfig(params CoreParams) (delivery.Config, error) {
	preset, ok := deliveryPresets[params.Fleet]
	if !ok {
		return delivery.Config{}, fmt.Errorf("fleet %q has no logos-delivery preset", params.Fleet)
	}
	mode := delivery.ModeCore
	if params.Mode == waku.ModeEdge {
		mode = delivery.ModeEdge
	}
	return delivery.Config{
		Preset:  preset,
		Mode:    mode,
		NodeKey: params.NodeKey,
		DataDir: params.DataDir,
		TCPPort: params.WakuConfig.Port,
		UDPPort: params.WakuConfig.UDPPort,
	}, nil
}

func newDeliveryBackend(params CoreParams, logger *zap.Logger) (*delivery.Adapter, error) {
	cfg, err := deliveryConfig(params)
	if err != nil {
		return nil, err
	}
	client, err := delivery.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return delivery.NewAdapter(client, params.Fleet, logger), nil
}
