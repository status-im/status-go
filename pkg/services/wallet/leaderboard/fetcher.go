package leaderboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	netUrl "net/url"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
)

const (
	MARKETS_ENDPOINT = "/v1/leaderboard/markets"
	PRICES_ENDPOINT  = "/v1/leaderboard/prices"

	// convertCurrencyParam asks the market proxy to convert the values it
	// serves (which are cached in USD) to another currency at request time.
	convertCurrencyParam = "convert_currency"
)

// DataFetcher defines the interface for fetching market and price data
type DataFetcher interface {
	// FetchMarkets fetches the full market data
	FetchMarkets(ctx context.Context) error
	// FetchPrices fetches the latest price data
	FetchPrices(ctx context.Context) error
	// StartRefreshLoops starts the data refresh loops
	StartRefreshLoops()
	// RefreshNow makes the running refresh loops fetch at once rather than at
	// their next tick. It is a no-op while they are stopped.
	RefreshNow()
	// Start begins the data refresh loops
	Start(ctx context.Context)
	// Stop halts all data refresh operations
	Stop()
}

// ProxyFetcher implements DataFetcher interface using HTTP proxy
type ProxyFetcher struct {
	client              *thirdparty.HTTPClient
	storage             *DataStorage
	subscriptionManager *SubscriptionManager
	config              ServiceConfig

	// mu guards every piece of mutable fetcher state below it.
	mu sync.Mutex
	// cancelFunc stops the refresh loops; nil while they are not running.
	cancelFunc context.CancelFunc
	// refreshMarkets/refreshPrices carry the "fetch now" trigger to the running
	// loops. They are nil while the loops are stopped.
	refreshMarkets chan struct{}
	refreshPrices  chan struct{}
	// unsupportedCurrencies remembers the currencies the proxy rejected with a
	// 400, so a rejected currency costs one failed request, not one per refresh.
	unsupportedCurrencies map[string]struct{}
}

// NewProxyFetcher creates a new proxy data fetcher
func NewProxyFetcher(config ServiceConfig, storage *DataStorage, subscriptionManager *SubscriptionManager) DataFetcher {
	// Configure HTTP client with detailed timeouts
	httpClient := thirdparty.NewHTTPClient(
		thirdparty.WithTimeout(10*time.Second),
		thirdparty.WithMaxRetries(1),
	)
	return &ProxyFetcher{
		client:                httpClient,
		storage:               storage,
		subscriptionManager:   subscriptionManager,
		config:                config,
		unsupportedCurrencies: make(map[string]struct{}),
	}
}

// Start begins the data refresh loops
func (f *ProxyFetcher) Start(ctx context.Context) {
	go func() {
		defer panics.LogOnPanic()
		<-ctx.Done()
		f.Stop() // gracefully stop if running
	}()
}

// Stop halts all data refresh operations
func (f *ProxyFetcher) Stop() {
	f.releaseLoops()
}

func (f *ProxyFetcher) StartRefreshLoops() {
	ctx, markets, prices, ok := f.claimLoops()
	if !ok {
		return
	}

	go func() {
		defer panics.LogOnPanic()
		f.refreshLoop(ctx, f.config.FullDataInterval, 0, markets, f.FetchMarkets, TickerFullDataUpdateSource)
	}()

	go func() {
		defer panics.LogOnPanic()
		// Prices wait a moment before their first tick, so that a fresh
		// subscription fetches the markets page first.
		f.refreshLoop(ctx, f.config.PriceUpdateInterval, time.Second, prices, f.FetchPrices, TickerPriceUpdateSource)
	}()
}

// RefreshNow makes both loops fetch as soon as they come round, rather than at
// their next tick. The sends are non-blocking against buffered channels, so
// triggers arriving faster than the loops can serve them collapse into one.
func (f *ProxyFetcher) RefreshNow() {
	markets, prices := f.triggers()
	for _, trigger := range []chan struct{}{markets, prices} {
		select {
		case trigger <- struct{}{}:
		default:
		}
	}
}

// claimLoops reserves the refresh loops and hands back what they need to run.
// ok is false when they are already running.
func (f *ProxyFetcher) claimLoops() (ctx context.Context, markets, prices chan struct{}, ok bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.cancelFunc != nil {
		return nil, nil, nil, false
	}

	ctx, f.cancelFunc = context.WithCancel(context.Background())
	f.refreshMarkets = make(chan struct{}, 1)
	f.refreshPrices = make(chan struct{}, 1)

	return ctx, f.refreshMarkets, f.refreshPrices, true
}

// releaseLoops cancels the refresh loops and drops their triggers, so that a
// trigger fired while they are stopped cannot queue a fetch for the next start.
func (f *ProxyFetcher) releaseLoops() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.cancelFunc == nil {
		return
	}
	f.cancelFunc()
	f.cancelFunc = nil
	f.refreshMarkets = nil
	f.refreshPrices = nil
}

// triggers returns the loop triggers, or nils while the loops are stopped.
// A send on a nil channel never proceeds, so the non-blocking selects in
// RefreshNow fall through and the trigger is dropped.
func (f *ProxyFetcher) triggers() (markets, prices chan struct{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.refreshMarkets, f.refreshPrices
}

// refreshLoop drives one endpoint: it fetches on its own tick and whenever it
// is triggered, and reports each success to the subscribers, which is what
// carries the new values out to the client.
func (f *ProxyFetcher) refreshLoop(ctx context.Context, interval, startDelay time.Duration, trigger <-chan struct{}, fetch func(context.Context) error, source int) {
	if startDelay > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(startDelay):
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f.refresh(ctx, fetch, source)
		case <-trigger:
			f.refresh(ctx, fetch, source)
		}
	}
}

