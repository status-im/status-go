package preferences

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/rpc"
)

type Service struct {
	store PreferenceStore
}

func NewService(db *sql.DB) *Service {
	return NewServiceWithStore(NewStore(db))
}

func NewServiceWithStore(store PreferenceStore) *Service {
	return &Service{
		store: store,
	}
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
			Namespace: "preferences",
			Version:   "0.1.0",
			Service:   NewAPI(s.store),
		},
	}
}
