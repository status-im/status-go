package cryptocompare

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
)

func TestIDs(t *testing.T) {
	stdClient := NewClient()
	require.Equal(t, baseID, stdClient.ID())

	clientWithParams := NewClientWithParams(Params{
		ID: "testID",
	})
	require.Equal(t, "testID", clientWithParams.ID())
}

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		path     string
		expected string
	}{
		{
			name:     "base URL without trailing slash, path without leading slash",
			baseURL:  "https://example.com",
			path:     "api/v1/endpoint",
			expected: "https://example.com/api/v1/endpoint",
		},
		{
			name:     "base URL with trailing slash, path without leading slash",
			baseURL:  "https://example.com/",
			path:     "api/v1/endpoint",
			expected: "https://example.com/api/v1/endpoint",
		},
		{
			name:     "base URL without trailing slash, path with leading slash",
			baseURL:  "https://example.com",
			path:     "/api/v1/endpoint",
			expected: "https://example.com/api/v1/endpoint",
		},
		{
			name:     "base URL with trailing slash, path with leading slash",
			baseURL:  "https://example.com/",
			path:     "/api/v1/endpoint",
			expected: "https://example.com/api/v1/endpoint",
		},
		{
			name:     "base URL with multiple trailing slashes",
			baseURL:  "https://example.com///",
			path:     "api/v1/endpoint",
			expected: "https://example.com/api/v1/endpoint",
		},
		{
			name:     "path with multiple leading slashes",
			baseURL:  "https://example.com",
			path:     "///api/v1/endpoint",
			expected: "https://example.com/api/v1/endpoint",
		},
		{
			name:     "empty path",
			baseURL:  "https://example.com",
			path:     "",
			expected: "https://example.com/",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the NewClientWithParams behavior which trims trailing slashes
			baseURL := strings.TrimSuffix(tc.baseURL, "/")
			client := &Client{baseURL: baseURL}

			result := client.buildURL(tc.path)
			require.Equal(t, tc.expected, result)
		})
	}
}

var testTokens = []*tokentypes.Token{
	{
		Token: &types.Token{
			Name:    "USDC",
			Symbol:  "USDC",
			ChainID: common.EthereumMainnet,
			Address: gethcommon.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
		},
	},
	{
		Token: &types.Token{
			Name:    "USDC",
			Symbol:  "USDC",
			ChainID: common.OptimismMainnet,
			Address: gethcommon.HexToAddress("0x0b2c639c533813f4aa9d7837caf62653d097ff85"),
		},
	},
	{
		Token: &types.Token{
			Name:    "Status",
			Symbol:  "SNT",
			ChainID: common.EthereumMainnet,
			Address: gethcommon.HexToAddress("0x744d70fdbe2ba4cf95131626614a1763df805b9e"),
		},
	},
	{
		Token: &types.Token{
			Name:    "Dai",
			Symbol:  "DAI",
			ChainID: common.EthereumMainnet,
			Address: gethcommon.HexToAddress("0x6b175474e89094c44da98b954eedeac495271d0f"),
		},
	},
	{
		Token: &types.Token{
			Name:    "Ethereum",
			Symbol:  "ETH",
			ChainID: common.EthereumMainnet,
			Address: gethcommon.HexToAddress("0x0000000000000000000000000000000000000000"),
		},
	},
	{
		Token: &types.Token{
			Name:    "Ethereum",
			Symbol:  "ETH",
			ChainID: common.OptimismMainnet,
			Address: gethcommon.HexToAddress("0x0000000000000000000000000000000000000000"),
		},
	},
}

