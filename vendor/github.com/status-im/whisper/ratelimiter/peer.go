package ratelimiter

import (
	"net"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
)

const (
	// IDMode enables rate limiting based on peers public key identity.
	IDMode = 1 + iota
	// IPMode enables rate limiting based on peer external ip address.
	IPMode
)

func ipModeFunc(peer *p2p.Peer) []byte {
	addr := peer.RemoteAddr().Network()
	ip := net.ParseIP(strings.Split(addr, ":")[0])
	return []byte(ip)
}

func idModeFunc(peer *p2p.Peer) []byte {
	return peer.ID().Bytes()
}

// selectFunc returns idModeFunc by default.
func selectFunc(mode int) func(*p2p.Peer) []byte {
	if mode == IPMode {
		return ipModeFunc
	}
	return idModeFunc
}

// NewP2PRateLimiter returns an instance of P2PRateLimiter.
func NewP2PRateLimiter(mode int, ratelimiter Interface) P2PRateLimiter {
	return P2PRateLimiter{
		modeFunc:    selectFunc(mode),
		ratelimiter: ratelimiter,
	}
}

// P2PRateLimiter implements rate limiter that accepts p2p.Peer as identifier.
type P2PRateLimiter struct {
	modeFunc    func(*p2p.Peer) []byte
	ratelimiter Interface
}

func (r P2PRateLimiter) Config() Config {
	return r.ratelimiter.Config()
}

func (r P2PRateLimiter) Create(peer *p2p.Peer) error {
	return r.ratelimiter.Create(r.modeFunc(peer))
}

func (r P2PRateLimiter) Remove(peer *p2p.Peer, duration time.Duration) error {
	return r.ratelimiter.Remove(r.modeFunc(peer), duration)
}

func (r P2PRateLimiter) TakeAvailable(peer *p2p.Peer, count int64) int64 {
	return r.ratelimiter.TakeAvailable(r.modeFunc(peer), count)
}

func (r P2PRateLimiter) Available(peer *p2p.Peer) int64 {
	return r.ratelimiter.Available(r.modeFunc(peer))
}

func (r P2PRateLimiter) UpdateConfig(peer *p2p.Peer, config Config) error {
	return r.ratelimiter.UpdateConfig(r.modeFunc(peer), config)
}

type Whisper struct {
	ingress, egress P2PRateLimiter
}

func ForWhisper(mode int, db DBInterface, ingress, egress Config) Whisper {
	return Whisper{
		ingress: NewP2PRateLimiter(mode, NewPersisted(WithPrefix(db, []byte("i")), ingress)),
		egress:  NewP2PRateLimiter(mode, NewPersisted(WithPrefix(db, []byte("e")), egress)),
	}
}

func (w Whisper) I() P2PRateLimiter {
	return w.ingress
}

func (w Whisper) E() P2PRateLimiter {
	return w.egress
}
