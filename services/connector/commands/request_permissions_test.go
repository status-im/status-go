package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

				if permission, ok := response.(Permission); ok {
					assert.Equal(t, permission.ParentCapability, tc.expectedCapability)
				} else {
					assert.Fail(t, "Can't parse permission from the response")
				}
			}
		})
	}
}
