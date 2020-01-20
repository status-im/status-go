package shhext

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/ext"
	"github.com/status-im/status-go/sqlite"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/whisper/v6"
)

func TestCreateSyncMailRequest(t *testing.T) {
	testCases := []struct {
		Name   string
		Req    SyncMessagesRequest
		Verify func(*testing.T, types.SyncMailRequest)
		Error  string
	}{
		{
			Name: "no topics",
			Req:  SyncMessagesRequest{},
			Verify: func(t *testing.T, r types.SyncMailRequest) {
				require.Equal(t, types.MakeFullNodeBloom(), r.Bloom)
			},
		},
		{
			Name: "some topics",
			Req: SyncMessagesRequest{
				Topics: []types.TopicType{{0x01, 0xff, 0xff, 0xff}},
			},
			Verify: func(t *testing.T, r types.SyncMailRequest) {
				expectedBloom := types.TopicToBloom(types.TopicType{0x01, 0xff, 0xff, 0xff})
				require.Equal(t, expectedBloom, r.Bloom)
			},
		},
		{
			Name: "decode cursor",
			Req: SyncMessagesRequest{
				Cursor: hex.EncodeToString([]byte{0x01, 0x02, 0x03}),
			},
			Verify: func(t *testing.T, r types.SyncMailRequest) {
				require.Equal(t, []byte{0x01, 0x02, 0x03}, r.Cursor)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			r, err := createSyncMailRequest(tc.Req)
			if tc.Error != "" {
				require.EqualError(t, err, tc.Error)
			}
			tc.Verify(t, r)
		})
	}
}

func TestSyncMessagesErrors(t *testing.T) {
	validEnode := "enode://e8a7c03b58911e98bbd66accb2a55d57683f35b23bf9dfca89e5e244eb5cc3f25018b4112db507faca34fb69ffb44b362f79eda97a669a8df29c72e654416784@127.0.0.1:30404"

	testCases := []struct {
		Name  string
		Req   SyncMessagesRequest
		Resp  SyncMessagesResponse
		Error string
	}{
		{
			Name:  "invalid MailServerPeer",
			Req:   SyncMessagesRequest{MailServerPeer: "invalid-scheme://"},
			Error: `invalid MailServerPeer: invalid URL scheme, want "enode"`,
		},
		{
			Name: "failed to create SyncMailRequest",
			Req: SyncMessagesRequest{
				MailServerPeer: validEnode,
				Cursor:         "a", // odd number of characters is an invalid hex representation
			},
			Error: "failed to create a sync mail request: encoding/hex: odd length hex string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			api := PublicAPI{}
			resp, err := api.SyncMessages(context.TODO(), tc.Req)
			if tc.Error != "" {
				require.EqualError(t, err, tc.Error)
			}
			require.EqualValues(t, tc.Resp, resp)
		})
	}
}

func TestRequestMessagesErrors(t *testing.T) {
	var err error

	shh := gethbridge.NewGethWhisperWrapper(whisper.New(nil))
	aNode, err := node.New(&node.Config{
		P2P: p2p.Config{
			MaxPeers:    math.MaxInt32,
			NoDiscovery: true,
		},
		NoUSB: true,
	}) // in-memory node as no data dir
	require.NoError(t, err)
	err = aNode.Register(func(*node.ServiceContext) (node.Service, error) {
		return gethbridge.GetGethWhisperFrom(shh), nil
	})
	require.NoError(t, err)

	err = aNode.Start()
	require.NoError(t, err)
	defer func() { require.NoError(t, aNode.Stop()) }()

	handler := ext.NewHandlerMock(1)
	config := params.ShhextConfig{
		InstallationID:        "1",
		BackupDisabledDataDir: os.TempDir(),
		PFSEnabled:            true,
	}
	nodeWrapper := ext.NewTestNodeWrapper(shh, nil)
	service := New(config, nodeWrapper, nil, handler, nil)
	api := NewPublicAPI(service)

	const (
		mailServerPeer = "enode://b7e65e1bedc2499ee6cbd806945af5e7df0e59e4070c96821570bd581473eade24a489f5ec95d060c0db118c879403ab88d827d3766978f28708989d35474f87@[::]:51920"
	)

	var hash []byte

	// invalid MailServer enode address
	hash, err = api.RequestMessages(context.TODO(), ext.MessagesRequest{MailServerPeer: "invalid-address"})
	require.Nil(t, hash)
	require.EqualError(t, err, "invalid mailServerPeer value: invalid URL scheme, want \"enode\"")

	// non-existent symmetric key
	hash, err = api.RequestMessages(context.TODO(), ext.MessagesRequest{
		MailServerPeer: mailServerPeer,
		SymKeyID:       "invalid-sym-key-id",
	})
	require.Nil(t, hash)
	require.EqualError(t, err, "invalid symKeyID value: non-existent key ID")

	// with a symmetric key
	symKeyID, symKeyErr := shh.AddSymKeyFromPassword("some-pass")
	require.NoError(t, symKeyErr)
	hash, err = api.RequestMessages(context.TODO(), ext.MessagesRequest{
		MailServerPeer: mailServerPeer,
		SymKeyID:       symKeyID,
	})
	require.Nil(t, hash)
	require.Contains(t, err.Error(), "Could not find peer with ID")

	// from is greater than to
	hash, err = api.RequestMessages(context.TODO(), ext.MessagesRequest{
		From: 10,
		To:   5,
	})
	require.Nil(t, hash)
	require.Contains(t, err.Error(), "Query range is invalid: from > to (10 > 5)")
}

