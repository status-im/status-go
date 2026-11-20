package transport

import (
	"reflect"
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
)

var (
	testRequestID  = wakutypes.RequestID("req-1")
	testRequestIDs = []wakutypes.RequestID{testRequestID}
	testIDs        = [][]byte{[]byte("id")}
)

type envelopeEventsHandlerMock struct {
	envelopeSentCalls    [][][]byte // slice of EnvelopeSent arguments
	envelopeExpiredCalls [][][]byte // slice of EnvelopeExpired arguments
}

func (h *envelopeEventsHandlerMock) EnvelopeSent(identifiers [][]byte) {
	h.envelopeSentCalls = append(h.envelopeSentCalls, identifiers)
}
func (h *envelopeEventsHandlerMock) EnvelopeExpired(identifiers [][]byte, _ error) {
	h.envelopeExpiredCalls = append(h.envelopeExpiredCalls, identifiers)
}
func (h *envelopeEventsHandlerMock) MailServerRequestCompleted(cryptotypes.Hash, cryptotypes.Hash, []byte, error) {
}
func (h *envelopeEventsHandlerMock) MailServerRequestExpired(cryptotypes.Hash) {
}

type EnvelopesMonitorSuite struct {
	suite.Suite

	monitor           *EnvelopesMonitor
	eventsHandlerMock *envelopeEventsHandlerMock
}

func TestEnvelopesMonitorSuite(t *testing.T) {
	suite.Run(t, new(EnvelopesMonitorSuite))
}

func (s *EnvelopesMonitorSuite) SetupTest() {
	s.eventsHandlerMock = &envelopeEventsHandlerMock{}
	s.monitor = NewEnvelopesMonitor(
		nil,
		EnvelopesMonitorConfig{
			EnvelopeEventsHandler:            s.eventsHandlerMock,
			AwaitOnlyMailServerConfirmations: false,
			IsMailserver:                     func(wakutypes.EnodeID) bool { return false },
			Logger:                           zap.NewNop(),
		},
	)
}

func (s *EnvelopesMonitorSuite) TestEnvelopePosted() {
	err := s.monitor.Add(testIDs, testRequestIDs, []*wakutypes.NewMessage{{}})
	s.Require().NoError(err)
	s.Contains(s.monitor.envelopes, testRequestID)
	s.Equal(EnvelopePosted, s.monitor.envelopes[testRequestID].state)
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event:     wakutypes.EventEnvelopeSent,
		RequestID: testRequestID,
	})
	s.Contains(s.monitor.envelopes, testRequestID)
	s.Equal(EnvelopeSent, s.monitor.envelopes[testRequestID].state)
}

func (s *EnvelopesMonitorSuite) TestEnvelopeExpiredBeforeAdded() {
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event:     wakutypes.EventEnvelopeExpired,
		RequestID: testRequestID,
	})
	s.Require().NoError(s.monitor.Add(testIDs, testRequestIDs, []*wakutypes.NewMessage{{}}))
	s.Equal([][][]byte{testIDs}, s.eventsHandlerMock.envelopeExpiredCalls)
	s.NotContains(s.monitor.envelopes, testRequestID)
	s.Empty(s.monitor.earlyFailures)
}

func (s *EnvelopesMonitorSuite) TestSDSAliasDoesNotProduceSeparateSentMessage() {
	applicationMessageID := []byte{0x01, 0x02}
	sdsMessageID := []byte{0x03, 0x04}

	err := s.monitor.Add([][]byte{applicationMessageID}, testRequestIDs, []*wakutypes.NewMessage{{}})
	s.Require().NoError(err)
	s.monitor.AddSDSAlias(applicationMessageID, sdsMessageID)

	resolvedMessageID, ok := s.monitor.TakeApplicationMessageIDForSDS(sdsMessageID)
	s.Require().True(ok)
	s.Equal(applicationMessageID, resolvedMessageID)
	_, ok = s.monitor.TakeApplicationMessageIDForSDS(sdsMessageID)
	s.False(ok)

	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event:     wakutypes.EventEnvelopeSent,
		RequestID: testRequestID,
	})
	s.Require().Len(s.eventsHandlerMock.envelopeSentCalls, 1)
	s.Equal([][]byte{applicationMessageID}, s.eventsHandlerMock.envelopeSentCalls[0])

	_, ok = s.monitor.TakeApplicationMessageIDForSDS([]byte{0xff})
	s.False(ok)
}

func (s *EnvelopesMonitorSuite) TestEnvelopePostedOutOfOrder() {
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event:     wakutypes.EventEnvelopeSent,
		RequestID: testRequestID,
	})

	err := s.monitor.Add(testIDs, testRequestIDs, []*wakutypes.NewMessage{{}})
	s.Require().NoError(err)
	s.Require().Contains(s.monitor.envelopes, testRequestID)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[testRequestID].state)
}

