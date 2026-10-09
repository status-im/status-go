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
	fleets.StatusProd: "status.prod",
}

func newDeliveryBackend(params CoreParams, logger *zap.Logger) (*delivery.Adapter, error) {
	preset, ok := deliveryPresets[params.Fleet]
	if !ok {
		return nil, fmt.Errorf("fleet %q has no logos-delivery preset", params.Fleet)
	}
	mode := delivery.ModeCore
	if params.Mode == waku.ModeEdge {
		mode = delivery.ModeEdge
	}
	client, err := delivery.NewClient(delivery.Config{
		Preset:  preset,
		Mode:    mode,
		NodeKey: params.NodeKey,
		DataDir: params.DataDir,
		TCPPort: params.WakuConfig.Port,
		UDPPort: params.WakuConfig.UDPPort,
	})
	if err != nil {
		return nil, err
	}
	return delivery.NewAdapter(client, logger), nil
}
