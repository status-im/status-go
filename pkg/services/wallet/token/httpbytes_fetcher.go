package token

import (
	"context"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/fetcher"

	"github.com/status-im/status-go/internal/metrics/httpbytes"
)

type countingTokenFetcher struct {
	inner fetcher.Fetcher
}

func newCountingTokenFetcher(inner fetcher.Fetcher) fetcher.Fetcher {
	return &countingTokenFetcher{inner: inner}
}

func (f *countingTokenFetcher) Fetch(ctx context.Context, details fetcher.FetchDetails) (fetcher.FetchedData, error) {
	data, err := f.inner.Fetch(ctx, details)
	httpbytes.AddResponseBytes(httpbytes.HostFromURL(data.SourceURL), len(data.JsonData))
	return data, err
}

func (f *countingTokenFetcher) FetchConcurrent(ctx context.Context, details []fetcher.FetchDetails) ([]fetcher.FetchedData, error) {
	data, err := f.inner.FetchConcurrent(ctx, details)
	for _, item := range data {
		httpbytes.AddResponseBytes(httpbytes.HostFromURL(item.SourceURL), len(item.JsonData))
	}
	return data, err
}
