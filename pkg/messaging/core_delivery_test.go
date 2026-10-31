package messaging

import (
	"testing"

	"github.com/stretchr/testify/require"

	bindings "github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"

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
		require.Equal(t, bindings.Overrides{"reliability": false, "backfill-enabled": false, "local-storage-path": "/data"}, cfg.MessagingOverrides)
	}
}

func TestDeliveryConfigRejectsFleetWithoutPreset(t *testing.T) {
	_, err := deliveryConfig(CoreParams{Fleet: "unknown.fleet"})
	require.Error(t, err)
}
