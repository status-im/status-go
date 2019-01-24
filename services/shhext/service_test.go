package shhext

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/t/helpers"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/suite"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

const (
	// internal whisper protocol codes
	statusCode             = 0
	p2pRequestCompleteCode = 125
)

func newHandlerMock(buf int) handlerMock {
	return handlerMock{
		confirmations:     make(chan common.Hash, buf),
		expirations:       make(chan common.Hash, buf),
		requestsCompleted: make(chan common.Hash, buf),
		requestsExpired:   make(chan common.Hash, buf),
		requestsFailed:    make(chan common.Hash, buf),
	}
}

type handlerMock struct {
	confirmations     chan common.Hash
	expirations       chan common.Hash
	requestsCompleted chan common.Hash
	requestsExpired   chan common.Hash
	requestsFailed    chan common.Hash
}

func (t handlerMock) EnvelopeSent(hash common.Hash) {
	t.confirmations <- hash
}

func (t handlerMock) EnvelopeExpired(hash common.Hash) {
	t.expirations <- hash
}

func (t handlerMock) MailServerRequestCompleted(requestID common.Hash, lastEnvelopeHash common.Hash, cursor []byte, err error) {
	if err == nil {
		t.requestsCompleted <- requestID
	} else {
		t.requestsFailed <- requestID
	}
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

	directory, err := ioutil.TempDir("", "status-go-testing")
	s.Require().NoError(err)

	for i := range s.nodes {
		i := i // bind i to be usable in service constructors
		cfg := &node.Config{
			Name: fmt.Sprintf("node-%d", i),
			P2P: p2p.Config{
				NoDiscovery: true,
				MaxPeers:    1,
				ListenAddr:  ":0",
			},
			NoUSB: true,
		}
		stack, err := node.New(cfg)
		s.NoError(err)
		s.whisper[i] = whisper.New(nil)
		s.NoError(stack.Register(func(n *node.ServiceContext) (node.Service, error) {
			return s.whisper[i], nil
		}))

		config := params.ShhextConfig{
			InstallationID:          "1",
			BackupDisabledDataDir:   directory,
			DebugAPIEnabled:         true,
			PFSEnabled:              true,
			MailServerConfirmations: true,
			ConnectionTarget:        10,
		}
		db, err := leveldb.Open(storage.NewMemStorage(), nil)
		s.Require().NoError(err)
		s.services[i] = New(s.whisper[i], nil, db, config)
		s.NoError(stack.Register(func(n *node.ServiceContext) (node.Service, error) {
			return s.services[i], nil
		}))
		s.Require().NoError(stack.Start())
		s.nodes[i] = stack
	}
	s.services[0].tracker.handler = newHandlerMock(1)
}

func (s *ShhExtSuite) TestInitProtocol() {
	err := s.services[0].InitProtocol("example-address", "`090///\nhtaa\rhta9x8923)$$'23")
	s.NoError(err)
}

