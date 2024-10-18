package connector

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/p2p"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/network"
)

func NewService(db *sql.DB, rpc rpc.ClientInterface, nm *network.Manager) *Service {
	return &Service{
		db:  db,
		rpc: rpc,
		nm:  nm,
	}
}

type Service struct {
	db  *sql.DB
	rpc rpc.ClientInterface
	nm  *network.Manager
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
