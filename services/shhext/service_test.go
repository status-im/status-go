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
	gethbridge "github.com/status-im/status-go/protocol/bridge/geth"
	whispertypes "github.com/status-im/status-go/protocol/transport/whisper/types"
	protocol "github.com/status-im/status-go/protocol/types"
	"github.com/status-im/status-go/sqlite"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/t/utils"
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

type failureMessage struct {
	IDs   [][]byte
	Error error
}

func newHandlerMock(buf int) handlerMock {
	return handlerMock{
		confirmations:     make(chan [][]byte, buf),
		expirations:       make(chan failureMessage, buf),
		requestsCompleted: make(chan protocol.Hash, buf),
		requestsExpired:   make(chan protocol.Hash, buf),
		requestsFailed:    make(chan protocol.Hash, buf),
	}
}

type handlerMock struct {
	confirmations     chan [][]byte
	expirations       chan failureMessage
	requestsCompleted chan protocol.Hash
	requestsExpired   chan protocol.Hash
	requestsFailed    chan protocol.Hash
}

func (t handlerMock) EnvelopeSent(ids [][]byte) {
	t.confirmations <- ids
}

func (t handlerMock) EnvelopeExpired(ids [][]byte, err error) {
	t.expirations <- failureMessage{IDs: ids, Error: err}
}

func (t handlerMock) MailServerRequestCompleted(requestID protocol.Hash, lastEnvelopeHash protocol.Hash, cursor []byte, err error) {
	if err == nil {
		t.requestsCompleted <- requestID
	} else {
		t.requestsFailed <- requestID
	}
}

func (t handlerMock) MailServerRequestExpired(hash protocol.Hash) {
	t.requestsExpired <- hash
}

func TestShhExtSuite(t *testing.T) {
	suite.Run(t, new(ShhExtSuite))
}

type ShhExtSuite struct {
	suite.Suite

	nodes    []*node.Node
	services []*Service
	whisper  []whispertypes.Whisper
}

