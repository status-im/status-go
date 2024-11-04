package activitycenter

import (
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/protocol"
)

type Service struct {
	messenger          *protocol.Messenger
	walletConnectsFeed *event.Feed
	controller         *Controller
}

func NewService(walletConnectFeed *event.Feed) *Service {
	return &Service{
		walletConnectsFeed: walletConnectFeed,
		controller:         NewController(walletConnectFeed),
	}
}

func (s *Service) Init(messenger *protocol.Messenger) {
	logutils.ZapLogger().Debug("Initializing activity center service")
	s.messenger = messenger
}

func (s *Service) Start() error {
	if s.messenger != nil {
		logutils.ZapLogger().Debug("Starting activity center service")
		s.controller.Start(s.messenger)
	} else {
		logutils.ZapLogger().Error("Activity center service not started, messenger is nil")
	}
	return nil
}

func (s *Service) Stop() error {
	s.controller.Stop()
	return nil
}

func (s *Service) Protocols() []p2p.Protocol {
	return nil
}

func (s *Service) APIs() []rpc.API {
	return nil
}
