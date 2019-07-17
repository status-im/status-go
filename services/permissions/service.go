package permissions

import (
	"sync"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

// NewService initializes service instance.
func NewService() *Service {
	return &Service{}
}

type Service struct {
	mu sync.Mutex
	db *Database
}

// Start a service.
func (s *Service) Start(*p2p.Server) error {
	return nil
}

// StartDatabase after dbpath and password will become known.
func (s *Service) StartDatabase(dbpath, password string) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db, err = InitializeDB(dbpath, password)
	return err
}

func (s *Service) StopDatabase() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Stop a service.
func (s *Service) Stop() error {
	return s.StopDatabase()
}

// APIs returns list of available RPC APIs.
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "permissions",
			Version:   "0.1.0",
			Service:   NewAPI(s),
			Public:    true,
		},
	}
}

// Protocols returns list of p2p protocols.
func (s *Service) Protocols() []p2p.Protocol {
	return nil
}
