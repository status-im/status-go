package pairing

import (
	"runtime"

	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/server/pairing/peers"
	"github.com/status-im/status-go/signal"
)

type PeerNotifier struct {
	logger *zap.Logger
	stop   chan struct{}
}

func NewPeerNotifier() *PeerNotifier {
	logger := logutils.ZapLogger().Named("PeerNotifier")
	stop := make(chan struct{})

	return &PeerNotifier{
		logger: logger,
		stop:   stop,
	}
}

func (p *PeerNotifier) handler(hello *peers.LocalPairingPeerHello) {
	signal.SendLocalPairingEvent(Event{Type: EventPeerDiscovered, Action: ActionPeerDiscovery, Data: hello})
	p.logger.Debug("received peers.LocalPairingPeerHello message", zap.Any("hello message", hello))
	// TODO p.stop <- struct{}{} Don't do this immediately start a countdown to kill after 5 seconds to allow the
	//  peer to discover us.
}

func (p *PeerNotifier) Search() error {
	dn, err := server.GetDeviceName()
	if err != nil {
		return err
	}

	return peers.Search(dn, runtime.GOOS, p.handler, p.stop, p.logger)
}
