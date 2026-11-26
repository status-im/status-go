package delivery

import (
	"fmt"

	bindings "github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"

	"github.com/status-im/status-go/pkg/messaging/waku/fleets"
)

// builtinPresets are the fleets logos-delivery has a built-in preset for.
var builtinPresets = map[string]bool{
	fleets.StatusProd:    true,
	fleets.StatusStaging: true,
}

// RegisterFleetPresets registers a logos-delivery network preset, named after
// the fleet, for each fleet logos-delivery has no built-in preset for, so a
// node joins it by the fleet's name.
func RegisterFleetPresets(fleetsMap fleets.Map) error {
	for name, info := range fleetsMap {
		if builtinPresets[name] {
			continue
		}
		if err := bindings.AddPreset(name, FleetPreset(info)); err != nil {
			return fmt.Errorf("register the logos-delivery preset of fleet %q: %w", name, err)
		}
	}
	return nil
}

// FleetPreset is the logos-delivery preset of a fleet: the Status network
// settings with the fleet's cluster id, nodes and store nodes. Its waku and
// discv5 bootstrap nodes become entry nodes, which logos-delivery sorts by
// kind: enrtree URLs, ENRs and multiaddrs.
func FleetPreset(info fleets.Info) bindings.PresetFields {
	entryNodes := []string{}
	seen := map[string]bool{}
	for _, node := range append(append([]string{}, info.WakuNodes...), info.DiscV5BootstrapNodes...) {
		if !seen[node] {
			seen[node] = true
			entryNodes = append(entryNodes, node)
		}
	}

	storeNodes := []string{}
	for _, node := range info.StoreNodes {
		if node.Addr != nil {
			storeNodes = append(storeNodes, (*node.Addr).String())
		}
	}

	return bindings.PresetFields{
		"clusterId":         info.ClusterID,
		"maxMessageSize":    "1024KiB",
		"shardingConf":      []uint16{32},
		"extraListenShards": []uint16{64},
		"discv5Discovery":   true,
		"entryNodes":        entryNodes,
		"storeNodes":        storeNodes,
	}
}
