package communities

import "sync"

// communityKeyEntitlement caches, per community, whether this node holds any
// hash-ratchet decryption key material. Invalidated when new keys are saved so
// the gate lifts the moment the node acquires keys.
type communityKeyEntitlement struct {
	mu     sync.Mutex
	holds  map[string]bool // communityID(hex) -> holds any decryption key
	logged map[string]bool // communityID(hex) -> skip already logged this session
}

func newCommunityKeyEntitlement() *communityKeyEntitlement {
	return &communityKeyEntitlement{
		holds:  make(map[string]bool),
		logged: make(map[string]bool),
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

// forgetAll is called when new key material arrives (community- or
// channel-scoped) so all gates re-evaluate.
func (c *communityKeyEntitlement) forgetAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.holds = make(map[string]bool)
}

// shouldLogDrop reports true once per community per session — the skip fires
// per redelivered description, far too often to log each time.
func (c *communityKeyEntitlement) shouldLogDrop(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.logged[id] {
		return false
	}
	c.logged[id] = true
	return true
}
