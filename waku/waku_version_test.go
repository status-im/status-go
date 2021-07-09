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
	"errors"
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
	"github.com/status-im/status-go/protocol/tt"
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
	newPeer func(common.WakuHost, *p2p.Peer, p2p.MsgReadWriter, *zap.Logger) common.Peer
}

// Set up random seed
func (s *WakuTestSuite) SetupTest() {
	s.seed = time.Now().Unix()
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
		s.Require().Error(w1.HandlePeer(s.newPeer(w1, p2p.NewPeer(enode.ID{}, "test", []p2p.Cap{}), rw1, nil), rw1))
	}()

	timer := time.AfterFunc(time.Second*5, func() {
		handleError(s.T(), rw1.Close())
		handleError(s.T(), rw2.Close())
	})

	peer1 := s.newPeer(w2, p2p.NewPeer(enode.ID{}, "test", []p2p.Cap{}), rw2, nil)
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

	p1 := s.newPeer(w1, p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"waku", 1}}), rw1, nil)

	go func() {
		// This will always fail eventually as we close the channels
		s.Require().Error(w1.HandlePeer(p1, rw1))
	}()

	p2 := s.newPeer(w2, p2p.NewPeer(enode.ID{}, "test-2", []p2p.Cap{}), rw2, nil)
	err := p2.Start()
	s.Require().NoError(err)
	peers := w1.getPeers()
	s.Require().Len(peers, 1)
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
	p1 := s.newPeer(w1, p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"waku", 0}}), rw2, nil)
	p2 := s.newPeer(w2, p2p.NewPeer(enode.ID{2}, "2", []p2p.Cap{{"waku", 0}}), rw1, nil)

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

func (s *WakuTestSuite) testConfirmationEvents(envelope common.Envelope, envelopeErrors []common.EnvelopeError) {
	conf := &Config{
		MinimumAcceptedPoW:  0,
		MaxMessageSize:      10 << 20,
		EnableConfirmations: true,
	}
	w1 := New(conf, nil)
	w2 := New(conf, nil)
	events := make(chan common.EnvelopeEvent, 2)
	sub := w1.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	rw1, rw2 := p2p.MsgPipe()

	p1 := s.newPeer(w1, p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"waku", 0}}), rw2, nil)
	p2 := s.newPeer(w2, p2p.NewPeer(enode.ID{2}, "2", []p2p.Cap{{"waku", 0}}), rw1, nil)

	errorc := make(chan error, 1)
	go func() { errorc <- w1.HandlePeer(p1, rw2) }()

	timer := time.AfterFunc(5*time.Second, func() {
		if err := rw1.Close(); err != nil {
			s.T().Errorf("error closing MsgPipe 1, '%s'", err)
		}
		if err := rw2.Close(); err != nil {
			s.T().Errorf("error closing MsgPipe 2, '%s'", err)
		}

	})

	// Start peer
	err := p2.Start()
	s.Require().NoError(err)

	// And run mainloop
	go func() { errorc <- p2.Run() }()

	w1.addEnvelope(&envelope)

	var e1, e2 *common.EnvelopeEvent
	var count int
	for count < 2 {
		select {
		case ev := <-events:
			switch ev.Event {
			case common.EventEnvelopeSent:
				if e1 == nil {
					e1 = &ev
					count++
				}
			case common.EventBatchAcknowledged:
				if e2 == nil {
					e2 = &ev
					count++
				}

			}

		case <-time.After(5 * time.Second):
			s.Require().FailNow("timed out waiting for an envelope.sent event")
		}
	}
	s.Require().Equal(p1.EnodeID(), e1.Peer)
	s.Require().NotEqual(gethcommon.Hash{}, e1.Batch)
	s.Require().Equal(p1.EnodeID(), e2.Peer)
	s.Require().Equal(e1.Batch, e2.Batch)
	s.Require().Equal(envelopeErrors, e2.Data)
	s.Require().NoError(rw1.Close())
	s.Require().NoError(rw2.Close())
	timer.Stop()
}

func (s *WakuTestSuite) TestConfirmationEventsReceived() {
	e := common.Envelope{
		Expiry: uint32(time.Now().Add(10 * time.Second).Unix()),
		TTL:    10,
		Topic:  common.TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}
	s.testConfirmationEvents(e, []common.EnvelopeError{})
}

