package shhext

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupOneConnection(t *testing.T, conf *whisper.Config) (*whisper.Whisper, *p2p.MsgPipeRW, chan error) {
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
	_, rw1, errorc := setupOneConnection(t, conf)

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
	w, rw1, _ := setupOneConnection(t, conf)
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
		time.Sleep(time.Second)
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

func TestRandomizedDelivery(t *testing.T) {
	conf := &whisper.Config{
		MinimumAcceptedPOW: 0,
		MaxMessageSize:     100 << 10,
		IngressRateLimit:   whisper.RateLimitConfig{uint64(time.Hour), 10 << 10, 1},
		EgressRateLimit:    whisper.RateLimitConfig{uint64(time.Hour), 10 << 10, 1},
		TimeSource:         time.Now,
	}
	w1, rw1, _ := setupOneConnection(t, conf)
	w2, rw2, _ := setupOneConnection(t, conf)
	w3, rw3, _ := setupOneConnection(t, conf)
	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		sent     = map[common.Hash]int{}
		received = map[int]int64{}
	)
	for i := uint64(1); i < 15; i++ {
		e := &whisper.Envelope{
			Expiry: uint32(time.Now().Add(10 * time.Second).Unix()),
			TTL:    10,
			Topic:  whisper.TopicType{1},
			Data:   make([]byte, 1<<10-whisper.EnvelopeHeaderLength), // so that 10 envelopes are exactly 10kb
			Nonce:  i,
		}
		sent[e.Hash()] = 0
		for _, w := range []*whisper.Whisper{w1, w2, w3} {
			go func(w *whisper.Whisper, i *whisper.Envelope) {
				time.Sleep(time.Duration(rand.Int63n(10)) * time.Millisecond)
				assert.NoError(t, w.Send(e))
			}(w, e)
		}
	}
	for i, rw := range []*p2p.MsgPipeRW{rw1, rw2, rw3} {
		received[i] = 0
		wg.Add(2)
		go func(rw *p2p.MsgPipeRW) {
			time.Sleep(time.Second)
			rw.Close()
			wg.Done()
		}(rw)
		go func(i int, rw *p2p.MsgPipeRW) {
			defer wg.Done()
			for {
				msg, err := rw.ReadMsg()
				if err != nil {
					return
				}
				if !assert.Equal(t, uint64(1), msg.Code) {
					return
				}
				var rst []*whisper.Envelope
				if !assert.NoError(t, msg.Decode(&rst)) {
					return
				}
				mu.Lock()
				for _, e := range rst {
					received[i] += int64(len(e.Data))
					received[i] += whisper.EnvelopeHeaderLength
					sent[e.Hash()]++
				}
				mu.Unlock()
			}
		}(i, rw)
	}
	wg.Wait()
	for i := range received {
		require.Equal(t, received[i], int64(10)<<10, "peer %d didnt' receive 10 kb of data: %d", i, received[i])
	}
	total := 0
	for h := range sent {
		total += sent[h]
		assert.True(t, sent[h] > 0, "every envelope(%s) should be sent atleat 1", h.String())
	}
	require.Equal(t, 30, total)
}

func TestTopicDrained(t *testing.T) {
	conf := &whisper.Config{
		MinimumAcceptedPOW: 0,
		MaxMessageSize:     100 << 10,
		IngressRateLimit:   whisper.RateLimitConfig{uint64(time.Hour), 10 << 10, 1},
		EgressRateLimit:    whisper.RateLimitConfig{uint64(time.Hour), 10 << 10, 1},
		TopicRateLimit:     whisper.RateLimitConfig{uint64(time.Hour), 1024, 1},
		TimeSource:         time.Now,
	}
	w1, rw1, errc := setupOneConnection(t, conf)
	topic := whisper.TopicType{1}
	require.False(t, w1.Drained(topic))
	envs := []*whisper.Envelope{}
	for i := uint64(1); i <= 4; i++ {
		envs = append(envs, &whisper.Envelope{
			Expiry: uint32(time.Now().Add(10 * time.Second).Unix()),
			TTL:    10,
			Topic:  topic,
			Data:   make([]byte, 300),
			Nonce:  i,
		})
	}
	require.NoError(t, p2p.Send(rw1, 1, envs))
	timer := time.After(time.Second)
	for {
		select {
		case <-timer:
			require.FailNow(t, "expected topic wasn't drained during allowed period")
		case err := <-errc:
			require.NoError(t, err, "whisper handler returned unexpected error")
		default:
			if w1.Drained(topic) {
				return
			}
		}
	}
}

func TestBloomFilterUpdatedOnUnsub(t *testing.T) {
	conf := &whisper.Config{
		MinimumAcceptedPOW: 0,
		MaxMessageSize:     100 << 10,
		IngressRateLimit:   whisper.RateLimitConfig{uint64(time.Hour), 10 << 10, 1},
		EgressRateLimit:    whisper.RateLimitConfig{uint64(time.Hour), 10 << 10, 1},
		TimeSource:         time.Now,
	}
	w1, rw1, _ := setupOneConnection(t, conf)
	go func() {
		assert.NoError(t, w1.SetBloomFilter(make([]byte, whisper.BloomFilterSize)))
	}()
	require.NoError(t, errorOnTimeout(time.Second, func() error {
		return p2p.ExpectMsg(rw1, 3, make([]byte, whisper.BloomFilterSize))
	}))
	bloomTopics := []whisper.TopicType{{1, 1, 1, 1}, {255, 255, 255, 255}}
	go func() {
		_, err := w1.Subscribe(&whisper.Filter{
			Topics: [][]byte{bloomTopics[0][:]},
			KeySym: []byte{1, 2, 3},
		})
		assert.NoError(t, err)
	}()
	topic1Bloom := whisper.TopicToBloom(bloomTopics[0])
	require.NoError(t, errorOnTimeout(time.Second, func() error {
		return p2p.ExpectMsg(rw1, 3, topic1Bloom)
	}))
	topic2Id := make(chan string, 1)
	go func() {
		id, err := w1.Subscribe(&whisper.Filter{
			Topics: [][]byte{bloomTopics[1][:]},
			KeySym: []byte{3, 4, 5},
		})
		assert.NoError(t, err)
		topic2Id <- id
	}()
	topic2Bloom := whisper.TopicToBloom(bloomTopics[1])
	for i := 0; i < whisper.BloomFilterSize; i++ { // update in place, no need for removal
		topic2Bloom[i] = topic1Bloom[i] | topic2Bloom[i]
	}
	require.NoError(t, errorOnTimeout(time.Second, func() error {
		return p2p.ExpectMsg(rw1, 3, topic2Bloom)
	}))
	go func() {
		require.NoError(t, w1.Unsubscribe(<-topic2Id))
	}()
	require.NoError(t, errorOnTimeout(time.Second, func() error {
		return p2p.ExpectMsg(rw1, 3, topic1Bloom)
	}))
}

func errorOnTimeout(timeout time.Duration, f func() error) error {
	errc := make(chan error, 1)
	go func() {
		errc <- f()
	}()
	select {
	case <-time.After(timeout):
		return fmt.Errorf("function didn't complete in timeout %v", timeout)
	case err := <-errc:
		return err
	}
}
