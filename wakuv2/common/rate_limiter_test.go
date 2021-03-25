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

package common

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

func TestPeerRateLimiterDecorator(t *testing.T) {
	in, out := p2p.MsgPipe()
	payload := []byte{0x01, 0x02, 0x03}
	msg := p2p.Msg{
		Code:       1,
		Size:       uint32(len(payload)),
		Payload:    bytes.NewReader(payload),
		ReceivedAt: time.Now(),
	}

	go func() {
		err := in.WriteMsg(msg)
		require.NoError(t, err)
	}()

	messages := make(chan p2p.Msg, 1)
	runLoop := func(rw p2p.MsgReadWriter) error {
		msg, err := rw.ReadMsg()
		if err != nil {
			return err
		}
		messages <- msg
		return nil
	}

	r := NewPeerRateLimiter(nil, &mockRateLimiterHandler{})
	err := r.Decorate(nil, out, runLoop)
	require.NoError(t, err)

	receivedMsg := <-messages
	receivedPayload := make([]byte, receivedMsg.Size)
	_, err = receivedMsg.Payload.Read(receivedPayload)
	require.NoError(t, err)
	require.Equal(t, msg.Code, receivedMsg.Code)
	require.Equal(t, payload, receivedPayload)
}

func TestPeerLimiterThrottlingWithZeroLimit(t *testing.T) {
	r := NewPeerRateLimiter(&PeerRateLimiterConfig{}, &mockRateLimiterHandler{})
	for i := 0; i < 1000; i++ {
		throttle := r.throttleIP("<nil>", 0)
		require.False(t, throttle)
		throttle = r.throttlePeer([]byte{0x01, 0x02, 0x03}, 0)
		require.False(t, throttle)
	}
}

func TestPeerPacketLimiterHandler(t *testing.T) {
	h := &mockRateLimiterHandler{}
	r := NewPeerRateLimiter(nil, h)
	p := &TestWakuPeer{
		peer: p2p.NewPeer(enode.ID{0xaa, 0xbb, 0xcc}, "test-peer", nil),
	}
	rw1, rw2 := p2p.MsgPipe()
	count := 100

	go func() {
		err := echoMessages(r, p, rw2)
		require.NoError(t, err)
	}()

	done := make(chan struct{})
	go func() {
		for i := 0; i < count; i++ {
			msg, err := rw1.ReadMsg()
			require.NoError(t, err)
			require.EqualValues(t, 101, msg.Code)
		}
		close(done)
	}()

	for i := 0; i < count; i++ {
		err := rw1.WriteMsg(p2p.Msg{Code: 101})
		require.NoError(t, err)
	}

	<-done

	require.EqualValues(t, 100-defaultPeerRateLimiterConfig.PacketLimitPerSecIP, h.exceedIPLimit)
	require.EqualValues(t, 100-defaultPeerRateLimiterConfig.PacketLimitPerSecPeerID, h.exceedPeerLimit)
}

func TestPeerBytesLimiterHandler(t *testing.T) {
	h := &mockRateLimiterHandler{}
	r := NewPeerRateLimiter(&PeerRateLimiterConfig{
		BytesLimitPerSecIP:     30,
		BytesLimitPerSecPeerID: 30,
	}, h)
	p := &TestWakuPeer{
		peer: p2p.NewPeer(enode.ID{0xaa, 0xbb, 0xcc}, "test-peer", nil),
	}
	rw1, rw2 := p2p.MsgPipe()
	count := 6

	go func() {
		err := echoMessages(r, p, rw2)
		require.NoError(t, err)
	}()

	done := make(chan struct{})
	go func() {
		for i := 0; i < count; i++ {
			msg, err := rw1.ReadMsg()
			require.NoError(t, err)
			require.EqualValues(t, 101, msg.Code)
			require.NoError(t, msg.Discard())
		}
		close(done)
	}()

	for i := 0; i < count; i++ {
		payload := make([]byte, 10)
		msg := p2p.Msg{
			Code:    101,
			Size:    uint32(len(payload)),
			Payload: bytes.NewReader(payload),
		}

		err := rw1.WriteMsg(msg)
		require.NoError(t, err)
	}

	<-done

	require.EqualValues(t, 3, h.exceedIPLimit)
	require.EqualValues(t, 3, h.exceedPeerLimit)
}

func TestPeerPacketLimiterHandlerWithWhitelisting(t *testing.T) {
	h := &mockRateLimiterHandler{}
	r := NewPeerRateLimiter(&PeerRateLimiterConfig{
		PacketLimitPerSecIP:     1,
		PacketLimitPerSecPeerID: 1,
		WhitelistedIPs:          []string{"<nil>"}, // no IP is represented as <nil> string
		WhitelistedPeerIDs:      []enode.ID{{0xaa, 0xbb, 0xcc}},
	}, h)
	p := &TestWakuPeer{
		peer: p2p.NewPeer(enode.ID{0xaa, 0xbb, 0xcc}, "test-peer", nil),
	}
	rw1, rw2 := p2p.MsgPipe()
	count := 100

	go func() {
		err := echoMessages(r, p, rw2)
		require.NoError(t, err)
	}()

	done := make(chan struct{})
	go func() {
		for i := 0; i < count; i++ {
			msg, err := rw1.ReadMsg()
			require.NoError(t, err)
			require.EqualValues(t, 101, msg.Code)
		}
		close(done)
	}()

	for i := 0; i < count; i++ {
		err := rw1.WriteMsg(p2p.Msg{Code: 101})
		require.NoError(t, err)
	}

	<-done

	require.Equal(t, 0, h.exceedIPLimit)
	require.Equal(t, 0, h.exceedPeerLimit)
}

func echoMessages(r *PeerRateLimiter, p RateLimiterPeer, rw p2p.MsgReadWriter) error {
	return r.Decorate(p, rw, func(rw p2p.MsgReadWriter) error {
		for {
			msg, err := rw.ReadMsg()
			if err != nil {
				return err
			}
			err = rw.WriteMsg(msg)
			if err != nil {
				return err
			}
		}
	})
}

type mockRateLimiterHandler struct {
	exceedPeerLimit int
	exceedIPLimit   int
}

func (m *mockRateLimiterHandler) ExceedPeerLimit() error { m.exceedPeerLimit++; return nil }
func (m *mockRateLimiterHandler) ExceedIPLimit() error   { m.exceedIPLimit++; return nil }

type TestWakuPeer struct {
	peer *p2p.Peer
}

func (p *TestWakuPeer) IP() net.IP {
	return p.peer.Node().IP()
}

func (p *TestWakuPeer) ID() []byte {
	id := p.peer.ID()
	return id[:]
}
