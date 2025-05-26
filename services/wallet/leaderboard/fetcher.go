package leaderboard

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

const (
	MARKETS_ENDPOINT = "/v1/leaderboard/markets"
	PRICES_ENDPOINT  = "/v1/leaderboard/prices"

	// Host suffix for market proxy
	MarketProxyHostSuffix = "market.status.im"
)

// getMarketProxyHost creates market proxy URL based on stage name
// Similar to getProxyHost in api/default_networks.go but for market proxy
func getMarketProxyHost(customUrl, stageName string) string {
	if customUrl != "" {
		return strings.TrimRight(customUrl, "/")
	}
	// For now always use "test" as prod is not deployed yet
	// TODO: Uncomment the line below when prod is deployed
	// return fmt.Sprintf("https://%s.%s", stageName, MarketProxyHostSuffix)
	return fmt.Sprintf("https://test.%s", MarketProxyHostSuffix)
}

// DataFetcher defines the interface for fetching market and price data
type DataFetcher interface {
	// FetchMarkets fetches the full market data
	FetchMarkets(ctx context.Context) error
	// FetchPrices fetches the latest price data
	FetchPrices(ctx context.Context) error
	// StartRefreshLoops starts the data refresh loops
	StartRefreshLoops()
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

	// Background polling state
	contextMutex sync.Mutex
	cancelFunc   context.CancelFunc
}

// NewProxyFetcher creates a new proxy data fetcher
func NewProxyFetcher(config ServiceConfig, storage *DataStorage, subscriptionManager *SubscriptionManager) DataFetcher {
	// Configure HTTP client with detailed timeouts
	httpClient := thirdparty.NewHTTPClient(
		thirdparty.WithTimeout(10*time.Second),
		thirdparty.WithMaxRetries(1),
	)
	return &ProxyFetcher{
		client:              httpClient,
		storage:             storage,
		subscriptionManager: subscriptionManager,
		config:              config,
	}
}

// Start begins the data refresh loops
func (f *ProxyFetcher) Start(ctx context.Context) {
	go func() {
		defer common.LogOnPanic()
		<-ctx.Done()
		f.Stop() // gracefully stop if running
	}()
}

// Stop halts all data refresh operations
func (f *ProxyFetcher) Stop() {
	f.contextMutex.Lock()
	defer f.contextMutex.Unlock()

	// Cancel the context to stop all loops
	if f.cancelFunc != nil {
		f.cancelFunc()
		f.cancelFunc = nil
	}
}

func (f *ProxyFetcher) StartRefreshLoops() {
	f.contextMutex.Lock()
	defer f.contextMutex.Unlock()

	if f.cancelFunc != nil {
		return
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	f.cancelFunc = cancelFunc

	// Start crypto data refresh loop
	go func() {
		defer common.LogOnPanic()
		f.cryptoRefreshLoop(ctx)
	}()

	// Start price data refresh loop
	go func() {
		defer common.LogOnPanic()
		f.priceRefreshLoop(ctx)
	}()
}

// cryptoRefreshLoop periodically fetches the full cryptocurrency data
func (f *ProxyFetcher) cryptoRefreshLoop(ctx context.Context) {
	// Set up ticker for periodic updates
	ticker := time.NewTicker(f.config.FullDataInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return // Context cancelled, stop the loop
		case <-ticker.C:
			if err := f.FetchMarkets(ctx); err != nil {
				logutils.ZapLogger().Error("Error fetching crypto data", zap.Error(err))
			} else {
				f.subscriptionManager.Emit(ctx, TickerFullDataUpdateSource)
			}
		}
	}
}

// priceRefreshLoop periodically fetches price updates
func (f *ProxyFetcher) priceRefreshLoop(ctx context.Context) {
	// Wait a short time before starting price updates
	time.Sleep(1 * time.Second)

	// Set up ticker for periodic updates
	ticker := time.NewTicker(f.config.PriceUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return // Context cancelled, stop the loop
		case <-ticker.C:
			if err := f.FetchPrices(ctx); err != nil {
				logutils.ZapLogger().Error("Error fetching price data", zap.Error(err))
			} else {
				f.subscriptionManager.Emit(ctx, TickerPriceUpdateSource)
			}
		}
	}
}

// FetchMarkets fetches the full market data
func (f *ProxyFetcher) FetchMarkets(ctx context.Context) error {
	etag := f.storage.GetCryptoEtag()

	body, newEtag, updated := f.fetchData(ctx, MARKETS_ENDPOINT, etag)
	if !updated {
		return nil
	}

	var data CryptoResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	// Store data and etag atomically
	f.storage.UpdateCryptoDataWithEtag(data.Data, newEtag)

	return nil
}

// FetchPrices fetches the latest price data
func (f *ProxyFetcher) FetchPrices(ctx context.Context) error {
	etag := f.storage.GetPriceEtag()

	body, newEtag, updated := f.fetchData(ctx, PRICES_ENDPOINT, etag)
	if !updated {
		return nil
	}

	type ProxyPriceData struct {
		ID                       string  `json:"id"`
		CurrentPrice             float64 `json:"price"`
		PriceChangePercentage24h float64 `json:"percent_change_24h"`
	}

	var tempPriceMap map[string]ProxyPriceData

	if err := json.Unmarshal(body, &tempPriceMap); err != nil {
		return err
	}

	priceData := PriceMap{}
	for key, tempPrice := range tempPriceMap {
		priceData[key] = PriceData{
			ID:               tempPrice.ID,
			Price:            tempPrice.CurrentPrice,
			PercentChange24h: tempPrice.PriceChangePercentage24h,
		}
	}

	// Store data and etag atomically
	f.storage.UpdatePriceDataWithEtag(priceData, newEtag)

	return nil
}

func (f *ProxyFetcher) fetchData(ctx context.Context, endpoint string, etag string) ([]byte, string, bool) {
	baseUrl := getMarketProxyHost(f.config.UrlOverride.Reveal(), f.config.StageName)
	url := f.client.BuildURL(baseUrl, endpoint)

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

	body, newEtag, err := f.client.DoGetRequestWithEtag(ctx, url, nil, etag, options...)
	if err != nil || body == nil {
		return nil, newEtag, false
	}
	return body, newEtag, true
}
