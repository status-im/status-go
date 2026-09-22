package relay

import (
	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
)

const (
	MainnetBaseURL = "https://api.relay.link"
	TestnetBaseURL = "https://api.testnets.relay.link"

	ReferrerDev  = "status-app"
	ReferrerProd = "status-app-prod"

	// prodStageName matches WalletConfig.StatusProxyStageName as set by release builds.
	prodStageName = "prod"

	apiKeyHeader = "x-api-key"

	appFeeBps = "65" // 0.65%
)

func ReferrerForStage(stageName string) string {
	if stageName == prodStageName {
		return ReferrerProd
	}
	return ReferrerDev
}

type Client struct {
	httpClient *thirdparty.HTTPClient
	baseURL    string
	// fixedBaseURL, when set, is used regardless of the chain (tests, proxies).
	fixedBaseURL string
	chainID      uint64
	referrer     string
	apiKey       string
}

func NewClient(chainID uint64, referrer string, apiKey string) *Client {
	return NewClientWithBaseURL("", chainID, referrer, apiKey)
}

// NewClientWithBaseURL is NewClient with a fixed API host; an empty baseURL selects the
// mainnet/testnet host from the chain.
func NewClientWithBaseURL(baseURL string, chainID uint64, referrer string, apiKey string) *Client {
	c := &Client{
		httpClient:   thirdparty.NewHTTPClient(),
		fixedBaseURL: baseURL,
		referrer:     referrer,
		apiKey:       apiKey,
	}
	c.SetChainID(chainID)
	return c
}

// SetChainID selects the chain the next quote originates from and, with it, the mainnet or
// testnet API host (unless a fixed host was given).
func (c *Client) SetChainID(chainID uint64) {
	c.chainID = chainID
	switch {
	case c.fixedBaseURL != "":
		c.baseURL = c.fixedBaseURL
	case walletcommon.SupportedTestNetworks[chainID]:
		c.baseURL = TestnetBaseURL
	default:
		c.baseURL = MainnetBaseURL
	}
}

// BaseURL returns the API host requests currently go to.
func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) requestOptions() []thirdparty.RequestOption {
	options := []thirdparty.RequestOption{}
	if c.apiKey != "" {
		options = append(options, thirdparty.WithHeader(apiKeyHeader, c.apiKey))
	}
	return options
}
