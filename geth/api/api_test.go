package api

import (
	"testing"

	"github.com/status-im/status-go/geth/common"
	"github.com/stretchr/testify/require"
)

func TestValidateNodeConfig(t *testing.T) {
	noErrorsCallback := func(resp common.APIDetailedResponse) {
		require.True(t, resp.Status, "expected status equal true")
		require.Empty(t, resp.FieldErrors)
		require.Empty(t, resp.Message)
	}

	testCases := []struct {
		Name     string
		Config   string
		Callback func(common.APIDetailedResponse)
	}{
		{
			Name: "response for valid config",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/tmp"
			}`,
			Callback: noErrorsCallback,
		},
		{
			Name:   "response for invalid JSON string",
			Config: `{"Network": }`,
			Callback: func(resp common.APIDetailedResponse) {
				require.False(t, resp.Status)
				require.Contains(t, resp.Message, "validation: invalid character '}'")
			},
		},
		{
			Name:   "response for config with multiple errors",
			Config: `{}`,
			Callback: func(resp common.APIDetailedResponse) {
				required := map[string]string{
					"NodeConfig.NetworkID": "required",
					"NodeConfig.DataDir":   "required",
				}

				require.False(t, resp.Status)
				require.Contains(t, resp.Message, "validation: validation failed")
				require.Equal(t, 2, len(resp.FieldErrors))

				for _, err := range resp.FieldErrors {
					require.Contains(t, required, err.Parameter)
					require.Contains(t, err.Error(), required[err.Parameter])
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Logf("TestValidateNodeConfig: %s", tc.Name)
		statusAPI := StatusAPI{}
		resp := statusAPI.ValidateJSONConfig(tc.Config)
		tc.Callback(resp)
	}
}