func TestFetchPrices(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/data/pricemulti", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		response := `{"USDC":{"USD":0.9999},"SNT":{"USD":0.02605},"ETH":{"USD":4528.86}}`
		_, _ = w.Write([]byte(response))
	})

	srv := httptest.NewServer(mux)

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	prices, err := geckoClient.FetchPrices(testTokens, []string{"USD"})
	require.NoError(t, err)
	require.NotNil(t, prices)
	require.Len(t, prices, len(testTokens))

	require.Equal(t, map[string]float64{"USD": 0.9999}, prices[testTokens[0].Key()])
	require.Equal(t, map[string]float64{"USD": 0.9999}, prices[testTokens[1].Key()])
	require.Equal(t, map[string]float64{"USD": 0.02605}, prices[testTokens[2].Key()])
	require.Equal(t, map[string]float64{"USD": 0}, prices[testTokens[3].Key()])
	require.Equal(t, map[string]float64{"USD": 4528.86}, prices[testTokens[4].Key()])
	require.Equal(t, map[string]float64{"USD": 4528.86}, prices[testTokens[5].Key()])
}

func TestFetchTokenDetails(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/data/all/coinlist", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		response := `{
"Response": "Success",
"Message": "Coin list successfully returned!",
	"Data": {
		"USDC": {
				"Id": "925809",
				"Url": "/coins/usdc/overview",
				"ImageUrl": "/media/34835941/usdc.png",
				"ContentCreatedOn": 1538573358,
				"Name": "USDC",
				"Symbol": "USDC",
				"CoinName": "USD Coin",
				"FullName": "USD Coin (USDC)",
				"Description": "Who Created USDC?The cryptocurrency is an open-source project that anyone can view and contribute to and is managed by the Centre consortium, which was co-founded by fintech firm Circle and Nasdaq-listed cryptocurrency exchange Coinbase.Accounting firm Grant Thornton oversees the segregated accounts with regulated U.S. financial institutions that hold the cryptocurrency’s reserves, held in dollars and dollar-denominated assets. In USDC’s case, these dollar-denominated assets are short-term U.S. Treasury securities.How Does USDC Remain at $1!?Because USDC is a fully collateralized stablecoin backed by dollar-denominated assets and allows token holders to redeem USDC tokens for dollars, it can almost be seen as a digital version of the U.S. dollar.Investors can initiate a transaction to buy USDC using fiat currency, with the fiat currency they send over being deposited at a U.S. financial institution while USDC tokens in the same nominal value are minted. If the USDC is redeemed for the fiat currency, the tokens are burned and the dollars are transferred to investors’ bank accounts, according to USDC’s whitepaper. What is USDC Used For?USDC is a widely used stablecoin being adopted throughout the cryptocurrency market as it competes with the leading stablecoin USDT. Some of the cryptocurrency’s use cases include:Hedging against volatilityStable price-peggingRemittancesCrowdfundingPayments for products and servicesLending, borrowing, and other financial servicesBecause USDC is a blockchain-based digital currency, it doesn’t require a bank account, users don’t need to be in a specific location or have an account with a specific institution to use it. Moreover, it isn’t restricted by banking hours or borders.The cryptocurrency is available on a number of blockchains, including Ethereum, Algorand, BNB Chain, Polygon Avalanche, Cronos, Solana, Stellar, and TRON. It’s widely used in the decentralized finance (DeFi) space.GitHub | Medium",
				"AssetTokenStatus": "N/A",
				"Algorithm": "N/A",
				"ProofType": "N/A",
				"SortOrder": "3382",
				"Sponsored": false,
				"Taxonomy": {
						"Access": "Permissioned",
						"FCA": "Exchange,Asset",
						"FINMA": "Payment,Asset",
						"Industry": "Financial and Insurance Activities",
						"CollateralizedAsset": "Yes",
						"CollateralizedAssetType": "Stablecoin",
						"CollateralType": "Currency",
						"CollateralInfo": ""
				},
				"Rating": {
						"Weiss": {
								"Rating": "",
								"TechnologyAdoptionRating": "",
								"MarketPerformanceRating": ""
						}
				},
				"IsTrading": true
		},
		"SNT": {
				"Id": "137013",
				"Url": "/coins/snt/overview",
				"ImageUrl": "/media/37747590/snt.png",
				"ContentCreatedOn": 1497962382,
				"Name": "SNT",
				"Symbol": "SNT",
				"CoinName": "Status Network Token",
				"FullName": "Status Network Token (SNT)",
				"Description": "Status is an open source messaging platform and mobile browser that allows users to interact with decentralized applications (dApps) that run on the Ethereum Network.In Status, users own and control their own data, wealth and digital identity. The Status Network Token (&#39;SNT&#39;) is an Ethereum-based token that is required to interact with the Status Network.Status strives to be a secure communication tool that upholds human rights. Designed to enable the free flow of information, protect the right to private, secure conversations, and promote the sovereignty of individuals.Discord | YouTube | Instagram | Facebook | WeiboWhitepaper",
				"AssetTokenStatus": "Finished",
				"Algorithm": "N/A",
				"ProofType": "N/A",
				"SortOrder": "1264",
				"Sponsored": false,
				"Taxonomy": {
						"Access": "Permissionless",
						"FCA": "Utility",
						"FINMA": "Utility",
						"Industry": "Arts, Entertainment and Recreation",
						"CollateralizedAsset": "No",
						"CollateralizedAssetType": "",
						"CollateralType": "",
						"CollateralInfo": ""
				},
				"Rating": {
						"Weiss": {
								"Rating": "C",
								"TechnologyAdoptionRating": "C",
								"MarketPerformanceRating": "D-"
						}
				},
				"IsTrading": true
		}
	}
}`
		_, _ = w.Write([]byte(response))
	})

	srv := httptest.NewServer(mux)

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	details, err := geckoClient.FetchTokenDetails(testTokens)
	require.NoError(t, err)
	require.NotNil(t, details)
	require.Len(t, details, len(testTokens))

	received := details[testTokens[0].Key()]
	require.Equal(t, "925809", received.ID)
	require.Equal(t, "USDC", received.Name)
	require.Equal(t, "USDC", received.Symbol)
	require.Equal(t, float64(0), received.TotalCoinsMined)

	received = details[testTokens[1].Key()]
	require.Equal(t, "925809", received.ID)
	require.Equal(t, "USDC", received.Name)
	require.Equal(t, "USDC", received.Symbol)
	require.Equal(t, float64(0), received.TotalCoinsMined)

	received = details[testTokens[2].Key()]
	require.Equal(t, "137013", received.ID)
	require.Equal(t, "SNT", received.Name)
	require.Equal(t, "SNT", received.Symbol)
	require.Equal(t, float64(0), received.TotalCoinsMined)

	received = details[testTokens[3].Key()]
	require.Equal(t, thirdparty.TokenDetails{}, received)
	received = details[testTokens[4].Key()]
	require.Equal(t, thirdparty.TokenDetails{}, received)
	received = details[testTokens[5].Key()]
	require.Equal(t, thirdparty.TokenDetails{}, received)
}

func TestFetchTokenMarketValues(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/data/pricemultifull", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		response := `{
	"RAW": {
			"ETH": {
					"USD": {
							"TYPE": "5",
							"MARKET": "CCCAGG",
							"FROMSYMBOL": "ETH",
							"TOSYMBOL": "USD",
							"FLAGS": "2",
							"LASTMARKET": "CCCAGG",
							"MEDIAN": 4516.14416425753,
							"TOPTIERVOLUME24HOUR": 410168.95285623,
							"TOPTIERVOLUME24HOURTO": 1825127962.69741,
							"LASTTRADEID": "754495303",
							"PRICE": 4516.14416425753,
							"LASTUPDATE": 1757667157,
							"LASTVOLUME": 0.07697372,
							"LASTVOLUMETO": 347.605622148,
							"VOLUMEHOUR": 6657.95654658,
							"VOLUMEHOURTO": 30120608.7948991,
							"OPENHOUR": 4527.26272975992,
							"HIGHHOUR": 4534.8406102931,
							"LOWHOUR": 4514.93155765281,
							"VOLUMEDAY": 109541.21454776,
							"VOLUMEDAYTO": 494825176.02217,
							"OPENDAY": 4461.10754887017,
							"HIGHDAY": 4564.28612417753,
							"LOWDAY": 4452.72268362409,
							"VOLUME24HOUR": 410168.95285623,
							"VOLUME24HOURTO": 1825127962.69741,
							"OPEN24HOUR": 4432.7608628562,
							"HIGH24HOUR": 4564.28612417753,
							"LOW24HOUR": 4373.18427790377,
							"CHANGE24HOUR": 83.38330140132985,
							"CHANGEPCT24HOUR": 1.8810692473855388,
							"CHANGEDAY": 55.03661538736014,
							"CHANGEPCTDAY": 1.2336984657834318,
							"CHANGEHOUR": -11.118565502390084,
							"CHANGEPCTHOUR": -0.24559134660558343,
							"CONVERSIONTYPE": "direct",
							"CONVERSIONSYMBOL": "",
							"CONVERSIONLASTUPDATE": 1757667157,
							"SUPPLY": 120704779.826411,
							"MKTCAP": 545120187011.0361,
							"MKTCAPPENALTY": 0,
							"CIRCULATINGSUPPLY": 120704779.826411,
							"CIRCULATINGSUPPLYMKTCAP": 545120187011.0361,
							"TOTALVOLUME24H": 4716921.79345383,
							"TOTALVOLUME24HTO": 21275044670.661804,
							"TOTALTOPTIERVOLUME24H": 2809715.26861036,
							"TOTALTOPTIERVOLUME24HTO": 12661825053.456081,
							"IMAGEURL": "/media/37746238/eth.png"
					}
			}
	}
}`
		_, _ = w.Write([]byte(response))
	})

	srv := httptest.NewServer(mux)

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	marketValues, err := geckoClient.FetchTokenMarketValues(testTokens, "USD")
	require.NoError(t, err)
	require.NotNil(t, marketValues)
	require.Len(t, marketValues, len(testTokens))

	require.Equal(t, thirdparty.TokenMarketValues{}, marketValues[testTokens[0].Key()])
	require.Equal(t, thirdparty.TokenMarketValues{}, marketValues[testTokens[1].Key()])
	require.Equal(t, thirdparty.TokenMarketValues{}, marketValues[testTokens[2].Key()])
	require.Equal(t, thirdparty.TokenMarketValues{}, marketValues[testTokens[3].Key()])

	ethereumValues := thirdparty.TokenMarketValues{
		MKTCAP:          5.451201870110361e+11,
		HIGHDAY:         4564.28612417753,
		LOWDAY:          4452.72268362409,
		CHANGEPCTHOUR:   -0.24559134660558343,
		CHANGEPCTDAY:    1.2336984657834318,
		CHANGEPCT24HOUR: 1.8810692473855388,
		CHANGE24HOUR:    83.38330140132985,
	}

	for _, index := range []int{4, 5} {
		require.InDelta(t, ethereumValues.MKTCAP, marketValues[testTokens[index].Key()].MKTCAP, 1e-10)
		require.InDelta(t, ethereumValues.HIGHDAY, marketValues[testTokens[index].Key()].HIGHDAY, 1e-10)
		require.InDelta(t, ethereumValues.LOWDAY, marketValues[testTokens[index].Key()].LOWDAY, 1e-10)
		require.InDelta(t, ethereumValues.CHANGEPCTHOUR, marketValues[testTokens[index].Key()].CHANGEPCTHOUR, 1e-10)
		require.InDelta(t, ethereumValues.CHANGEPCTDAY, marketValues[testTokens[index].Key()].CHANGEPCTDAY, 1e-10)
		require.InDelta(t, ethereumValues.CHANGEPCT24HOUR, marketValues[testTokens[index].Key()].CHANGEPCT24HOUR, 1e-10)
		require.InDelta(t, ethereumValues.CHANGE24HOUR, marketValues[testTokens[index].Key()].CHANGE24HOUR, 1e-10)
	}
}
