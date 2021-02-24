package appmetrics

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/appmetrics"
)

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

type Service struct {
	db *sql.DB
}

func (s *Service) Start(*p2p.Server) error {
	return nil
}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "appmetrics",
			Version:   "0.1.0",
			Service:   appmetrics.NewDB(s.db),
			Public:    true,
		},
	}
}

func (s *Service) Protocols() []p2p.Protocol {
	return nil
}
