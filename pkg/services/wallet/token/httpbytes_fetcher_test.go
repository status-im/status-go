package token

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/fetcher"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"
	"github.com/stretchr/testify/require"
)

type stubFetcher struct {
	data fetcher.FetchedData
}

func (s stubFetcher) Fetch(context.Context, fetcher.FetchDetails) (fetcher.FetchedData, error) {
	return s.data, nil
}

func (s stubFetcher) FetchConcurrent(context.Context, []fetcher.FetchDetails) ([]fetcher.FetchedData, error) {
	return []fetcher.FetchedData{s.data}, nil
}

func TestCountingTokenFetcherAddsResponseBytes(t *testing.T) {
	const payload = "token-list-json"
	const host = "prod.market.status.im"
	before := httpInBytes(t, host)

	inner := stubFetcher{data: fetcher.FetchedData{
		FetchDetails: fetcher.FetchDetails{
			ListDetails: types.ListDetails{SourceURL: "https://prod.market.status.im/static/lists.json"},
		},
		JsonData: []byte(payload),
	}}
	got, err := newCountingTokenFetcher(inner).Fetch(context.Background(), fetcher.FetchDetails{})
	require.NoError(t, err)
	require.Equal(t, payload, string(got.JsonData))
	require.Equal(t, before+len(payload), httpInBytes(t, host))
}

func httpInBytes(t *testing.T, host string) int {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != "status_http_bytes" {
			continue
		}
		for _, metric := range family.GetMetric() {
			if labelValue(metric, "host") == host && labelValue(metric, "dir") == "in" {
				return int(metric.GetCounter().GetValue())
			}
		}
	}
	return 0
}

func labelValue(metric *dto.Metric, name string) string {
	for _, label := range metric.GetLabel() {
		if label.GetName() == name {
			return label.GetValue()
		}
	}
	return ""
}
