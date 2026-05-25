package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"

	persistence "github.com/status-im/status-go/services/connector/database"
)

func TestFailToRequestPermissionsWithMissingDAppFields(t *testing.T) {
	state, close := setupCommand(t, Method_RequestPermissions)
	t.Cleanup(close)

	// Missing DApp fields
	request, err := ConstructRPCRequest("wallet_requestPermissions", []interface{}{}, nil)
	assert.NoError(t, err)

	result, err := state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrRequestMissingDAppData, err)
	assert.Empty(t, result)
}

func TestRequestPermissionsResponse(t *testing.T) {
	state, close := setupCommand(t, Method_RequestPermissions)
	t.Cleanup(close)

	dApp := &persistence.DApp{
		URL:           testDAppData.URL,
		Name:          testDAppData.Name,
		IconURL:       testDAppData.IconURL,
		ClientID:      testDAppData.ClientID,
		SharedAccount: [20]byte{0x01},
		ChainID:       1,
	}
	err := persistence.UpsertDApp(state.walletDb, dApp)
	assert.NoError(t, err)

	testCases := []struct {
		name               string
		params             []interface{}
		expectedError      error
		expectedCapability string
	}{
		{
			name: "Single valid key",
			params: []interface{}{
				map[string]interface{}{
					"eth_requestAccounts": struct{}{},
				},
			},
			expectedError:      nil,
			expectedCapability: "eth_requestAccounts",
		},
		{
			name: "Single valid key",
			params: []interface{}{
				map[string]interface{}{
					"eth_accounts": struct{}{},
				},
			},
			expectedError:      nil,
			expectedCapability: "eth_accounts",
		},
		{
			name: "Multiple keys",
			params: []interface{}{
				map[string]interface{}{
					"eth_requestAccounts": struct{}{},
					"eth_sendTransaction": struct{}{},
				},
			},
			expectedError:      ErrMultipleKeysFound,
			expectedCapability: "",
		},
		{
			name: "No keys",
			params: []interface{}{
				map[string]interface{}{},
			},
			expectedError:      ErrNoRequestPermissionsParamsFound,
			expectedCapability: "",
		},
		{
			name:               "Nil params",
			params:             nil,
			expectedError:      ErrEmptyRPCParams,
			expectedCapability: "",
		},
		{
			name: "Invalid param type",
			params: []interface{}{
				"invalid_param_type",
			},
			expectedError:      ErrInvalidParamType,
			expectedCapability: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := ConstructRPCRequest("wallet_requestPermissions", tc.params, &testDAppData)
			assert.NoError(t, err)

			response, err := state.cmd.Execute(state.ctx, request)
			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)

				if permissions, ok := response.([]persistence.Permission); ok {
					assert.Len(t, permissions, 1)
					assert.Equal(t, permissions[0].ParentCapability, tc.expectedCapability)
					assert.Equal(t, permissions[0].Invoker, testDAppData.URL)
				} else {
					assert.Fail(t, "Can't parse permissions array from the response")
				}
			}
		})
	}
}

func TestRequestPermissionsWithCaveats(t *testing.T) {
	state, close := setupCommand(t, Method_RequestPermissions)
	t.Cleanup(close)

	dApp := &persistence.DApp{
		URL:           testDAppData.URL,
		Name:          testDAppData.Name,
		IconURL:       testDAppData.IconURL,
		ClientID:      testDAppData.ClientID,
		SharedAccount: [20]byte{0x01},
		ChainID:       1,
	}
	err := persistence.UpsertDApp(state.walletDb, dApp)
	assert.NoError(t, err)

	// Test with caveats as map
	params := []interface{}{
		map[string]interface{}{
			"eth_accounts": map[string]interface{}{
				"requiredMethods": []string{"signTypedData_v4"},
				"expiry":          float64(1234567890),
			},
		},
	}

	request, err := ConstructRPCRequest("wallet_requestPermissions", params, &testDAppData)
	assert.NoError(t, err)

	response, err := state.cmd.Execute(state.ctx, request)
	assert.NoError(t, err)

	permissions, ok := response.([]persistence.Permission)
	assert.True(t, ok)
	assert.Len(t, permissions, 1)
	assert.Equal(t, "eth_accounts", permissions[0].ParentCapability)
	assert.Len(t, permissions[0].Caveats, 2)
}

func TestRequestPermissionsWithNonMapCaveats(t *testing.T) {
	state, close := setupCommand(t, Method_RequestPermissions)
	t.Cleanup(close)

	dApp := &persistence.DApp{
		URL:           testDAppData.URL,
		Name:          testDAppData.Name,
		IconURL:       testDAppData.IconURL,
		ClientID:      testDAppData.ClientID,
		SharedAccount: [20]byte{0x01},
		ChainID:       1,
	}
	err := persistence.UpsertDApp(state.walletDb, dApp)
	assert.NoError(t, err)

	// Test with non-map caveats value
	params := []interface{}{
		map[string]interface{}{
			"eth_accounts": "not-a-map",
		},
	}

	request, err := ConstructRPCRequest("wallet_requestPermissions", params, &testDAppData)
	assert.NoError(t, err)

	response, err := state.cmd.Execute(state.ctx, request)
	assert.NoError(t, err)

	permissions, ok := response.([]persistence.Permission)
	assert.True(t, ok)
	assert.Len(t, permissions, 1)
	assert.Equal(t, "eth_accounts", permissions[0].ParentCapability)
	assert.Empty(t, permissions[0].Caveats)
}

func TestRequestPermissionsNoDAppFound(t *testing.T) {
	state, close := setupCommand(t, Method_RequestPermissions)
	t.Cleanup(close)

	// No dApp row: eth_accounts would auto-share (UI flow); use another capability to assert ErrDAppNotFound.
	params := []interface{}{
		map[string]interface{}{
			"personal_sign": map[string]interface{}{},
		},
	}

	request, err := ConstructRPCRequest("wallet_requestPermissions", params, &testDAppData)
	assert.NoError(t, err)

	response, err := state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrDAppNotFound, err)
	assert.Empty(t, response)
}
