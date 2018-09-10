package shhext

import (
	"math"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/stretchr/testify/require"
)

func setupTestWithOnePeer(t *testing.T, conf *whisper.Config) (*whisper.Whisper, *p2p.MsgPipeRW, chan error) {
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
	require.NoError(t, p2p.SendItems(rw1, 0, whisper.ProtocolVersion, math.Float64bits(w.MinPow()), w.BloomFilter(), true))
	require.NoError(t, p2p.ExpectMsg(rw1, 8, conf.IngressRateLimit), "peer must send ingress rate limit after handshake")
	return w, rw1, errorc
}

func TestPeerDropsConnection(t *testing.T) {
	conf := &whisper.Config{
		MinimumAcceptedPOW: 0,
		MaxMessageSize:     100 << 10,
		IngressRateLimit:   whisper.RateLimitConfig{uint64(time.Hour), 10 << 10, 1 << 10},
		EgressRateLimit:    whisper.RateLimitConfig{uint64(time.Hour), 10 << 10, 1 << 10},
	}
	_, rw1, errorc := setupTestWithOnePeer(t, conf)

	require.NoError(t, p2p.Send(rw1, 42, make([]byte, 11<<10))) // limit is 1024
	select {
	case err := <-errorc:
		require.Error(t, err, "error must be related to reaching rate limit")
	case <-time.After(time.Second):
		require.FailNow(t, "failed waiting for HandlePeer to exit")
	}
}

func TestRateLimitedDelivery(t *testing.T) {
	conf := &whisper.Config{
		MinimumAcceptedPOW: 0,
		MaxMessageSize:     100 << 10,
		IngressRateLimit:   whisper.RateLimitConfig{uint64(time.Hour), 10 << 10, 1 << 10},
		EgressRateLimit:    whisper.RateLimitConfig{uint64(time.Hour), 10 << 10, 1 << 10},
		TimeSource:         time.Now,
	}
	w, rw1, _ := setupTestWithOnePeer(t, conf)
	small1 := whisper.Envelope{
		Expiry: uint32(time.Now().Add(10 * time.Second).Unix()),
		TTL:    10,
		Topic:  whisper.TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}
	small2 := small1
	small2.Nonce = 2
	big := small1
	big.Nonce = 3
	big.Data = make([]byte, 11<<10)

	require.NoError(t, w.Send(&small1))
	require.NoError(t, w.Send(&big))
	require.NoError(t, w.Send(&small2))

	received := map[common.Hash]struct{}{}
	// we can not guarantee that all expected envelopes will be delivered in a one batch
	// so allow whisper to write multiple times and read every message
	go func() {
		time.Sleep(2 * time.Second)
		rw1.Close()
	}()
	for {
		msg, err := rw1.ReadMsg()
		if err == p2p.ErrPipeClosed {
			require.Contains(t, received, small1.Hash())
			require.Contains(t, received, small2.Hash())
			require.NotContains(t, received, big.Hash())
			break
		}
		require.NoError(t, err)
		require.Equal(t, uint64(1), msg.Code)
		var rst []*whisper.Envelope
		require.NoError(t, msg.Decode(&rst))
		for _, e := range rst {
			received[e.Hash()] = struct{}{}
		}

	}
}
