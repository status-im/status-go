// +build e2e_test

// Tests in `./lib` package will run only when `e2e_test` build tag is provided.
// It's required to prevent some files from being included in the binary.
// Check out `lib/utils.go` for more details.

package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// the actual test functions are in non-_test.go files (so that they can use cgo i.e. import "C")
// the only intent of these wrappers is for gotest can find what tests are exposed.
func TestExportedAPI(t *testing.T) {
	allTestsDone := make(chan struct{}, 1)
	go testExportedAPI(t, allTestsDone)

	<-allTestsDone
}

func TestValidateNodeConfig(t *testing.T) {
	noErrorsCallback := func(resp APIDetailedResponse) {
		require.True(t, resp.Status, "expected status equal true")
		require.Empty(t, resp.FieldErrors)
		require.Empty(t, resp.Message)
	}

	testCases := []struct {
		Name     string
		Config   string
		Callback func(APIDetailedResponse)
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
			Callback: func(resp APIDetailedResponse) {
				require.False(t, resp.Status)
				require.Contains(t, resp.Message, "validation: invalid character '}'")
			},
		},
		{
			Name:   "response for config with multiple errors",
			Config: `{}`,
			Callback: func(resp APIDetailedResponse) {
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
		testValidateNodeConfig(t, tc.Config, tc.Callback)
	}
}
