package shhext

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/t/helpers"
	"github.com/stretchr/testify/suite"
)

func newHandlerMock(buf int) handlerMock {
	return handlerMock{
		confirmations:     make(chan common.Hash, buf),
		expirations:       make(chan common.Hash, buf),
		requestsCompleted: make(chan common.Hash, buf),
		requestsExpired:   make(chan common.Hash, buf),
	}
}

type handlerMock struct {
	confirmations     chan common.Hash
	expirations       chan common.Hash
	requestsCompleted chan common.Hash
	requestsExpired   chan common.Hash
}

func (t handlerMock) EnvelopeSent(hash common.Hash) {
	t.confirmations <- hash
}

func (t handlerMock) EnvelopeExpired(hash common.Hash) {
	t.expirations <- hash
}

func (t handlerMock) MailServerRequestCompleted(requestID common.Hash, lastEnvelopeHash common.Hash, cursor []byte) {
	t.requestsCompleted <- requestID
}

func (t handlerMock) MailServerRequestExpired(hash common.Hash) {
	t.requestsExpired <- hash
}

func TestShhExtSuite(t *testing.T) {
	suite.Run(t, new(ShhExtSuite))
}

type ShhExtSuite struct {
	suite.Suite

	nodes    []*node.Node
	services []*Service
	whisper  []*whisper.Whisper
}

func (s *ShhExtSuite) SetupTest() {
	s.nodes = make([]*node.Node, 2)
	s.services = make([]*Service, 2)
	s.whisper = make([]*whisper.Whisper, 2)
	for i := range s.nodes {
		i := i // bind i to be usable in service constructors
		cfg := &node.Config{
			Name: fmt.Sprintf("node-%d", i),
			P2P: p2p.Config{
				NoDiscovery: true,
				MaxPeers:    1,
				ListenAddr:  ":0",
			},
		}
		stack, err := node.New(cfg)
		s.NoError(err)
		s.whisper[i] = whisper.New(nil)
		s.NoError(stack.Register(func(n *node.ServiceContext) (node.Service, error) {
			return s.whisper[i], nil
		}))
		s.services[i] = New(s.whisper[i], nil, nil, os.TempDir(), true)
		s.NoError(stack.Register(func(n *node.ServiceContext) (node.Service, error) {
			return s.services[i], nil
		}))
		s.Require().NoError(stack.Start())
		s.nodes[i] = stack
	}
	s.services[0].tracker.handler = newHandlerMock(1)
}

func (s *ShhExtSuite) TestPostMessageWithConfirmation() {
	mock := newHandlerMock(1)
	s.services[0].tracker.handler = mock
	s.nodes[0].Server().AddPeer(s.nodes[1].Server().Self())
	symID, err := s.whisper[0].GenerateSymKey()
	s.NoError(err)
	client, err := s.nodes[0].Attach()
	s.NoError(err)
	var hash common.Hash
	s.NoError(client.Call(&hash, "shhext_post", whisper.NewMessage{
		SymKeyID:  symID,
		PowTarget: whisper.DefaultMinimumPoW,
		PowTime:   200,
		Topic:     whisper.TopicType{0x01, 0x01, 0x01, 0x01},
		Payload:   []byte("hello"),
	}))
	s.NoError(err)
	select {
	case confirmed := <-mock.confirmations:
		s.Equal(hash, confirmed)
	case <-time.After(5 * time.Second):
		s.Fail("timed out while waiting for confirmation")
	}
}

func (s *ShhExtSuite) TestWaitMessageExpired() {
	mock := newHandlerMock(1)
	s.services[0].tracker.handler = mock
	symID, err := s.whisper[0].GenerateSymKey()
	s.NoError(err)
	client, err := s.nodes[0].Attach()
	s.NoError(err)
	var hash common.Hash
	s.NoError(client.Call(&hash, "shhext_post", whisper.NewMessage{
		SymKeyID:  symID,
		PowTarget: whisper.DefaultMinimumPoW,
		PowTime:   200,
		TTL:       1,
		Topic:     whisper.TopicType{0x01, 0x01, 0x01, 0x01},
		Payload:   []byte("hello"),
	}))
	s.NoError(err)
	select {
	case expired := <-mock.expirations:
		s.Equal(hash, expired)
	case confirmed := <-mock.confirmations:
		s.Fail("unexpected confirmation for hash", confirmed)
	case <-time.After(10 * time.Second):
		s.Fail("timed out while waiting for confirmation")
	}
}

