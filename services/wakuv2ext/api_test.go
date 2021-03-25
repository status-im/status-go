package wakuv2ext

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

	"go.uber.org/zap"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/appdatabase"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/ext"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/waku"
	wakucommon "github.com/status-im/status-go/waku/common"
	v0 "github.com/status-im/status-go/waku/v0"
)

func TestRequestMessagesErrors(t *testing.T) {
	var err error

	waku := gethbridge.NewGethWakuWrapper(waku.New(nil, nil))
	aNode, err := node.New(&node.Config{
		P2P: p2p.Config{
			MaxPeers:    math.MaxInt32,
			NoDiscovery: true,
		},
		NoUSB: true,
	}) // in-memory node as no data dir
	require.NoError(t, err)
	err = aNode.Register(func(*node.ServiceContext) (node.Service, error) {
		return gethbridge.GetGethWakuFrom(waku), nil
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
	nodeWrapper := ext.NewTestNodeWrapper(nil, waku)
	service := New(config, nodeWrapper, nil, handler, nil)
	api := NewPublicAPI(service)

	const mailServerPeer = "enode://b7e65e1bedc2499ee6cbd806945af5e7df0e59e4070c96821570bd581473eade24a489f5ec95d060c0db118c879403ab88d827d3766978f28708989d35474f87@[::]:51920"

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
	symKeyID, symKeyErr := waku.AddSymKeyFromPassword("some-pass")
	require.NoError(t, symKeyErr)
	hash, err = api.RequestMessages(context.TODO(), ext.MessagesRequest{
		MailServerPeer: mailServerPeer,
		SymKeyID:       symKeyID,
	})
	require.Nil(t, hash)
	require.Contains(t, err.Error(), "could not find peer with ID")

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

	waku := gethbridge.NewGethWakuWrapper(waku.New(nil, nil))
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	nodeWrapper := ext.NewTestNodeWrapper(nil, waku)
	service := New(config, nodeWrapper, nil, nil, db)

	tmpdir, err := ioutil.TempDir("", "test-shhext-service-init-protocol")
	require.NoError(t, err)

	sqlDB, err := appdatabase.InitializeDB(fmt.Sprintf("%s/db.sql", tmpdir), "password")
	require.NoError(t, err)

	tmpfile, err := ioutil.TempFile("", "multi-accounts-tests-")
	require.NoError(t, err)
	multiAccounts, err := multiaccounts.InitializeDB(tmpfile.Name())
	require.NoError(t, err)

	acc := &multiaccounts.Account{KeyUID: "0xdeadbeef"}

	err = service.InitProtocol(privateKey, sqlDB, multiAccounts, acc, zap.NewNop())
	require.NoError(t, err)
}

func TestShhExtSuite(t *testing.T) {
	suite.Run(t, new(ShhExtSuite))
}

type ShhExtSuite struct {
	suite.Suite

	dir      string
	nodes    []*node.Node
	wakus    []types.Waku
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
	w := waku.New(nil, nil)
	err = stack.Register(func(n *node.ServiceContext) (node.Service, error) {
		return w, nil
	})
	s.NoError(err)

	// set up protocol
	config := params.ShhextConfig{
		InstallationID:          "1",
		BackupDisabledDataDir:   s.dir,
		PFSEnabled:              true,
		MailServerConfirmations: true,
		ConnectionTarget:        10,
	}
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	s.Require().NoError(err)
	nodeWrapper := ext.NewTestNodeWrapper(nil, gethbridge.NewGethWakuWrapper(w))
	service := New(config, nodeWrapper, nil, nil, db)
	sqlDB, err := appdatabase.InitializeDB(fmt.Sprintf("%s/%d", s.dir, idx), "password")
	s.Require().NoError(err)

	tmpfile, err := ioutil.TempFile("", "multi-accounts-tests-")
	s.Require().NoError(err)
	multiAccounts, err := multiaccounts.InitializeDB(tmpfile.Name())
	s.Require().NoError(err)

	privateKey, err := crypto.GenerateKey()
	s.NoError(err)

	acc := &multiaccounts.Account{KeyUID: "0xdeadbeef"}

	err = service.InitProtocol(privateKey, sqlDB, multiAccounts, acc, zap.NewNop())
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
	s.wakus = append(s.wakus, gethbridge.NewGethWakuWrapper(w))
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
	s.wakus = nil
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
	s.EqualError(err, "could not find peer with ID: 10841e6db5c02fc331bf36a8d2a9137a1696d9d3b6b1f872f780e02aa8ec5bba")
}

const (
	// internal waku protocol codes
	p2pRequestCompleteCode = 125
)

type WakuNodeMockSuite struct {
	suite.Suite

	localWakuAPI *waku.PublicWakuAPI
	localAPI     *PublicAPI
	localNode    *enode.Node
	remoteRW     *p2p.MsgPipeRW

	localService *Service
}

func (s *WakuNodeMockSuite) SetupTest() {
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	s.Require().NoError(err)
	conf := &waku.Config{
		MinimumAcceptedPoW:  0,
		MaxMessageSize:      100 << 10,
		EnableConfirmations: true,
	}
	w := waku.New(conf, nil)
	w2 := waku.New(nil, nil)
	s.Require().NoError(w.Start(nil))
	pkey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	node := enode.NewV4(&pkey.PublicKey, net.ParseIP("127.0.0.1"), 1, 1)
	rw1, rw2 := p2p.MsgPipe()
	peer := v0.NewPeer(w, p2p.NewPeer(node.ID(), "1", []p2p.Cap{{"shh", 6}}), rw2, nil)
	go func() {
		err := w.HandlePeer(peer, rw2)
		panic(err)
	}()
	wakuWrapper := gethbridge.NewGethWakuWrapper(w)

	peer1 := v0.NewPeer(w2, p2p.NewPeer(enode.ID{}, "test", []p2p.Cap{}), rw1, nil)
	err = peer1.Start()
	s.Require().NoError(err, "failed run message loop")

	nodeWrapper := ext.NewTestNodeWrapper(nil, wakuWrapper)
	s.localService = New(
		params.ShhextConfig{MailServerConfirmations: true, MaxMessageDeliveryAttempts: 3},
		nodeWrapper,
		nil,
		nil,
		db,
	)
	s.Require().NoError(s.localService.UpdateMailservers([]*enode.Node{node}))

	s.localWakuAPI = waku.NewPublicWakuAPI(w)
	s.localAPI = NewPublicAPI(s.localService)
	s.localNode = node
	s.remoteRW = rw1
}

func TestRequestMessagesSync(t *testing.T) {
	suite.Run(t, new(RequestMessagesSyncSuite))
}

type RequestMessagesSyncSuite struct {
	WakuNodeMockSuite
}

// NOTE: Disabling this for now as too flaky
/*
func (s *RequestMessagesSyncSuite) TestExpired() {
	// intentionally discarding all requests, so that request will timeout
	go func() {
		for {
			msg, err := s.remoteRW.ReadMsg()
			s.Require().NoError(err)
			s.Require().NoError(msg.Discard())
		}
	}()
	_, err := s.localAPI.RequestMessagesSync(
		ext.RetryConfig{
			BaseTimeout: time.Millisecond * 100,
		},
		ext.MessagesRequest{
			MailServerPeer: s.localNode.String(),
			Topics:         []common.TopicType{{0x01, 0x02, 0x03, 0x04}},
		},
	)
	s.Require().EqualError(err, "failed to request messages after 1 retries")
}
*/

func (s *RequestMessagesSyncSuite) testCompletedFromAttempt(target int) {
	const cursorSize = 36 // taken from mailserver_response.go from waku package
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
			var e wakucommon.Envelope
			s.Require().NoError(msg.Decode(&e))
			s.Require().NoError(p2p.Send(s.remoteRW, p2pRequestCompleteCode, waku.CreateMailServerRequestCompletedPayload(e.Hash(), common.Hash{}, cursor[:])))
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
