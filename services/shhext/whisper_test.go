package shhext

import (
	"math"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/stretchr/testify/require"
)

func TestPeerDropsConnection(t *testing.T) {
	conf := &whisper.Config{
		MinimumAcceptedPOW: 0,
		MaxMessageSize:     100 << 10,
		IngressRateLimit:   whisper.RateLimitConfig{uint64(time.Hour), 10 << 10, 1 << 10},
		EgressRateLimit:    whisper.RateLimitConfig{uint64(time.Hour), 10 << 10, 1 << 10},
	}
	w := whisper.New(conf)
	idx, _ := discover.BytesID([]byte{0x01})
	p := p2p.NewPeer(idx, "1", []p2p.Cap{{"shh", 6}})
	rw1, rw2 := p2p.MsgPipe()
	errorc := make(chan error, 1)
	go func() {
		errorc <- w.HandlePeer(p, rw2)
	}()
	msg, err := rw1.ReadMsg()
	require.NoError(t, err)
	require.Equal(t, uint64(0), msg.Code)
	require.NoError(t, msg.Discard())
	require.NoError(t, p2p.SendItems(rw1, 0, whisper.ProtocolVersion, math.Float64bits(w.MinPow()), w.BloomFilter()))
	require.NoError(t, p2p.ExpectMsg(rw1, 8, conf.IngressRateLimit), "peer must send ingress rate limit after handshake")

	require.NoError(t, p2p.Send(rw1, 42, make([]byte, 11<<10)))
	select {
	case err := <-errorc:
		require.Error(t, err, "error must be related to reaching rate limit")
	case <-time.After(time.Second):
		require.FailNow(t, "failed waiting for HandlePeer to exit")
	}
}
