package logosstorage

type LogosStorageNode struct {
	NodeID  string  `json:"nodeId"`
	PeerID  string  `json:"peerId"`
	Record  string  `json:"record"`
	Address *string `json:"address"`
	Seen    bool    `json:"seen"`
}

type LogosStorageRoutingTable struct {
	LocalNode LogosStorageNode   `json:"localNode"`
	Nodes     []LogosStorageNode `json:"nodes"`
}

type LogosStorageDebugInfo struct {
	// Peer ID.
	ID string `json:"id"`

	// Peer info addresses configured for the node.
	Addrs []string `json:"addrs"`

	Spr               string                   `json:"spr"`
	AnnounceAddresses []string                 `json:"announceAddresses"`
	PeersTable        LogosStorageRoutingTable `json:"table"`
}