func (s *WakuTestSuite) TestConfirmationEventsExtendedWithErrors() {
	e := common.Envelope{
		Expiry: uint32(time.Now().Unix()) - 4*common.DefaultSyncAllowance,
		TTL:    10,
		Topic:  common.TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}
	s.testConfirmationEvents(e, []common.EnvelopeError{
		{
			Hash:        e.Hash(),
			Code:        common.EnvelopeTimeNotSynced,
			Description: "very old envelope",
		}},
	)
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
	p1 := s.newPeer(w1, p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"waku", 0}}), rw2, nil)

	go func() { handleError(s.T(), w1.HandlePeer(p1, rw2)) }()

	timer := time.AfterFunc(5*time.Second, func() {
		handleError(s.T(), rw1.Close())
	})
	peer2 := s.newPeer(w2, p2p.NewPeer(enode.ID{1}, "1", nil), rw1, nil)
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

func discardPipe() *p2p.MsgPipeRW {
	rw1, rw2 := p2p.MsgPipe()
	go func() {
		for {
			msg, err := rw1.ReadMsg()
			if err != nil {
				return
			}
			msg.Discard() // nolint: errcheck
		}
	}()
	return rw2
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
	p1 := s.newPeer(w2, p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"waku", 1}}), rw1, nil)
	p2 := s.newPeer(w1, p2p.NewPeer(enode.ID{2}, "2", []p2p.Cap{{"waku", 1}}), rw2, nil)

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

func (s *WakuTestSuite) TestRequestSentEventWithExpiry() {
	w := New(nil, nil)
	p := p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"waku", 1}})
	rw := discardPipe()
	defer func() { handleError(s.T(), rw.Close()) }()
	w.peers[s.newPeer(w, p, rw, nil)] = struct{}{}
	events := make(chan common.EnvelopeEvent, 1)
	sub := w.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()
	e := &common.Envelope{Nonce: 1}
	s.Require().NoError(w.RequestHistoricMessagesWithTimeout(p.ID().Bytes(), e, time.Millisecond))
	verifyEvent := func(etype common.EventType) {
		select {
		case <-time.After(time.Second):
			s.Require().FailNow("error waiting for a event type %s", etype)
		case ev := <-events:
			s.Require().Equal(etype, ev.Event)
			s.Require().Equal(p.ID(), ev.Peer)
			s.Require().Equal(e.Hash(), ev.Hash)
		}
	}
	verifyEvent(common.EventMailServerRequestSent)
	verifyEvent(common.EventMailServerRequestExpired)
}

type MockMailserver struct {
	deliverMail func([]byte, *common.Envelope)
}

func (*MockMailserver) Archive(e *common.Envelope) {
}

func (*MockMailserver) Deliver(peerID []byte, r common.MessagesRequest) {
}

func (m *MockMailserver) DeliverMail(peerID []byte, e *common.Envelope) {

	if m.deliverMail != nil {
		m.deliverMail(peerID, e)
	}
}

func (s *WakuTestSuite) TestDeprecatedDeliverMail() {

	w1 := New(nil, nil)
	w2 := New(nil, nil)

	var deliverMailCalled bool

	w2.RegisterMailServer(&MockMailserver{
		deliverMail: func(peerID []byte, e *common.Envelope) {
			deliverMailCalled = true
		},
	})

	rw1, rw2 := p2p.MsgPipe()
	p1 := s.newPeer(w1, p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"waku", 0}}), rw2, nil)

	go func() { handleError(s.T(), w1.HandlePeer(p1, rw2)) }()

	timer := time.AfterFunc(5*time.Second, func() {
		handleError(s.T(), rw1.Close())
	})
	peer2 := s.newPeer(w2, p2p.NewPeer(enode.ID{1}, "1", nil), rw1, nil)
	s.Require().NoError(peer2.Start())

	go func() { handleError(s.T(), peer2.Run()) }()

	s.Require().NoError(w1.RequestHistoricMessages(p1.ID(), &common.Envelope{Data: []byte{1}}))

	err := tt.RetryWithBackOff(func() error {
		if !deliverMailCalled {
			return errors.New("DeliverMail not called")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().NoError(rw1.Close())
	s.Require().NoError(rw2.Close())

	timer.Stop()

}

func (s *WakuTestSuite) TestSendMessagesRequest() {
	validMessagesRequest := common.MessagesRequest{
		ID:    make([]byte, 32),
		From:  0,
		To:    10,
		Bloom: []byte{0x01},
	}

	s.Run("InvalidID", func() {
		w := New(nil, nil)
		err := w.SendMessagesRequest([]byte{0x01, 0x02}, common.MessagesRequest{})
		s.Require().EqualError(err, "invalid 'ID', expected a 32-byte slice")
	})

	s.Run("WithoutPeer", func() {
		w := New(nil, nil)
		err := w.SendMessagesRequest([]byte{0x01, 0x02}, validMessagesRequest)
		s.Require().EqualError(err, "could not find peer with ID: 0102")
	})

	s.Run("AllGood", func() {
		p := p2p.NewPeer(enode.ID{0x01}, "peer01", nil)
		rw1, rw2 := p2p.MsgPipe()
		w := New(nil, nil)
		w.peers[s.newPeer(w, p, rw1, nil)] = struct{}{}

		go func() {
			// Read out so that it's consumed
			_, err := rw2.ReadMsg()
			s.Require().NoError(err)
			s.Require().NoError(rw1.Close())
			s.Require().NoError(rw2.Close())

		}()
		err := w.SendMessagesRequest(p.ID().Bytes(), validMessagesRequest)
		s.Require().NoError(err)
	})
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
	p := s.newPeer(w, p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"waku", 0}}), rw2, nil)
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

