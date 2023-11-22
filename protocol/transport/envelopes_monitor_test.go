package transport

import (
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
)

var (
	testHash = types.Hash{0x01}
	testIDs  = [][]byte{[]byte("id")}
)

type EnvelopesMonitorSuite struct {
	suite.Suite

	monitor *EnvelopesMonitor
}

func TestEnvelopesMonitorSuite(t *testing.T) {
	suite.Run(t, new(EnvelopesMonitorSuite))
}

func (s *EnvelopesMonitorSuite) SetupTest() {
	s.monitor = NewEnvelopesMonitor(
		nil,
		EnvelopesMonitorConfig{
			EnvelopeEventsHandler:            nil,
			MaxAttempts:                      0,
			AwaitOnlyMailServerConfirmations: false,
			IsMailserver:                     func(types.EnodeID) bool { return false },
			Logger:                           zap.NewNop(),
		},
	)
}

func (s *EnvelopesMonitorSuite) TestEnvelopePosted() {
	s.monitor.Add(testIDs, testHash, types.NewMessage{})
	s.Contains(s.monitor.envelopes, testHash)
	s.Equal(EnvelopePosted, s.monitor.envelopes[testHash].state)
	s.monitor.handleEvent(types.EnvelopeEvent{
		Event: types.EventEnvelopeSent,
		Hash:  testHash,
	})
	s.Contains(s.monitor.envelopes, testHash)
	s.Equal(EnvelopeSent, s.monitor.envelopes[testHash].state)
}

func (s *EnvelopesMonitorSuite) TestEnvelopePostedOutOfOrder() {
	s.monitor.handleEvent(types.EnvelopeEvent{
		Event: types.EventEnvelopeSent,
		Hash:  testHash,
	})

	s.monitor.Add(testIDs, testHash, types.NewMessage{})
	s.Require().Contains(s.monitor.envelopes, testHash)
	s.Require().Equal(EnvelopeSent, s.monitor.envelopes[testHash].state)
}

func (s *EnvelopesMonitorSuite) TestConfirmedWithAcknowledge() {
	testBatch := types.Hash{1}
	pkey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	node := enode.NewV4(&pkey.PublicKey, nil, 0, 0)
	s.monitor.Add(testIDs, testHash, types.NewMessage{})
	s.Contains(s.monitor.envelopes, testHash)
	s.Equal(EnvelopePosted, s.monitor.envelopes[testHash].state)
	s.monitor.handleEvent(types.EnvelopeEvent{
		Event: types.EventEnvelopeSent,
		Hash:  testHash,
		Batch: testBatch,
	})
	s.Equal(EnvelopePosted, s.monitor.envelopes[testHash].state)
	s.monitor.handleEvent(types.EnvelopeEvent{
		Event: types.EventBatchAcknowledged,
		Batch: testBatch,
		Peer:  types.EnodeID(node.ID()),
	})
	s.Contains(s.monitor.envelopes, testHash)
	s.Equal(EnvelopeSent, s.monitor.envelopes[testHash].state)
}

func (s *EnvelopesMonitorSuite) TestRemoved() {
	s.monitor.Add(testIDs, testHash, types.NewMessage{})
	s.Contains(s.monitor.envelopes, testHash)
	s.monitor.handleEvent(types.EnvelopeEvent{
		Event: types.EventEnvelopeExpired,
		Hash:  testHash,
	})
	s.NotContains(s.monitor.envelopes, testHash)
}

func (s *EnvelopesMonitorSuite) TestIgnoreNotFromMailserver() {
	// enables filter in the tracker to drop confirmations from non-mailserver peers
	s.monitor.awaitOnlyMailServerConfirmations = true
	s.monitor.Add(testIDs, testHash, types.NewMessage{})
	s.monitor.handleEvent(types.EnvelopeEvent{
		Event: types.EventEnvelopeSent,
		Hash:  testHash,
		Peer:  types.EnodeID{1}, // could be empty, doesn't impact test behaviour
	})
	s.Require().Equal(EnvelopePosted, s.monitor.GetState(testHash))
}

func (s *EnvelopesMonitorSuite) TestReceived() {
	s.monitor.isMailserver = func(peer types.EnodeID) bool {
		return true
	}
	s.monitor.Add(testIDs, testHash, types.NewMessage{})
	s.Contains(s.monitor.envelopes, testHash)
	s.monitor.handleEvent(types.EnvelopeEvent{
		Event: types.EventEnvelopeReceived,
		Hash:  testHash,
	})
	s.Require().Equal(EnvelopeSent, s.monitor.GetState(testHash))
}
