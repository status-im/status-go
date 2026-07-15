package communities

import (
	"crypto/sha256"
	"sync"
)

type descriptionGateEntry struct {
	clock       uint64
	contentHash [32]byte
}

// communityDescriptionGate short-circuits reprocessing of redelivered community
// descriptions that cannot change local state. It remembers the clock and
// content hash of the last fully processed description per community and fails
// open — anything it does not positively know to be a redelivery proceeds.
// Entries are cleared on key arrival (forgetAll) and community deletion
// (forget) so legitimate reprocessing is never starved.
type communityDescriptionGate struct {
	mu      sync.Mutex
	entries map[string]descriptionGateEntry // communityID(hex) -> last processed
}

func newCommunityDescriptionGate() *communityDescriptionGate {
	return &communityDescriptionGate{
		entries: make(map[string]descriptionGateEntry),
	}
}

func hashDescriptionPayload(payload []byte) [32]byte {
	return sha256.Sum256(payload)
}

// shouldSkip reports whether a description can be skipped: older clocks always,
// equal clocks only when byte-identical for a keys-held node (a same-clock,
// different-content description must reprocess so it can be re-evaluated once a
// decryption key arrives), and regardless of content for a keyless node
// (re-encrypted republishes carry the same clock with different bytes).
// holdsKeys must be fail-open (true) on entitlement-lookup errors.
func (g *communityDescriptionGate) shouldSkip(id string, clock uint64, hash [32]byte, holdsKeys bool) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	entry, ok := g.entries[id]
	if !ok {
		return false
	}
	if clock < entry.clock {
		return true
	}
	if clock == entry.clock {
		if !holdsKeys {
			return true
		}
		return hash == entry.contentHash
	}
	return false
}

// recordProcessed only ever moves the recorded clock forward, so an
// out-of-order older success cannot shadow a newer one.
func (g *communityDescriptionGate) recordProcessed(id string, clock uint64, hash [32]byte) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if entry, ok := g.entries[id]; ok && clock < entry.clock {
		return
	}
	g.entries[id] = descriptionGateEntry{clock: clock, contentHash: hash}
}

func (g *communityDescriptionGate) forget(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.entries, id)
}

// forgetAll is called when new hash-ratchet key material arrives (community- or
// channel-scoped) so every community's next description is reprocessed.
func (g *communityDescriptionGate) forgetAll() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.entries = make(map[string]descriptionGateEntry)
}
