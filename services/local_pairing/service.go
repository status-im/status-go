package local_pairing

import (
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

type Service struct {}

func (s *Service) Start() error {
	return nil
}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "localPairing",
			Version:   "0.1.0",
			Service:   NewAPI(),
			Public:    true,
		},
	}
}

func (s *Service) Protocols() []p2p.Protocol {
	return nil
}

