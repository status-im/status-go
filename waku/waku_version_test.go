// Copyright 2019 The Waku Library Authors.
//
// The Waku library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Waku library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty off
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Waku library. If not, see <http://www.gnu.org/licenses/>.
//
// This software uses the go-ethereum library, which is licensed
// under the GNU Lesser General Public Library, version 3 or any later.

package waku

import (
	mrand "math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/waku/common"
	v0 "github.com/status-im/status-go/waku/v0"
	v1 "github.com/status-im/status-go/waku/v1"

	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

func TestWakuV0(t *testing.T) {
	ws := new(WakuTestSuite)
	ws.newPeer = v0.NewPeer

	suite.Run(t, ws)
}

func TestWakuV1(t *testing.T) {
	ws := new(WakuTestSuite)
	ws.newPeer = v1.NewPeer

	suite.Run(t, ws)
}

type WakuTestSuite struct {
	suite.Suite
	seed    int64
	stats   *common.StatsTracker
	newPeer func(common.WakuHost, *p2p.Peer, p2p.MsgReadWriter, *zap.Logger, *common.StatsTracker) common.Peer
}

// Set up random seed
func (s *WakuTestSuite) SetupTest() {
	s.seed = time.Now().Unix()
	s.stats = &common.StatsTracker{}
	mrand.Seed(s.seed)
}

func (s *WakuTestSuite) TestHandleP2PMessageCode() {

	w1 := New(nil, nil)
	s.Require().NoError(w1.SetMinimumPoW(0.0000001, false))
	s.Require().NoError(w1.Start())

	go func() { handleError(s.T(), w1.Stop()) }()

	w2 := New(nil, nil)
	s.Require().NoError(w2.SetMinimumPoW(0.0000001, false))
	s.Require().NoError(w2.Start())
	go func() { handleError(s.T(), w2.Stop()) }()

	envelopeEvents := make(chan common.EnvelopeEvent, 10)
	sub := w1.SubscribeEnvelopeEvents(envelopeEvents)
	defer sub.Unsubscribe()

	params, err := generateMessageParams()
	s.Require().NoError(err, "failed generateMessageParams with seed", s.seed)

	params.TTL = 1

	msg, err := common.NewSentMessage(params)
	s.Require().NoError(err, "failed to create new message with seed", seed)

	env, err := msg.Wrap(params, time.Now())
	s.Require().NoError(err, "failed Wrap with seed", seed)

	rw1, rw2 := p2p.MsgPipe()

	go func() {
		s.Require().Error(w1.HandlePeer(s.newPeer(w1, p2p.NewPeer(enode.ID{}, "test", []p2p.Cap{}), rw1, nil, s.stats), rw1))
	}()

	timer := time.AfterFunc(time.Second*5, func() {
		handleError(s.T(), rw1.Close())
		handleError(s.T(), rw2.Close())
	})

	peer1 := s.newPeer(w2, p2p.NewPeer(enode.ID{}, "test", []p2p.Cap{}), rw2, nil, s.stats)
	peer1.SetPeerTrusted(true)

	err = peer1.Start()
	s.Require().NoError(err, "failed run message loop")

	// Simulate receiving the new envelope
	_, err = w2.add(env, true)
	s.Require().NoError(err)

	e := <-envelopeEvents
	s.Require().Equal(e.Hash, env.Hash(), "envelopes not equal")
	peer1.Stop()
	s.Require().NoError(rw1.Close())
	s.Require().NoError(rw2.Close())
	timer.Stop()
}

func (s *WakuTestSuite) testConfirmationsHandshake(expectConfirmations bool) {
	conf := &Config{
		MinimumAcceptedPoW:  0,
		EnableConfirmations: expectConfirmations,
	}
	w1 := New(nil, nil)
	w2 := New(conf, nil)
	rw1, rw2 := p2p.MsgPipe()

	// so that actual read won't hang forever
	timer := time.AfterFunc(5*time.Second, func() {
		handleError(s.T(), rw1.Close())
		handleError(s.T(), rw2.Close())
	})

	p1 := s.newPeer(w1, p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{Name: "waku", Version: 1}}), rw1, nil, s.stats)

	go func() {
		// This will always fail eventually as we close the channels
		s.Require().Error(w1.HandlePeer(p1, rw1))
	}()

	p2 := s.newPeer(w2, p2p.NewPeer(enode.ID{}, "test-2", []p2p.Cap{}), rw2, nil, s.stats)
	err := p2.Start()
	s.Require().NoError(err)
	peers := w1.getPeers()
	s.Require().Len(peers, 1)
	// We need to let the loop run, not very elegant, but otherwise is
	// flaky
	time.Sleep(10 * time.Millisecond)
	s.Require().Equal(expectConfirmations, peers[0].ConfirmationsEnabled())
	timer.Stop()
	s.Require().NoError(rw1.Close())
	s.Require().NoError(rw2.Close())
}

