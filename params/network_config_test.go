package params

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRpcProvider_GetHost(t *testing.T) {
	provider := RpcProvider{URL: "https://api.example.com/path"}
	expectedHost := "api.example.com"
	assert.Equal(t, expectedHost, provider.GetHost())
}

func TestRpcProvider_GetFullURL(t *testing.T) {
	provider := RpcProvider{URL: "https://api.example.com", AuthType: TokenAuth, AuthToken: "mytoken"}
	expectedFullURL := "https://api.example.com/mytoken"
	assert.Equal(t, expectedFullURL, provider.GetFullURL())

	provider.AuthType = NoAuth
	expectedFullURL = "https://api.example.com"
	assert.Equal(t, expectedFullURL, provider.GetFullURL())
}
