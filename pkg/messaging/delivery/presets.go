package delivery

import (
	"fmt"

	bindings "github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"

	"github.com/status-im/status-go/pkg/messaging/waku/fleets"
)

// fleetPresetBase maps a fleet to the logos-delivery preset its definition is
// based on. A fleet without a matching preset is based on status.staging, the
// network settings every Status fleet shares.
var fleetPresetBase = map[string]string{
	fleets.StatusProd:    bindings.PresetStatusProd,
	fleets.StatusStaging: bindings.PresetStatusStaging,
}

// RegisterFleetPresets registers a logos-delivery network preset for each
// fleet, named after the fleet, so a node joins it by the fleet's name. The
// fleet's cluster id, nodes and store nodes replace the base preset's, so the
// node reaches only the fleet's nodes.
func RegisterFleetPresets(fleetsMap fleets.Map) error {
	for name, info := range fleetsMap {
		if err := bindings.AddPreset(name, FleetPreset(name, info)); err != nil {
			return fmt.Errorf("register the logos-delivery preset of fleet %q: %w", name, err)
		}
	}
	return nil
}

// FleetPreset is the logos-delivery preset of the fleet name. Its waku and
// discv5 bootstrap nodes become entry nodes, which logos-delivery sorts by
// kind: enrtree URLs, ENRs and multiaddrs.
func FleetPreset(name string, info fleets.Info) bindings.PresetFields {
	base, ok := fleetPresetBase[name]
	if !ok {
		base = bindings.PresetStatusStaging
	}

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
		"base":                 base,
		"clusterId":            info.ClusterID,
		"entryNodes":           entryNodes,
		"discv5BootstrapNodes": []string{},
		"storeNodes":           storeNodes,
	}
}
