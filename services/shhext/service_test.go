package shhext

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/mailserver"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/t/utils"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/suite"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"golang.org/x/crypto/sha3"
)

const (
	// internal whisper protocol codes
	statusCode             = 0
	messagesCode           = 1
	batchAcknowledgeCode   = 11
	p2pRequestCompleteCode = 125
)

type failureMessage struct {
	Hash  common.Hash
	Error error
}

func newHandlerMock(buf int) handlerMock {
	return handlerMock{
		confirmations:     make(chan common.Hash, buf),
		expirations:       make(chan failureMessage, buf),
		requestsCompleted: make(chan common.Hash, buf),
		requestsExpired:   make(chan common.Hash, buf),
		requestsFailed:    make(chan common.Hash, buf),
	}
}

type handlerMock struct {
	confirmations     chan common.Hash
	expirations       chan failureMessage
	requestsCompleted chan common.Hash
	requestsExpired   chan common.Hash
	requestsFailed    chan common.Hash
}

func (t handlerMock) EnvelopeSent(hash common.Hash) {
	t.confirmations <- hash
}

func (t handlerMock) EnvelopeExpired(hash common.Hash, err error) {
	t.expirations <- failureMessage{Hash: hash, Error: err}
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
	s.services[0].envelopesMonitor.handler = newHandlerMock(1)
}

func (s *ShhExtSuite) TestInitProtocol() {
	err := s.services[0].InitProtocolWithPassword("example-address", "`090///\nhtaa\rhta9x8923)$$'23")
	s.NoError(err)

	digest := sha3.Sum256([]byte("`090///\nhtaa\rhta9x8923)$$'23"))
	encKey := fmt.Sprintf("%x", digest)
	err = s.services[0].InitProtocolWithEncyptionKey("example-address", encKey)
	s.NoError(err)
}

func (s *ShhExtSuite) TestPostMessageWithConfirmation() {
	mock := newHandlerMock(1)
	s.services[0].envelopesMonitor.handler = mock
	s.Require().NoError(s.services[0].UpdateMailservers([]*enode.Node{s.nodes[1].Server().Self()}))
	s.nodes[0].Server().AddPeer(s.nodes[1].Server().Self())
	symID, err := s.whisper[0].GenerateSymKey()
	s.NoError(err)
	client, err := s.nodes[0].Attach()
	s.NoError(err)
	var hash common.Hash
	message := whisper.NewMessage{
		SymKeyID:  symID,
		PowTarget: whisper.DefaultMinimumPoW,
		PowTime:   200,
		Topic:     whisper.TopicType{0x01, 0x01, 0x01, 0x01},
		Payload:   []byte("hello"),
	}
	mid := messageID(message)
	s.NoError(client.Call(&hash, "shhext_post", message))
	s.NoError(err)
	select {
	case confirmed := <-mock.confirmations:
		s.Equal(mid, confirmed)
	case <-time.After(5 * time.Second):
		s.Fail("timed out while waiting for confirmation")
	}
}

func (s *ShhExtSuite) testWaitMessageExpired(expectedError string, ttl uint32) {
	mock := newHandlerMock(1)
	s.services[0].envelopesMonitor.handler = mock
	symID, err := s.whisper[0].GenerateSymKey()
	s.NoError(err)
	client, err := s.nodes[0].Attach()
	s.NoError(err)
	var hash common.Hash
	message := whisper.NewMessage{
		SymKeyID:  symID,
		PowTarget: whisper.DefaultMinimumPoW,
		PowTime:   200,
		TTL:       ttl,
		Topic:     whisper.TopicType{0x01, 0x01, 0x01, 0x01},
		Payload:   []byte("hello"),
	}
	mid := messageID(message)
	s.NoError(client.Call(&hash, "shhext_post", message))
	s.NoError(err)
	select {
	case expired := <-mock.expirations:
		s.Equal(mid, expired.Hash)
		s.EqualError(expired.Error, expectedError)
	case confirmed := <-mock.confirmations:
		s.Fail("unexpected confirmation for hash", confirmed)
	case <-time.After(10 * time.Second):
		s.Fail("timed out while waiting for confirmation")
	}
}

func (s *ShhExtSuite) TestWaitMessageExpired() {
	s.testWaitMessageExpired("envelope expired due to connectivity issues", 1)
}

