package traffic

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/go-wallet-sdk/pkg/httptraffic"
)

func TestAttribution_SkipsPlumbingAndRPCEntryPoints(t *testing.T) {
	for function, want := range map[string]bool{
		statusGoModule + "pkg/services/wallet/multistandardbalance.(*Controller).fetchChain.func1": true,
		walletSDKModule + "pkg/multistandardfetcher.FetchBalances.func2":                           true,
		statusGoModule + "pkg/services/wallet/market.(*Manager).FetchPrices":                       true,
		statusGoModule + "pkg/services/wallet/thirdparty/market/coingecko.(*Client).fetchTokens":   true,
		statusGoModule + "pkg/services/wallet/thirdparty.(*HTTPClient).doGetRequest":               false,
		statusGoModule + "pkg/services/wallet.(*API).FetchPrices":                                  false,
		statusGoModule + "internal/rpc/chain.(*ClientWithFallback).CallContext":                    false,
		statusGoModule + "internal/traffic.WithSource":                                             false,
		statusGoModule + "mobile.CallPrivateRPC":                                                   false,
		walletSDKModule + "pkg/httptraffic.(*roundTripper).RoundTrip":                              false,
		"github.com/ethereum/go-ethereum/rpc.(*Client).CallContext":                                false,
		"net/http.(*Client).Do": false,
	} {
		require.Equal(t, want, attribution.IsFeature(function), function)
	}
}

func TestAttribution_NamesStatusGoFeatures(t *testing.T) {
	sg := statusGoModule + "pkg/services/"
	for _, c := range []struct{ function, path, want string }{
		{sg + "wallet/multistandardbalance.(*Controller).fetchChain", "/ethereum/mainnet/#eth_call", Balances},
		{walletSDKModule + "pkg/balance/multistandardfetcher.FetchBalances.func1", "/ethereum/mainnet/#eth_call", Balances},
		{sg + "wallet/market.(*Manager).FetchPrices", "/v1/coins/list", MarketTokenList},
		{sg + "wallet/market.(*Manager).FetchPrices", "/v1/simple/price", MarketPrices},
		{sg + "wallet/thirdparty/activity/alchemy.(*Client).FetchTransfers", "/v2/transfers", Activity},
		{sg + "wallet/thirdparty/collectibles/alchemy.(*Client).FetchOwned", "/nft/v3/getNFTsForOwner", Collectibles},
		{walletSDKModule + "pkg/tokens/fetcher.(*fetcher).FetchConcurrent.func1", "/static/lists.json", "Token lists"},
		{sg + "wallet/thirdparty/decoder/fourbyte.(*Client).Run", "/api/v1/signatures/", "Transaction decoding"},
		{sg + "linkpreview/unfurlers.(*OEmbedUnfurler).Unfurl", "/oembed", "Link previews"},
		{sg + "newsfeed.(*Service).Start", "/desktop-news/rss/v2", "News feed"},
		{"", "/auth/solve", "Proxy auth"},
		{"", "/x", "Other"},
		{sg + "stickers.(*API).fetch", "/stickers", "stickers"},
	} {
		require.Equal(t, c.want, attribution.SourceOf(c.function, c.path), c.function)
	}
}

func TestAttribution_KeepsNoPathOfAddressesUsersGave(t *testing.T) {
	for _, caller := range []string{
		statusGoModule + "pkg/services/linkpreview.(*Unfurler).Unfurl",
		statusGoModule + "internal/images.DownloadImage",
		statusGoModule + "internal/protocol/discord.DownloadAsset",
	} {
		require.True(t, attribution.IsPrivate(caller), caller)
	}
	require.False(t, attribution.IsPrivate(statusGoModule+"pkg/services/wallet/thirdparty/market/coingecko.(*Client).fetchTokens"))
}

func TestPrivateTransport_RecordsNeitherHostNorPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	Default.SetEnabled(true)
	Default.Reset()
	defer func() {
		Default.SetEnabled(false)
		Default.Reset()
	}()

	req, err := http.NewRequestWithContext(WithSource(context.Background(), "RPC URL check"), http.MethodGet, server.URL+"/v2/my-key", nil)
	require.NoError(t, err)
	resp, err := (&http.Client{Transport: PrivateTransport}).Do(req)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	require.NoError(t, resp.Body.Close())

	s := Default.Snapshot()
	require.Len(t, s.Endpoints, 1)
	require.Equal(t, "RPC URL check", s.Endpoints[0].Source, "the bytes still count under their source")
	require.NotContains(t, s.Endpoints[0].Host, "127.0.0.1")
	require.NotContains(t, s.Endpoints[0].Path, "my-key")
	for _, h := range s.Hosts {
		require.NotContains(t, h.Host, "127.0.0.1")
	}
}

func TestDefault_StartsDisabled(t *testing.T) {
	require.False(t, Default.Enabled(), "the shared recorder waits for the client to turn it on")
}

func TestSampler_SamplesUntilStopped(t *testing.T) {
	rec := httptraffic.NewRecorder()
	s := NewSampler(rec, 10*time.Millisecond, 1000, zap.NewNop())
	s.Start()
	s.Start()
	require.Eventually(t, func() bool { return len(rec.Snapshot().Series) >= 2 }, 2*time.Second, 5*time.Millisecond)
	s.Stop()
	s.Stop()

	n := len(rec.Snapshot().Series)
	time.Sleep(40 * time.Millisecond)
	require.Equal(t, n, len(rec.Snapshot().Series), "a stopped sampler takes no samples")
}