func (f *ProxyFetcher) refresh(ctx context.Context, fetch func(context.Context) error, source int) {
	if err := fetch(ctx); err != nil {
		logutils.ZapLogger().Error("Market - error fetching data",
			zap.Int("source", source), zap.Error(err))
		return
	}
	f.subscriptionManager.Emit(ctx, source)
}

// FetchMarkets fetches the full market data
func (f *ProxyFetcher) FetchMarkets(ctx context.Context) error {
	etag := f.storage.GetCryptoEtag()
	// The currency this response is requested in has to travel with it: by the
	// time it lands the user may have selected another one.
	currency := f.storage.GetCurrency()

	body, newEtag, updated := f.fetchData(ctx, MARKETS_ENDPOINT, etag, currency)
	if !updated {
		return nil
	}

	var data CryptoResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	// Store data and etag atomically
	f.storage.UpdateCryptoDataWithEtag(data.Data, newEtag, currency)

	return nil
}

// FetchPrices fetches the latest price data
func (f *ProxyFetcher) FetchPrices(ctx context.Context) error {
	etag := f.storage.GetPriceEtag()
	currency := f.storage.GetCurrency()

	body, newEtag, updated := f.fetchData(ctx, PRICES_ENDPOINT, etag, currency)
	if !updated {
		return nil
	}

	type ProxyPriceData struct {
		CurrentPrice             float64 `json:"price"`
		MarketCap                float64 `json:"market_cap"`
		Volume24h                float64 `json:"volume_24h"`
		PriceChangePercentage24h float64 `json:"percent_change_24h"`
	}

	var tempPriceMap map[string]ProxyPriceData
	if err := json.Unmarshal(body, &tempPriceMap); err != nil {
		return fmt.Errorf("failed to unmarshal price data: %w", err)
	}

	priceData := PriceMap{}
	for key, tempPrice := range tempPriceMap {
		// The key is the cryptocurrency ID (e.g., "bitcoin", "ethereum")
		priceData[key] = PriceData{
			ID:               key,
			Price:            tempPrice.CurrentPrice,
			MarketCap:        tempPrice.MarketCap,
			Volume24h:        tempPrice.Volume24h,
			PercentChange24h: tempPrice.PriceChangePercentage24h,
		}
	}

	// Store data and etag atomically
	f.storage.UpdatePriceDataWithEtag(priceData, newEtag, currency)

	return nil
}

// fetchData requests an endpoint converted to the given display currency.
// If the proxy rejects the currency it retries once without the conversion, so
// an unsupported currency degrades to USD values instead of an empty tab.
func (f *ProxyFetcher) fetchData(ctx context.Context, endpoint string, etag string, currency string) ([]byte, string, bool) {
	convertCurrency := f.convertCurrencyFor(currency)

	body, newEtag, err := f.doFetch(ctx, endpoint, etag, convertCurrency)
	if err != nil && isUnsupportedCurrencyError(err) {
		logutils.ZapLogger().Error("Market - proxy rejected the display currency, falling back to USD values",
			zap.String("endpoint", endpoint),
			zap.String("currency", convertCurrency),
			zap.Error(err))
		f.markCurrencyUnsupported(convertCurrency)
		body, newEtag, err = f.doFetch(ctx, endpoint, etag, "")
	}

	if err != nil || body == nil {
		return nil, newEtag, false
	}
	return body, newEtag, true
}

func (f *ProxyFetcher) doFetch(ctx context.Context, endpoint string, etag string, convertCurrency string) ([]byte, string, error) {
	baseUrl := GetMarketProxyHost(f.config.UrlOverride.Reveal(), f.config.StageName)
	url := f.client.BuildURL(baseUrl, endpoint)

	var params netUrl.Values
	if convertCurrency != "" {
		params = netUrl.Values{convertCurrencyParam: []string{convertCurrency}}
	}

	options := []thirdparty.RequestOption{}

	if f.config.AllowGzip {
		options = append(options, thirdparty.WithGzip())
	}
	if f.config.AllowETag {
		options = append(options, thirdparty.WithEtag(etag))
	}

	options = append(options, thirdparty.WithCredentials(&thirdparty.BasicCreds{
		User:     f.config.User,
		Password: f.config.Password,
	}))

	return f.client.DoGetRequestWithEtag(ctx, url, params, etag, options...)
}

// convertCurrencyFor returns the value to send as convert_currency, or an
// empty string when no conversion is needed or possible: USD is what the proxy
// already caches, and a currency it rejected once is not asked for again.
func (f *ProxyFetcher) convertCurrencyFor(currency string) string {
	currency = normalizeCurrency(currency)
	if currency == DefaultCurrency {
		return ""
	}

	if f.isCurrencyUnsupported(currency) {
		return ""
	}
	return currency
}

func (f *ProxyFetcher) isCurrencyUnsupported(currency string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, unsupported := f.unsupportedCurrencies[currency]
	return unsupported
}

func (f *ProxyFetcher) markCurrencyUnsupported(currency string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.unsupportedCurrencies == nil {
		f.unsupportedCurrencies = make(map[string]struct{})
	}
	f.unsupportedCurrencies[currency] = struct{}{}
}

// isUnsupportedCurrencyError reports whether the proxy answered
// `400 {"error":"unsupported convert_currency: xyz"}`.
func isUnsupportedCurrencyError(err error) bool {
	var statusErr *thirdparty.HTTPStatusError
	if !errors.As(err, &statusErr) {
		return false
	}
	return statusErr.StatusCode == http.StatusBadRequest &&
		strings.Contains(strings.ToLower(statusErr.Body), convertCurrencyParam)
}
