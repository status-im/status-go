package coingecko

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/common"

	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"

	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"

	"github.com/stretchr/testify/require"
)

const testCoinsList = `[
{"id":"usd-coin","symbol":"usdc","name":"USDC","platforms":{"ethereum":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"}},
{"id":"status","symbol":"snt","name":"Status","platforms":{"ethereum":"0x744d70fdbe2ba4cf95131626614a1763df805b9e"}}
]`

type fakeMarketServer struct {
	srv *httptest.Server

	coinsListRequests atomic.Int32
	notModified       atomic.Int32
	failCoinsList     atomic.Bool
	etag              string
}

func newFakeMarketServer(t *testing.T) *fakeMarketServer {
	f := &fakeMarketServer{etag: `W/"coins-list-v1"`}
	mux := http.NewServeMux()
	mux.HandleFunc("/coins/list", func(w http.ResponseWriter, r *http.Request) {
		f.coinsListRequests.Add(1)
		if f.failCoinsList.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if r.Header.Get("If-None-Match") == f.etag {
			f.notModified.Add(1)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", f.etag)
		_, _ = w.Write([]byte(testCoinsList))
	})
	mux.HandleFunc("/simple/price", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"usd-coin":{"usd":1.001},"status":{"usd":0.02}}`))
	})
	mux.HandleFunc("/coins/markets", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"usd-coin","market_cap":1},{"id":"status","market_cap":2}]`))
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func newTestClient(f *fakeMarketServer) *Client {
	return &Client{
		httpClient: thirdparty.NewHTTPClient(thirdparty.WithMaxRetries(1)),
		baseURL:    f.srv.URL,
	}
}

func testMarketTokens() []*tokentypes.Token {
	return []*tokentypes.Token{
		{Token: &types.Token{Symbol: "USDC", ChainID: common.EthereumMainnet, Address: gethcommon.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")}},
		{Token: &types.Token{Symbol: "SNT", ChainID: common.EthereumMainnet, Address: gethcommon.HexToAddress("0x744d70fdbe2ba4cf95131626614a1763df805b9e")}},
	}
}

func TestCoinsListFetchedOnceForBurstOfPriceAndMarketCalls(t *testing.T) {
	f := newFakeMarketServer(t)
	client := newTestClient(f)
	tokens := testMarketTokens()

	const burst = 20
	var wg sync.WaitGroup
	errs := make(chan error, 2*burst)
	for i := 0; i < burst; i++ {
		wg.Add(2)
		go func() {
			defer panics.LogOnPanic()
			defer wg.Done()
			prices, err := client.FetchPrices(tokens, []string{"usd"})
			if err == nil && prices[tokens[0].Key()]["usd"] != 1.001 {
				err = errUnexpected
			}
			errs <- err
		}()
		go func() {
			defer panics.LogOnPanic()
			defer wg.Done()
			values, err := client.FetchTokenMarketValues(tokens, "usd")
			if err == nil && values[tokens[1].Key()].MKTCAP != 2 {
				err = errUnexpected
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	_, err := client.FetchPrices(tokens, []string{"usd"})
	require.NoError(t, err)

	require.Equal(t, int32(1), f.coinsListRequests.Load())
}

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newClientWithClock(f *fakeMarketServer) (*Client, *fakeClock) {
	clock := &fakeClock{t: time.Unix(1_800_000_000, 0)}
	client := newTestClient(f)
	client.coinsList.now = clock.now
	return client, clock
}

func TestCoinsListRevalidatedWithEtagAfterTTL(t *testing.T) {
	f := newFakeMarketServer(t)
	client, clock := newClientWithClock(f)
	tokens := testMarketTokens()

	_, err := client.FetchPrices(tokens, []string{"usd"})
	require.NoError(t, err)

	clock.advance(coinsListTTL - time.Second)
	_, err = client.FetchPrices(tokens, []string{"usd"})
	require.NoError(t, err)
	require.Equal(t, int32(1), f.coinsListRequests.Load())

	clock.advance(2 * time.Second)
	prices, err := client.FetchPrices(tokens, []string{"usd"})
	require.NoError(t, err)
	require.Equal(t, 1.001, prices[tokens[0].Key()]["usd"])
	require.Equal(t, int32(2), f.coinsListRequests.Load())
	require.Equal(t, int32(1), f.notModified.Load())

	clock.advance(coinsListTTL - time.Second)
	_, err = client.FetchPrices(tokens, []string{"usd"})
	require.NoError(t, err)
	require.Equal(t, int32(2), f.coinsListRequests.Load())
}

func TestCoinsListKeepsStaleMappingWhenRefreshFails(t *testing.T) {
	f := newFakeMarketServer(t)
	client, clock := newClientWithClock(f)
	tokens := testMarketTokens()

	_, err := client.FetchPrices(tokens, []string{"usd"})
	require.NoError(t, err)

	f.failCoinsList.Store(true)
	clock.advance(coinsListTTL + time.Second)
	prices, err := client.FetchPrices(tokens, []string{"usd"})
	require.NoError(t, err)
	require.Equal(t, 1.001, prices[tokens[0].Key()]["usd"])
	require.Equal(t, int32(2), f.coinsListRequests.Load())

	clock.advance(coinsListRetryDelay - time.Second)
	_, err = client.FetchPrices(tokens, []string{"usd"})
	require.NoError(t, err)
	require.Equal(t, int32(2), f.coinsListRequests.Load())

	f.failCoinsList.Store(false)
	clock.advance(2 * time.Second)
	_, err = client.FetchPrices(tokens, []string{"usd"})
	require.NoError(t, err)
	require.Equal(t, int32(3), f.coinsListRequests.Load())
}

func TestCoinsListFailureWithoutCacheIsRetriedOnNextCall(t *testing.T) {
	f := newFakeMarketServer(t)
	client := newTestClient(f)
	tokens := testMarketTokens()

	f.failCoinsList.Store(true)
	_, err := client.FetchPrices(tokens, []string{"usd"})
	require.Error(t, err)

	f.failCoinsList.Store(false)
	_, err = client.FetchPrices(tokens, []string{"usd"})
	require.NoError(t, err)
	require.Equal(t, int32(2), f.coinsListRequests.Load())
}

var errUnexpected = &thirdparty.HTTPStatusError{StatusCode: -1, Body: "unexpected market data"}
