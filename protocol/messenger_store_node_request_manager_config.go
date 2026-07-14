package protocol

import "time"

// communityNotFoundCooldownTTL is how long a community id stays on the
// negative-result cooldown after a store-node request finalized without
// producing a community (issue #21470-hf). The dev curated directory carries
// "dead" ids whose descriptions never arrive; without a cooldown any client-side
// loop (Communities Portal re-requesting per delegate, curated refreshes, …)
// re-issues FetchCommunity for them back-to-back, each run paging to the cap and
// finalizing not-found — a sustained CPU/GC storm that overheats the device.
//
// Trade-off: a user's manual retry within the TTL of a definitive not-found
// no-ops (returns not-found immediately without hitting the network). This is
// acceptable because the fetch would fail identically anyway — nothing on the
// store node changed in two minutes — and a successful fetch or an active
// in-flight request are both unaffected (a live request is joined, not
// suppressed; a success clears the entry immediately).
const communityNotFoundCooldownTTL = 2 * time.Minute

// communityNotFoundCooldown records, per community id, the time its last
// store-node request finalized without producing a community. A fresh request
// for an id whose last nil-result finalize is younger than ttl is suppressed
// (delivered an immediate not-found) instead of spawning another pager. Entries
// expire lazily on read — there is no janitor goroutine.
//
// It carries no lock of its own: the owning StoreNodeRequestManager guards every
// access with its activeRequestsLock (the same lock that guards activeRequests
// and is already held across finalize()), so record/clear/suppress are always
// called under that lock. It must therefore never be copied by value once
// populated; the manager holds it by pointer.
type communityNotFoundCooldown struct {
	ttl     time.Duration
	entries map[string]time.Time
}

func newCommunityNotFoundCooldown(ttl time.Duration) *communityNotFoundCooldown {
	return &communityNotFoundCooldown{
		ttl:     ttl,
		entries: map[string]time.Time{},
	}
}

// recordNilFinalize arms (or re-arms) the cooldown for communityID at now. Called
// when a community request finalizes with a nil community (not-found, cap-tripped
// or queued-stop). Re-arming slides the window to the newest finalize.
func (c *communityNotFoundCooldown) recordNilFinalize(communityID string, now time.Time) {
	c.entries[communityID] = now
}

// clear drops any cooldown for communityID. Called when a community request
// finalizes WITH a community, so a subsequent legitimate re-fetch is never
// suppressed by a stale negative result.
func (c *communityNotFoundCooldown) clear(communityID string) {
	delete(c.entries, communityID)
}

// suppress reports whether a fresh request for communityID should be suppressed,
// and the remaining cooldown (for logging). An entry at or past the TTL does not
// suppress and is evicted on read. A missing entry never suppresses.
func (c *communityNotFoundCooldown) suppress(communityID string, now time.Time) (bool, time.Duration) {
	last, ok := c.entries[communityID]
	if !ok {
		return false, 0
	}
	remaining := c.ttl - now.Sub(last)
	if remaining <= 0 {
		delete(c.entries, communityID)
		return false, 0
	}
	return true, remaining
}

// maxStoreNodeRequestPageCount is a hard ceiling on the number of store-node
// response pages a single storeNodeRequest may process before giving up (issue
// #21470-hf). Normal community/contact fetches resolve in 1-2 pages; this is
// deliberately generous (~15x) so it never affects healthy requests, while
// still bounding a request that can never terminate — e.g. a token-owned
// community whose owner validation keeps failing, which otherwise drains the
// full 31-day store-node window and burns CPU/battery on the device.
const maxStoreNodeRequestPageCount = 30

type StoreNodeRequestConfig struct {
	WaitForResponse   bool
	StopWhenDataFound bool
	InitialPageSize   uint64
	FurtherPageSize   uint64
	// MaxPageCount caps how many pages a single request may process. A value
	// of 0 disables the cap.
	MaxPageCount int
}

// gateNextPageByCap applies the per-request page ceiling to a natural
// pagination decision. It only ever converts a "fetch next page" decision into
// a stop; it never forces additional paging, and it is a no-op both when the
// natural decision is already to stop and when the cap is disabled
// (MaxPageCount <= 0). capTripped reports whether the cap itself forced the
// stop, so the caller can log it distinctly from an ordinary completion.
func (c StoreNodeRequestConfig) gateNextPageByCap(fetchNext bool, nextPageSize uint64, fetchedPagesCount int) (shouldFetch bool, pageSize uint64, capTripped bool) {
	if fetchNext && c.MaxPageCount > 0 && fetchedPagesCount >= c.MaxPageCount {
		return false, 0, true
	}
	return fetchNext, nextPageSize, false
}

