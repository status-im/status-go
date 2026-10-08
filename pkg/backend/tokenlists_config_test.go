package backend

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/requests"
)

func TestTokenCatalogueLoginConfig(t *testing.T) {
	for _, obsolete := range []string{`{}`, `{"tokenListsUseNim":false,"tokenListsShadow":false}`, `{"tokenListsUseNim":true,"tokenListsShadow":true}`} {
		var request requests.Login
		require.NoError(t, json.Unmarshal([]byte(obsolete), &request))
		config := buildWalletConfig(&request.WalletConfig, &request.WalletSecretsConfig)
		require.Equal(t, buildWalletConfig(&requests.WalletConfig{}, &requests.WalletSecretsConfig{}), config)
		encoded, err := json.Marshal(config)
		require.NoError(t, err)
		require.NotContains(t, string(encoded), "TokenListsUseNim")
		require.NotContains(t, string(encoded), "TokenListsShadow")
		encoded, err = json.Marshal(request)
		require.NoError(t, err)
		require.NotContains(t, string(encoded), "tokenListsUseNim")
		require.NotContains(t, string(encoded), "tokenListsShadow")
	}
}