func (s *ShhExtSuite) SetupTest() {
	s.nodes = make([]*node.Node, 2)
	s.services = make([]*Service, 2)
	s.whisper = make([]whispertypes.Whisper, 2)

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
		s.whisper[i] = gethbridge.NewGethWhisperWrapper(whisper.New(nil))

		privateKey, err := crypto.GenerateKey()
		s.NoError(err)
		err = s.whisper[i].SelectKeyPair(privateKey)
		s.NoError(err)

		s.NoError(stack.Register(func(n *node.ServiceContext) (node.Service, error) {
			return gethbridge.GetGethWhisperFrom(s.whisper[i]), nil
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

		tmpdir, err := ioutil.TempDir("", "test-shhext-service")
		s.Require().NoError(err)

		sqlDB, err := sqlite.OpenDB(fmt.Sprintf("%s/%d", tmpdir, i), "password")
		s.Require().NoError(err)

		s.Require().NoError(s.services[i].InitProtocol(sqlDB))
		s.NoError(stack.Register(func(n *node.ServiceContext) (node.Service, error) {
			return s.services[i], nil
		}))
		s.Require().NoError(stack.Start())
		s.nodes[i] = stack
	}
}

func (s *ShhExtSuite) TestInitProtocol() {
	directory, err := ioutil.TempDir("", "status-go-testing")
	s.Require().NoError(err)

	config := params.ShhextConfig{
		InstallationID:          "2",
		BackupDisabledDataDir:   directory,
		PFSEnabled:              true,
		MailServerConfirmations: true,
		ConnectionTarget:        10,
	}
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	s.Require().NoError(err)

	shh := gethbridge.NewGethWhisperWrapper(whisper.New(nil))
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	err = shh.SelectKeyPair(privateKey)
	s.Require().NoError(err)

	service := New(shh, nil, db, config)

	tmpdir, err := ioutil.TempDir("", "test-shhext-service-init-protocol")
	s.Require().NoError(err)

	sqlDB, err := sqlite.OpenDB(fmt.Sprintf("%s/db.sql", tmpdir), "password")
	s.Require().NoError(err)

	err = service.InitProtocol(sqlDB)
	s.NoError(err)
}

func (s *ShhExtSuite) TestRequestMessagesErrors() {
	var err error

	shh := gethbridge.NewGethWhisperWrapper(whisper.New(nil))
	aNode, err := node.New(&node.Config{
		P2P: p2p.Config{
			MaxPeers:    math.MaxInt32,
			NoDiscovery: true,
		},
		NoUSB: true,
	}) // in-memory node as no data dir
	s.NoError(err)
	err = aNode.Register(func(*node.ServiceContext) (node.Service, error) {
		return gethbridge.GetGethWhisperFrom(shh), nil
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
	waitErr := helpers.WaitForPeerAsync(s.nodes[0].Server(), s.nodes[1].Server().Self().URLv4(), p2p.PeerEventTypeAdd, time.Second)
	s.nodes[0].Server().AddPeer(s.nodes[1].Server().Self())
	s.Require().NoError(<-waitErr)
	client, err := s.nodes[0].Attach()
	s.NoError(err)
	s.NoError(client.Call(nil, "shhext_requestMessages", MessagesRequest{
		MailServerPeer: s.nodes[1].Server().Self().URLv4(),
		Topics:         []whispertypes.TopicType{{1}},
	}))
	s.EqualError(client.Call(nil, "shhext_requestMessages", MessagesRequest{
		MailServerPeer: s.nodes[1].Server().Self().URLv4(),
		Topics:         []whispertypes.TopicType{{1}},
	}), "another request with the same topics was sent less than 3s ago. Please wait for a bit longer, or set `force` to true in request parameters")
	s.NoError(client.Call(nil, "shhext_requestMessages", MessagesRequest{
		MailServerPeer: s.nodes[1].Server().Self().URLv4(),
		Topics:         []whispertypes.TopicType{{2}},
	}))
}

func (s *ShhExtSuite) TestFailedRequestUnregistered() {
	waitErr := helpers.WaitForPeerAsync(s.nodes[0].Server(), s.nodes[1].Server().Self().URLv4(), p2p.PeerEventTypeAdd, time.Second)
	s.nodes[0].Server().AddPeer(s.nodes[1].Server().Self())
	s.Require().NoError(<-waitErr)
	client, err := s.nodes[0].Attach()
	topics := []whispertypes.TopicType{{1}}
	s.NoError(err)
	s.EqualError(client.Call(nil, "shhext_requestMessages", MessagesRequest{
		MailServerPeer: "enode://19872f94b1e776da3a13e25afa71b47dfa99e658afd6427ea8d6e03c22a99f13590205a8826443e95a37eee1d815fc433af7a8ca9a8d0df7943d1f55684045b7@0.0.0.0:30305",
		Topics:         topics,
	}), "Could not find peer with ID: 10841e6db5c02fc331bf36a8d2a9137a1696d9d3b6b1f872f780e02aa8ec5bba")
	s.NoError(client.Call(nil, "shhext_requestMessages", MessagesRequest{
		MailServerPeer: s.nodes[1].Server().Self().URLv4(),
		Topics:         topics,
	}))
}

func (s *ShhExtSuite) TestRequestMessagesSuccess() {
	var err error

	shh := gethbridge.NewGethWhisperWrapper(whisper.New(nil))
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	err = shh.SelectKeyPair(privateKey)
	s.Require().NoError(err)
	aNode, err := node.New(&node.Config{
		P2P: p2p.Config{
			MaxPeers:    math.MaxInt32,
			NoDiscovery: true,
		},
		NoUSB: true,
	}) // in-memory node as no data dir
	s.Require().NoError(err)
	err = aNode.Register(func(*node.ServiceContext) (node.Service, error) { return gethbridge.GetGethWhisperFrom(shh), nil })
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

	tmpdir, err := ioutil.TempDir("", "test-shhext-service-request-messages")
	s.Require().NoError(err)

	sqlDB, err := sqlite.OpenDB(fmt.Sprintf("%s/db.sql", tmpdir), "password")
	s.Require().NoError(err)

	s.Require().NoError(service.InitProtocol(sqlDB))
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
	waitErr := helpers.WaitForPeerAsync(aNode.Server(), mailNode.Server().Self().URLv4(), p2p.PeerEventTypeAdd, time.Second)
	aNode.Server().AddPeer(mailNode.Server().Self())
	s.Require().NoError(<-waitErr)

	var hash []byte

	// send a request with a symmetric key
	symKeyID, symKeyErr := shh.AddSymKeyFromPassword("some-pass")
	s.Require().NoError(symKeyErr)
	hash, err = api.RequestMessages(context.TODO(), MessagesRequest{
		MailServerPeer: mailNode.Server().Self().URLv4(),
		SymKeyID:       symKeyID,
		Force:          true,
	})
	s.Require().NoError(err)
	s.Require().NotNil(hash)
	// Send a request without a symmetric key. In this case,
	// a public key extracted from MailServerPeer will be used.
	hash, err = api.RequestMessages(context.TODO(), MessagesRequest{
		MailServerPeer: mailNode.Server().Self().URLv4(),
		Force:          true,
	})
	s.Require().NoError(err)
	s.Require().NotNil(hash)
}

func (s *ShhExtSuite) TearDown() {
	for _, n := range s.nodes {
		s.NoError(n.Stop())
	}
}

type WhisperNodeMockSuite struct {
	suite.Suite

	localWhisperAPI *whisper.PublicWhisperAPI
	localAPI        *PublicAPI
	localNode       *enode.Node
	remoteRW        *p2p.MsgPipeRW

	localService *Service
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
	whisperWrapper := gethbridge.NewGethWhisperWrapper(w)
	s.Require().NoError(p2p.ExpectMsg(rw1, statusCode, []interface{}{whisper.ProtocolVersion, math.Float64bits(whisperWrapper.MinPow()), whisperWrapper.BloomFilter(), false, true}))
	s.Require().NoError(p2p.SendItems(rw1, statusCode, whisper.ProtocolVersion, whisper.ProtocolVersion, math.Float64bits(whisperWrapper.MinPow()), whisperWrapper.BloomFilter(), true, true))

	s.localService = New(whisperWrapper, nil, db, params.ShhextConfig{MailServerConfirmations: true, MaxMessageDeliveryAttempts: 3})
	s.Require().NoError(s.localService.UpdateMailservers([]*enode.Node{node}))

	s.localWhisperAPI = whisper.NewPublicWhisperAPI(w)
	s.localAPI = NewPublicAPI(s.localService)
	s.localNode = node
	s.remoteRW = rw1
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

func TestWhisperRetriesSuite(t *testing.T) {
	suite.Run(t, new(WhisperRetriesSuite))
}

type WhisperRetriesSuite struct {
	WhisperNodeMockSuite
}

func TestRequestWithTrackingHistorySuite(t *testing.T) {
	suite.Run(t, new(RequestWithTrackingHistorySuite))
}

type RequestWithTrackingHistorySuite struct {
	suite.Suite

	envelopeSymkey   string
	envelopeSymkeyID string

	localWhisperAPI whispertypes.PublicWhisperAPI
	localAPI        *PublicAPI
	localService    *Service
	localContext    Context
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
	localSHH := whisper.New(conf)
	local := gethbridge.NewGethWhisperWrapper(localSHH)
	s.Require().NoError(localSHH.Start(nil))

	s.localWhisperAPI = local.PublicWhisperAPI()
	s.localService = New(local, nil, db, params.ShhextConfig{})
	s.localContext = NewContextFromService(context.Background(), s.localService, s.localService.storage)
	localPkey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	tmpdir, err := ioutil.TempDir("", "test-shhext-service")
	s.Require().NoError(err)

	sqlDB, err := sqlite.OpenDB(fmt.Sprintf("%s/db.sql", tmpdir), "password")
	s.Require().NoError(err)

	s.Require().NoError(s.localService.InitProtocol(sqlDB))
	s.Require().NoError(s.localService.Start(&p2p.Server{Config: p2p.Config{PrivateKey: localPkey}}))
	s.localAPI = NewPublicAPI(s.localService)

	remoteSHH := whisper.New(conf)
	s.remoteWhisper = remoteSHH
	s.Require().NoError(remoteSHH.Start(nil))
	s.remoteMailserver = &mailserver.WMailServer{}
	remoteSHH.RegisterServer(s.remoteMailserver)
	password := "test"
	tmpdir, err = ioutil.TempDir("", "tracking-history-tests-")
	s.Require().NoError(err)
	s.Require().NoError(s.remoteMailserver.Init(remoteSHH, &params.WhisperConfig{
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
		err := localSHH.HandlePeer(remotePeer, rw1)
		s.Require().NoError(err)
	}()
	go func() {
		err := remoteSHH.HandlePeer(localPeer, rw2)
		s.Require().NoError(err)
	}()
	s.mailSymKey, err = s.localWhisperAPI.GenerateSymKeyFromPassword(context.Background(), password)
	s.Require().NoError(err)

	s.envelopeSymkey = "topics"
	s.envelopeSymkeyID, err = s.localWhisperAPI.GenerateSymKeyFromPassword(context.Background(), s.envelopeSymkey)
	s.Require().NoError(err)
}

func (s *RequestWithTrackingHistorySuite) postEnvelopes(topics ...whispertypes.TopicType) []hexutil.Bytes {
	var (
		rst = make([]hexutil.Bytes, len(topics))
		err error
	)
	for i, t := range topics {
		rst[i], err = s.localWhisperAPI.Post(context.Background(), whispertypes.NewMessage{
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

func (s *RequestWithTrackingHistorySuite) createEmptyFilter(topics ...whispertypes.TopicType) string {
	filterid, err := s.localWhisperAPI.NewMessageFilter(whispertypes.Criteria{
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

func (s *RequestWithTrackingHistorySuite) initiateHistoryRequest(topics ...TopicRequest) []protocol.HexBytes {
	requests, err := s.localAPI.InitiateHistoryRequests(context.Background(), InitiateHistoryRequestParams{
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

func (s *RequestWithTrackingHistorySuite) waitNoRequests() {
	store := s.localContext.HistoryStore()
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
	topic1 := whispertypes.TopicType{1, 1, 1, 1}
	topic2 := whispertypes.TopicType{2, 2, 2, 2}
	topic3 := whispertypes.TopicType{3, 3, 3, 3}
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
	s.Require().NotEqual(requests[0], requests[1])
	s.waitMessagesDelivered(filterid, hexes...)

	s.Require().NoError(s.localService.historyUpdates.UpdateTopicHistory(s.localContext, topic1, time.Now()))
	s.Require().NoError(s.localService.historyUpdates.UpdateTopicHistory(s.localContext, topic2, time.Now()))
	s.Require().NoError(s.localService.historyUpdates.UpdateTopicHistory(s.localContext, topic3, time.Now()))
	for _, r := range requests {
		s.Require().NoError(s.localAPI.CompleteRequest(context.TODO(), r.String()))
	}
	s.waitNoRequests()

	requests = s.initiateHistoryRequest(
		TopicRequest{Topic: topic1, Duration: time.Hour},
		TopicRequest{Topic: topic2, Duration: time.Hour},
		TopicRequest{Topic: topic3, Duration: 10 * time.Hour},
	)
	s.Len(requests, 1)
}

func (s *RequestWithTrackingHistorySuite) TestSingleRequest() {
	topic1 := whispertypes.TopicType{1, 1, 1, 1}
	topic2 := whispertypes.TopicType{255, 255, 255, 255}
	hexes := s.postEnvelopes(topic1, topic2)
	s.waitForArchival(hexes)

	filterid := s.createEmptyFilter(topic1, topic2)
	requests := s.initiateHistoryRequest(
		TopicRequest{Topic: topic1, Duration: time.Hour},
		TopicRequest{Topic: topic2, Duration: time.Hour},
	)
	s.Require().Len(requests, 1)
	s.waitMessagesDelivered(filterid, hexes...)
}

func (s *RequestWithTrackingHistorySuite) TestPreviousRequestReplaced() {
	topic1 := whispertypes.TopicType{1, 1, 1, 1}
	topic2 := whispertypes.TopicType{255, 255, 255, 255}

	requests := s.initiateHistoryRequest(
		TopicRequest{Topic: topic1, Duration: time.Hour},
		TopicRequest{Topic: topic2, Duration: time.Hour},
	)
	s.Require().Len(requests, 1)
	s.localService.requestsRegistry.Clear()
	replaced := s.initiateHistoryRequest(
		TopicRequest{Topic: topic1, Duration: time.Hour},
		TopicRequest{Topic: topic2, Duration: time.Hour},
	)
	s.Require().Len(replaced, 1)
	s.Require().NotEqual(requests[0], replaced[0])
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