// gateNextPageByValidationQueue decides whether store-node pagination for a
// community request should stop because the community's description is already
// in hand and only blocked on owner validation (issue #21470-hf).
//
// For a token-owned community the pager stops when the community is persisted,
// but persistence is withheld until on-chain owner verification succeeds. On a
// transport failure (RPC timeout, dead proxy, rate limit) verification neither
// succeeds nor rejects, so the pager — which only knows "community not in the
// table yet" — wrongly requests another page. A new page cannot revive a dead
// RPC; it only floods the network and starves the very verification calls whose
// success would end the request. Once we hold the description (it is queued for
// validation), fetching more pages is pointless: verification retries out-of-band
// on the owner-verification loop and publishes once it eventually succeeds.
//
// queuedClock is the highest clock among the descriptions queued for validation
// for this community (0 if none are queued). A queued description only supersedes
// further paging when it is newer than the clock we already hold
// (minimumDataClock) — mirroring the "is this description newer" test the pager
// applies when the community IS found. communityInDB short-circuits to false so
// the found path (with its own clock check) remains solely responsible for that
// case.
func gateNextPageByValidationQueue(communityInDB bool, queuedClock, minimumDataClock uint64) bool {
	if communityInDB {
		return false
	}
	return queuedClock > minimumDataClock
}

// gateNextPageByDescriptionSeen decides whether store-node pagination for a
// community request should stop because a description for the target community
// has already been processed in this request (issue #21470-hf).
//
// Store-node history is paged newest-first: the go-waku HistoryRetriever builds
// each StoreQueryRequest with PaginationForward unset (proto3 default false =
// backward) and walks the time window from newest to oldest. So the first
// description encountered for the community is the newest available, and every
// later page can only carry an older one. Once any description has been
// processed, continuing to page cannot yield a newer description; each extra
// page merely re-unmarshals and re-preprocesses the (large, ~1.4MB) description,
// driving a GC storm that overheats the device. This is the flood that remains
// after the page-cap and validation-queue gates: the "community persisted but
// not newer than minimumDataClock" branch keeps paging to the cap and is
// re-issued perpetually as the spectated community's description is re-fetched.
//
// Like the other gates it only ever converts a "fetch next page" decision into a
// stop; it never forces paging. When StopWhenDataFound is false the caller wants
// a full-window walk regardless of what is found (mirroring the success-path
// `!StopWhenDataFound` return), so this gate is disabled and returns the natural
// decision unchanged. gateTripped reports whether this gate forced the stop, so
// the caller can log it distinctly from an ordinary completion.
func (c StoreNodeRequestConfig) gateNextPageByDescriptionSeen(fetchNext, descriptionSeen bool) (shouldFetch, gateTripped bool) {
	if fetchNext && c.StopWhenDataFound && descriptionSeen {
		return false, true
	}
	return fetchNext, false
}

type StoreNodeRequestOption func(*StoreNodeRequestConfig)

func defaultStoreNodeRequestConfig() StoreNodeRequestConfig {
	return StoreNodeRequestConfig{
		WaitForResponse:   true,
		StopWhenDataFound: true,
		InitialPageSize:   initialStoreNodeRequestPageSize,
		FurtherPageSize:   defaultStoreNodeRequestPageSize,
		MaxPageCount:      maxStoreNodeRequestPageCount,
	}
}

func buildStoreNodeRequestConfig(opts []StoreNodeRequestOption) StoreNodeRequestConfig {
	cfg := defaultStoreNodeRequestConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

func WithWaitForResponseOption(waitForResponse bool) StoreNodeRequestOption {
	return func(c *StoreNodeRequestConfig) {
		c.WaitForResponse = waitForResponse
	}
}

func WithStopWhenDataFound(stopWhenDataFound bool) StoreNodeRequestOption {
	return func(c *StoreNodeRequestConfig) {
		c.StopWhenDataFound = stopWhenDataFound
	}
}

func WithInitialPageSize(initialPageSize uint64) StoreNodeRequestOption {
	return func(c *StoreNodeRequestConfig) {
		c.InitialPageSize = initialPageSize
	}
}

func WithFurtherPageSize(furtherPageSize uint64) StoreNodeRequestOption {
	return func(c *StoreNodeRequestConfig) {
		c.FurtherPageSize = furtherPageSize
	}
}

func WithMaxPageCount(maxPageCount int) StoreNodeRequestOption {
	return func(c *StoreNodeRequestConfig) {
		c.MaxPageCount = maxPageCount
	}
}
