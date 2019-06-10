package wallet

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

// NewService initializes service instance.
func NewService() *Service {
	feed := &event.Feed{}
	return &Service{
		feed:    feed,
		signals: &SignalsTransmitter{publisher: feed},
	}
}

// Service is a wallet service.
type Service struct {
	feed    *event.Feed
	db      *Database
	reactor *Reactor
	signals *SignalsTransmitter
}

// Start signals transmitter.
func (s *Service) Start(*p2p.Server) error {
	return s.signals.Start()
}

// StartReactor separately because it requires known ethereum address, which will become available only after login.
func (s *Service) StartReactor(dbpath, password string, client *ethclient.Client, accounts []common.Address, chain *big.Int) error {
	db, err := InitializeDB(dbpath, password)
	if err != nil {
		return err
	}
	reactor := NewReactor(db, s.feed, client, accounts, chain)
	err = reactor.Start()
	if err != nil {
		return err
	}
	s.db = db
	s.reactor = reactor
	return nil
}

// StopReactor stops reactor and closes database.
func (s *Service) StopReactor() error {
	if s.reactor == nil {
		return nil
	}
	s.reactor.Stop()
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Stop reactor, signals transmitter and close db.
func (s *Service) Stop() error {
	log.Info("wallet will be stopped")
	err := s.StopReactor()
	s.signals.Stop()
	log.Info("wallet stopped")
	return err
}

// APIs returns list of available RPC APIs.
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "wallet",
			Version:   "0.1.0",
			Service:   &API{s},
			Public:    true,
		},
	}
}

// Protocols returns list of p2p protocols.
func (s *Service) Protocols() []p2p.Protocol {
	return nil
}
