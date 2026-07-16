package protocol

import "time"

const (
	// storeNodeRequestMaxPages bounds how many pages a single request may drain.
	// Store queries are newest-first, so the first pages carry the latest data.
	storeNodeRequestMaxPages = 2

	storeNodeRequestPageSize = 25

	communityStoreNodeFetchWindow = 7 * oneDayDuration
	contactStoreNodeFetchWindow   = oneMonthDuration
)

// storeNodeRequestOutcome describes how a store node request ended.
type storeNodeRequestOutcome int

const (
	// storeNodeRequestUnresolved means nothing usable was fetched, or the fetched
	// data could not be persisted (e.g. owner verification failure / missing keys).
	storeNodeRequestUnresolved storeNodeRequestOutcome = iota
	// storeNodeRequestResolved means newer data was fetched and persisted.
	storeNodeRequestResolved
	// storeNodeRequestAlreadyUpToDate means the newest fetched data was not newer
	// than the locally stored one. This is a success, not a failure.
	storeNodeRequestAlreadyUpToDate
)

type StoreNodeRequestConfig struct {
	WaitForResponse   bool
	StopWhenDataFound bool
	InitialPageSize   uint64
	FurtherPageSize   uint64
	MaxPages          uint64
	FetchWindow       time.Duration
}

type StoreNodeRequestOption func(*StoreNodeRequestConfig)

func defaultStoreNodeRequestConfig(requestType storeNodeRequestType) StoreNodeRequestConfig {
	fetchWindow := communityStoreNodeFetchWindow
	if requestType == storeNodeContactRequest {
		fetchWindow = contactStoreNodeFetchWindow
	}

	return StoreNodeRequestConfig{
		WaitForResponse:   true,
		StopWhenDataFound: true,
		InitialPageSize:   storeNodeRequestPageSize,
		FurtherPageSize:   storeNodeRequestPageSize,
		MaxPages:          storeNodeRequestMaxPages,
		FetchWindow:       fetchWindow,
	}
}

func buildStoreNodeRequestConfig(requestType storeNodeRequestType, opts []StoreNodeRequestOption) StoreNodeRequestConfig {
	cfg := defaultStoreNodeRequestConfig(requestType)

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
