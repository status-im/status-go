package eth

import (
	geth_rpc "github.com/ethereum/go-ethereum/rpc"

	rpc_client "github.com/status-im/status-go/rpc"
)

type Service struct {
	rpcClient *rpc_client.Client
}

func NewService(
	rpcClient *rpc_client.Client,
) *Service {
	return &Service{
		rpcClient: rpcClient,
	}
}

func (s *Service) APIs() []geth_rpc.API {
	return privateAPIs(s.rpcClient)
}

func (s *Service) Start() error {
	return nil
}

func (s *Service) Stop() error {
	return nil
}
