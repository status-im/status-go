package connector

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/p2p"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/services/connector/commands"
)

func NewService(db *sql.DB, rpc commands.RPCClientInterface, nm commands.NetworkManagerInterface) *Service {
	return &Service{
		db:  db,
		rpc: rpc,
		nm:  nm,
	}
}

type Service struct {
	db  *sql.DB
	rpc commands.RPCClientInterface
	nm  commands.NetworkManagerInterface
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
