package reliability

import (
	"sync"
	"time"

	"github.com/status-im/mvds/state"
)

// Presence is broadcast every five minutes and automatic status expires after
// another five-second receiving allowance. Authenticated MVDS traffic renews
// this local lease too, including ACKs from otherwise quiet chat participants.
const peerOnlineTimeout = 5*time.Minute + 5*time.Second

type peerActivity struct {
	mu        sync.Mutex
	lastSeen  map[state.PeerID]time.Time
	nextPrune time.Time
}

func (p *peerActivity) record(peer state.PeerID, now time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pruneLocked(now)
	p.lastSeen[peer] = now
}

// enqueue must be nonblocking. Holding the activity lock makes the online
// transition and successful notification atomic with concurrent packet activity.
// A full notification channel leaves an unknown/expired peer eligible to retry.
func (p *peerActivity) reportOnline(peer state.PeerID, now time.Time, enqueue func() bool) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pruneLocked(now)
	if last, exists := p.lastSeen[peer]; exists && now.Sub(last) < peerOnlineTimeout {
		p.lastSeen[peer] = now
		return "already_online"
	}
	if !enqueue() {
		return "queue_full"
	}
	p.lastSeen[peer] = now
	return "queued"
}

func (p *peerActivity) pruneLocked(now time.Time) {
	if p.lastSeen == nil {
		p.lastSeen = make(map[state.PeerID]time.Time)
	}
	if now.Before(p.nextPrune) {
		return
	}
	cutoff := now.Add(-peerOnlineTimeout)
	for peer, last := range p.lastSeen {
		if !last.After(cutoff) {
			delete(p.lastSeen, peer)
		}
	}
	p.nextPrune = now.Add(peerOnlineTimeout)
}
