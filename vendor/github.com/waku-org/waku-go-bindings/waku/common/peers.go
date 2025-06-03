package common

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

type PeersData map[peer.ID]PeerInfo
type PeerInfo struct {
	Protocols []protocol.ID         `json:"protocols"`
	Addresses []multiaddr.Multiaddr `json:"addresses"`
}

func ParsePeerInfoFromJSON(jsonStr string) (PeersData, error) {
	/*
		We expect a JSON string with the format:

			{
				<peerId1>: {
				  "protocols": [
					"protocol1",
					"protocol2",
					 ...
				  ],
				  "addresses": [
					"address1",
					"address2",
					...
				  ]
				},
				<peerId2>: ...
			}
	*/
	// Create a temporary map to unmarshal the JSON data
	var rawMap map[string]struct {
		Protocols []string `json:"protocols"`
		Addresses []string `json:"addresses"`
	}

	// Unmarshal the JSON string to our temporary map
	if err := json.Unmarshal([]byte(jsonStr), &rawMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Create the result map
	result := make(PeersData)

	// Process each peer entry
	for peerIDStr, rawPeer := range rawMap {
		// Parse the peer ID
		peerID, err := peer.Decode(peerIDStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode peer ID %s: %w", peerIDStr, err)
		}

		// Convert protocols to libp2pproto.ID
		protocols := make([]protocol.ID, len(rawPeer.Protocols))
		for i, protoStr := range rawPeer.Protocols {
			protocols[i] = protocol.ID(protoStr)
		}

		// Convert addresses to multiaddr.Multiaddr
		addresses := make([]multiaddr.Multiaddr, 0, len(rawPeer.Addresses))
		for _, addrStr := range rawPeer.Addresses {
			addr, err := multiaddr.NewMultiaddr(addrStr)
			if err != nil {
				// Log the error but continue with other addresses
				log.Printf("failed to parse multiaddress %s: %v", addrStr, err)
				continue
			}
			addresses = append(addresses, addr)
		}

		// Add the peer to the result map
		result[peerID] = PeerInfo{
			Protocols: protocols,
			Addresses: addresses,
		}
	}

	return result, nil
}

// EncapsulatePeerID takes a peer.ID and adds a p2p component to all multiaddresses it receives
func EncapsulatePeerID(peerID peer.ID, addrs ...multiaddr.Multiaddr) []multiaddr.Multiaddr {
	hostInfo, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", peerID.String()))
	var result []multiaddr.Multiaddr
	for _, addr := range addrs {
		result = append(result, addr.Encapsulate(hostInfo))
	}
	return result
}
