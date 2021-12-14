package provider

import (
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

func NewService() *Service {
	return &Service{}
}

type Service struct {
}

func (s *Service) Start() error {
	return nil
}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "provider",
			Version:   "0.1.0",
			Service:   NewAPI(),
		},
	}
}

func (s *Service) Protocols() []p2p.Protocol {
	return nil
}
