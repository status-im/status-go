package token

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	sdktypes "github.com/status-im/go-wallet-sdk/pkg/tokens/types"
	"github.com/stretchr/testify/require"
)

func TestSDKCatalogueConversion(t *testing.T) {
	require.Nil(t, catalogueToken(nil))
	require.Nil(t, catalogueTokens(nil))
	require.NotNil(t, catalogueTokens([]*sdktypes.Token{}))
	require.Nil(t, catalogueList(nil))
	source := &sdktypes.TokenList{
		ID: "example", Name: "Tokens", Timestamp: "2026-01-01T00:00:00Z",
		FetchedTimestamp: "2026-01-02T00:00:00Z", Source: "https://example.org/list",
		Version: sdktypes.Version{Major: 1, Minor: 2, Patch: 3}, LogoURI: "ipfs://logo",
		Tags:     map[string]interface{}{"stable": map[string]interface{}{"name": "Stable"}},
		Keywords: []string{"test"}, Tokens: []*sdktypes.Token{nil, {
			ChainID: 1, Address: common.HexToAddress("0xabcd"), CrossChainID: "example",
			Name: "Example", Symbol: "EX", Decimals: 24, LogoURI: "ipfs://token", CustomToken: true,
		}},
	}
	converted := catalogueList(source)
	want, err := json.Marshal(source)
	require.NoError(t, err)
	got, err := json.Marshal(converted)
	require.NoError(t, err)
	require.JSONEq(t, string(want), string(got))
	converted.Tokens[1].Symbol = "CHANGED"
	require.Equal(t, "EX", source.Tokens[1].Symbol)
}
