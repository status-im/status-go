package protocol

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
