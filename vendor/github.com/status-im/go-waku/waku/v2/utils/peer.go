package utils

import (
	"errors"
	"math/rand"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

var log = logging.Logger("utils")

// SelectPeer is used to return a random peer that supports a given protocol.
func SelectPeer(host host.Host, protocolId string) (*peer.ID, error) {
	// @TODO We need to be more strategic about which peers we dial. Right now we just set one on the service.
	// Ideally depending on the query and our set  of peers we take a subset of ideal peers.
	// This will require us to check for various factors such as:
	//  - which topics they track
	//  - latency?
	//  - default store peer?
	var peers peer.IDSlice
	for _, peer := range host.Peerstore().Peers() {
		protocols, err := host.Peerstore().SupportsProtocols(peer, protocolId)
		if err != nil {
			log.Error("error obtaining the protocols supported by peers", err)
			return nil, err
		}

		if len(protocols) > 0 {
			peers = append(peers, peer)
		}
	}

	if len(peers) >= 1 {
		// TODO: proper heuristic here that compares peer scores and selects "best" one. For now a random peer for the given protocol is returned
		return &peers[rand.Intn(len(peers))], nil // nolint: gosec
	}

	return nil, errors.New("no suitable peers found")
}
