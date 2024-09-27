package requests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/multiaccounts"
)

func TestMigrateKeystoreDir_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		req         MigrateKeystoreDir
		expectedErr string
	}{
		{
			name: "valid request",
			req: MigrateKeystoreDir{
				Account:  multiaccounts.Account{Name: "test-account"},
				Password: "test-password",
				OldDir:   "/old/keystore/dir",
				NewDir:   "/new/keystore/dir",
			},
		},
		{
			name: "empty account",
			req: MigrateKeystoreDir{
				Password: "test-password",
				OldDir:   "/old/keystore/dir",
				NewDir:   "/new/keystore/dir",
			},
		},
		{
			name: "empty password",
			req: MigrateKeystoreDir{
				Account: multiaccounts.Account{Name: "test-account"},
				OldDir:  "/old/keystore/dir",
				NewDir:  "/new/keystore/dir",
			},
			expectedErr: "Password",
		},
		{
			name: "empty old dir",
			req: MigrateKeystoreDir{
				Account:  multiaccounts.Account{Name: "test-account"},
				Password: "test-password",
				NewDir:   "/new/keystore/dir",
			},
			expectedErr: "OldDir",
		},
		{
			name: "empty new dir",
			req: MigrateKeystoreDir{
				Account:  multiaccounts.Account{Name: "test-account"},
				Password: "test-password",
				OldDir:   "/old/keystore/dir",
			},
			expectedErr: "NewDir",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