func (s *WakuTestSuite) TestConfirmationHandshakeExtension() {
	s.testConfirmationsHandshake(true)
}

func (s *WakuTestSuite) TestHandshakeWithConfirmationsDisabled() {
	s.testConfirmationsHandshake(false)
}

func (s *WakuTestSuite) TestMessagesResponseWithError() {
	conf := &Config{
		MinimumAcceptedPoW:  0,
		MaxMessageSize:      10 << 20,
		EnableConfirmations: true,
	}
	w1 := New(conf, nil)
	w2 := New(conf, nil)

	rw1, rw2 := p2p.MsgPipe()
	defer func() {
		if err := rw1.Close(); err != nil {
			s.T().Errorf("error closing MsgPipe 1, '%s'", err)
		}
		if err := rw2.Close(); err != nil {
			s.T().Errorf("error closing MsgPipe 2, '%s'", err)
		}
	}()
	p1 := s.newPeer(w1, p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{Name: "waku", Version: 0}}), rw2, nil, s.stats)
	p2 := s.newPeer(w2, p2p.NewPeer(enode.ID{2}, "2", []p2p.Cap{{Name: "waku", Version: 0}}), rw1, nil, s.stats)

	errorc := make(chan error, 1)
	go func() { errorc <- w1.HandlePeer(p1, rw2) }()
	s.Require().NoError(p2.Start())

	failed := common.Envelope{
		Expiry: uint32(time.Now().Add(time.Hour).Unix()),
		TTL:    10,
		Topic:  common.TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}
	normal := common.Envelope{
		Expiry: uint32(time.Now().Unix()) + 5,
		TTL:    10,
		Topic:  common.TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}

	events := make(chan common.EnvelopeEvent, 2)
	sub := w1.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	w2.addEnvelope(&failed)
	w2.addEnvelope(&normal)
	count := 0
	// Wait for the two envelopes to be received
	for count < 2 {
		select {
		case <-time.After(5 * time.Second):
			s.Require().FailNow("didnt receive events")

		case ev := <-events:
			switch ev.Event {
			case common.EventEnvelopeReceived:
				count++
			default:
				s.Require().FailNow("invalid event message", ev.Event)

			}
		}
	}
	// Make sure only one envelope is saved and one is discarded
	s.Require().Len(w1.Envelopes(), 1)
}

func (s *WakuTestSuite) TestEventsWithoutConfirmation() {
	conf := &Config{
		MinimumAcceptedPoW: 0,
		MaxMessageSize:     10 << 20,
	}
	w1 := New(conf, nil)
	w2 := New(conf, nil)
	events := make(chan common.EnvelopeEvent, 2)
	sub := w1.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	rw1, rw2 := p2p.MsgPipe()
	p1 := s.newPeer(w1, p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{Name: "waku", Version: 0}}), rw2, nil, s.stats)

	go func() { handleError(s.T(), w1.HandlePeer(p1, rw2)) }()

	timer := time.AfterFunc(5*time.Second, func() {
		handleError(s.T(), rw1.Close())
	})
	peer2 := s.newPeer(w2, p2p.NewPeer(enode.ID{1}, "1", nil), rw1, nil, s.stats)
	s.Require().NoError(peer2.Start())

	go func() { handleError(s.T(), peer2.Run()) }()

	e := common.Envelope{
		Expiry: uint32(time.Now().Add(10 * time.Second).Unix()),
		TTL:    10,
		Topic:  common.TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}
	s.Require().NoError(w1.Send(&e))

	select {
	case ev := <-events:
		s.Require().Equal(common.EventEnvelopeSent, ev.Event)
		s.Require().Equal(p1.EnodeID(), ev.Peer)
		s.Require().Equal(gethcommon.Hash{}, ev.Batch)
	case <-time.After(5 * time.Second):
		s.Require().FailNow("timed out waiting for an envelope.sent event")
	}
	s.Require().NoError(rw1.Close())
	timer.Stop()
}

