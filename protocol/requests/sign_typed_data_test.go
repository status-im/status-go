package requests

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/typeddata"
)

func TestSignTypedData_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		req         SignTypedData
		expectedErr string
	}{
		{
			name: "valid request",
			req: SignTypedData{
				TypedData: typeddata.TypedData{
					Types: typeddata.Types{
						"EIP712Domain": []typeddata.Field{
							{Name: "name", Type: "string"},
						},
					},
					Domain: map[string]json.RawMessage{
						"name": json.RawMessage(`"test"`),
					},
					PrimaryType: "EIP712Domain",
					Message: map[string]json.RawMessage{
						"name": json.RawMessage(`"test"`),
					},
				},
				Address:  "0x1234567890123456789012345678901234567890",
				Password: "password",
			},
			expectedErr: "",
		},
		{
			name: "missing typed data",
			req: SignTypedData{
				TypedData: typeddata.TypedData{
					Types:       typeddata.Types{},
					Domain:      map[string]json.RawMessage{},
					PrimaryType: "",
					Message:     map[string]json.RawMessage{},
				},
				Address:  "0x1234567890123456789012345678901234567890",
				Password: "password",
			},
			expectedErr: "`EIP712Domain` must be in `types`",
		},
		{
			name: "missing address",
			req: SignTypedData{
				TypedData: typeddata.TypedData{
					Types: typeddata.Types{
						"EIP712Domain": []typeddata.Field{
							{Name: "name", Type: "string"},
						},
					},
					Domain: map[string]json.RawMessage{
						"name": json.RawMessage(`"test"`),
					},
					PrimaryType: "EIP712Domain",
					Message: map[string]json.RawMessage{
						"name": json.RawMessage(`"test"`),
					},
				},
				Password: "password",
			},
			expectedErr: "Key: 'SignTypedData.Address' Error:Field validation for 'Address' failed on the 'required' tag",
		},
		{
			name: "missing password",
			req: SignTypedData{
				TypedData: typeddata.TypedData{
					Types: typeddata.Types{
						"EIP712Domain": []typeddata.Field{
							{Name: "name", Type: "string"},
						},
					},
					Domain: map[string]json.RawMessage{
						"name": json.RawMessage(`"test"`),
					},
					PrimaryType: "EIP712Domain",
					Message: map[string]json.RawMessage{
						"name": json.RawMessage(`"test"`),
					},
				},
				Address: "0x1234567890123456789012345678901234567890",
			},
			expectedErr: "Key: 'SignTypedData.Password' Error:Field validation for 'Password' failed on the 'required' tag",
		},
		{
			name: "invalid typed data",
			req: SignTypedData{
				TypedData: typeddata.TypedData{
					Types: typeddata.Types{
						"EIP712Domain": []typeddata.Field{
							{Name: "name", Type: "string"},
						},
					},
					Domain: map[string]json.RawMessage{
						"name": json.RawMessage(`"test"`),
					},
					PrimaryType: "InvalidType",
					Message: map[string]json.RawMessage{
						"name": json.RawMessage(`"test"`),
					},
				},
				Address:  "0x1234567890123456789012345678901234567890",
				Password: "password",
			},
			expectedErr: "primary type `InvalidType` not defined in types",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.expectedErr != "" {
				require.EqualError(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
