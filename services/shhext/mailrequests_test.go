package shhext

import (
	"errors"
	"testing"
	"time"

	whispertypes "github.com/status-im/status-protocol-go/transport/whisper/types"
	statusproto "github.com/status-im/status-protocol-go/types"
	"github.com/stretchr/testify/suite"
)

var (
	testHash = statusproto.Hash{0x01}
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
		cache:            map[statusproto.Hash]EnvelopeState{},
		requestsRegistry: NewRequestsRegistry(0),
	}
}

func (s *MailRequestMonitorSuite) TestRequestCompleted() {
	mock := newHandlerMock(1)
	s.monitor.handler = mock
	s.monitor.cache[testHash] = MailServerRequestSent
	s.monitor.handleEvent(whispertypes.EnvelopeEvent{
		Event: whispertypes.EventMailServerRequestCompleted,
		Hash:  testHash,
		Data:  &whispertypes.MailServerResponse{},
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
	mock := newHandlerMock(1)
	s.monitor.handler = mock
	s.monitor.cache[testHash] = MailServerRequestSent
	s.monitor.handleEvent(whispertypes.EnvelopeEvent{
		Event: whispertypes.EventMailServerRequestCompleted,
		Hash:  testHash,
		Data:  &whispertypes.MailServerResponse{Error: errors.New("test error")},
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
	mock := newHandlerMock(1)
	s.monitor.handler = mock
	s.monitor.cache[testHash] = MailServerRequestSent
	s.monitor.handleEvent(whispertypes.EnvelopeEvent{
		Event: whispertypes.EventMailServerRequestExpired,
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
