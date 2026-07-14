package protocol

// maxStoreNodeRequestPageCount is a hard ceiling on the number of store-node
// response pages a single storeNodeRequest may process before giving up (issue
// #21470-hf). Normal community/contact fetches resolve in 1-2 pages; this is
// deliberately generous (~15x) so it never affects healthy requests, while
// still bounding a request that can never terminate — e.g. a token-owned
// community whose owner validation keeps failing, which otherwise drains the
// full 31-day store-node window and burns CPU/battery on the device.
const maxStoreNodeRequestPageCount = 30

// maxCommunityStoreNodeRequestPageCount caps community fetches far more tightly
// than the generic contact cap (issue #21470-hf). Store queries page newest-first
// (verified in go-waku history.go: PaginationForward unset => proto3 default false
// => backward) and community descriptions are republished periodically, so a live
// community's description lands within the first pages; the description-seen gate
// (6cd6ced1a) then stops the request on that hit. Deeper paging can therefore only
// ever chew empty/irrelevant history — exactly the device failure mode: dead
// curated ids each cost a ~minutes-long 30-page run finalizing with
// fetchedEnvelopesCount=0. 3 pages keeps a healthy fetch (1-2 pages) untouched
// while collapsing a dead-id run to a fraction of its former cost. Applied only at
// the community construction site; the contact/default cap stays at 30.
const maxCommunityStoreNodeRequestPageCount = 3

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

// shouldProcessStoreNodePage reports whether the forced per-page
// Messenger.ProcessAllMessages should run for a page that carried envelopesCount
// envelopes (issue #21470-hf). ProcessAllMessages walks the messenger's whole
// pending backlog; the store-node pager only forces it to flush THIS request's
// just-fetched envelopes into the DB before the GetByID check. A page with zero
// envelopes has nothing of ours to flush, so processing it only re-walks the
// (possibly fat) backfill queue for no gain — the CPU/GC amplification observed on
// device. Skip it; the subsequent DB check still runs so a description delivered
// by a parallel channel-sync page is not missed.
func shouldProcessStoreNodePage(envelopesCount int) bool {
	return envelopesCount > 0
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

// defaultCommunityStoreNodeRequestConfig is the base config for community fetches.
// It matches the shared default in every field except the page cap, which is
// tightened to maxCommunityStoreNodeRequestPageCount (issue #21470-hf).
func defaultCommunityStoreNodeRequestConfig() StoreNodeRequestConfig {
	cfg := defaultStoreNodeRequestConfig()
	cfg.MaxPageCount = maxCommunityStoreNodeRequestPageCount
	return cfg
}

func applyStoreNodeRequestOptions(cfg StoreNodeRequestConfig, opts []StoreNodeRequestOption) StoreNodeRequestConfig {
	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

// buildStoreNodeRequestConfig builds the config for a contact fetch (the generic
// default, 30-page cap). Caller options are folded on top.
func buildStoreNodeRequestConfig(opts []StoreNodeRequestOption) StoreNodeRequestConfig {
	return applyStoreNodeRequestOptions(defaultStoreNodeRequestConfig(), opts)
}

// buildCommunityStoreNodeRequestConfig builds the config for a community fetch,
// starting from the tighter community default (3-page cap). Caller options are
// still folded on top and can override the cap (issue #21470-hf).
func buildCommunityStoreNodeRequestConfig(opts []StoreNodeRequestOption) StoreNodeRequestConfig {
	return applyStoreNodeRequestOptions(defaultCommunityStoreNodeRequestConfig(), opts)
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
