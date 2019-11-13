package whisperv6

import (
	"bytes"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/tsenart/tb"
)

type runLoop func(p *Peer, rw p2p.MsgReadWriter) error

type RateLimiterHandler interface {
	ExceedPeerLimit()
	ExceedIPLimit()
}

type MetricsRateLimiterHandler struct{}

func (MetricsRateLimiterHandler) ExceedPeerLimit() { rateLimiterPeerExceeded.Inc(1) }
func (MetricsRateLimiterHandler) ExceedIPLimit()   { rateLimiterIPExceeded.Inc(1) }

type PeerRateLimiterConfig struct {
	LimitPerSecIP      int64
	LimitPerSecPeerID  int64
	WhitelistedIPs     []string
	WhitelistedPeerIDs []enode.ID
}

var defaultPeerRateLimiterConfig = PeerRateLimiterConfig{
	LimitPerSecIP:      10,
	LimitPerSecPeerID:  5,
	WhitelistedIPs:     nil,
	WhitelistedPeerIDs: nil,
}

type PeerRateLimiter struct {
	peerIDThrottler *tb.Throttler
	ipThrottler     *tb.Throttler

	limitPerSecIP     int64
	limitPerSecPeerID int64

	whitelistedPeerIDs []enode.ID
	whitelistedIPs     []string

	handler RateLimiterHandler
}

func NewPeerRateLimiter(handler RateLimiterHandler, cfg *PeerRateLimiterConfig) *PeerRateLimiter {
	if cfg == nil {
		copy := defaultPeerRateLimiterConfig
		cfg = &copy
	}

	return &PeerRateLimiter{
		peerIDThrottler:    tb.NewThrottler(time.Millisecond * 100),
		ipThrottler:        tb.NewThrottler(time.Millisecond * 100),
		limitPerSecIP:      cfg.LimitPerSecIP,
		limitPerSecPeerID:  cfg.LimitPerSecPeerID,
		whitelistedPeerIDs: cfg.WhitelistedPeerIDs,
		whitelistedIPs:     cfg.WhitelistedIPs,
		handler:            handler,
	}
}

func (r *PeerRateLimiter) decorate(p *Peer, rw p2p.MsgReadWriter, runLoop runLoop) error {
	in, out := p2p.MsgPipe()
	defer in.Close()
	defer out.Close()
	errC := make(chan error, 1)

	// Read from the original reader and write to the message pipe.
	go func() {
		for {
			packet, err := rw.ReadMsg()
			if err != nil {
				errC <- fmt.Errorf("failed to read packet: %v", err)
				return
			}

			rateLimiterProcessed.Inc(1)

			var ip string
			if p != nil && p.peer != nil {
				ip = p.peer.Node().IP().String()
			}
			if halted := r.throttleIP(ip); halted {
				r.handler.ExceedIPLimit()
			}

			var peerID []byte
			if p != nil {
				peerID = p.ID()
			}
			if halted := r.throttlePeer(peerID); halted {
				r.handler.ExceedPeerLimit()
			}

			if err := in.WriteMsg(packet); err != nil {
				errC <- fmt.Errorf("failed to write packet to pipe: %v", err)
				return
			}
		}
	}()

	// Read from the message pipe and write to the original writer.
	go func() {
		for {
			packet, err := in.ReadMsg()
			if err != nil {
				errC <- fmt.Errorf("failed to read packet from pipe: %v", err)
				return
			}
			if err := rw.WriteMsg(packet); err != nil {
				errC <- fmt.Errorf("failed to write packet: %v", err)
				return
			}
		}
	}()

	go func() {
		errC <- runLoop(p, out)
	}()

	return <-errC
}

// throttleIP throttles a number of messages incoming from a given IP.
// It allows 10 packets per second.
func (r *PeerRateLimiter) throttleIP(ip string) bool {
	if stringSliceContains(r.whitelistedIPs, ip) {
		return false
	}
	return r.ipThrottler.Halt(ip, 1, r.limitPerSecIP)
}

// throttlePeer throttles a number of messages incoming from a peer.
// It allows 3 packets per second.
func (r *PeerRateLimiter) throttlePeer(peerID []byte) bool {
	var id enode.ID
	copy(id[:], peerID)
	if enodeIDSliceContains(r.whitelistedPeerIDs, id) {
		return false
	}
	return r.peerIDThrottler.Halt(id.String(), 1, r.limitPerSecPeerID)
}

func stringSliceContains(s []string, searched string) bool {
	for _, item := range s {
		if item == searched {
			return true
		}
	}
	return false
}

func enodeIDSliceContains(s []enode.ID, searched enode.ID) bool {
	for _, item := range s {
		if bytes.Equal(item.Bytes(), searched.Bytes()) {
			return true
		}
	}
	return false
}
