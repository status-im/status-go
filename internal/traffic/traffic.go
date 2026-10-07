// Package traffic records the HTTP traffic of status-go's clients with
// go-wallet-sdk's httptraffic, and names its sources after status-go's
// features.
package traffic

import (
	"context"
	"net/http"
	"regexp"

	"github.com/status-im/go-wallet-sdk/pkg/httptraffic"
	"github.com/status-im/go-wallet-sdk/pkg/httptraffic/jsonrpc"
)

const (
	statusGoModule  = "github.com/status-im/status-go/"
	walletSDKModule = "github.com/status-im/go-wallet-sdk/"
)

// The sources features tag their requests with, where they share clients
// that would otherwise hide them.
const (
	Balances          = "Balances"
	MarketPrices      = "Market: prices"
	MarketTokenList   = "Market: token list"
	MarketLeaderboard = "Market: leaderboard"
	Collectibles      = "Collectibles"
	Activity          = "Activity"
)

// attribution names the sources of status-go's untagged requests.
var attribution = httptraffic.Attribution{
	Modules: []string{statusGoModule, walletSDKModule},
	// Packages and functions that carry requests for someone else.
	Plumbing: []string{
		statusGoModule + "internal/traffic.",
		statusGoModule + "internal/rpc",
		statusGoModule + "internal/circuitbreaker.",
		statusGoModule + "internal/panics.",
		statusGoModule + "mobile.",
		statusGoModule + "pkg/backend",
		statusGoModule + "pkg/services/wallet/thirdparty.",
		statusGoModule + "pkg/services/wallet/puzzleauth.",
		statusGoModule + "pkg/services/wallet/async.",
		walletSDKModule + "pkg/ethclient.",
	},
	// The methods of the RPC services the app calls, e.g.
	// wallet.(*API).FetchPrices: entry points, not the feature behind them.
	EntryPoints: regexp.MustCompile(`\.\(\*(Public|Private)?API\)\.`),
	// The first match wins, so more specific rules come first.
	Sources: []httptraffic.SourceRule{
		{Path: "/coins/list", Source: MarketTokenList},
		{Path: "/auth/", Source: "Proxy auth"},
		{Function: "/multistandardbalance.", Source: Balances},
		{Function: "/multistandardfetcher.", Source: Balances},
		{Function: "/multicall.", Source: Balances},
		{Function: "/transferdetector.", Source: "Transfer detection"},
		{Function: "/activityfetcher", Source: Activity},
		{Function: "/activity/", Source: Activity},
		{Function: "/activity.", Source: Activity},
		{Function: "/leaderboard.", Source: MarketLeaderboard},
		{Function: "/market.", Source: MarketPrices},
		{Function: "/coingecko.", Source: MarketPrices},
		{Function: "/currency.", Source: MarketPrices},
		{Function: "/collectibles", Source: Collectibles},
		{Function: "/rarible.", Source: Collectibles},
		{Function: "/router.", Source: "Swap & bridge"},
		{Function: "/pathprocessor.", Source: "Swap & bridge"},
		{Function: "/routeexecution.", Source: "Swap & bridge"},
		{Function: "/paraswap.", Source: "Swap & bridge"},
		{Function: "/lifi.", Source: "Swap & bridge"},
		{Function: "/onramp.", Source: "On-ramp"},
		{Function: "/mercuryo.", Source: "On-ramp"},
		{Function: "/following.", Source: "Following (EFP)"},
		{Function: "/efp.", Source: "Following (EFP)"},
		{Function: "/ens", Source: "ENS"},
		{Function: "/tokens/", Source: "Token lists"},
		{Function: "/token.", Source: "Token lists"},
		{Function: "/decoder/", Source: "Transaction decoding"},
		{Function: "/blockchainstate.", Source: "Chain state"},
		{Function: "/transfer.", Source: "Chain state"},
		{Function: "/connector/", Source: "dApp connector"},
		{Function: "/linkpreview", Source: "Link previews"},
		{Function: "/gif.", Source: "GIFs"},
		{Function: "/images.", Source: "Images"},
		{Function: "/ipfs.", Source: "IPFS"},
		{Function: "/discord.", Source: "Discord import"},
		{Function: "/updates.", Source: "App updates"},
		{Function: "/pairing.", Source: "Device pairing"},
		{Function: "/newsfeed.", Source: "News feed"},
	},
	// They fetch addresses users gave, such as links to preview: where people
	// go is not ours to log.
	PrivateCallers: []string{
		statusGoModule + "pkg/services/linkpreview",
		statusGoModule + "internal/images.",
		statusGoModule + "internal/protocol/discord.",
	},
	Unattributed: "Other",
}

// Default records the traffic of status-go's HTTP clients. It starts disabled:
// the client turns it on where it wants the stats, as it costs work on every
// request.
var Default = func() *httptraffic.Recorder {
	r := httptraffic.NewRecorder(
		httptraffic.WithAttribution(attribution),
		httptraffic.WithInspector(jsonrpc.Inspector{}),
	)
	r.SetEnabled(false)
	return r
}()

// Transport is a clone of http.DefaultTransport recorded by Default. Clients
// that would use http.DefaultTransport share this one, so they keep one pool
// of connections among themselves; it is separate from http.DefaultTransport.
var Transport = Default.Instrument(nil)

// PrivateTransport records requests like Transport, but not where they go:
// for addresses users gave, such as their own RPC providers.
var PrivateTransport http.RoundTripper = privateTransport{Transport}

type privateTransport struct{ next http.RoundTripper }

func (t privateTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.next.RoundTrip(req.WithContext(httptraffic.WithPrivateDestination(req.Context())))
}

// WithSource tags the requests made with ctx as coming from source.
func WithSource(ctx context.Context, source string) context.Context {
	return httptraffic.WithSource(ctx, source)
}
