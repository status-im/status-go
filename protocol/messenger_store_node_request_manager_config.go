package protocol

// Healthy fetches resolve in 1-2 pages; the cap only bounds requests that can
// never terminate on their own (e.g. a token-owned community whose owner
// validation keeps failing).
const maxStoreNodeRequestPageCount = 30

// Community fetches have their own cap knob: a community's content topic
// carries all of its channels' messages, so on a high-traffic community the
// periodically-republished description can sit many pages deep — a cap that is
// too small makes the description unreachable.
const maxCommunityStoreNodeRequestPageCount = 30

type StoreNodeRequestConfig struct {
	WaitForResponse   bool
	StopWhenDataFound bool
	InitialPageSize   uint64
	FurtherPageSize   uint64
	// MaxPageCount caps how many pages a single request may process. A value
	// of 0 disables the cap.
	MaxPageCount int
}

// gateNextPageByCap applies the page ceiling to a pagination decision. It only
// ever turns "fetch next page" into a stop; capTripped distinguishes a
// cap-forced stop from ordinary completion.
func (c StoreNodeRequestConfig) gateNextPageByCap(fetchNext bool, nextPageSize uint64, fetchedPagesCount int) (shouldFetch bool, pageSize uint64, capTripped bool) {
	if fetchNext && c.MaxPageCount > 0 && fetchedPagesCount >= c.MaxPageCount {
		return false, 0, true
	}
	return fetchNext, nextPageSize, false
}

// gateNextPageByValidationQueue stops community pagination once a description
// is already in hand but not yet persisted: for token-owned communities,
// persistence waits on on-chain owner verification, so the pager's "community
// not in DB yet" test keeps requesting pages that cannot help — verification
// retries out-of-band on the owner-verification loop.
//
// queuedClock is the highest clock among descriptions queued for validation
// for this community (0 if none). When the community is already in the DB the
// found path owns the newer-clock decision, so this gate stands down.
func gateNextPageByValidationQueue(communityInDB bool, queuedClock, minimumDataClock uint64) bool {
	if communityInDB {
		return false
	}
	return queuedClock > minimumDataClock
}

// gateNextPageByDescriptionSeen stops community pagination once any description
// for the target community has been processed in this request: store-node
// history is paged newest-first (go-waku HistoryRetriever leaves
// PaginationForward unset = backward), so later pages can only carry older
// descriptions. Disabled when StopWhenDataFound is false — the caller asked for
// a full-window walk. gateTripped distinguishes this stop from ordinary
// completion.
func (c StoreNodeRequestConfig) gateNextPageByDescriptionSeen(fetchNext, descriptionSeen bool) (shouldFetch, gateTripped bool) {
	if fetchNext && c.StopWhenDataFound && descriptionSeen {
		return false, true
	}
	return fetchNext, false
}

// shouldProcessStoreNodePage skips the forced per-page ProcessAllMessages for
// empty pages: it walks the messenger's whole pending backlog, and the pager
// only forces it to flush this request's just-fetched envelopes before the
// GetByID check. The DB check still runs either way.
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

func buildStoreNodeRequestConfig(opts []StoreNodeRequestOption) StoreNodeRequestConfig {
	return applyStoreNodeRequestOptions(defaultStoreNodeRequestConfig(), opts)
}

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
