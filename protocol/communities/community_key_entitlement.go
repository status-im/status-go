package communities

import "sync"

// dropLogInterval is how often (in gated descriptions) a per-community DEBUG
// checkpoint is emitted after the initial INFO line.
const dropLogInterval = 1000

// communityKeyEntitlement caches, per community, whether this node holds ANY
// hash-ratchet decryption key material for that community. A keyless (spectator)
// node uses the cache to skip the per-key decryption attempts that a re-published
// community description would otherwise trigger on every fetch — each of which is
// a guaranteed "no ratchet key" failure (issue #21470). The cache is invalidated
// when new keys are saved (Manager.NewHashRatchetKeys), so the gate lifts the
// moment the node acquires keys.
type communityKeyEntitlement struct {
	mu      sync.Mutex
	holds   map[string]bool   // communityID(hex) -> holds any decryption key
	dropped map[string]uint64 // communityID(hex) -> count of gated descriptions
	logged  map[string]bool   // communityID(hex) -> INFO already emitted this session
}

func newCommunityKeyEntitlement() *communityKeyEntitlement {
	return &communityKeyEntitlement{
		holds:   make(map[string]bool),
		dropped: make(map[string]uint64),
		logged:  make(map[string]bool),
	}
}

// known reports the cached entitlement for id and whether it was cached at all.
func (c *communityKeyEntitlement) known(id string) (holds bool, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	holds, ok = c.holds[id]
	return holds, ok
}

// remember stores the entitlement result for id.
func (c *communityKeyEntitlement) remember(id string, holds bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.holds[id] = holds
}

// forget drops the cached entitlement for the given ids, forcing a fresh lookup.
// Called when new keys are saved for a community so the gate re-evaluates and lifts.
func (c *communityKeyEntitlement) forget(ids ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range ids {
		delete(c.holds, id)
	}
}

// forgetAll clears every cached entitlement. Called when new key material arrives
// (which may be community- or channel-scoped) so all gates re-evaluate and lift.
func (c *communityKeyEntitlement) forgetAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.holds = make(map[string]bool)
}

// dropLogDecision tells the caller what (if anything) to log for a gated description.
type dropLogDecision struct {
	logInfo  bool   // first drop for this community this session
	logDebug bool   // periodic checkpoint (every dropLogInterval drops)
	count    uint64 // total gated descriptions for this community this session
}

// recordDrop increments the gated-description counter for id and returns what to
// log: an INFO line on the first drop of the session, then a DEBUG checkpoint
// every dropLogInterval drops. Never logs per envelope.
func (c *communityKeyEntitlement) recordDrop(id string) dropLogDecision {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dropped[id]++
	count := c.dropped[id]
	d := dropLogDecision{count: count}
	switch {
	case !c.logged[id]:
		c.logged[id] = true
		d.logInfo = true
	case count%dropLogInterval == 0:
		d.logDebug = true
	}
	return d
}

// shouldSkipPrivateDataDecryption reports whether the per-key decryption attempts
// over a community description's PrivateData should be skipped. They are skipped
// only when there is encrypted private data to attempt AND the node holds no key
// material for the community — i.e. a keyless spectator, for whom every attempt is
// a guaranteed "no ratchet key" failure.
func shouldSkipPrivateDataDecryption(hasPrivateData, holdsKeys bool) bool {
	return hasPrivateData && !holdsKeys
}
