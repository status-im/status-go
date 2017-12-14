package whisper

import (
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/geth/common"
)

// MailService is a service that provides some additional Whisper API.
type MailService struct {
	nodeManager common.NodeManager
}

// NewMailService returns a new MailService.
func NewMailService(nodeManager common.NodeManager) *MailService {
	return &MailService{nodeManager}
}

// Protocols returns a new protocols list. In this case, there are none.
func (s *MailService) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns a list of new APIs.
func (s *MailService) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "shh",
			Version:   "1.0",
			Service:   NewMailServicePublicAPI(s.nodeManager),
			Public:    true,
		},
	}
}

// Start is run when a service is started.
// It does nothing in this case but is required by `node.Service` interface.
func (s *MailService) Start(server *p2p.Server) error {
	return nil
}

// Stop is run when a service is stopped.
// It does nothing in this case but is required by `node.Service` interface.
func (s *MailService) Stop() error {
	return nil
}