func (s *WakuTestSuite) TestWakuTimeDesyncEnvelopeIgnored() {
	c := &Config{
		MaxMessageSize:     common.DefaultMaxMessageSize,
		MinimumAcceptedPoW: 0,
	}
	rw1, rw2 := p2p.MsgPipe()
	defer func() {
		if err := rw1.Close(); err != nil {
			s.T().Errorf("error closing MsgPipe, '%s'", err)
		}
		if err := rw2.Close(); err != nil {
			s.T().Errorf("error closing MsgPipe, '%s'", err)
		}
	}()
	w1, w2 := New(c, nil), New(c, nil)
	p1 := s.newPeer(w2, p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{Name: "waku", Version: 1}}), rw1, nil, s.stats)
	p2 := s.newPeer(w1, p2p.NewPeer(enode.ID{2}, "2", []p2p.Cap{{Name: "waku", Version: 1}}), rw2, nil, s.stats)

	errc := make(chan error)
	go func() { errc <- w1.HandlePeer(p2, rw2) }()
	go func() { errc <- w2.HandlePeer(p1, rw1) }()
	w1.SetTimeSource(func() time.Time {
		return time.Now().Add(time.Hour)
	})
	env := &common.Envelope{
		Expiry: uint32(time.Now().Add(time.Hour).Unix()),
		TTL:    30,
		Topic:  common.TopicType{1},
		Data:   []byte{1, 1, 1},
	}
	s.Require().NoError(w1.Send(env))
	select {
	case err := <-errc:
		s.Require().NoError(err)
	case <-time.After(time.Second):
	}
	s.Require().NoError(rw2.Close())
	select {
	case err := <-errc:
		s.Require().Error(err, "p2p: read or write on closed message pipe")
	case <-time.After(time.Second):
		s.Require().FailNow("connection wasn't closed in expected time")
	}
}

func (s *WakuTestSuite) TestRateLimiterIntegration() {
	conf := &Config{
		MinimumAcceptedPoW: 0,
		MaxMessageSize:     10 << 20,
	}
	w := New(conf, nil)
	w.RegisterRateLimiter(common.NewPeerRateLimiter(nil, &common.MetricsRateLimiterHandler{}))
	rw1, rw2 := p2p.MsgPipe()
	defer func() {
		if err := rw1.Close(); err != nil {
			s.T().Errorf("error closing MsgPipe, '%s'", err)
		}
		if err := rw2.Close(); err != nil {
			s.T().Errorf("error closing MsgPipe, '%s'", err)
		}
	}()
	p := s.newPeer(w, p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{Name: "waku", Version: 0}}), rw2, nil, s.stats)
	errorc := make(chan error, 1)
	go func() { errorc <- w.HandlePeer(p, rw2) }()

	_, err := rw1.ReadMsg()
	s.Require().NoError(err)

	select {
	case err := <-errorc:
		s.Require().NoError(err)
	default:
	}
}

// two generic waku node handshake
func (s *WakuTestSuite) TestPeerHandshakeWithTwoFullNode() {
	rw1, rw2 := p2p.MsgPipe()
	defer func() { handleError(s.T(), rw1.Close()) }()
	defer func() { handleError(s.T(), rw2.Close()) }()

	w1 := New(nil, nil)
	var pow = 0.1
	err := w1.SetMinimumPoW(pow, true)
	s.Require().NoError(err)

	w2 := New(nil, nil)

	go func() {
		handleError(s.T(), w1.HandlePeer(s.newPeer(w1, p2p.NewPeer(enode.ID{}, "test-1", []p2p.Cap{}), rw1, nil, s.stats), rw1))
	}()

	p2 := s.newPeer(w2, p2p.NewPeer(enode.ID{}, "test-2", []p2p.Cap{}), rw2, nil, s.stats)
	err = p2.Start()
	s.Require().NoError(err)

	s.Require().Equal(pow, p2.PoWRequirement())
}

