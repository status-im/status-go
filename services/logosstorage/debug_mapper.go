//go:build use_logos_storage && !lint
// +build use_logos_storage,!lint

package logosstorage

import "github.com/logos-storage/logos-storage-go-bindings/storage"

func toLogosStorageNode(node storage.Node) LogosStorageNode {
	return LogosStorageNode{
		NodeID:  node.NodeId,
		PeerID:  node.PeerId,
		Record:  node.Record,
		Address: node.Address,
		Seen:    node.Seen,
	}
}

func toLogosStorageRoutingTable(table storage.RoutingTable) LogosStorageRoutingTable {
	nodes := make([]LogosStorageNode, 0, len(table.Nodes))
	for _, node := range table.Nodes {
		nodes = append(nodes, toLogosStorageNode(node))
	}

	return LogosStorageRoutingTable{
		LocalNode: toLogosStorageNode(table.LocalNode),
		Nodes:     nodes,
	}
}

func toLogosStorageDebugInfo(info storage.DebugInfo) LogosStorageDebugInfo {
	return LogosStorageDebugInfo{
		ID:                info.ID,
		Addrs:             append([]string(nil), info.Addrs...),
		Spr:               info.Spr,
		AnnounceAddresses: append([]string(nil), info.AnnounceAddresses...),
		PeersTable:        toLogosStorageRoutingTable(info.PeersTable),
	}
}
