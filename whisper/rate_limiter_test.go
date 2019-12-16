package whisper

import (
	"bytes"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/stretchr/testify/require"
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
	runLoop := func(p *Peer, rw p2p.MsgReadWriter) error {
		msg, err := rw.ReadMsg()
		if err != nil {
			return err
		}
		messages <- msg
		return nil
	}

	r := NewPeerRateLimiter(nil, &mockRateLimiterHandler{})
	err := r.decorate(nil, out, runLoop)
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
		throttle := r.throttleIP("<nil>")
		require.False(t, throttle)
		throttle = r.throttlePeer([]byte{0x01, 0x02, 0x03})
		require.False(t, throttle)
	}
}

func TestPeerLimiterHandler(t *testing.T) {
	h := &mockRateLimiterHandler{}
	r := NewPeerRateLimiter(nil, h)
	p := &Peer{
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

	for i := 0; i < count; i += 1 {
		err := rw1.WriteMsg(p2p.Msg{Code: 101})
		require.NoError(t, err)
	}

	<-done

	require.EqualValues(t, 100-defaultPeerRateLimiterConfig.LimitPerSecIP, h.exceedIPLimit)
	require.EqualValues(t, 100-defaultPeerRateLimiterConfig.LimitPerSecPeerID, h.exceedPeerLimit)
}

func TestPeerLimiterHandlerWithWhitelisting(t *testing.T) {
	h := &mockRateLimiterHandler{}
	r := NewPeerRateLimiter(&PeerRateLimiterConfig{
		LimitPerSecIP:      1,
		LimitPerSecPeerID:  1,
		WhitelistedIPs:     []string{"<nil>"}, // no IP is represented as <nil> string
		WhitelistedPeerIDs: []enode.ID{enode.ID{0xaa, 0xbb, 0xcc}},
	}, h)
	p := &Peer{
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

	for i := 0; i < count; i += 1 {
		err := rw1.WriteMsg(p2p.Msg{Code: 101})
		require.NoError(t, err)
	}

	<-done

	require.Equal(t, 0, h.exceedIPLimit)
	require.Equal(t, 0, h.exceedPeerLimit)
}

func echoMessages(r *PeerRateLimiter, p *Peer, rw p2p.MsgReadWriter) error {
	return r.decorate(p, rw, func(p *Peer, rw p2p.MsgReadWriter) error {
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

func (m *mockRateLimiterHandler) ExceedPeerLimit() error { m.exceedPeerLimit += 1; return nil }
func (m *mockRateLimiterHandler) ExceedIPLimit() error   { m.exceedIPLimit += 1; return nil }
