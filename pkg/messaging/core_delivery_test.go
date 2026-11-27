package messaging

import (
	"testing"

	"github.com/stretchr/testify/require"

	bindings "github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/messaging/waku"
	"github.com/status-im/status-go/pkg/messaging/waku/fleets"
)

func TestDeliveryConfigUsesTheFleetPreset(t *testing.T) {
	for fleet, preset := range map[string]string{
		fleets.StatusProd:    "status.prod",
		fleets.StatusStaging: "status.staging",
	} {
		cfg, err := deliveryConfig(CoreParams{Fleet: fleet, Mode: waku.ModeEdge, DataDir: "/data"})
		require.NoError(t, err)

		require.Equal(t, preset, cfg.Preset)
		require.Equal(t, bindings.ModeEdge, cfg.Mode)
		require.Equal(t, bindings.Overrides{"reliability": false, "backfill-enabled": false, "configured-store-nodes-only": true, "local-storage-path": "/data"}, cfg.MessagingOverrides)
	}
}

func TestDeliveryConfigRejectsFleetWithoutPreset(t *testing.T) {
	_, err := deliveryConfig(CoreParams{Fleet: "unknown.fleet"})
	require.Error(t, err)
}

func TestDeliveryConfigTurnsQUICOffOnTheDiscv5Port(t *testing.T) {
	params := CoreParams{Fleet: fleets.StatusProd, WakuConfig: params.WakuV2Config{Port: 30303, UDPPort: 30303}}
	cfg, err := deliveryConfig(params)
	require.NoError(t, err)
	require.Equal(t, false, cfg.MessagingOverrides["quic-support"])

	params.WakuConfig.UDPPort = 30304
	cfg, err = deliveryConfig(params)
	require.NoError(t, err)
	require.NotContains(t, cfg.MessagingOverrides, "quic-support")
}
