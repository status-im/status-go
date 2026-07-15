package communities

import "sync"

// dropLogInterval is how often (in gated descriptions) a per-community DEBUG
// checkpoint is emitted after the initial INFO line.
const dropLogInterval = 1000

// communityKeyEntitlement caches, per community, whether this node holds any
// hash-ratchet decryption key material. Invalidated when new keys are saved so
// the gate lifts the moment the node acquires keys.
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

func (c *communityKeyEntitlement) known(id string) (holds bool, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	holds, ok = c.holds[id]
	return holds, ok
}

func (c *communityKeyEntitlement) remember(id string, holds bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.holds[id] = holds
}

func (c *communityKeyEntitlement) forget(ids ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range ids {
		delete(c.holds, id)
	}
}

// forgetAll is called when new key material arrives (community- or
// channel-scoped) so all gates re-evaluate.
func (c *communityKeyEntitlement) forgetAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.holds = make(map[string]bool)
}

type dropLogDecision struct {
	logInfo  bool
	logDebug bool
	count    uint64
}

// recordDrop returns what to log for a gated description: INFO on the first
// drop of the session, a DEBUG checkpoint every dropLogInterval after — never
// per envelope.
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

// shouldSkipPrivateDataDecryption skips the per-key decryption attempts over a
// description's PrivateData only when there is something to attempt and the
// node holds no key material for the community.
func shouldSkipPrivateDataDecryption(hasPrivateData, holdsKeys bool) bool {
	return hasPrivateData && !holdsKeys
}
