package appgeneral

import (
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

type Service struct{}

func New() *Service {
	return &Service{}
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "appgeneral",
			Version:   "0.1.0",
			Service:   NewAPI(s),
		},
	}
}

func (s *Service) Protocols() []p2p.Protocol {
	return nil
}

func (s *Service) Start() error {
	return nil
}

func (s *Service) Stop() error {
	return nil
}