func (s *WakuTestSuite) TestMailserverCompletionEvent() {
	w1 := New(nil, nil)
	s.Require().NoError(w1.Start())
	defer func() { handleError(s.T(), w1.Stop()) }()

	rw1, rw2 := p2p.MsgPipe()
	errorc := make(chan error, 1)
	go func() {
		err := w1.HandlePeer(s.newPeer(w1, p2p.NewPeer(enode.ID{}, "1", []p2p.Cap{}), rw1, nil), rw1)
		errorc <- err
	}()

	w2 := New(nil, nil)
	s.Require().NoError(w2.Start())
	defer func() { handleError(s.T(), w2.Stop()) }()

	peer2 := s.newPeer(w2, p2p.NewPeer(enode.ID{1}, "1", nil), rw2, nil)
	peer2.SetPeerTrusted(true)

	events := make(chan common.EnvelopeEvent)
	sub := w1.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	envelopes := []*common.Envelope{{Data: []byte{1}}, {Data: []byte{2}}}
	s.Require().NoError(peer2.Start())
	// Set peer trusted, we know the peer has been added as handshake was successful
	w1.getPeers()[0].SetPeerTrusted(true)

	s.Require().NoError(peer2.SendP2PMessages(envelopes))
	s.Require().NoError(peer2.SendHistoricMessageResponse(make([]byte, 100)))
	s.Require().NoError(rw2.Close())

	// Wait for all messages to be read
	err := <-errorc
	s.Require().EqualError(err, "p2p: read or write on closed message pipe")

	after := time.After(2 * time.Second)
	count := 0
	for {
		select {
		case <-after:
			s.Require().FailNow("timed out waiting for all events")
		case ev := <-events:
			switch ev.Event {
			case common.EventEnvelopeAvailable:
				count++
			case common.EventMailServerRequestCompleted:
				s.Require().Equal(count, len(envelopes),
					"all envelope.avaiable events mut be recevied before request is compelted")
				return
			}
		}
	}
}

//two generic waku node handshake
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
		handleError(s.T(), w1.HandlePeer(s.newPeer(w1, p2p.NewPeer(enode.ID{}, "test-1", []p2p.Cap{}), rw1, nil), rw1))
	}()

	p2 := s.newPeer(w2, p2p.NewPeer(enode.ID{}, "test-2", []p2p.Cap{}), rw2, nil)
	err = p2.Start()
	s.Require().NoError(err)

	s.Require().Equal(pow, p2.PoWRequirement())
}

//two generic waku node handshake. one don't send light flag
func (s *WakuTestSuite) TestHandshakeWithOldVersionWithoutLightModeFlag() {
	rw1, rw2 := p2p.MsgPipe()
	defer func() { handleError(s.T(), rw1.Close()) }()
	defer func() { handleError(s.T(), rw2.Close()) }()

	w1 := New(nil, nil)
	w1.SetLightClientMode(true)

	w2 := New(nil, nil)

	go func() {
		handleError(s.T(), w1.HandlePeer(s.newPeer(w1, p2p.NewPeer(enode.ID{}, "test-1", []p2p.Cap{}), rw1, nil), rw1))
	}()

	p2 := s.newPeer(w2, p2p.NewPeer(enode.ID{}, "test-2", []p2p.Cap{}), rw2, nil)
	err := p2.Start()
	s.Require().NoError(err)
}

//two light nodes handshake. restriction enable
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
		handleError(s.T(), w1.HandlePeer(s.newPeer(w1, p2p.NewPeer(enode.ID{}, "test-1", []p2p.Cap{}), rw1, nil), rw1))
	}()

	p2 := s.newPeer(w2, p2p.NewPeer(enode.ID{}, "test-2", []p2p.Cap{}), rw2, nil)
	s.Require().NoError(p2.Start())
}

//two light nodes handshake. restriction enabled
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
		handleError(s.T(), w1.HandlePeer(s.newPeer(w1, p2p.NewPeer(enode.ID{}, "test-1", []p2p.Cap{}), rw1, nil), rw1))
	}()

	p2 := s.newPeer(w2, p2p.NewPeer(enode.ID{}, "test-2", []p2p.Cap{}), rw2, nil)
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
