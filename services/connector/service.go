package connector

import (
	"github.com/ethereum/go-ethereum/p2p"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/rpc"
)

func NewService(rpcClient *rpc.Client, connectorSrvc *Service) *Service {
	return &Service{
		rpcClient:     rpcClient,
		connectorSrvc: connectorSrvc,
	}
}

type Service struct {
	rpcClient     *rpc.Client
	connectorSrvc *Service
}

func (s *Service) Start() error {
	return nil
}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) APIs() []gethrpc.API {
	return []gethrpc.API{
		{
			Namespace: "connector",
			Version:   "0.1.0",
			Service:   NewAPI(s),
		},
	}
}

func (s *Service) Protocols() []p2p.Protocol {
	return nil
}
