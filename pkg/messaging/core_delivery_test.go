package messaging

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/pkg/messaging/delivery"
	"github.com/status-im/status-go/pkg/messaging/waku"
	"github.com/status-im/status-go/pkg/messaging/waku/fleets"
)

func TestDeliveryConfigPublishesOn32AndListensOn64(t *testing.T) {
	cfg, err := deliveryConfig(CoreParams{Fleet: fleets.StatusProd, Mode: waku.ModeEdge, DataDir: "/data"})
	require.NoError(t, err)

	require.Equal(t, "status.prod", cfg.Preset)
	require.Equal(t, delivery.ModeEdge, cfg.Mode)
	require.Equal(t, "/data", cfg.DataDir)
	require.Equal(t, []uint16{32}, cfg.Overrides["autosharding-shards"])
	require.Equal(t, []uint16{64}, cfg.Overrides["extra-listen-shard"])
}

func TestDeliveryConfigRejectsFleetWithoutPreset(t *testing.T) {
	_, err := deliveryConfig(CoreParams{Fleet: "unknown.fleet"})
	require.Error(t, err)
}
