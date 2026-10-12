package messaging

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/status-im/status-go/pkg/messaging/delivery"
	"github.com/status-im/status-go/pkg/messaging/waku"
	"github.com/status-im/status-go/pkg/messaging/waku/fleets"
)

// deliveryNetwork is a fleet's logos-delivery network preset and the node
// configuration Status sets on top of it.
type deliveryNetwork struct {
	preset    string
	overrides map[string]any
}

// deliveryNetworks maps the fleets logos-delivery has a network preset for.
var deliveryNetworks = map[string]deliveryNetwork{
	fleets.StatusProd: {
		preset: "status.prod",
		overrides: map[string]any{
			"autosharding-shards": []uint16{32},
			// Remove after Status 2.41 (status-im/status-go#7498).
			"extra-listen-shard": []uint16{64},
		},
	},
}

func deliveryConfig(params CoreParams) (delivery.Config, error) {
	network, ok := deliveryNetworks[params.Fleet]
	if !ok {
		return delivery.Config{}, fmt.Errorf("fleet %q has no logos-delivery preset", params.Fleet)
	}
	mode := delivery.ModeCore
	if params.Mode == waku.ModeEdge {
		mode = delivery.ModeEdge
	}
	return delivery.Config{
		Preset:    network.preset,
		Mode:      mode,
		NodeKey:   params.NodeKey,
		DataDir:   params.DataDir,
		TCPPort:   params.WakuConfig.Port,
		UDPPort:   params.WakuConfig.UDPPort,
		Overrides: network.overrides,
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
