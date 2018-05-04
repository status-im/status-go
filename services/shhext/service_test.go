package shhext

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/stretchr/testify/suite"
)

func newHandlerMock(buf int) handlerMock {
	return handlerMock{
		confirmations: make(chan common.Hash, buf),
		expirations:   make(chan common.Hash, buf),
	}
}

type handlerMock struct {
	confirmations chan common.Hash
	expirations   chan common.Hash
}

func (t handlerMock) EnvelopeSent(hash common.Hash) {
	t.confirmations <- hash
}

func (t handlerMock) EnvelopeExpired(hash common.Hash) {
	t.expirations <- hash
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
		s.services[i] = New(s.whisper[i], nil)
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
	case <-time.After(time.Second):
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
	case <-time.After(2 * time.Second):
		s.Fail("timed out while waiting for confirmation")
	}
}

func (s *ShhExtSuite) TestRequestMessagesDefaults() {
	r := MessagesRequest{}
	r.setDefaults(time.Now())
	s.NotZero(r.From)
	s.InEpsilon(uint32(time.Now().UTC().Unix()), r.To, 1.0)
}

func (s *ShhExtSuite) TestRequestMessages() {
	var err error

	shh := whisper.New(nil)
	aNode, err := node.New(&node.Config{
		P2P: p2p.Config{
			MaxPeers:    math.MaxInt32,
			NoDiscovery: true,
		},
	}) // in-memory node as no data dir
	s.NoError(err)
	err = aNode.Register(func(_ *node.ServiceContext) (node.Service, error) {
		return shh, nil
	})
	s.NoError(err)

	err = aNode.Start()
	s.NoError(err)
	defer func() {
		err := aNode.Stop()
		s.NoError(err)
	}()

	mock := newHandlerMock(1)
	service := New(shh, mock)
	api := NewPublicAPI(service)

	const (
		mailServerPeer = "enode://b7e65e1bedc2499ee6cbd806945af5e7df0e59e4070c96821570bd581473eade24a489f5ec95d060c0db118c879403ab88d827d3766978f28708989d35474f87@[::]:51920"
	)

	var result bool

	// invalid MailServer enode address
	result, err = api.RequestMessages(context.TODO(), MessagesRequest{MailServerPeer: "invalid-address"})
	s.False(result)
	s.EqualError(err, "invalid mailServerPeer value: invalid URL scheme, want \"enode\"")

	// non-existent symmetric key
	result, err = api.RequestMessages(context.TODO(), MessagesRequest{
		MailServerPeer: mailServerPeer,
	})
	s.False(result)
	s.EqualError(err, "invalid symKeyID value: non-existent key ID")

	// with a symmetric key
	symKeyID, symKeyErr := shh.AddSymKeyFromPassword("some-pass")
	s.NoError(symKeyErr)
	result, err = api.RequestMessages(context.TODO(), MessagesRequest{
		MailServerPeer: mailServerPeer,
		SymKeyID:       symKeyID,
	})
	s.Contains(err.Error(), "Could not find peer with ID")
	s.False(result)

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
	err = mailNode.Register(func(_ *node.ServiceContext) (node.Service, error) {
		return whisper.New(nil), nil
	})
	s.NoError(err)
	err = mailNode.Start()
	s.NoError(err)
	defer func() {
		err := mailNode.Stop()
		s.NoError(err)
	}()

	// add mailPeer as a peer
	aNode.Server().AddPeer(mailNode.Server().Self())
	time.Sleep(time.Second) // wait for the peer to be added

	// send a request
	result, err = api.RequestMessages(context.TODO(), MessagesRequest{
		MailServerPeer: mailNode.Server().Self().String(),
		SymKeyID:       symKeyID,
	})
	s.NoError(err)
	s.True(result)
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
