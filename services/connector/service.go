package connector

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/p2p"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/rpc"
)

func NewService(db *sql.DB, rpcClient *rpc.Client, connectorSrvc *Service) *Service {
	return &Service{
		rpcClient:     rpcClient,
		connectorSrvc: connectorSrvc,
		db:            db,
	}
}

type Service struct {
	rpcClient     *rpc.Client
	connectorSrvc *Service
	db            *sql.DB
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
