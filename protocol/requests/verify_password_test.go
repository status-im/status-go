package requests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifyPassword_Validate(t *testing.T) {
	tests := []struct {
		name        string
		req         VerifyPassword
		expectError bool
		errContains string
	}{
		{
			name:        "valid password",
			req:         VerifyPassword{Password: "secret"},
			expectError: false,
		},
		{
			name:        "empty password",
			req:         VerifyPassword{Password: ""},
			expectError: true,
			errContains: "Password",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if !tc.expectError {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errContains)
		})
	}
}
