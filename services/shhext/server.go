package shhext

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

type server struct {
	server *p2p.Server
}

func (s *server) NodeID() *ecdsa.PrivateKey {
	return s.server.PrivateKey
}

func (s *server) Online() bool {
	return s.server.PeerCount() != 0
}

func (s *server) AddPeer(url string) error {
	parsedNode, err := enode.ParseV4(url)
	if err != nil {
		return err
	}
	s.server.AddPeer(parsedNode)
	return nil
}

func (s *server) Connected(id enode.ID) (bool, error) {
	for _, p := range s.server.Peers() {
		if p.ID() == id {
			return true, nil
		}
	}
	return false, nil
}
