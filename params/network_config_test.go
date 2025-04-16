package params_test

import (
	"github.com/status-im/status-go/internal/security"
	"testing"

	"github.com/status-im/status-go/params"

	"github.com/stretchr/testify/assert"
)

func TestRpcProvider_GetHost(t *testing.T) {
	provider := params.RpcProvider{URL: security.NewSensitiveString("https://api.example.com/path")}
	expectedHost := "api.example.com"
	assert.Equal(t, expectedHost, provider.GetHost())
}

func TestRpcProvider_GetFullURL(t *testing.T) {
	provider := params.RpcProvider{
		URL: security.NewSensitiveString("https://api.example.com"),
		AuthType: params.TokenAuth,
		AuthToken: security.NewSensitiveString("mytoken")}
	expectedFullURL := "https://api.example.com/mytoken"
	assert.Equal(t, expectedFullURL, provider.GetFullURL().Reveal())

	provider.AuthType = params.NoAuth
	expectedFullURL = "https://api.example.com"
	assert.Equal(t, expectedFullURL, provider.GetFullURL().Reveal())
}
