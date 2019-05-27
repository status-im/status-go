package wallet

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

// NewService initializes database and creates new service instance.
func NewService(dbpath string, client *ethclient.Client, address common.Address, chain *big.Int) (*Service, error) {
	db, err := InitializeDB(dbpath)
	if err != nil {
		return nil, err
	}
	feed := &event.Feed{}
	return &Service{
		db:      db,
		reactor: NewReactor(db, feed, client, address, chain),
		signals: &SignalsTransmitter{publisher: feed},
	}, nil
}

// Service is a wallet service.
type Service struct {
	db      *Database
	reactor *Reactor
	signals *SignalsTransmitter
}

// Start reactor and signals transmitter.
func (s *Service) Start(*p2p.Server) error {
	err := s.signals.Start()
	if err != nil {
		return err
	}
	err = s.reactor.Start()
	if err != nil {
		return err
	}
	return nil
}

// Stop reactor, signals  transmitter and close db.
func (s *Service) Stop() error {
	s.reactor.Stop()
	s.signals.Stop()
	return s.db.Close()
}

// APIs returns list of available RPC APIs.
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "wallet",
			Version:   "0.1.0",
			Service:   API{s.db},
			Public:    true,
		},
	}
}

// Protocols returns list of p2p protocols.
func (s *Service) Protocols() []p2p.Protocol {
	return nil
}
