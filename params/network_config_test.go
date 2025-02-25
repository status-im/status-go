package params_test

import (
	"testing"

	"github.com/status-im/status-go/params"
	"github.com/stretchr/testify/assert"
)

func TestRpcProvider_GetHost(t *testing.T) {
	provider := params.RpcProvider{URL: "https://api.example.com/path"}
	expectedHost := "api.example.com"
	assert.Equal(t, expectedHost, provider.GetHost())
}

func TestRpcProvider_GetFullURL(t *testing.T) {
	provider := params.RpcProvider{URL: "https://api.example.com", AuthType: params.TokenAuth, AuthToken: "mytoken"}
	expectedFullURL := "https://api.example.com/mytoken"
	assert.Equal(t, expectedFullURL, provider.GetFullURL())

	provider.AuthType = params.NoAuth
	expectedFullURL = "https://api.example.com"
	assert.Equal(t, expectedFullURL, provider.GetFullURL())
}
