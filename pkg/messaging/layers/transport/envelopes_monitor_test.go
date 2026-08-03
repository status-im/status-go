package transport

import (
	"reflect"
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/suite"

	types3 "github.com/status-im/status-go/internal/crypto/types"
	types2 "github.com/status-im/status-go/pkg/messaging/waku/types"
)

var (
	testHash   = types3.Hash{0x01}
	testHashes = []types3.Hash{testHash}
	testIDs    = [][]byte{[]byte("id")}
)

type messageEventsHandlerMock struct {
	messagesSentCalls [][][]byte // slice of MessagesSent arguments
}

func (h *messageEventsHandlerMock) MessagesSent(messageIDs [][]byte) {
	h.messagesSentCalls = append(h.messagesSentCalls, messageIDs)
}
func (h *messageEventsHandlerMock) MessagesExpired([][]byte, error) {
}

type EnvelopesMonitorSuite struct {
	suite.Suite

	monitor           *EnvelopesMonitor
	eventsHandlerMock *messageEventsHandlerMock
}

func TestEnvelopesMonitorSuite(t *testing.T) {
	suite.Run(t, new(EnvelopesMonitorSuite))
}

func (s *EnvelopesMonitorSuite) SetupTest() {
	s.eventsHandlerMock = &messageEventsHandlerMock{}
	s.monitor = NewEnvelopesMonitor(
		nil,
		EnvelopesMonitorConfig{
			Logger: zap.NewNop(),
		},
	)
	s.monitor.handler = s.eventsHandlerMock
}

func (s *EnvelopesMonitorSuite) TestEnvelopePosted() {
	err := s.monitor.Add(testIDs, testHashes, []*types2.NewMessage{{}})
	s.Require().NoError(err)
	s.Contains(s.monitor.envelopes, testHash)
	s.Equal(EnvelopePosted, s.monitor.envelopes[testHash].state)
	s.monitor.handleEvent(types2.EnvelopeEvent{
		Event: types2.EventEnvelopeSent,
		Hash:  testHash,
	})
	s.Contains(s.monitor.envelopes, testHash)
	s.Equal(EnvelopeSent, s.monitor.envelopes[testHash].state)
}

func (s *EnvelopesMonitorSuite) TestEnvelopePostedOutOfOrder() {
	s.monitor.handleEvent(types2.EnvelopeEvent{
		Event: types2.EventEnvelopeSent,
		Hash:  testHash,
	})

	err := s.monitor.Add(testIDs, testHashes, []*types2.NewMessage{{}})
	s.Require().NoError(err)
	s.Require().Contains(s.monitor.envelopes, testHash)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[testHash].state)
}

func (s *EnvelopesMonitorSuite) TestRemoved() {
	err := s.monitor.Add(testIDs, testHashes, []*types2.NewMessage{{}})
	s.Require().NoError(err)
	s.Contains(s.monitor.envelopes, testHash)
	s.monitor.handleEvent(types2.EnvelopeEvent{
		Event: types2.EventEnvelopeExpired,
		Hash:  testHash,
	})
	s.NotContains(s.monitor.envelopes, testHash)
}

func (s *EnvelopesMonitorSuite) TestMultipleHashes() {
	messageIDs := [][]byte{[]byte("id1"), []byte("id2")}
	hashes := []types3.Hash{{0x01}, {0x02}, {0x03}}
	messages := []*types2.NewMessage{{}, {}, {}}

	err := s.monitor.Add(messageIDs, hashes, messages)
	s.Require().NoError(err)
	for _, hash := range hashes {
		s.Contains(s.monitor.envelopes, hash)
	}
	s.Require().Empty(s.eventsHandlerMock.messagesSentCalls)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[hashes[0]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[hashes[1]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[hashes[2]].state)

	s.monitor.handleEvent(types2.EnvelopeEvent{
		Event: types2.EventEnvelopeSent,
		Hash:  hashes[0],
	})
	s.Require().Empty(s.eventsHandlerMock.messagesSentCalls)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[hashes[0]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[hashes[1]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[hashes[2]].state)

	s.monitor.handleEvent(types2.EnvelopeEvent{
		Event: types2.EventEnvelopeSent,
		Hash:  hashes[1],
	})
	s.Require().Empty(s.eventsHandlerMock.messagesSentCalls)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[hashes[0]].state)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[hashes[1]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[hashes[2]].state)

	s.monitor.handleEvent(types2.EnvelopeEvent{
		Event: types2.EventEnvelopeSent,
		Hash:  hashes[2],
	})
	// Identifiers should be marked as sent only if all corresponding envelopes are sent
	s.Require().Len(s.eventsHandlerMock.messagesSentCalls, 1)
	s.Require().True(reflect.DeepEqual(messageIDs, s.eventsHandlerMock.messagesSentCalls[0]))
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[hashes[0]].state)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[hashes[1]].state)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[hashes[2]].state)
}

func (s *EnvelopesMonitorSuite) TestMultipleHashes_EnvelopeExpired() {
	messageIDs := [][]byte{[]byte("id1"), []byte("id2")}
	hashes := []types3.Hash{{0x01}, {0x02}, {0x03}}
	messages := []*types2.NewMessage{{}, {}, {}}

	err := s.monitor.Add(messageIDs, hashes, messages)
	s.Require().NoError(err)

	// If any envelope fails, then messageIDs are considered as not sent
	s.monitor.handleEvent(types2.EnvelopeEvent{
		Event: types2.EventEnvelopeExpired,
		Hash:  hashes[0],
	})
	s.monitor.handleEvent(types2.EnvelopeEvent{
		Event: types2.EventEnvelopeSent,
		Hash:  hashes[1],
	})
	s.monitor.handleEvent(types2.EnvelopeEvent{
		Event: types2.EventEnvelopeSent,
		Hash:  hashes[2],
	})

	s.Require().Empty(s.eventsHandlerMock.messagesSentCalls)
	s.Require().Empty(s.monitor.messageEnvelopeHashes)
	s.Require().Len(s.monitor.envelopes, 2)
}

func (s *EnvelopesMonitorSuite) TestMultipleHashes_Failure() {
	err := s.monitor.Add(testIDs, []types3.Hash{{0x01}, {0x02}}, []*types2.NewMessage{{}})
	s.Require().Error(err)
}
