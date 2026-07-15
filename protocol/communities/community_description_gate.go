package communities

import (
	"crypto/sha256"
	"sync"
)

// descriptionGateEntry records, for one community, the last community description
// this node FULLY processed: its clock and a content hash of the raw description
// payload bytes that were handled.
type descriptionGateEntry struct {
	clock       uint64
	contentHash [32]byte
}

// communityDescriptionGate short-circuits the expensive per-description pipeline
// for redelivered community descriptions that cannot change local state
// (issue #21470-hf).
//
// status-go re-publishes each community description into every fetch/spectate
// window, so the same byte-identical ~1.4MB blob is handed to
// Manager.HandleCommunityDescriptionMessage hundreds of times over a long history
// sync. Each of those calls otherwise pays, before any clock comparison:
// proto.Unmarshal of the blob, GetDecryptedCommunityDescription (read) +
// SaveDecryptedCommunityDescription (write) in preprocessDescription, and GetByID,
// which re-reads and re-unmarshals the stored community blob — ~10s of CPU per
// redelivered copy.
//
// The gate remembers, per community, the clock+content-hash of the last
// description that was fully processed, and lets the handler return early for:
//   - strictly-older clocks (can never win the downstream clock comparison in
//     Community.UpdateCommunityDescription, which rejects them as outdated), and
//   - the byte-identical same-clock redelivery (a guaranteed no-op).
//
// It deliberately does NOT gate a same-clock description whose content differs:
// Community.UpdateCommunityDescription intentionally reprocesses identical clocks
// so a previously-encrypted description can be re-evaluated once the decryption key
// arrives. For that same reason the gate is cleared when new hash-ratchet keys
// arrive (Manager.NewHashRatchetKeys) and when a community is deleted
// (Manager.DeleteCommunity), so those legitimate reprocessing paths are never
// starved.
//
// A community that has only ever been QUEUED for owner validation (never fully
// processed) has no gate entry, so validateCommunity's re-entry into the handler
// is never blocked. The gate fails OPEN: any state it does not positively know to
// be a redelivery proceeds to full processing.
type communityDescriptionGate struct {
	mu      sync.Mutex
	entries map[string]descriptionGateEntry // communityID(hex) -> last processed
}

func newCommunityDescriptionGate() *communityDescriptionGate {
	return &communityDescriptionGate{
		entries: make(map[string]descriptionGateEntry),
	}
}

// hashDescriptionPayload returns a content hash of the raw community description
// payload bytes. SHA-256 makes a collision astronomically unlikely, so a matching
// (clock, hash) pair is safely treated as a byte-identical redelivery.
func hashDescriptionPayload(payload []byte) [32]byte {
	return sha256.Sum256(payload)
}

// shouldSkip reports whether a description with the given clock and content hash
// can be skipped because it can neither advance nor change local state. It returns
// true only for a strictly-older clock, or an equal clock whose content hash matches
// the last fully-processed description for this community. Everything else (unknown
// community, newer clock, equal clock with different content) returns false so the
// handler proceeds.
func (g *communityDescriptionGate) shouldSkip(id string, clock uint64, hash [32]byte) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	entry, ok := g.entries[id]
	if !ok {
		return false
	}
	if clock < entry.clock {
		return true
	}
	if clock == entry.clock && hash == entry.contentHash {
		return true
	}
	return false
}

// recordProcessed advances the gate for a community after a description was fully
// and successfully processed. It only ever moves the recorded clock forward, so an
// out-of-order older success can never shadow a newer one.
func (g *communityDescriptionGate) recordProcessed(id string, clock uint64, hash [32]byte) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if entry, ok := g.entries[id]; ok && clock < entry.clock {
		return
	}
	g.entries[id] = descriptionGateEntry{clock: clock, contentHash: hash}
}

// forget drops the gate entry for a single community so its next description is
// fully reprocessed. Called when the community is deleted.
func (g *communityDescriptionGate) forget(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.entries, id)
}

// forgetAll clears every gate entry. Called when new hash-ratchet key material
// arrives (which may be community- or channel-scoped) so every community's next
// description is reprocessed and can decrypt now-available private data.
func (g *communityDescriptionGate) forgetAll() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.entries = make(map[string]descriptionGateEntry)
}
