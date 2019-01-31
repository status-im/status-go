package shhext

import (
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/suite"
)

var (
	testHash = common.Hash{0x01}
)

func TestTrackerSuite(t *testing.T) {
	suite.Run(t, new(TrackerSuite))
}

type TrackerSuite struct {
	suite.Suite

	tracker *tracker
}

func (s *TrackerSuite) SetupTest() {
	s.tracker = &tracker{
		cache:            map[common.Hash]EnvelopeState{},
		requestsRegistry: NewRequestsRegistry(0),
	}
}

func (s *TrackerSuite) TestRequestCompleted() {
	mock := newHandlerMock(1)
	s.tracker.handler = mock
	s.tracker.cache[testHash] = MailServerRequestSent
	s.tracker.handleEvent(whisper.EnvelopeEvent{
		Event: whisper.EventMailServerRequestCompleted,
		Hash:  testHash,
		Data:  &whisper.MailServerResponse{},
	})
	select {
	case requestID := <-mock.requestsCompleted:
		s.Equal(testHash, requestID)
		s.NotContains(s.tracker.cache, testHash)
	case <-time.After(10 * time.Second):
		s.Fail("timed out while waiting for a request to be completed")
	}
}

func (s *TrackerSuite) TestRequestFailed() {
	mock := newHandlerMock(1)
	s.tracker.handler = mock
	s.tracker.cache[testHash] = MailServerRequestSent
	s.tracker.handleEvent(whisper.EnvelopeEvent{
		Event: whisper.EventMailServerRequestCompleted,
		Hash:  testHash,
		Data:  &whisper.MailServerResponse{Error: errors.New("test error")},
	})
	select {
	case requestID := <-mock.requestsFailed:
		s.Equal(testHash, requestID)
		s.NotContains(s.tracker.cache, testHash)
	case <-time.After(10 * time.Second):
		s.Fail("timed out while waiting for a request to be failed")
	}
}

func (s *TrackerSuite) TestRequestExpiration() {
	mock := newHandlerMock(1)
	s.tracker.handler = mock
	s.tracker.cache[testHash] = MailServerRequestSent
	s.tracker.handleEvent(whisper.EnvelopeEvent{
		Event: whisper.EventMailServerRequestExpired,
		Hash:  testHash,
	})
	select {
	case requestID := <-mock.requestsExpired:
		s.Equal(testHash, requestID)
		s.NotContains(s.tracker.cache, testHash)
	case <-time.After(10 * time.Second):
		s.Fail("timed out while waiting for request expiration")
	}
}
