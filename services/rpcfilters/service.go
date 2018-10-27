package rpcfilters

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

// Make sure that Service implements node.Service interface.
var _ node.Service = (*Service)(nil)

// Service represents out own implementation of personal sign operations.
type Service struct {
	latestBlockChangedEvent        *latestBlockChangedEvent
	transactionSentToUpstreamEvent *transactionSentToUpstreamEvent
	rpc                            rpcProvider

	quit chan struct{}
}

// New returns a new Service.
func New(rpc rpcProvider) *Service {
	provider := &latestBlockProviderRPC{rpc}
	latestBlockChangedEvent := newLatestBlockChangedEvent(provider)
	transactionSentToUpstreamEvent := newTransactionSentToUpstreamEvent()
	return &Service{
		latestBlockChangedEvent:        latestBlockChangedEvent,
		transactionSentToUpstreamEvent: transactionSentToUpstreamEvent,
		rpc:                            rpc,
	}
}

// Protocols returns a new protocols list. In this case, there are none.
func (s *Service) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicAPI(s),
			Public:    true,
		},
	}
}

// Start is run when a service is started.
func (s *Service) Start(server *p2p.Server) error {
	s.quit = make(chan struct{})
	err := s.transactionSentToUpstreamEvent.Start()
	if err != nil {
		return err
	}
	return s.latestBlockChangedEvent.Start()
}

// Stop is run when a service is stopped.
func (s *Service) Stop() error {
	close(s.quit)
	s.transactionSentToUpstreamEvent.Stop()
	s.latestBlockChangedEvent.Stop()
	return nil
}

// TriggerTransactionSentToUpstreamEvent notifies the subscribers
// of the TransactionSentToUpstream event
func (s *Service) TriggerTransactionSentToUpstreamEvent(transactionHash common.Hash) {
	s.transactionSentToUpstreamEvent.Trigger(transactionHash)
}