func TestInitProtocol(t *testing.T) {
	directory, err := ioutil.TempDir("", "status-go-testing")
	require.NoError(t, err)

	config := params.ShhextConfig{
		InstallationID:          "2",
		BackupDisabledDataDir:   directory,
		PFSEnabled:              true,
		MailServerConfirmations: true,
		ConnectionTarget:        10,
	}
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	require.NoError(t, err)

	shh := gethbridge.NewGethWhisperWrapper(whisper.New(nil))
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	nodeWrapper := ext.NewTestNodeWrapper(shh, nil)
	service := New(config, nodeWrapper, nil, nil, db)

	tmpdir, err := ioutil.TempDir("", "test-shhext-service-init-protocol")
	require.NoError(t, err)

	sqlDB, err := sqlite.OpenDB(fmt.Sprintf("%s/db.sql", tmpdir), "password")
	require.NoError(t, err)

	err = service.InitProtocol(privateKey, sqlDB)
	require.NoError(t, err)
}

func TestShhExtSuite(t *testing.T) {
	suite.Run(t, new(ShhExtSuite))
}

type ShhExtSuite struct {
	suite.Suite

	dir      string
	nodes    []*node.Node
	whispers []types.Whisper
	services []*Service
}

func (s *ShhExtSuite) createAndAddNode() {
	idx := len(s.nodes)

	// create a node
	cfg := &node.Config{
		Name: strconv.Itoa(idx),
		P2P: p2p.Config{
			MaxPeers:    math.MaxInt32,
			NoDiscovery: true,
			ListenAddr:  ":0",
		},
		NoUSB: true,
	}
	stack, err := node.New(cfg)
	s.NoError(err)
	whisper := whisper.New(nil)
	err = stack.Register(func(n *node.ServiceContext) (node.Service, error) {
		return whisper, nil
	})
	s.NoError(err)

	// set up protocol
	config := params.ShhextConfig{
		InstallationID:          strconv.Itoa(idx),
		BackupDisabledDataDir:   s.dir,
		PFSEnabled:              true,
		MailServerConfirmations: true,
		ConnectionTarget:        10,
	}
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	s.Require().NoError(err)
	nodeWrapper := ext.NewTestNodeWrapper(gethbridge.NewGethWhisperWrapper(whisper), nil)
	service := New(config, nodeWrapper, nil, nil, db)
	sqlDB, err := sqlite.OpenDB(fmt.Sprintf("%s/%d", s.dir, idx), "password")
	s.Require().NoError(err)
	privateKey, err := crypto.GenerateKey()
	s.NoError(err)
	err = service.InitProtocol(privateKey, sqlDB)
	s.NoError(err)
	err = stack.Register(func(n *node.ServiceContext) (node.Service, error) {
		return service, nil
	})
	s.NoError(err)

	// start the node
	err = stack.Start()
	s.Require().NoError(err)

	// store references
	s.nodes = append(s.nodes, stack)
	s.whispers = append(s.whispers, gethbridge.NewGethWhisperWrapper(whisper))
	s.services = append(s.services, service)
}

func (s *ShhExtSuite) SetupTest() {
	var err error
	s.dir, err = ioutil.TempDir("", "status-go-testing")
	s.Require().NoError(err)
}

func (s *ShhExtSuite) TearDownTest() {
	for _, n := range s.nodes {
		s.NoError(n.Stop())
	}
	s.nodes = nil
	s.whispers = nil
	s.services = nil
}