func (s *ShhExtSuite) TestErrorOnEnvelopeDelivery() {
	// in the test we are sending message from peer 0 to peer 1
	s.nodes[0].Server().AddPeer(s.nodes[1].Server().Self())
	s.Require().NoError(s.services[0].UpdateMailservers([]*enode.Node{s.nodes[1].Server().Self()}))
	s.whisper[1].SetTimeSource(func() time.Time {
		return time.Now().Add(time.Hour)
	})
	s.testWaitMessageExpired("envelope wasn't delivered due to time sync issues", 100)
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

func (s *ShhExtSuite) TestFailedRequestUnregistered() {
	waitErr := helpers.WaitForPeerAsync(s.nodes[0].Server(), s.nodes[1].Server().Self().String(), p2p.PeerEventTypeAdd, time.Second)
	s.nodes[0].Server().AddPeer(s.nodes[1].Server().Self())
	s.Require().NoError(<-waitErr)
	client, err := s.nodes[0].Attach()
	topics := []whisper.TopicType{{1}}
	s.NoError(err)
	s.EqualError(client.Call(nil, "shhext_requestMessages", MessagesRequest{
		MailServerPeer: "enode://19872f94b1e776da3a13e25afa71b47dfa99e658afd6427ea8d6e03c22a99f13590205a8826443e95a37eee1d815fc433af7a8ca9a8d0df7943d1f55684045b7@0.0.0.0:30305",
		Topics:         topics,
	}), "Could not find peer with ID: 10841e6db5c02fc331bf36a8d2a9137a1696d9d3b6b1f872f780e02aa8ec5bba")
	s.NoError(client.Call(nil, "shhext_requestMessages", MessagesRequest{
		MailServerPeer: s.nodes[1].Server().Self().String(),
		Topics:         topics,
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
	s.Require().NoError(waitForHashInMonitor(api.service.mailMonitor, common.BytesToHash(hash), MailServerRequestSent, time.Second))
	// Send a request without a symmetric key. In this case,
	// a public key extracted from MailServerPeer will be used.
	hash, err = api.RequestMessages(context.TODO(), MessagesRequest{
		MailServerPeer: mailNode.Server().Self().String(),
		Force:          true,
	})
	s.Require().NoError(err)
	s.Require().NotNil(hash)
	s.Require().NoError(waitForHashInMonitor(api.service.mailMonitor, common.BytesToHash(hash), MailServerRequestSent, time.Second))
}

func (s *ShhExtSuite) TearDown() {
	for _, n := range s.nodes {
		s.NoError(n.Stop())
	}
}

func waitForHashInMonitor(mon *MailRequestMonitor, hash common.Hash, state EnvelopeState, deadline time.Duration) error {
	after := time.After(deadline)
	ticker := time.Tick(100 * time.Millisecond)
	for {
		select {
		case <-after:
			return fmt.Errorf("failed while waiting for %s to get into state %d", hash, state)
		case <-ticker:
			if mon.GetState(hash) == state {
				return nil
			}
		}
	}
}

type WhisperNodeMockSuite struct {
	suite.Suite

	localWhisperAPI *whisper.PublicWhisperAPI
	localAPI        *PublicAPI
	localNode       *enode.Node
	remoteRW        *p2p.MsgPipeRW

	localService          *Service
	localEnvelopesMonitor *EnvelopesMonitor
}

func (s *WhisperNodeMockSuite) SetupTest() {
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	s.Require().NoError(err)
	conf := &whisper.Config{
		MinimumAcceptedPOW: 0,
		MaxMessageSize:     100 << 10,
	}
	w := whisper.New(conf)
	s.Require().NoError(w.Start(nil))
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

	s.localService = New(w, nil, db, params.ShhextConfig{MailServerConfirmations: true, MaxMessageDeliveryAttempts: 3})
	s.Require().NoError(s.localService.UpdateMailservers([]*enode.Node{node}))

	s.localEnvelopesMonitor = s.localService.envelopesMonitor
	s.localEnvelopesMonitor.Start()

	s.localWhisperAPI = whisper.NewPublicWhisperAPI(w)
	s.localAPI = NewPublicAPI(s.localService)
	s.localNode = node
	s.remoteRW = rw1
}

func (s *WhisperNodeMockSuite) PostMessage(message whisper.NewMessage) common.Hash {
	envBytes, err := s.localAPI.Post(context.TODO(), message)
	envHash := common.BytesToHash(envBytes)
	s.Require().NoError(err)
	s.Require().NoError(utils.Eventually(func() error {
		if state := s.localEnvelopesMonitor.GetMessageState(envHash); state != EnvelopePosted {
			return fmt.Errorf("envelope with hash %s wasn't posted", envHash.String())
		}
		return nil
	}, 2*time.Second, 100*time.Millisecond))
	return envHash
}

func TestRequestMessagesSync(t *testing.T) {
	suite.Run(t, new(RequestMessagesSyncSuite))
}

type RequestMessagesSyncSuite struct {
	WhisperNodeMockSuite
}

func (s *RequestMessagesSyncSuite) TestExpired() {
	// intentionally discarding all requests, so that request will timeout
	go func() {
		msg, err := s.remoteRW.ReadMsg()
		s.Require().NoError(err)
		s.Require().NoError(msg.Discard())
	}()
	_, err := s.localAPI.RequestMessagesSync(
		RetryConfig{
			BaseTimeout: time.Second,
		},
		MessagesRequest{
			MailServerPeer: s.localNode.String(),
		},
	)
	s.Require().EqualError(err, "failed to request messages after 1 retries")
}

func (s *RequestMessagesSyncSuite) testCompletedFromAttempt(target int) {
	const cursorSize = 36 // taken from mailserver_response.go from whisperv6 package
	cursor := [cursorSize]byte{}
	cursor[0] = 0x01

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
			s.Require().NoError(p2p.Send(s.remoteRW, p2pRequestCompleteCode, whisper.CreateMailServerRequestCompletedPayload(e.Hash(), common.Hash{}, cursor[:])))
		}
	}()
	resp, err := s.localAPI.RequestMessagesSync(
		RetryConfig{
			BaseTimeout: time.Second,
			MaxRetries:  target,
		},
		MessagesRequest{
			MailServerPeer: s.localNode.String(),
			Force:          true, // force true is convenient here because timeout is less then default delay (3s)
		},
	)
	s.Require().NoError(err)
	s.Require().Equal(MessagesResponse{Cursor: hex.EncodeToString(cursor[:])}, resp)
}

func (s *RequestMessagesSyncSuite) TestCompletedFromFirstAttempt() {
	s.testCompletedFromAttempt(1)
}

func (s *RequestMessagesSyncSuite) TestCompletedFromSecondAttempt() {
	s.testCompletedFromAttempt(2)
}

func TestWhisperConfirmations(t *testing.T) {
	suite.Run(t, new(WhisperConfirmationSuite))
}

type WhisperConfirmationSuite struct {
	WhisperNodeMockSuite
}

func (s *WhisperConfirmationSuite) TestEnvelopeReceived() {
	symID, err := s.localWhisperAPI.GenerateSymKeyFromPassword(context.TODO(), "test")
	s.Require().NoError(err)
	envHash := s.PostMessage(whisper.NewMessage{
		SymKeyID: symID,
		TTL:      1000,
		Topic:    whisper.TopicType{0x01},
	})

	// enable auto-replies once message got registered internally
	go func() {
		for {
			msg, err := s.remoteRW.ReadMsg()
			s.Require().NoError(err)
			if msg.Code != messagesCode {
				s.Require().NoError(msg.Discard())
				continue
			}
			// reply with same envelopes. we could probably just write same data to remoteRW, but this works too.
			var envs []*whisper.Envelope
			s.Require().NoError(msg.Decode(&envs))
			s.Require().NoError(p2p.Send(s.remoteRW, messagesCode, envs))
		}
	}()

	// wait for message to be removed because it was delivered by remoteRW
	s.Require().NoError(utils.Eventually(func() error {
		if state := s.localEnvelopesMonitor.GetMessageState(envHash); state == EnvelopePosted {
			return fmt.Errorf("envelope with hash %s wasn't posted", envHash.String())
		}
		return nil
	}, 2*time.Second, 100*time.Millisecond))
}

func TestWhisperRetriesSuite(t *testing.T) {
	suite.Run(t, new(WhisperRetriesSuite))
}

type WhisperRetriesSuite struct {
	WhisperNodeMockSuite
}

func (s *WhisperRetriesSuite) TestUseAllAvaiableAttempts() {
	var attempts int32
	go func() {
		for {
			msg, err := s.remoteRW.ReadMsg()
			s.Require().NoError(err)
			s.Require().NoError(msg.Discard())
			if msg.Code != messagesCode {
				continue
			}
			atomic.AddInt32(&attempts, 1)
		}
	}()
	symID, err := s.localWhisperAPI.GenerateSymKeyFromPassword(context.TODO(), "test")
	s.Require().NoError(err)
	message := whisper.NewMessage{
		SymKeyID:  symID,
		PowTarget: whisper.DefaultMinimumPoW,
		PowTime:   200,
		TTL:       1,
		Topic:     whisper.TopicType{0x01, 0x01, 0x01, 0x01},
		Payload:   []byte("hello"),
	}
	s.Require().NotNil(s.PostMessage(message))
	s.Require().NoError(utils.Eventually(func() error {
		madeAttempts := atomic.LoadInt32(&attempts)
		if madeAttempts != int32(s.localEnvelopesMonitor.maxAttempts) {
			return fmt.Errorf("made unexpected number of attempts to deliver a message: %d != %d", s.localEnvelopesMonitor.maxAttempts, madeAttempts)
		}
		return nil
	}, 10*time.Second, 500*time.Millisecond))
}

func (s *WhisperRetriesSuite) testDelivery(target int) {
	go func() {
		attempt := 0
		for {
			msg, err := s.remoteRW.ReadMsg()
			s.Require().NoError(err)
			if msg.Code != messagesCode {
				s.Require().NoError(msg.Discard())
				continue
			}
			attempt++
			if attempt != target {
				s.Require().NoError(msg.Discard())
				continue
			}
			data, err := ioutil.ReadAll(msg.Payload)
			s.Require().NoError(err)
			// without this hack event from the whisper read loop will be sent sooner than event from write loop
			// i don't think that this is realistic situation and can be reproduced only in test with in-memory
			// connection mock
			time.Sleep(time.Nanosecond)
			s.Require().NoError(p2p.Send(s.remoteRW, batchAcknowledgeCode, crypto.Keccak256Hash(data)))
		}
	}()
	symID, err := s.localWhisperAPI.GenerateSymKeyFromPassword(context.TODO(), "test")
	s.Require().NoError(err)
	message := whisper.NewMessage{
		SymKeyID:  symID,
		PowTarget: whisper.DefaultMinimumPoW,
		PowTime:   200,
		TTL:       1,
		Topic:     whisper.TopicType{0x01, 0x01, 0x01, 0x01},
		Payload:   []byte("hello"),
	}
	mID := messageID(message)
	s.Require().NotNil(s.PostMessage(message))
	s.Require().NoError(utils.Eventually(func() error {
		if state := s.localEnvelopesMonitor.GetMessageState(mID); state != EnvelopeSent {
			return fmt.Errorf("message with ID %s wasn't sent", mID.String())
		}
		return nil
	}, 3*time.Second, 100*time.Millisecond))
}

func (s *WhisperRetriesSuite) TestDeliveredFromFirstAttempt() {
	s.testDelivery(1)
}

func (s *WhisperRetriesSuite) TestDeliveredFromSecondAttempt() {
	s.testDelivery(2)
}

func TestRequestWithTrackingHistorySuite(t *testing.T) {
	suite.Run(t, new(RequestWithTrackingHistorySuite))
}

type RequestWithTrackingHistorySuite struct {
	suite.Suite

	envelopeSymkey   string
	envelopeSymkeyID string

	localWhisperAPI *whisper.PublicWhisperAPI
	localAPI        *PublicAPI
	localService    *Service
	mailSymKey      string

	remoteMailserver *mailserver.WMailServer
	remoteNode       *enode.Node
	remoteWhisper    *whisper.Whisper
}

func (s *RequestWithTrackingHistorySuite) SetupTest() {
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	s.Require().NoError(err)
	conf := &whisper.Config{
		MinimumAcceptedPOW: 0,
		MaxMessageSize:     100 << 10,
	}
	local := whisper.New(conf)
	s.Require().NoError(local.Start(nil))

	s.localWhisperAPI = whisper.NewPublicWhisperAPI(local)
	s.localService = New(local, nil, db, params.ShhextConfig{})
	localPkey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.Require().NoError(s.localService.Start(&p2p.Server{Config: p2p.Config{PrivateKey: localPkey}}))
	s.localAPI = NewPublicAPI(s.localService)

	remote := whisper.New(conf)
	s.remoteWhisper = remote
	s.Require().NoError(remote.Start(nil))
	s.remoteMailserver = &mailserver.WMailServer{}
	remote.RegisterServer(s.remoteMailserver)
	password := "test"
	tmpdir, err := ioutil.TempDir("", "tracking-history-tests-")
	s.Require().NoError(err)
	s.Require().NoError(s.remoteMailserver.Init(remote, &params.WhisperConfig{
		DataDir:            tmpdir,
		MailServerPassword: password,
	}))

	pkey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	// we need proper enode for a remote node. it will be used when mail server request is made
	s.remoteNode = enode.NewV4(&pkey.PublicKey, net.ParseIP("127.0.0.1"), 1, 1)
	remotePeer := p2p.NewPeer(s.remoteNode.ID(), "1", []p2p.Cap{{"shh", 6}})
	localPeer := p2p.NewPeer(enode.ID{2}, "2", []p2p.Cap{{"shh", 6}})
	// FIXME close this in tear down
	rw1, rw2 := p2p.MsgPipe()
	go func() {
		err := local.HandlePeer(remotePeer, rw1)
		s.Require().NoError(err)
	}()
	go func() {
		err := remote.HandlePeer(localPeer, rw2)
		s.Require().NoError(err)
	}()

	s.mailSymKey, err = s.localWhisperAPI.GenerateSymKeyFromPassword(context.Background(), password)
	s.Require().NoError(err)

	s.envelopeSymkey = "topics"
	s.envelopeSymkeyID, err = s.localWhisperAPI.GenerateSymKeyFromPassword(context.Background(), s.envelopeSymkey)
	s.Require().NoError(err)
}

func (s *RequestWithTrackingHistorySuite) postEnvelopes(topics ...whisper.TopicType) []hexutil.Bytes {
	var (
		rst = make([]hexutil.Bytes, len(topics))
		err error
	)
	for i, t := range topics {
		rst[i], err = s.localWhisperAPI.Post(context.Background(), whisper.NewMessage{
			SymKeyID: s.envelopeSymkeyID,
			TTL:      10,
			Topic:    t,
		})
		s.Require().NoError(err)
	}
	return rst

}

func (s *RequestWithTrackingHistorySuite) waitForArchival(hexes []hexutil.Bytes) {
	events := make(chan whisper.EnvelopeEvent, 2)
	sub := s.remoteWhisper.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()
	s.Require().NoError(waitForArchival(events, 2*time.Second, hexes...))
}

func (s *RequestWithTrackingHistorySuite) createEmptyFilter(topics ...whisper.TopicType) string {
	filterid, err := s.localWhisperAPI.NewMessageFilter(whisper.Criteria{
		SymKeyID: s.envelopeSymkeyID,
		Topics:   topics,
		AllowP2P: true,
	})
	s.Require().NoError(err)
	s.Require().NotNil(filterid)

	messages, err := s.localWhisperAPI.GetFilterMessages(filterid)
	s.Require().NoError(err)
	s.Require().Empty(messages)
	return filterid
}

func (s *RequestWithTrackingHistorySuite) initiateHistoryRequest(topics ...TopicRequest) []hexutil.Bytes {
	requests, err := s.localAPI.InitiateHistoryRequests(InitiateHistoryRequest{
		Peer:     s.remoteNode.String(),
		SymKeyID: s.mailSymKey,
		Timeout:  10 * time.Second,
		Requests: topics,
	})
	s.Require().NoError(err)
	return requests
}

func (s *RequestWithTrackingHistorySuite) waitMessagesDelivered(filterid string, hexes ...hexutil.Bytes) {
	var received int
	s.Require().NoError(utils.Eventually(func() error {
		messages, err := s.localWhisperAPI.GetFilterMessages(filterid)
		if err != nil {
			return err
		}
		received += len(messages)
		if received != len(hexes) {
			return fmt.Errorf("expecting to receive %d messages, received %d", len(hexes), received)
		}
		return nil
	}, 2*time.Second, 200*time.Millisecond))

}

func (s *RequestWithTrackingHistorySuite) waitLastEnvelopeUpdated(requests ...hexutil.Bytes) {
	store := s.localService.historyUpdates.store
	s.Require().NoError(utils.Eventually(func() error {
		reqs, err := store.GetAllRequests()
		if err != nil {
			return err
		}
		if len(reqs) != len(requests) {
			return errors.New("one request should be in database")
		}
		if (reqs[0].LastEnvelopeHash == common.Hash{}) {
			return errors.New("last envelope hash is not set yet")
		}
		return nil
	}, 2*time.Second, 200*time.Millisecond))
}

func (s *RequestWithTrackingHistorySuite) waitNoRequests() {
	store := s.localService.historyUpdates.store
	s.Require().NoError(utils.Eventually(func() error {
		reqs, err := store.GetAllRequests()
		if err != nil {
			return err
		}
		if len(reqs) != 0 {
			return fmt.Errorf("not all requests were removed. count %d", len(reqs))
		}
		return nil
	}, 2*time.Second, 200*time.Millisecond))
}

func (s *RequestWithTrackingHistorySuite) TestMultipleMergeIntoOne() {
	topic1 := whisper.TopicType{1, 1, 1, 1}
	topic2 := whisper.TopicType{2, 2, 2, 2}
	topic3 := whisper.TopicType{3, 3, 3, 3}
	hexes := s.postEnvelopes(topic1, topic2, topic3)
	s.waitForArchival(hexes)

	filterid := s.createEmptyFilter(topic1, topic2, topic3)
	requests := s.initiateHistoryRequest(
		TopicRequest{Topic: topic1, Duration: time.Hour},
		TopicRequest{Topic: topic2, Duration: time.Hour},
		TopicRequest{Topic: topic3, Duration: 10 * time.Hour},
	)
	// since we are using different duration for 3rd topic there will be 2 requests
	s.Require().Len(requests, 2)
	s.waitMessagesDelivered(filterid, hexes...)

	s.Require().NoError(s.localService.historyUpdates.UpdateTopicHistory(topic1, time.Now(), common.BytesToHash(hexes[0])))
	s.Require().NoError(s.localService.historyUpdates.UpdateTopicHistory(topic2, time.Now(), common.BytesToHash(hexes[1])))
	s.Require().NoError(s.localService.historyUpdates.UpdateTopicHistory(topic3, time.Now(), common.BytesToHash(hexes[2])))
	s.waitNoRequests()

	requests = s.initiateHistoryRequest(
		TopicRequest{Topic: topic1, Duration: time.Hour},
		TopicRequest{Topic: topic2, Duration: time.Hour},
		TopicRequest{Topic: topic3, Duration: 10 * time.Hour},
	)
	s.Len(requests, 1)
}

func (s *RequestWithTrackingHistorySuite) TestSingleRequest() {
	topic1 := whisper.TopicType{1, 1, 1, 1}
	topic2 := whisper.TopicType{255, 255, 255, 255}
	hexes := s.postEnvelopes(topic1, topic2)
	s.waitForArchival(hexes)

	filterid := s.createEmptyFilter(topic1, topic2)
	requests := s.initiateHistoryRequest(
		TopicRequest{Topic: topic1, Duration: time.Hour},
		TopicRequest{Topic: topic2, Duration: time.Hour},
	)
	s.Require().Len(requests, 1)
	s.waitMessagesDelivered(filterid, hexes...)
	s.waitLastEnvelopeUpdated(requests...)
}

func waitForArchival(events chan whisper.EnvelopeEvent, duration time.Duration, hashes ...hexutil.Bytes) error {
	waiting := map[common.Hash]struct{}{}
	for _, hash := range hashes {
		waiting[common.BytesToHash(hash)] = struct{}{}
	}
	timeout := time.After(duration)
	for {
		select {
		case <-timeout:
			return errors.New("timed out while waiting for mailserver to archive envelopes")
		case ev := <-events:
			if ev.Event != whisper.EventMailServerEnvelopeArchived {
				continue
			}
			if _, exist := waiting[ev.Hash]; exist {
				delete(waiting, ev.Hash)
				if len(waiting) == 0 {
					return nil
				}
			}
		}
	}

}