func (s *ShhExtSuite) TestRequestMessagesErrors() {
	var err error

	shh := whisper.New(nil)
	aNode, err := node.New(&node.Config{
		P2P: p2p.Config{
			MaxPeers:    math.MaxInt32,
			NoDiscovery: true,
		},
	}) // in-memory node as no data dir
	s.NoError(err)
	err = aNode.Register(func(*node.ServiceContext) (node.Service, error) {
		return shh, nil
	})
	s.NoError(err)

	err = aNode.Start()
	s.NoError(err)
	defer func() { s.NoError(aNode.Stop()) }()

	mock := newHandlerMock(1)
	service := New(shh, mock, nil, os.TempDir(), false)
	api := NewPublicAPI(service)

	const (
		mailServerPeer = "enode://b7e65e1bedc2499ee6cbd806945af5e7df0e59e4070c96821570bd581473eade24a489f5ec95d060c0db118c879403ab88d827d3766978f28708989d35474f87@[::]:51920"
	)

	var hash []byte

	// invalid MailServer enode address
	hash, err = api.RequestMessages(context.TODO(), MessagesRequest{MailServerPeer: "invalid-address"})
	s.Nil(hash)
	s.EqualError(err, "invalid mailServerPeer value: invalid URL scheme, want \"enode\"")

	// non-existent symmetric key
	hash, err = api.RequestMessages(context.TODO(), MessagesRequest{
		MailServerPeer: mailServerPeer,
		SymKeyID:       "invalid-sym-key-id",
	})
	s.Nil(hash)
	s.EqualError(err, "invalid symKeyID value: non-existent key ID")

	// with a symmetric key
	symKeyID, symKeyErr := shh.AddSymKeyFromPassword("some-pass")
	s.NoError(symKeyErr)
	hash, err = api.RequestMessages(context.TODO(), MessagesRequest{
		MailServerPeer: mailServerPeer,
		SymKeyID:       symKeyID,
	})
	s.Nil(hash)
	s.Contains(err.Error(), "Could not find peer with ID")

	// from is greater than to
	hash, err = api.RequestMessages(context.TODO(), MessagesRequest{
		From: 10,
		To:   5,
	})
	s.Nil(hash)
	s.Contains(err.Error(), "Query range is invalid: from > to (10 > 5)")
}

func (s *ShhExtSuite) TestRequestMessagesSuccess() {
	var err error

	shh := whisper.New(nil)
	aNode, err := node.New(&node.Config{
		P2P: p2p.Config{
			MaxPeers:    math.MaxInt32,
			NoDiscovery: true,
		},
	}) // in-memory node as no data dir
	s.NoError(err)
	err = aNode.Register(func(*node.ServiceContext) (node.Service, error) { return shh, nil })
	s.NoError(err)

	err = aNode.Start()
	s.NoError(err)
	defer func() { err := aNode.Stop(); s.NoError(err) }()

	mock := newHandlerMock(1)
	service := New(shh, mock, nil, os.TempDir(), false)
	api := NewPublicAPI(service)

	// with a peer acting as a mailserver
	// prepare a node first
	mailNode, err := node.New(&node.Config{
		P2P: p2p.Config{
			MaxPeers:    math.MaxInt32,
			NoDiscovery: true,
			ListenAddr:  ":0",
		},
	}) // in-memory node as no data dir
	s.NoError(err)
	err = mailNode.Register(func(*node.ServiceContext) (node.Service, error) {
		return whisper.New(nil), nil
	})
	s.NoError(err)
	err = mailNode.Start()
	s.NoError(err)
	defer func() { s.NoError(mailNode.Stop()) }()

	// add mailPeer as a peer
	waitErr := helpers.WaitForPeerAsync(aNode.Server(), mailNode.Server().Self().String(), p2p.PeerEventTypeAdd, time.Second)
	aNode.Server().AddPeer(mailNode.Server().Self())
	s.NoError(<-waitErr)

	var hash []byte

	// send a request with a symmetric key
	symKeyID, symKeyErr := shh.AddSymKeyFromPassword("some-pass")
	s.NoError(symKeyErr)
	hash, err = api.RequestMessages(context.TODO(), MessagesRequest{
		MailServerPeer: mailNode.Server().Self().String(),
		SymKeyID:       symKeyID,
	})
	s.NoError(err)
	s.NotNil(hash)
	s.Contains(api.service.tracker.cache, common.BytesToHash(hash))

	// Send a request without a symmetric key. In this case,
	// a public key extracted from MailServerPeer will be used.
	hash, err = api.RequestMessages(context.TODO(), MessagesRequest{
		MailServerPeer: mailNode.Server().Self().String(),
	})
	s.NoError(err)
	s.NotNil(hash)
	s.Contains(api.service.tracker.cache, common.BytesToHash(hash))
}

