package pairing

import (
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/httpserver"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/platform"
	"github.com/status-im/status-go/internal/signal"
	"github.com/status-im/status-go/pkg/services/pairing/peers"
)

type PeerNotifier struct {
	// The backend session the notifier was started in; its events are sent in it.
	signalSession signal.Session
	logger        *zap.Logger
	stop          chan struct{}
	terminator    sync.Once
}

func NewPeerNotifier() *PeerNotifier {
	logger := logutils.ZapLogger().Named("PeerNotifier")
	stop := make(chan struct{})

	return &PeerNotifier{
		signalSession: signal.CurrentSession(),
		logger:        logger,
		stop:          stop,
	}
}

func (p *PeerNotifier) terminateIn(d time.Duration) {
	p.terminator.Do(func() {
		time.Sleep(d)
		p.stop <- struct{}{}
	})
}

func (p *PeerNotifier) handler(hello *peers.LocalPairingPeerHello) {
	p.signalSession.SendLocalPairingEvent(Event{Type: EventPeerDiscovered, Action: ActionPeerDiscovery, Data: hello})
	p.logger.Debug("received peers.LocalPairingPeerHello message", zap.Any("hello message", hello))
	p.terminateIn(5 * time.Second)
}

func (p *PeerNotifier) Search() error {
	// TODO until we can resolve Android errors when calling net.Interfaces() just noop. Sorry Android
	if platform.OperatingSystemIs(platform.AndroidPlatform) {
		return nil
	}

	dn, err := httpserver.GetDeviceName()
	if err != nil {
		return err
	}

	return peers.Search(dn, runtime.GOOS, p.handler, p.stop, p.logger)
}
