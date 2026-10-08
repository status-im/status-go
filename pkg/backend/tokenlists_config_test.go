package backend

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/requests"
)

func TestTokenCatalogueLoginConfig(t *testing.T) {
	var request requests.Login
	require.NoError(t, json.Unmarshal([]byte(`{"tokenListsUseNim":true}`), &request))
	config := buildWalletConfig(&request.WalletConfig, &request.WalletSecretsConfig)
	require.True(t, config.TokenListsUseNim)
	require.False(t, buildWalletConfig(&requests.WalletConfig{}, &requests.WalletSecretsConfig{}).TokenListsUseNim)
}
