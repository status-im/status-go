package requests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/multiaccounts"
)

func TestMigrateKeyStoreDir_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		req         MigrateKeyStoreDir
		expectedErr error
	}{
		{
			name: "valid request",
			req: MigrateKeyStoreDir{
				Account:  multiaccounts.Account{KeyUID: "0x1234"},
				Password: "password",
				OldDir:   "/old/dir",
				NewDir:   "/new/dir",
			},
			expectedErr: nil,
		},
		{
			name: "empty account",
			req: MigrateKeyStoreDir{
				Password: "password",
				OldDir:   "/old/dir",
				NewDir:   "/new/dir",
			},
			expectedErr: ErrMigrateKeyStoreDirEmptyAccount,
		},
		{
			name: "empty password",
			req: MigrateKeyStoreDir{
				Account: multiaccounts.Account{KeyUID: "0x1234"},
				OldDir:  "/old/dir",
				NewDir:  "/new/dir",
			},
			expectedErr: ErrMigrateKeyStoreDirEmptyPassword,
		},
		{
			name: "empty old dir",
			req: MigrateKeyStoreDir{
				Account:  multiaccounts.Account{KeyUID: "0x1234"},
				Password: "password",
				NewDir:   "/new/dir",
			},
			expectedErr: ErrMigrateKeyStoreDirEmptyOldDir,
		},
		{
			name: "empty new dir",
			req: MigrateKeyStoreDir{
				Account:  multiaccounts.Account{KeyUID: "0x1234"},
				Password: "password",
				OldDir:   "/old/dir",
			},
			expectedErr: ErrMigrateKeyStoreDirEmptyNewDir,
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