func (s *ShhExtSuite) TestPostMessageWithConfirmation() {
	mock := newHandlerMock(1)
	s.services[0].tracker.handler = mock
	s.Require().NoError(s.services[0].UpdateMailservers([]*enode.Node{s.nodes[1].Server().Self()}))
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
		NoUSB: true,
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
	config := params.ShhextConfig{
		InstallationID:        "1",
		BackupDisabledDataDir: os.TempDir(),
		PFSEnabled:            true,
	}
	service := New(shh, mock, nil, config)
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

func (s *ShhExtSuite) TestMultipleRequestMessagesWithoutForce() {
	waitErr := helpers.WaitForPeerAsync(s.nodes[0].Server(), s.nodes[1].Server().Self().String(), p2p.PeerEventTypeAdd, time.Second)
	s.nodes[0].Server().AddPeer(s.nodes[1].Server().Self())
	s.Require().NoError(<-waitErr)
	client, err := s.nodes[0].Attach()
	s.NoError(err)
	s.NoError(client.Call(nil, "shhext_requestMessages", MessagesRequest{
		MailServerPeer: s.nodes[1].Server().Self().String(),
		Topics:         []whisper.TopicType{{1}},
	}))
	s.EqualError(client.Call(nil, "shhext_requestMessages", MessagesRequest{
		MailServerPeer: s.nodes[1].Server().Self().String(),
		Topics:         []whisper.TopicType{{1}},
	}), "another request with the same topics was sent less than 3s ago. Please wait for a bit longer, or set `force` to true in request parameters")
	s.NoError(client.Call(nil, "shhext_requestMessages", MessagesRequest{
		MailServerPeer: s.nodes[1].Server().Self().String(),
		Topics:         []whisper.TopicType{{2}},
	}))
}

func (s *ShhExtSuite) TestRequestMessagesSuccess() {
	var err error

	shh := whisper.New(nil)
	aNode, err := node.New(&node.Config{
		P2P: p2p.Config{
			MaxPeers:    math.MaxInt32,
			NoDiscovery: true,
		},
		NoUSB: true,
	}) // in-memory node as no data dir
	s.Require().NoError(err)
	err = aNode.Register(func(*node.ServiceContext) (node.Service, error) { return shh, nil })
	s.Require().NoError(err)

	err = aNode.Start()
	s.Require().NoError(err)
	defer func() { err := aNode.Stop(); s.NoError(err) }()

	mock := newHandlerMock(1)
	config := params.ShhextConfig{
		InstallationID:        "1",
		BackupDisabledDataDir: os.TempDir(),
		PFSEnabled:            true,
	}
	service := New(shh, mock, nil, config)
	s.Require().NoError(service.Start(aNode.Server()))
	api := NewPublicAPI(service)

	// with a peer acting as a mailserver
	// prepare a node first
	mailNode, err := node.New(&node.Config{
		P2P: p2p.Config{
			MaxPeers:    math.MaxInt32,
			NoDiscovery: true,
			ListenAddr:  ":0",
		},
		NoUSB: true,
	}) // in-memory node as no data dir
	s.Require().NoError(err)
	err = mailNode.Register(func(*node.ServiceContext) (node.Service, error) {
		return whisper.New(nil), nil
	})
	s.NoError(err)
	err = mailNode.Start()
	s.Require().NoError(err)
	defer func() { s.NoError(mailNode.Stop()) }()

	// add mailPeer as a peer
	waitErr := helpers.WaitForPeerAsync(aNode.Server(), mailNode.Server().Self().String(), p2p.PeerEventTypeAdd, time.Second)
	aNode.Server().AddPeer(mailNode.Server().Self())
	s.Require().NoError(<-waitErr)

	var hash []byte

	// send a request with a symmetric key
	symKeyID, symKeyErr := shh.AddSymKeyFromPassword("some-pass")
	s.Require().NoError(symKeyErr)
	hash, err = api.RequestMessages(context.TODO(), MessagesRequest{
		MailServerPeer: mailNode.Server().Self().String(),
		SymKeyID:       symKeyID,
		Force:          true,
	})
	s.Require().NoError(err)
	s.Require().NotNil(hash)
	s.Require().NoError(waitForHashInTracker(api.service.tracker, common.BytesToHash(hash), MailServerRequestSent, time.Second))
	// Send a request without a symmetric key. In this case,
	// a public key extracted from MailServerPeer will be used.
	hash, err = api.RequestMessages(context.TODO(), MessagesRequest{
		MailServerPeer: mailNode.Server().Self().String(),
		Force:          true,
	})
	s.Require().NoError(err)
	s.Require().NotNil(hash)
	s.Require().NoError(waitForHashInTracker(api.service.tracker, common.BytesToHash(hash), MailServerRequestSent, time.Second))
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

func waitForHashInTracker(track *tracker, hash common.Hash, state EnvelopeState, deadline time.Duration) error {
	after := time.After(deadline)
	ticker := time.Tick(100 * time.Millisecond)
	for {
		select {
		case <-after:
			return fmt.Errorf("failed while waiting for %s to get into state %d", hash, state)
		case <-ticker:
			if track.GetState(hash) == state {
				return nil
			}
		}
	}
}

func TestRequestMessagesSync(t *testing.T) {
	suite.Run(t, new(RequestMessagesSyncSuite))
}

type RequestMessagesSyncSuite struct {
	suite.Suite

	localAPI  *PublicAPI
	localNode *enode.Node
	remoteRW  *p2p.MsgPipeRW
}

func (s *RequestMessagesSyncSuite) SetupTest() {
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	s.Require().NoError(err)
	conf := &whisper.Config{
		MinimumAcceptedPOW: 0,
		MaxMessageSize:     100 << 10,
	}
	w := whisper.New(conf)
	pkey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	node := enode.NewV4(&pkey.PublicKey, net.ParseIP("127.0.0.1"), 1, 1)
	peer := p2p.NewPeer(node.ID(), "1", []p2p.Cap{{"shh", 6}})
	rw1, rw2 := p2p.MsgPipe()
	errorc := make(chan error, 1)
	go func() {
		err := w.HandlePeer(peer, rw2)
		errorc <- err
	}()
	s.Require().NoError(p2p.ExpectMsg(rw1, statusCode, []interface{}{whisper.ProtocolVersion, math.Float64bits(w.MinPow()), w.BloomFilter(), false, true}))
	s.Require().NoError(p2p.SendItems(rw1, statusCode, whisper.ProtocolVersion, whisper.ProtocolVersion, math.Float64bits(w.MinPow()), w.BloomFilter(), true, true))

	service := New(w, nil, db, params.ShhextConfig{})

	s.localAPI = NewPublicAPI(service)
	s.localNode = node
	s.remoteRW = rw1
}

func (s *RequestMessagesSyncSuite) TestExpired() {
	// intentionally discarding all requests, so that request will timeout
	go func() {
		msg, err := s.remoteRW.ReadMsg()
		s.Require().NoError(err)
		s.Require().NoError(msg.Discard())
	}()
	s.Require().EqualError(s.localAPI.RequestMessagesSync(RetryConfig{
		BaseTimeout: time.Second,
	}, MessagesRequest{
		MailServerPeer: s.localNode.String(),
	}), "failed to request messages after 1 retries")
}

func (s *RequestMessagesSyncSuite) testCompletedFromAttempt(target int) {
	go func() {
		attempt := 0
		for {
			attempt++
			msg, err := s.remoteRW.ReadMsg()
			s.Require().NoError(err)
			if attempt < target {
				s.Require().NoError(msg.Discard())
				continue
			}
			var e whisper.Envelope
			s.Require().NoError(msg.Decode(&e))
			s.Require().NoError(p2p.Send(s.remoteRW, p2pRequestCompleteCode, whisper.CreateMailServerRequestCompletedPayload(e.Hash(), common.Hash{}, []byte{})))
		}
	}()
	s.Require().NoError(s.localAPI.RequestMessagesSync(RetryConfig{
		BaseTimeout: time.Second,
		MaxRetries:  target,
	}, MessagesRequest{
		MailServerPeer: s.localNode.String(),
		Force:          true, // force true is convenient here because timeout is less then default delay (3s)
	}))
}

func (s *RequestMessagesSyncSuite) TestCompletedFromFirstAttempt() {
	s.testCompletedFromAttempt(1)
}

func (s *RequestMessagesSyncSuite) TestCompletedFromSecondAttempt() {
	s.testCompletedFromAttempt(2)
}
