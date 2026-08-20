package coingecko

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
)

func TestFetchingTokens(t *testing.T) {
	expectedData := []GeckoToken{
		{
			ID:     "valobit",
			Symbol: "vbit",
			Name:   "VALOBIT",
			Platforms: map[string]string{
				"valobit": "",
			},
		},
		{
			ID:     "dmx",
			Symbol: "dmx",
			Name:   "DMX",
			Platforms: map[string]string{
				"factom": "0x0ec581b1f76ee71fb9feefd058e0ecf90ebab63e",
			},
		},
		{
			ID:     "pedro",
			Symbol: "pedro",
			Name:   "PEDRO",
			Platforms: map[string]string{
				"factom": "0x51165e8ce5d6e99c570f4601a6c8409394295065",
			},
		},
		{
			ID:     "volta-club",
			Symbol: "volta",
			Name:   "Volta Club",
			Platforms: map[string]string{
				"ethereum":     "0x9b06f3c5de42d4623d7a2bd940ec735103c68a76",
				"factom":       "0x9b06f3c5de42d4623d7a2bd940ec735103c68a76",
				"avalanche":    "0x9b06f3c5de42d4623d7a2bd940ec735103c68a76",
				"arbitrum-one": "0x9b06f3c5de42d4623d7a2bd940ec735103c68a76",
			},
		},
		{
			ID:     "don-t-sell-your-bitcoin",
			Symbol: "bitcoin",
			Name:   "DON'T SELL YOUR BITCOIN",
			Platforms: map[string]string{
				"solana": "RrUiMy3j9bzhgBJpXCqpF33vfrGD5Y9qAfbBVbRMkQv",
			},
		},
		{
			ID:     "harrypotterobamasonic10in",
			Symbol: "bitcoin",
			Name:   "HarryPotterObamaSonic10Inu (ETH)",
			Platforms: map[string]string{
				"ethereum": "0x72e4f9f808c49a2a61de9c5896298920dc4eeea9",
				"solana":   "CTgiaZUK12kCcB8sosn4Nt2NZtzLgtPqDwyQyr2syATC",
				"base":     "0x2a06a17cbc6d0032cac2c6696da90f29d39a1a29",
			},
		},
		{
			ID:     "dork-lord-coin",
			Symbol: "dlord",
			Name:   "DORK LORD COIN",
			Platforms: map[string]string{
				"solana": "3krWsXrweUbpsDJ9NKiwzNJSxLQKdPJNGzeEU5MZKkrb",
			},
		},
		{
			ID:     "dork-lord-eth",
			Symbol: "dorkl",
			Name:   "DORK LORD (SOL)",
			Platforms: map[string]string{
				"solana": "8uwcmeA46XfLUc4MJ1WFQeV81rDTHTVer1B5Rc6M4iyn",
			},
		},
		{
			ID:     "dotcom",
			Symbol: "y2k",
			Name:   "Dotcom",
			Platforms: map[string]string{
				"solana": "8YiB8B43EwDeSx5Jp91VQjgBU4mfCgVvyNahadtzpump",
			},
		},
		{
			ID:     "sentinel-bot-ai",
			Symbol: "snt",
			Name:   "Sentinel Bot Ai",
			Platforms: map[string]string{
				"ethereum": "0x78ba134c3ace18e69837b01703d07f0db6fb0a60",
			},
		},
		{
			ID:     "status",
			Symbol: "snt",
			Name:   "Status",
			Platforms: map[string]string{
				"ethereum": "0x744d70fdbe2ba4cf95131626614a1763df805b9e",
				"energi":   "0x6bb14afedc740dce4904b7a65807fe3b967f4c94",
			},
		},
		{
			ID:     "usd-coin",
			Symbol: "usdc",
			Name:   "USDC",
			Platforms: map[string]string{
				"ethereum":            "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"unichain":            "0x078d782b760474a361dda0af3839290b0ef57ad6",
				"zksync":              "0x1d17cbcf0d6d143135ae902365d2e5e2a16538d4",
				"optimistic-ethereum": "0x0b2c639c533813f4aa9d7837caf62653d097ff85",
				"polkadot":            "1337",
				"tron":                "TEkxiTehnzSmSe2XqrBj4w32RUN966rdz8",
				"near-protocol":       "17208628f84f5d6ad33f0da3bbbeb27ffcb398eac501a31bd6ad2011e36133a1",
				"hedera-hashgraph":    "0.0.456858",
				"aptos":               "0xbae207659db88bea0cbead6da0ed00aac12edcdda169e591cd41c94180b46f3b",
				"algorand":            "31566704",
				"stellar":             "USDC-GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
				"celo":                "0xceba9300f2b948710d2653dd7b07f33a8b32118c",
				"sui":                 "0xdba34672e30cb065b1f93e3ab55318768fd6fef66c15942c9f7cb846e2f900e7::usdc::USDC",
				"avalanche":           "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e",
				"arbitrum-one":        "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
				"polygon-pos":         "0x3c499c542cef5e3811e1192ce70d8cc03d5c3359",
				"base":                "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
				"solana":              "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
			},
		},
		{
			ID:     "bridged-usdt",
			Symbol: "usdt",
			Name:   "Bridged USDT",
			Platforms: map[string]string{
				"optimistic-ethereum": "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58",
				"kardiachain":         "0x551a5dcac57c66aa010940c2dcff5da9c53aa53b",
				"metis-andromeda":     "0xbb06dca3ae6887fabf931640f67cab3e3a16f4dc",
				"boba":                "0x5de1677344d3cb0d7d465c10b72a8f60699c062d",
				"rollux":              "0x28c9c7fb3fe3104d2116af26cc8ef7905547349c",
				"velas":               "0xb44a9b6905af7c801311e8f4e76932ee959c663c",
				"iotex":               "0x3cdb7c48e70b854ed2fa392e21687501d84b3afc",
				"harmony-shard-0":     "0x3c2b8be99c50593081eaa2a724f0b8285f5aba8f",
				"canto":               "0xd567b3d7b8fe3c79a1ad8da978812cfc4fa05e75",
				"zksync":              "0x493257fd37edb34451f62edf8d2a0c418852ba4c",
				"osmosis":             "ibc/4ABBEF4C8926DDDB320AE5188CFD63267ABBCEFC0583E4AE05D6E5AA2401DDAB",
				"bitgert":             "0xde14b85cf78f2add2e867fee40575437d5f10c06",
				"fuse":                "0xfadbbf8ce7d5b7041be672561bba99f79c532e10",
				"meter":               "0x5fa41671c48e3c951afc30816947126ccc8c162e",
				"ethereumpow":         "0x2ad7868ca212135c6119fd7ad1ce51cfc5702892",
				"okex-chain":          "0x382bb369d343125bfb2117af9c149795c6c65c50",
				"oasys":               "0xdc3af65ecbd339309ec55f109cb214e0325c5ed4",
				"milkomeda-cardano":   "0x80a16016cc4a2e6a2caca8a4a498b1699ff0f844",
			},
		},
	}

	srv, stop := setupTest(t, responseCoinsListData)
	defer stop()

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	received, err := geckoClient.fetchTokens(context.Background())
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(expectedData, received))
}

func TestErrorWhenFetchingTokens(t *testing.T) {
	srv, stop := setupTest(t, responseError)
	defer stop()

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	received, err := geckoClient.fetchTokens(context.Background())
	require.Error(t, err)
	require.Nil(t, received)
}