func (s *ShhExtSuite) TestRequestMessagesSuccess() {
	// two nodes needed: client and mailserver
	s.createAndAddNode()
	s.createAndAddNode()

	waitErr := helpers.WaitForPeerAsync(s.nodes[0].Server(), s.nodes[1].Server().Self().URLv4(), p2p.PeerEventTypeAdd, time.Second)
	s.nodes[0].Server().AddPeer(s.nodes[1].Server().Self())
	s.Require().NoError(<-waitErr)

	api := NewPublicAPI(s.services[0])

	_, err := api.RequestMessages(context.Background(), ext.MessagesRequest{
		MailServerPeer: s.nodes[1].Server().Self().URLv4(),
		Topics:         []types.TopicType{{1}},
	})
	s.NoError(err)
}

func (s *ShhExtSuite) TestMultipleRequestMessagesWithoutForce() {
	// two nodes needed: client and mailserver
	s.createAndAddNode()
	s.createAndAddNode()

	waitErr := helpers.WaitForPeerAsync(s.nodes[0].Server(), s.nodes[1].Server().Self().URLv4(), p2p.PeerEventTypeAdd, time.Second)
	s.nodes[0].Server().AddPeer(s.nodes[1].Server().Self())
	s.Require().NoError(<-waitErr)

	api := NewPublicAPI(s.services[0])

	_, err := api.RequestMessages(context.Background(), ext.MessagesRequest{
		MailServerPeer: s.nodes[1].Server().Self().URLv4(),
		Topics:         []types.TopicType{{1}},
	})
	s.NoError(err)
	_, err = api.RequestMessages(context.Background(), ext.MessagesRequest{
		MailServerPeer: s.nodes[1].Server().Self().URLv4(),
		Topics:         []types.TopicType{{1}},
	})
	s.EqualError(err, "another request with the same topics was sent less than 3s ago. Please wait for a bit longer, or set `force` to true in request parameters")
	_, err = api.RequestMessages(context.Background(), ext.MessagesRequest{
		MailServerPeer: s.nodes[1].Server().Self().URLv4(),
		Topics:         []types.TopicType{{2}},
	})
	s.NoError(err)
}

func (s *ShhExtSuite) TestFailedRequestWithUnknownMailServerPeer() {
	s.createAndAddNode()

	api := NewPublicAPI(s.services[0])

	_, err := api.RequestMessages(context.Background(), ext.MessagesRequest{
		MailServerPeer: "enode://19872f94b1e776da3a13e25afa71b47dfa99e658afd6427ea8d6e03c22a99f13590205a8826443e95a37eee1d815fc433af7a8ca9a8d0df7943d1f55684045b7@0.0.0.0:30305",
		Topics:         []types.TopicType{{1}},
	})
	s.EqualError(err, "Could not find peer with ID: 10841e6db5c02fc331bf36a8d2a9137a1696d9d3b6b1f872f780e02aa8ec5bba")
}

const (
	// internal whisper protocol codes
	statusCode             = 0
	p2pRequestCompleteCode = 125
)

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
	s.Require().NoError(p2p.ExpectMsg(rw1, statusCode, []interface{}{
		whisper.ProtocolVersion,
		math.Float64bits(whisperWrapper.MinPow()),
		whisperWrapper.BloomFilter(),
		false,
		true,
		whisper.RateLimits{},
	}))
	s.Require().NoError(p2p.SendItems(
		rw1,
		statusCode,
		whisper.ProtocolVersion,
		whisper.ProtocolVersion,
		math.Float64bits(whisperWrapper.MinPow()),
		whisperWrapper.BloomFilter(),
		true,
		true,
		whisper.RateLimits{},
	))

	nodeWrapper := ext.NewTestNodeWrapper(whisperWrapper, nil)
	s.localService = New(
		params.ShhextConfig{MailServerConfirmations: true, MaxMessageDeliveryAttempts: 3},
		nodeWrapper,
		nil,
		nil,
		db,
	)
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
		ext.RetryConfig{
			BaseTimeout: time.Second,
		},
		ext.MessagesRequest{
			MailServerPeer: s.localNode.String(),
		},
	)
	s.Require().EqualError(err, "failed to request messages after 1 retries")
}

func (s *RequestMessagesSyncSuite) testCompletedFromAttempt(target int) {
	const cursorSize = 36 // taken from mailserver_response.go from whisper package
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
		ext.RetryConfig{
			BaseTimeout: time.Second,
			MaxRetries:  target,
		},
		ext.MessagesRequest{
			MailServerPeer: s.localNode.String(),
			Force:          true, // force true is convenient here because timeout is less then default delay (3s)
		},
	)
	s.Require().NoError(err)
	s.Require().Equal(ext.MessagesResponse{Cursor: hex.EncodeToString(cursor[:])}, resp)
}

func (s *RequestMessagesSyncSuite) TestCompletedFromFirstAttempt() {
	s.testCompletedFromAttempt(1)
}

func (s *RequestMessagesSyncSuite) TestCompletedFromSecondAttempt() {
	s.testCompletedFromAttempt(2)
}
