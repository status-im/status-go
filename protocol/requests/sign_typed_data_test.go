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
		expectedErr error
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
			expectedErr: nil,
		},
		{
			name: "empty typed data",
			req: SignTypedData{
				Address:  "0x1234567890123456789012345678901234567890",
				Password: "password",
			},
			expectedErr: ErrSignTypedDataEmptyTypedData,
		},
		{
			name: "empty address",
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
			expectedErr: ErrSignTypedDataEmptyAddress,
		},
		{
			name: "empty password",
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
			expectedErr: ErrSignTypedDataEmptyPassword,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.expectedErr != nil {
				require.Equal(t, tc.expectedErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