func (s *EnvelopesMonitorSuite) TestConfirmedWithAcknowledge() {
	testBatch := cryptotypes.Hash{1}
	pkey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	node := enode.NewV4(&pkey.PublicKey, nil, 0, 0)
	err = s.monitor.Add(testIDs, testRequestIDs, []*wakutypes.NewMessage{{}})
	s.Require().NoError(err)
	s.Contains(s.monitor.envelopes, testRequestID)
	s.Equal(EnvelopePosted, s.monitor.envelopes[testRequestID].state)
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event:     wakutypes.EventEnvelopeSent,
		RequestID: testRequestID,
		Batch:     testBatch,
	})
	s.Equal(EnvelopePosted, s.monitor.envelopes[testRequestID].state)
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event: wakutypes.EventBatchAcknowledged,
		Batch: testBatch,
		Peer:  wakutypes.EnodeID(node.ID()),
	})
	s.Contains(s.monitor.envelopes, testRequestID)
	s.Equal(EnvelopeSent, s.monitor.envelopes[testRequestID].state)
}

func (s *EnvelopesMonitorSuite) TestRemoved() {
	err := s.monitor.Add(testIDs, testRequestIDs, []*wakutypes.NewMessage{{}})
	s.Require().NoError(err)
	s.Contains(s.monitor.envelopes, testRequestID)
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event:     wakutypes.EventEnvelopeExpired,
		RequestID: testRequestID,
	})
	s.NotContains(s.monitor.envelopes, testRequestID)
}

func (s *EnvelopesMonitorSuite) TestIgnoreNotFromMailserver() {
	// enables filter in the tracker to drop confirmations from non-mailserver peers
	s.monitor.awaitOnlyMailServerConfirmations = true
	err := s.monitor.Add(testIDs, testRequestIDs, []*wakutypes.NewMessage{{}})
	s.Require().NoError(err)
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event:     wakutypes.EventEnvelopeSent,
		RequestID: testRequestID,
		Peer:      wakutypes.EnodeID{1}, // could be empty, doesn't impact test behaviour
	})
	s.Require().Equal(EnvelopePosted, s.monitor.GetState(testRequestID))
}

func (s *EnvelopesMonitorSuite) TestReceived() {
	s.monitor.isMailserver = func(peer wakutypes.EnodeID) bool {
		return true
	}
	err := s.monitor.Add(testIDs, testRequestIDs, []*wakutypes.NewMessage{{}})
	s.Require().NoError(err)
	s.Contains(s.monitor.envelopes, testRequestID)
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event:     wakutypes.EventEnvelopeReceived,
		RequestID: testRequestID,
	})
	s.Require().Equal(EnvelopeSent, s.monitor.GetState(testRequestID))
}

func (s *EnvelopesMonitorSuite) TestMultipleHashes() {
	messageIDs := [][]byte{[]byte("id1"), []byte("id2")}
	requestIDs := []wakutypes.RequestID{"req-1", "req-2", "req-3"}
	messages := []*wakutypes.NewMessage{{}, {}, {}}

	err := s.monitor.Add(messageIDs, requestIDs, messages)
	s.Require().NoError(err)
	for _, requestID := range requestIDs {
		s.Contains(s.monitor.envelopes, requestID)
	}
	s.Require().Empty(s.eventsHandlerMock.envelopeSentCalls)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[requestIDs[0]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[requestIDs[1]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[requestIDs[2]].state)

	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event:     wakutypes.EventEnvelopeSent,
		RequestID: requestIDs[0],
	})
	s.Require().Empty(s.eventsHandlerMock.envelopeSentCalls)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[requestIDs[0]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[requestIDs[1]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[requestIDs[2]].state)

	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event:     wakutypes.EventEnvelopeSent,
		RequestID: requestIDs[1],
	})
	s.Require().Empty(s.eventsHandlerMock.envelopeSentCalls)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[requestIDs[0]].state)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[requestIDs[1]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[requestIDs[2]].state)

	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event:     wakutypes.EventEnvelopeSent,
		RequestID: requestIDs[2],
	})
	// Identifiers should be marked as sent only if all corresponding envelopes are sent
	s.Require().Len(s.eventsHandlerMock.envelopeSentCalls, 1)
	s.Require().True(reflect.DeepEqual(messageIDs, s.eventsHandlerMock.envelopeSentCalls[0]))
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[requestIDs[0]].state)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[requestIDs[1]].state)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[requestIDs[2]].state)
}

func (s *EnvelopesMonitorSuite) TestMultipleHashes_EnvelopeExpired() {
	messageIDs := [][]byte{[]byte("id1"), []byte("id2")}
	requestIDs := []wakutypes.RequestID{"req-1", "req-2", "req-3"}
	messages := []*wakutypes.NewMessage{{}, {}, {}}

	err := s.monitor.Add(messageIDs, requestIDs, messages)
	s.Require().NoError(err)

	// If any envelope fails, then messageIDs are considered as not sent
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event:     wakutypes.EventEnvelopeExpired,
		RequestID: requestIDs[0],
	})
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event:     wakutypes.EventEnvelopeSent,
		RequestID: requestIDs[1],
	})
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event:     wakutypes.EventEnvelopeSent,
		RequestID: requestIDs[2],
	})

	s.Require().Empty(s.eventsHandlerMock.envelopeSentCalls)
	s.Require().Empty(s.monitor.messageRequestIDs)
	s.Require().Len(s.monitor.envelopes, 2)
}

func (s *EnvelopesMonitorSuite) TestMultipleHashes_Failure() {
	err := s.monitor.Add(testIDs, []wakutypes.RequestID{"req-1", "req-2"}, []*wakutypes.NewMessage{{}})
	s.Require().Error(err)
}
