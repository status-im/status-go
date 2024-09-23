package requests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeleteImportedKey_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		req         DeleteImportedKey
		expectedErr error
	}{
		{
			name: "valid request",
			req: DeleteImportedKey{
				Address:     "0x1234567890123456789012345678901234567890",
				Password:    "password",
				KeyStoreDir: "/keystore/dir",
			},
			expectedErr: nil,
		},
		{
			name: "empty address",
			req: DeleteImportedKey{
				Password:    "password",
				KeyStoreDir: "/keystore/dir",
			},
			expectedErr: ErrDeleteImportedKeyEmptyAddress,
		},
		{
			name: "empty password",
			req: DeleteImportedKey{
				Address:     "0x1234567890123456789012345678901234567890",
				KeyStoreDir: "/keystore/dir",
			},
			expectedErr: ErrDeleteImportedKeyEmptyPassword,
		},
		{
			name: "empty keystore dir",
			req: DeleteImportedKey{
				Address:  "0x1234567890123456789012345678901234567890",
				Password: "password",
			},
			expectedErr: ErrDeleteImportedKeyEmptyKeyStoreDir,
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
