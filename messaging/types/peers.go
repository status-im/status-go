package types

import (
	"encoding/json"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

type ConnStatus struct {
	IsOnline bool      `json:"isOnline"`
	Peers    PeerStats `json:"peers"`
}

type PeerStats map[peer.ID]Peer

func (m PeerStats) MarshalJSON() ([]byte, error) {
	tmpMap := make(map[string]Peer)
	for k, v := range m {
		tmpMap[k.String()] = v
	}
	return json.Marshal(tmpMap)
}

type Peer struct {
	Protocols []protocol.ID         `json:"protocols"`
	Addresses []multiaddr.Multiaddr `json:"addresses"`
}

type PeerList struct {
	FullMeshPeers peer.IDSlice `json:"fullMesh"`
	AllPeers      peer.IDSlice `json:"all"`
}