// two generic waku node handshake. one don't send light flag
func (s *WakuTestSuite) TestHandshakeWithOldVersionWithoutLightModeFlag() {
	rw1, rw2 := p2p.MsgPipe()
	defer func() { handleError(s.T(), rw1.Close()) }()
	defer func() { handleError(s.T(), rw2.Close()) }()

	w1 := New(nil, nil)
	w1.SetLightClientMode(true)

	w2 := New(nil, nil)

	go func() {
		handleError(s.T(), w1.HandlePeer(s.newPeer(w1, p2p.NewPeer(enode.ID{}, "test-1", []p2p.Cap{}), rw1, nil, s.stats), rw1))
	}()

	p2 := s.newPeer(w2, p2p.NewPeer(enode.ID{}, "test-2", []p2p.Cap{}), rw2, nil, s.stats)
	err := p2.Start()
	s.Require().NoError(err)
}

// two light nodes handshake. restriction enable
func (s *WakuTestSuite) TestTwoLightPeerHandshakeRestrictionOff() {
	rw1, rw2 := p2p.MsgPipe()
	defer func() { handleError(s.T(), rw1.Close()) }()
	defer func() { handleError(s.T(), rw2.Close()) }()

	w1 := New(nil, nil)
	w1.SetLightClientMode(true)
	w1.settings.RestrictLightClientsConn = false

	w2 := New(nil, nil)
	w2.SetLightClientMode(true)
	w2.settings.RestrictLightClientsConn = false

	go func() {
		handleError(s.T(), w1.HandlePeer(s.newPeer(w1, p2p.NewPeer(enode.ID{}, "test-1", []p2p.Cap{}), rw1, nil, s.stats), rw1))
	}()

	p2 := s.newPeer(w2, p2p.NewPeer(enode.ID{}, "test-2", []p2p.Cap{}), rw2, nil, s.stats)
	s.Require().NoError(p2.Start())
}

// two light nodes handshake. restriction enabled
func (s *WakuTestSuite) TestTwoLightPeerHandshakeError() {
	rw1, rw2 := p2p.MsgPipe()
	defer func() { handleError(s.T(), rw1.Close()) }()
	defer func() { handleError(s.T(), rw2.Close()) }()

	w1 := New(nil, nil)
	w1.SetLightClientMode(true)
	w1.settings.RestrictLightClientsConn = true

	w2 := New(nil, nil)
	w2.SetLightClientMode(true)
	w2.settings.RestrictLightClientsConn = true

	go func() {
		handleError(s.T(), w1.HandlePeer(s.newPeer(w1, p2p.NewPeer(enode.ID{}, "test-1", []p2p.Cap{}), rw1, nil, s.stats), rw1))
	}()

	p2 := s.newPeer(w2, p2p.NewPeer(enode.ID{}, "test-2", []p2p.Cap{}), rw2, nil, s.stats)
	s.Require().Error(p2.Start())
}

func generateMessageParams() (*common.MessageParams, error) {
	// set all the parameters except p.Dst and p.Padding

	buf := make([]byte, 4)
	mrand.Read(buf)       // nolint: gosec
	sz := mrand.Intn(400) // nolint: gosec

	var p common.MessageParams
	p.PoW = 0.01
	p.WorkTime = 1
	p.TTL = uint32(mrand.Intn(1024)) // nolint: gosec
	p.Payload = make([]byte, sz)
	p.KeySym = make([]byte, common.AESKeyLength)
	mrand.Read(p.Payload) // nolint: gosec
	mrand.Read(p.KeySym)  // nolint: gosec
	p.Topic = common.BytesToTopic(buf)

	var err error
	p.Src, err = crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	return &p, nil
}
