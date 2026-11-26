package delivery

import (
	"testing"

	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"

	bindings "github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"

	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/pkg/messaging/waku/fleets"
)

const (
	testStoreAddr = "/ip4/127.0.0.1/tcp/60002/p2p/16Uiu2HAmCDqxtfF1DwBqs7UJ4TgSnjoh6j1RtE1hhQxLLao84jLi"
	testBootENR   = "enr:-boot"
	testExtraENR  = "enr:-extra"
)

func testFleet(t *testing.T) fleets.Info {
	addr, err := multiaddr.NewMultiaddr(testStoreAddr)
	require.NoError(t, err)
	return fleets.Info{
		ClusterID:            16,
		WakuNodes:            []string{testBootENR},
		DiscV5BootstrapNodes: []string{testBootENR, testExtraENR},
		StoreNodes: []messagingtypes.StoreNode{
			{ID: "store", Addr: &addr},
			{ID: "store-without-addr"},
		},
	}
}

func TestFleetPresetUsesTheFleetNodes(t *testing.T) {
	got := FleetPreset(testFleet(t))

	require.Equal(t, bindings.PresetFields{
		"clusterId":         uint16(16),
		"maxMessageSize":    "1024KiB",
		"shardingConf":      []uint16{32},
		"extraListenShards": []uint16{64},
		"discv5Discovery":   true,
		"entryNodes":        []string{testBootENR, testExtraENR},
		"storeNodes":        []string{testStoreAddr},
	}, got)
}

func TestRegisterFleetPresetsSkipsBuiltinPresets(t *testing.T) {
	require.NoError(t, RegisterFleetPresets(fleets.Map{
		fleets.StatusProd:    testFleet(t),
		fleets.StatusStaging: testFleet(t),
	}))
}

func TestRegisteredFleetPresetReachesTheNode(t *testing.T) {
	const fleet = "status-go.unit-test"
	info := testFleet(t)
	info.WakuNodes = nil
	info.DiscV5BootstrapNodes = nil
	require.NoError(t, RegisterFleetPresets(fleets.Map{fleet: info}))

	client, err := bindings.New(bindings.Config{
		Mode:   bindings.ModeEdge,
		Preset: fleet,
		MessagingOverrides: bindings.Overrides{
			"local-storage-path": t.TempDir(),
			"listen-address":     "127.0.0.1",
			"nat":                "none",
		},
	})
	require.NoError(t, err)
	defer client.Close()

	storeNodes, err := client.Node().Debug().StoreNodes()
	require.NoError(t, err)
	require.Len(t, storeNodes, 1)
	require.Equal(t, testStoreAddr, storeNodes[0].String())
}
