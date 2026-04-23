package chain

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/security"
)

func TestCreateEthClientFromProvider_PuzzleAuth(t *testing.T) {
	provider := params.RpcProvider{
		ChainID:  1,
		Name:     "p",
		URL:      security.NewSensitiveString("https://example.com/eth/mainnet/"),
		Enabled:  true,
		Type:     params.EmbeddedEthRpcProxyProviderType,
		AuthType: params.PuzzleAuth,
	}
	cl, err := CreateEthClientFromProvider(provider, "ua/1.0")
	require.NoError(t, err)
	require.NotNil(t, cl)
	cl.Close()
}
