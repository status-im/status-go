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
	testHash   = cryptotypes.Hash{0x01}
	testHashes = []cryptotypes.Hash{testHash}
	testIDs    = [][]byte{[]byte("id")}
)

type envelopeEventsHandlerMock struct {
	envelopeSentCalls [][][]byte // slice of EnvelopeSent arguments
}

func (h *envelopeEventsHandlerMock) EnvelopeSent(identifiers [][]byte) {
	h.envelopeSentCalls = append(h.envelopeSentCalls, identifiers)
}
func (h *envelopeEventsHandlerMock) EnvelopeExpired([][]byte, error) {
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
	err := s.monitor.Add(testIDs, testHashes, []*wakutypes.NewMessage{{}})
	s.Require().NoError(err)
	s.Contains(s.monitor.envelopes, testHash)
	s.Equal(EnvelopePosted, s.monitor.envelopes[testHash].state)
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event: wakutypes.EventEnvelopeSent,
		Hash:  testHash,
	})
	s.Contains(s.monitor.envelopes, testHash)
	s.Equal(EnvelopeSent, s.monitor.envelopes[testHash].state)
}

func (s *EnvelopesMonitorSuite) TestSDSAliasDoesNotProduceSeparateSentMessage() {
	applicationMessageID := []byte{0x01, 0x02}
	sdsMessageID := []byte{0x03, 0x04}

	err := s.monitor.Add([][]byte{applicationMessageID}, testHashes, []*wakutypes.NewMessage{{}})
	s.Require().NoError(err)
	s.monitor.AddSDSAlias(applicationMessageID, sdsMessageID)

	resolvedMessageID, ok := s.monitor.TakeApplicationMessageIDForSDS(sdsMessageID)
	s.Require().True(ok)
	s.Equal(applicationMessageID, resolvedMessageID)
	_, ok = s.monitor.TakeApplicationMessageIDForSDS(sdsMessageID)
	s.False(ok)

	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event: wakutypes.EventEnvelopeSent,
		Hash:  testHash,
	})
	s.Require().Len(s.eventsHandlerMock.envelopeSentCalls, 1)
	s.Equal([][]byte{applicationMessageID}, s.eventsHandlerMock.envelopeSentCalls[0])

	_, ok = s.monitor.TakeApplicationMessageIDForSDS([]byte{0xff})
	s.False(ok)
}

func (s *EnvelopesMonitorSuite) TestEnvelopePostedOutOfOrder() {
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event: wakutypes.EventEnvelopeSent,
		Hash:  testHash,
	})

	err := s.monitor.Add(testIDs, testHashes, []*wakutypes.NewMessage{{}})
	s.Require().NoError(err)
	s.Require().Contains(s.monitor.envelopes, testHash)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[testHash].state)
}

func (s *EnvelopesMonitorSuite) TestConfirmedWithAcknowledge() {
	testBatch := cryptotypes.Hash{1}
	pkey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	node := enode.NewV4(&pkey.PublicKey, nil, 0, 0)
	err = s.monitor.Add(testIDs, testHashes, []*wakutypes.NewMessage{{}})
	s.Require().NoError(err)
	s.Contains(s.monitor.envelopes, testHash)
	s.Equal(EnvelopePosted, s.monitor.envelopes[testHash].state)
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event: wakutypes.EventEnvelopeSent,
		Hash:  testHash,
		Batch: testBatch,
	})
	s.Equal(EnvelopePosted, s.monitor.envelopes[testHash].state)
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event: wakutypes.EventBatchAcknowledged,
		Batch: testBatch,
		Peer:  wakutypes.EnodeID(node.ID()),
	})
	s.Contains(s.monitor.envelopes, testHash)
	s.Equal(EnvelopeSent, s.monitor.envelopes[testHash].state)
}

func (s *EnvelopesMonitorSuite) TestRemoved() {
	err := s.monitor.Add(testIDs, testHashes, []*wakutypes.NewMessage{{}})
	s.Require().NoError(err)
	s.Contains(s.monitor.envelopes, testHash)
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event: wakutypes.EventEnvelopeExpired,
		Hash:  testHash,
	})
	s.NotContains(s.monitor.envelopes, testHash)
}