func (s *ShhExtSuite) TestDebugPostSync() {
	mock := newHandlerMock(1)
	s.services[0].tracker.handler = mock
	symID, err := s.whisper[0].GenerateSymKey()
	s.NoError(err)
	s.nodes[0].Server().AddPeer(s.nodes[1].Server().Self())
	client, err := s.nodes[0].Attach()
	s.NoError(err)
	var hash common.Hash

	var testCases = []struct {
		name            string
		msg             whisper.NewMessage
		postSyncTimeout time.Duration
		expectedErr     error
	}{
		{
			name: "timeout",
			msg: whisper.NewMessage{
				SymKeyID:  symID,
				PowTarget: whisper.DefaultMinimumPoW,
				PowTime:   200,
				Topic:     whisper.TopicType{0x01, 0x01, 0x01, 0x01},
				Payload:   []byte("hello"),
			},
			postSyncTimeout: postSyncTimeout,
			expectedErr:     nil,
		},
		{
			name: "invalid message",
			msg: whisper.NewMessage{
				PowTarget: whisper.DefaultMinimumPoW,
				PowTime:   200,
				Topic:     whisper.TopicType{0x01, 0x01, 0x01, 0x01},
				Payload:   []byte("hello"),
			},
			postSyncTimeout: postSyncTimeout,
			expectedErr:     whisper.ErrSymAsym,
		},
		{
			name: "context deadline exceeded",
			msg: whisper.NewMessage{
				SymKeyID:  symID,
				PowTarget: whisper.DefaultMinimumPoW,
				PowTime:   10,
				Topic:     whisper.TopicType{0x01, 0x01, 0x01, 0x01},
				TTL:       100,
				Payload:   []byte("hello"),
			},
			postSyncTimeout: 1 * time.Millisecond,
			expectedErr:     errors.New("context deadline exceeded"),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tc.postSyncTimeout)
			defer cancel()
			err := client.CallContext(ctx, &hash, "debug_postSync", tc.msg)

			if tc.expectedErr != nil {
				s.Equal(tc.expectedErr.Error(), err.Error())
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *ShhExtSuite) TestEnvelopeExpiredOnDebugPostSync() {
	mock := newHandlerMock(1)
	s.services[0].tracker.handler = mock
	symID, err := s.whisper[0].GenerateSymKey()
	s.NoError(err)
	client, err := s.nodes[0].Attach()
	s.NoError(err)
	var hash common.Hash

	ctx, cancel := context.WithTimeout(context.Background(), postSyncTimeout)
	defer cancel()
	err = client.CallContext(ctx, &hash, "debug_postSync", whisper.NewMessage{
		SymKeyID:  symID,
		PowTarget: whisper.DefaultMinimumPoW,
		PowTime:   200,
		Topic:     whisper.TopicType{0x01, 0x01, 0x01, 0x01},
		Payload:   []byte("hello"),
		TTL:       1,
	})

	s.Equal(errEnvelopeExpired.Error(), err.Error())
}

func (s *ShhExtSuite) TearDown() {
	for _, n := range s.nodes {
		s.NoError(n.Stop())
	}
}

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
		cache: map[common.Hash]EnvelopeState{},
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
	s.tracker.AddRequest(testHash, time.After(defaultRequestTimeout*time.Second))
	s.Contains(s.tracker.cache, testHash)
	s.Equal(MailServerRequestSent, s.tracker.cache[testHash])
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

func (s *TrackerSuite) TestRequestExpiration() {
	mock := newHandlerMock(1)
	s.tracker.handler = mock
	c := make(chan time.Time)
	s.tracker.AddRequest(testHash, c)
	s.Contains(s.tracker.cache, testHash)
	s.Equal(MailServerRequestSent, s.tracker.cache[testHash])
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
