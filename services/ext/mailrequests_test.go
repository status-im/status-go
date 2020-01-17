package ext

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/types"
)

var (
	testHash = types.Hash{0x01}
)

func TestMailRequestMonitorSuite(t *testing.T) {
	suite.Run(t, new(MailRequestMonitorSuite))
}

type MailRequestMonitorSuite struct {
	suite.Suite

	monitor *MailRequestMonitor
}

func (s *MailRequestMonitorSuite) SetupTest() {
	s.monitor = &MailRequestMonitor{
		cache:            map[types.Hash]EnvelopeState{},
		requestsRegistry: NewRequestsRegistry(0),
	}
}

func (s *MailRequestMonitorSuite) TestRequestCompleted() {
	mock := NewHandlerMock(1)
	s.monitor.handler = mock
	s.monitor.cache[testHash] = MailServerRequestSent
	s.monitor.handleEvent(types.EnvelopeEvent{
		Event: types.EventMailServerRequestCompleted,
		Hash:  testHash,
		Data:  &types.MailServerResponse{},
	})
	select {
	case requestID := <-mock.requestsCompleted:
		s.Equal(testHash, requestID)
		s.NotContains(s.monitor.cache, testHash)
	case <-time.After(10 * time.Second):
		s.Fail("timed out while waiting for a request to be completed")
	}
}

func (s *MailRequestMonitorSuite) TestRequestFailed() {
	mock := NewHandlerMock(1)
	s.monitor.handler = mock
	s.monitor.cache[testHash] = MailServerRequestSent
	s.monitor.handleEvent(types.EnvelopeEvent{
		Event: types.EventMailServerRequestCompleted,
		Hash:  testHash,
		Data:  &types.MailServerResponse{Error: errors.New("test error")},
	})
	select {
	case requestID := <-mock.requestsFailed:
		s.Equal(testHash, requestID)
		s.NotContains(s.monitor.cache, testHash)
	case <-time.After(10 * time.Second):
		s.Fail("timed out while waiting for a request to be failed")
	}
}

func (s *MailRequestMonitorSuite) TestRequestExpiration() {
	mock := NewHandlerMock(1)
	s.monitor.handler = mock
	s.monitor.cache[testHash] = MailServerRequestSent
	s.monitor.handleEvent(types.EnvelopeEvent{
		Event: types.EventMailServerRequestExpired,
		Hash:  testHash,
	})
	select {
	case requestID := <-mock.requestsExpired:
		s.Equal(testHash, requestID)
		s.NotContains(s.monitor.cache, testHash)
	case <-time.After(10 * time.Second):
		s.Fail("timed out while waiting for request expiration")
	}
}
