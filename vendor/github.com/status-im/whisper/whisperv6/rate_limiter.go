package whisperv6

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/tsenart/tb"
)

const (
	rateLimitPerSecIP     = 10
	rateLimitPerSecPeerID = 3
)

type runLoop func(p *Peer, rw p2p.MsgReadWriter) error

type rateLimiterHandler interface {
	ExceedPeerLimit()
	ExceedIPLimit()
}

type metricsRateLimiterHandler struct{}

func (metricsRateLimiterHandler) ExceedPeerLimit() { rateLimiterPeerExceeded.Inc(1) }
func (metricsRateLimiterHandler) ExceedIPLimit()   { rateLimiterIPExceeded.Inc(1) }

type peerRateLimiter struct {
	peerIDThrottler *tb.Throttler
	ipThrottler     *tb.Throttler

	handler rateLimiterHandler
}

func newPeerRateLimiter(handler rateLimiterHandler) *peerRateLimiter {
	return &peerRateLimiter{
		peerIDThrottler: tb.NewThrottler(time.Millisecond * 100),
		ipThrottler:     tb.NewThrottler(time.Millisecond * 100),
		handler:         handler,
	}
}

func (r *peerRateLimiter) Decorate(p *Peer, rw p2p.MsgReadWriter, runLoop runLoop) error {
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

			// TODO: use whitelisting for cluster peers.

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
func (r *peerRateLimiter) throttleIP(ip string) bool {
	return r.ipThrottler.Halt(ip, 1, rateLimitPerSecIP)
}

// throttlePeer throttles a number of messages incoming from a peer.
// It allows 3 packets per second.
func (r *peerRateLimiter) throttlePeer(peerID []byte) bool {
	var id enode.ID
	copy(id[:], peerID)
	return r.peerIDThrottler.Halt(id.String(), 1, rateLimitPerSecPeerID)
}
