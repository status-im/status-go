package destructive

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/event"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/t/e2e"
	. "github.com/status-im/status-go/t/utils"
)

func TestSyncSuiteNetworkConnection(t *testing.T) {
	suite.Run(t, &SyncTestSuite{controller: new(NetworkConnectionController)})
}

type SyncTestSuite struct {
	suite.Suite

	backend    *api.StatusBackend
	controller *NetworkConnectionController

	tempDir string
}

func (s *SyncTestSuite) SetupTest() {
	s.backend = api.NewStatusBackend()
	config, err := e2e.MakeTestNodeConfig(GetNetworkID())
	s.Require().NoError(err)
	s.tempDir, err = ioutil.TempDir("/tmp", "status-sync-chain")
	s.Require().NoError(err)
	config.LightEthConfig.Enabled = true
	config.WhisperConfig.Enabled = false
	s.Require().NoError(s.backend.StartNode(config))
}

func (s *SyncTestSuite) TearDown() {
	err := s.backend.StopNode()
	if len(s.tempDir) != 0 {
		err = os.RemoveAll(s.tempDir)
	}
	s.Require().NoError(err)
}

func (s *SyncTestSuite) waitForProgress(d *downloader.Downloader) {
	initialBlock := d.Progress().CurrentBlock
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-time.After(30 * time.Second):
			s.Require().Fail("timed out waiting for fetching new headers")
		case <-ticker.C:
			log.Info("sync progress", "current", d.Progress().CurrentBlock, "initial", initialBlock)
			if d.Progress().CurrentBlock > initialBlock {
				return
			}
		}
	}
}

func (s *SyncTestSuite) consumeExpectedEvent(subscription *event.TypeMuxSubscription, expectedEvent interface{}) {
	select {
	case ev := <-subscription.Chan():
		if reflect.TypeOf(expectedEvent) != reflect.TypeOf(ev.Data) {
			s.Require().Fail("received unexpected event")
		}
	case <-time.After(60 * time.Second):
		s.Require().Fail(("timeout waiting for an event"))
	}
}

func (s *SyncTestSuite) TestSyncChain() {
	les, err := s.backend.NodeManager().LightEthereumService()
	s.Require().NoError(err)
	subscription := les.EventMux().Subscribe(
		downloader.StartEvent{}, downloader.DoneEvent{}, downloader.FailedEvent{})
	defer subscription.Unsubscribe()
	s.consumeExpectedEvent(subscription, downloader.StartEvent{})
	// wait for downloader to start festching new headers
	s.waitForProgress(les.Downloader())
	s.Require().NoError(s.controller.Enable())
	s.consumeExpectedEvent(subscription, downloader.FailedEvent{})
	s.Require().NoError(s.controller.Disable())
	s.consumeExpectedEvent(subscription, downloader.StartEvent{})
	s.waitForProgress(les.Downloader())
}
