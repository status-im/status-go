package shhext

import (
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/services/shhext/mailservers"
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
		cache:                  map[common.Hash]EnvelopeState{},
		batches:                map[common.Hash]map[common.Hash]struct{}{},
		mailPeers:              mailservers.NewPeerStore(),
		mailServerConfirmation: true,
	}
}

func (s *TrackerSuite) TestConfirmed() {
	s.tracker.Add(testHash)
	s.Contains(s.tracker.cache, testHash)
	s.Equal(EnvelopePosted, s.tracker.cache[testHash])
	s.tracker.handleEvent(whisper.EnvelopeEvent{
		Event: whisper.EventEnvelopeSent,
		Hash:  testHash,
	})
	s.Contains(s.tracker.cache, testHash)
	s.Equal(EnvelopeSent, s.tracker.cache[testHash])
}

func (s *TrackerSuite) TestConfirmedWithAcknowledge() {
	testBatch := common.Hash{1}
	pkey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	node := enode.NewV4(&pkey.PublicKey, nil, 0, 0)
	s.tracker.mailPeers.Update([]*enode.Node{node})
	s.tracker.Add(testHash)
	s.Contains(s.tracker.cache, testHash)
	s.Equal(EnvelopePosted, s.tracker.cache[testHash])
	s.tracker.handleEvent(whisper.EnvelopeEvent{
		Event: whisper.EventEnvelopeSent,
		Hash:  testHash,
		Batch: testBatch,
	})
	s.Equal(EnvelopePosted, s.tracker.cache[testHash])
	s.tracker.handleEvent(whisper.EnvelopeEvent{
		Event: whisper.EventBatchAcknowledged,
		Batch: testBatch,
		Peer:  node.ID(),
	})
	s.Contains(s.tracker.cache, testHash)
	s.Equal(EnvelopeSent, s.tracker.cache[testHash])
}

func (s *TrackerSuite) TestIgnored() {
	s.tracker.handleEvent(whisper.EnvelopeEvent{
		Event: whisper.EventEnvelopeSent,
		Hash:  testHash,
	})
	s.NotContains(s.tracker.cache, testHash)
}

func (s *TrackerSuite) TestRemoved() {
	s.tracker.Add(testHash)
	s.Contains(s.tracker.cache, testHash)
	s.tracker.handleEvent(whisper.EnvelopeEvent{
		Event: whisper.EventEnvelopeExpired,
		Hash:  testHash,
	})
	s.NotContains(s.tracker.cache, testHash)
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
