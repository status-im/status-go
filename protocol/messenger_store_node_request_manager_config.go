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