func (s *EnvelopesMonitorSuite) TestIgnoreNotFromMailserver() {
	// enables filter in the tracker to drop confirmations from non-mailserver peers
	s.monitor.awaitOnlyMailServerConfirmations = true
	err := s.monitor.Add(testIDs, testHashes, []*wakutypes.NewMessage{{}})
	s.Require().NoError(err)
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event: wakutypes.EventEnvelopeSent,
		Hash:  testHash,
		Peer:  wakutypes.EnodeID{1}, // could be empty, doesn't impact test behaviour
	})
	s.Require().Equal(EnvelopePosted, s.monitor.GetState(testHash))
}

func (s *EnvelopesMonitorSuite) TestReceived() {
	s.monitor.isMailserver = func(peer wakutypes.EnodeID) bool {
		return true
	}
	err := s.monitor.Add(testIDs, testHashes, []*wakutypes.NewMessage{{}})
	s.Require().NoError(err)
	s.Contains(s.monitor.envelopes, testHash)
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event: wakutypes.EventEnvelopeReceived,
		Hash:  testHash,
	})
	s.Require().Equal(EnvelopeSent, s.monitor.GetState(testHash))
}

func (s *EnvelopesMonitorSuite) TestMultipleHashes() {
	messageIDs := [][]byte{[]byte("id1"), []byte("id2")}
	hashes := []cryptotypes.Hash{{0x01}, {0x02}, {0x03}}
	messages := []*wakutypes.NewMessage{{}, {}, {}}

	err := s.monitor.Add(messageIDs, hashes, messages)
	s.Require().NoError(err)
	for _, hash := range hashes {
		s.Contains(s.monitor.envelopes, hash)
	}
	s.Require().Empty(s.eventsHandlerMock.envelopeSentCalls)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[hashes[0]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[hashes[1]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[hashes[2]].state)

	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event: wakutypes.EventEnvelopeSent,
		Hash:  hashes[0],
	})
	s.Require().Empty(s.eventsHandlerMock.envelopeSentCalls)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[hashes[0]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[hashes[1]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[hashes[2]].state)

	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event: wakutypes.EventEnvelopeSent,
		Hash:  hashes[1],
	})
	s.Require().Empty(s.eventsHandlerMock.envelopeSentCalls)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[hashes[0]].state)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[hashes[1]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[hashes[2]].state)

	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event: wakutypes.EventEnvelopeSent,
		Hash:  hashes[2],
	})
	// Identifiers should be marked as sent only if all corresponding envelopes are sent
	s.Require().Len(s.eventsHandlerMock.envelopeSentCalls, 1)
	s.Require().True(reflect.DeepEqual(messageIDs, s.eventsHandlerMock.envelopeSentCalls[0]))
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[hashes[0]].state)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[hashes[1]].state)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[hashes[2]].state)
}

func (s *EnvelopesMonitorSuite) TestMultipleHashes_EnvelopeExpired() {
	messageIDs := [][]byte{[]byte("id1"), []byte("id2")}
	hashes := []cryptotypes.Hash{{0x01}, {0x02}, {0x03}}
	messages := []*wakutypes.NewMessage{{}, {}, {}}

	err := s.monitor.Add(messageIDs, hashes, messages)
	s.Require().NoError(err)

	// If any envelope fails, then messageIDs are considered as not sent
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event: wakutypes.EventEnvelopeExpired,
		Hash:  hashes[0],
	})
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event: wakutypes.EventEnvelopeSent,
		Hash:  hashes[1],
	})
	s.monitor.handleEvent(wakutypes.EnvelopeEvent{
		Event: wakutypes.EventEnvelopeSent,
		Hash:  hashes[2],
	})

	s.Require().Empty(s.eventsHandlerMock.envelopeSentCalls)
	s.Require().Empty(s.monitor.messageEnvelopeHashes)
	s.Require().Len(s.monitor.envelopes, 2)
}

func (s *EnvelopesMonitorSuite) TestMultipleHashes_Failure() {
	err := s.monitor.Add(testIDs, []cryptotypes.Hash{{0x01}, {0x02}}, []*wakutypes.NewMessage{{}})
	s.Require().Error(err)
}
