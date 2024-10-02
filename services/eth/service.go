package eth

import (
	"github.com/ethereum/go-ethereum/p2p"
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

func (s *Service) Protocols() []p2p.Protocol {
	return nil
}

func (s *Service) Start() error {
	return nil
}

func (s *Service) Stop() error {
	return nil
}
