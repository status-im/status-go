package transport

import (
	"context"
	"reflect"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/status-im/status-go/internal/crypto"
	types3 "github.com/status-im/status-go/internal/crypto/types"
	types2 "github.com/status-im/status-go/pkg/messaging/waku/types"
)

var (
	testHash   = types3.Hash{0x01}
	testHashes = []types3.Hash{testHash}
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
func (h *envelopeEventsHandlerMock) MailServerRequestCompleted(types3.Hash, types3.Hash, []byte, error) {
}
func (h *envelopeEventsHandlerMock) MailServerRequestExpired(types3.Hash) {
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
			MaxAttempts:                      6,
			AwaitOnlyMailServerConfirmations: false,
			IsMailserver:                     func(types2.EnodeID) bool { return false },
			Logger:                           zap.NewNop(),
		},
	)
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

func (s *EnvelopesMonitorSuite) TestConfirmedWithAcknowledge() {
	testBatch := types3.Hash{1}
	pkey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	node := enode.NewV4(&pkey.PublicKey, nil, 0, 0)
	err = s.monitor.Add(testIDs, testHashes, []*types2.NewMessage{{}})
	s.Require().NoError(err)
	s.Contains(s.monitor.envelopes, testHash)
	s.Equal(EnvelopePosted, s.monitor.envelopes[testHash].state)
	s.monitor.handleEvent(types2.EnvelopeEvent{
		Event: types2.EventEnvelopeSent,
		Hash:  testHash,
		Batch: testBatch,
	})
	s.Equal(EnvelopePosted, s.monitor.envelopes[testHash].state)
	s.monitor.handleEvent(types2.EnvelopeEvent{
		Event: types2.EventBatchAcknowledged,
		Batch: testBatch,
		Peer:  types2.EnodeID(node.ID()),
	})
	s.Contains(s.monitor.envelopes, testHash)
	s.Equal(EnvelopeSent, s.monitor.envelopes[testHash].state)
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

func (s *EnvelopesMonitorSuite) TestIgnoreNotFromMailserver() {
	// enables filter in the tracker to drop confirmations from non-mailserver peers
	s.monitor.awaitOnlyMailServerConfirmations = true
	err := s.monitor.Add(testIDs, testHashes, []*types2.NewMessage{{}})
	s.Require().NoError(err)
	s.monitor.handleEvent(types2.EnvelopeEvent{
		Event: types2.EventEnvelopeSent,
		Hash:  testHash,
		Peer:  types2.EnodeID{1}, // could be empty, doesn't impact test behaviour
	})
	s.Require().Equal(EnvelopePosted, s.monitor.GetState(testHash))
}

func (s *EnvelopesMonitorSuite) TestReceived() {
	s.monitor.isMailserver = func(peer types2.EnodeID) bool {
		return true
	}
	err := s.monitor.Add(testIDs, testHashes, []*types2.NewMessage{{}})
	s.Require().NoError(err)
	s.Contains(s.monitor.envelopes, testHash)
	s.monitor.handleEvent(types2.EnvelopeEvent{
		Event: types2.EventEnvelopeReceived,
		Hash:  testHash,
	})
	s.Require().Equal(EnvelopeSent, s.monitor.GetState(testHash))
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
	s.Require().Empty(s.eventsHandlerMock.envelopeSentCalls)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[hashes[0]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[hashes[1]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[hashes[2]].state)

	s.monitor.handleEvent(types2.EnvelopeEvent{
		Event: types2.EventEnvelopeSent,
		Hash:  hashes[0],
	})
	s.Require().Empty(s.eventsHandlerMock.envelopeSentCalls)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[hashes[0]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[hashes[1]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[hashes[2]].state)

	s.monitor.handleEvent(types2.EnvelopeEvent{
		Event: types2.EventEnvelopeSent,
		Hash:  hashes[1],
	})
	s.Require().Empty(s.eventsHandlerMock.envelopeSentCalls)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[hashes[0]].state)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[hashes[1]].state)
	s.Require().Equal(EnvelopePosted, s.monitor.envelopes[hashes[2]].state)

	s.monitor.handleEvent(types2.EnvelopeEvent{
		Event: types2.EventEnvelopeSent,
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

	s.Require().Empty(s.eventsHandlerMock.envelopeSentCalls)
	s.Require().Empty(s.monitor.messageEnvelopeHashes)
	s.Require().Len(s.monitor.envelopes, 2)
}

func (s *EnvelopesMonitorSuite) TestMultipleHashes_Failure() {
	err := s.monitor.Add(testIDs, []types3.Hash{{0x01}, {0x02}}, []*types2.NewMessage{{}})
	s.Require().Error(err)
}

func (s *EnvelopesMonitorSuite) TestRetryOnce() {
	s.monitor.api = &mockWakuAPI{}
	err := s.monitor.Add(testIDs, testHashes, []*types2.NewMessage{{}})
	s.Require().NoError(err)
	envelope := s.monitor.envelopes[testHash]
	envelope.attempts = 2
	envelope.lastAttemptTime = time.Now().Add(-20 * time.Second)
	s.monitor.retryQueue = append(s.monitor.retryQueue, envelope)

	s.monitor.retryOnce()

	s.Require().Equal(3, envelope.attempts)
	s.Require().Len(s.monitor.retryQueue, 0)
	s.Require().Equal(envelope.envelopeHashID, s.monitor.envelopes[envelope.envelopeHashID].envelopeHashID)
}

func (s *EnvelopesMonitorSuite) TestRetryLoopSkipsWhenPaused() {
	s.monitor.MarkPaused()
	defer s.monitor.MarkResumed()

	mockAPI := &mockWakuAPI{}
	s.monitor.api = mockAPI
	err := s.monitor.Add(testIDs, testHashes, []*types2.NewMessage{{}})
	s.Require().NoError(err)
	envelope := s.monitor.envelopes[testHash]
	envelope.attempts = 2
	envelope.lastAttemptTime = time.Now().Add(-20 * time.Second)
	s.monitor.retryQueue = append(s.monitor.retryQueue, envelope)

	s.monitor.quit = make(chan struct{})
	go s.monitor.retryLoop()
	time.Sleep(1100 * time.Millisecond)
	close(s.monitor.quit)
	time.Sleep(50 * time.Millisecond)

	s.Require().Equal(0, mockAPI.postCalls)
}

type mockWakuAPI struct {
	postCalls int
}

func (m *mockWakuAPI) Post(ctx context.Context, msg types2.NewMessage) ([]byte, error) {
	m.postCalls++
	return []byte{0x01}, nil
}

func (m *mockWakuAPI) AddPrivateKey(ctx context.Context, privateKey types3.HexBytes) (string, error) {
	return "", nil
}
func (m *mockWakuAPI) DeleteKeyPair(ctx context.Context, key string) (bool, error) {
	return false, nil
}
func (m *mockWakuAPI) NewMessageFilter(req types2.Criteria) (string, error) {
	return "", nil
}
func (m *mockWakuAPI) GetFilterMessages(id string) ([]*types2.Message, error) {
	return nil, nil
}
