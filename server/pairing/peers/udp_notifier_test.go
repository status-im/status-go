package peers

import (
	"runtime"
	"sync"
	"testing"
	"time"

	udpp2p "github.com/schollz/peerdiscovery"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/server/servertest"
)

func TestUDPPeerDiscoverySuite(t *testing.T) {
	suite.Run(t, new(UDPPeerDiscoverySuite))
}

type UDPPeerDiscoverySuite struct {
	suite.Suite
	servertest.TestLoggerComponents
}

func (s *UDPPeerDiscoverySuite) SetupSuite() {
	s.SetupLoggerComponents()
}

type testSignalLogger struct {
	log  map[string]map[string]bool
	lock sync.Mutex
}

func newTestSignalLogger() *testSignalLogger {
	tsl := new(testSignalLogger)
	tsl.log = make(map[string]map[string]bool)
	return tsl
}

func (t *testSignalLogger) testSignal(h *LocalPairingPeerHello) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if _, ok := t.log[h.Discovered.Address]; !ok {
		t.log[h.Discovered.Address] = make(map[string]bool)
	}
	t.log[h.Discovered.Address][h.DeviceName] = true
}

func (s *UDPPeerDiscoverySuite) TestUDPNotifier() {
	tsl := newTestSignalLogger()

	u1, err := NewUDPNotifier(s.Logger, tsl.testSignal)
	s.Require().NoError(err)

	u2, err := NewUDPNotifier(s.Logger, tsl.testSignal)
	s.Require().NoError(err)

	n1 := "device 1"
	n2 := "device 2"

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		settings, err := u1.MakeUDPP2PSettings(n1, runtime.GOOS)
		s.Require().NoError(err)

		settings.TimeLimit = 2 * time.Second
		settings.Limit = 4
		settings.AllowSelf = true

		_, err = udpp2p.Discover(*settings)
		s.Require().NoError(err)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		settings, err := u2.MakeUDPP2PSettings(n2, runtime.GOOS)
		s.Require().NoError(err)

		settings.TimeLimit = 2 * time.Second
		settings.Limit = 4
		settings.AllowSelf = true

		_, err = udpp2p.Discover(*settings)
		s.Require().NoError(err)
		wg.Done()
	}()

	wg.Wait()

	s.Require().Len(tsl.log, 2)
	for _, address := range tsl.log {
		s.Require().Len(address, 2)

		for device := range address {
			if !(device == n1 || device == n2) {
				s.Require().Failf("unknown device name", device)
			}
		}
	}
}
