package wallet

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

func NewService(db *Database, client *ethclient.Client, address common.Address) *Service {
	feed := &event.Feed{}
	return &Service{
		db:      db,
		reactor: NewReactor(db, feed, client, address),
		signals: &SignalsTransmitter{publisher: feed},
	}
}

type Service struct {
	db      *Database
	reactor *Reactor
	signals *SignalsTransmitter
}

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

func (s *Service) Stop() error {
	s.reactor.Stop()
	s.signals.Stop()
	return nil
}

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

func (s *Service) Protocols() []p2p.Protocol {
	return nil
}
