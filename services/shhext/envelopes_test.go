package shhext

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/services/shhext/mailservers"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/suite"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

type EnvelopesMonitorSuite struct {
	suite.Suite

	monitor *EnvelopesMonitor
}

func (s *EnvelopesMonitorSuite) SetupTest() {
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	s.Require().NoError(err)
	s.monitor = &EnvelopesMonitor{
		envelopes: map[common.Hash]EnvelopeState{},
		batches:   map[common.Hash]map[common.Hash]struct{}{},
		mailPeers: mailservers.NewPeerStore(mailservers.NewCache(db)),
	}
}

func (s *EnvelopesMonitorSuite) TestConfirmed() {
	s.monitor.Add(testHash, whisper.NewMessage{})
	s.Contains(s.monitor.envelopes, testHash)
	s.Equal(EnvelopePosted, s.monitor.envelopes[testHash])
	s.monitor.handleEvent(whisper.EnvelopeEvent{
		Event: whisper.EventEnvelopeSent,
		Hash:  testHash,
	})
	s.Contains(s.monitor.envelopes, testHash)
	s.Equal(EnvelopeSent, s.monitor.envelopes[testHash])
}

func (s *EnvelopesMonitorSuite) TestConfirmedWithAcknowledge() {
	testBatch := common.Hash{1}
	pkey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	node := enode.NewV4(&pkey.PublicKey, nil, 0, 0)
	s.Require().NoError(s.monitor.mailPeers.Update([]*enode.Node{node}))
	s.monitor.Add(testHash, whisper.NewMessage{})
	s.Contains(s.monitor.envelopes, testHash)
	s.Equal(EnvelopePosted, s.monitor.envelopes[testHash])
	s.monitor.handleEvent(whisper.EnvelopeEvent{
		Event: whisper.EventEnvelopeSent,
		Hash:  testHash,
		Batch: testBatch,
	})
	s.Equal(EnvelopePosted, s.monitor.envelopes[testHash])
	s.monitor.handleEvent(whisper.EnvelopeEvent{
		Event: whisper.EventBatchAcknowledged,
		Batch: testBatch,
		Peer:  node.ID(),
	})
	s.Contains(s.monitor.envelopes, testHash)
	s.Equal(EnvelopeSent, s.monitor.envelopes[testHash])
}

func (s *EnvelopesMonitorSuite) TestIgnored() {
	s.monitor.handleEvent(whisper.EnvelopeEvent{
		Event: whisper.EventEnvelopeSent,
		Hash:  testHash,
	})
	s.NotContains(s.monitor.envelopes, testHash)
}

func (s *EnvelopesMonitorSuite) TestRemoved() {
	s.monitor.Add(testHash, whisper.NewMessage{})
	s.Contains(s.monitor.envelopes, testHash)
	s.monitor.handleEvent(whisper.EnvelopeEvent{
		Event: whisper.EventEnvelopeExpired,
		Hash:  testHash,
	})
	s.NotContains(s.monitor.envelopes, testHash)
}

func (s *EnvelopesMonitorSuite) TestIgnoreNotFromMailserver() {
	// enables filter in the tracker to drop confirmations from non-mailserver peers
	s.monitor.mailServerConfirmation = true
	s.monitor.Add(testHash, whisper.NewMessage{})
	s.monitor.handleEvent(whisper.EnvelopeEvent{
		Event: whisper.EventEnvelopeSent,
		Hash:  testHash,
		Peer:  enode.ID{1}, // could be empty, doesn't impact test behaviour
	})
	s.Require().Equal(EnvelopePosted, s.monitor.GetState(testHash))
}

func (s *EnvelopesMonitorSuite) TestReceived() {
	s.monitor.Add(testHash, whisper.NewMessage{})
	s.Contains(s.monitor.envelopes, testHash)
	s.monitor.handleEvent(whisper.EnvelopeEvent{
		Event: whisper.EventEnvelopeReceived,
		Hash:  testHash})
	s.NotContains(s.monitor.envelopes, testHash)
}
