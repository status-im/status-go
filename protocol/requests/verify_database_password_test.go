package requests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifyDatabasePassword_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request VerifyDatabasePassword
		wantErr string
	}{
		{
			name:    "Empty KeyUID",
			request: VerifyDatabasePassword{KeyUID: "", Password: "password"},
			wantErr: "KeyUID",
		},
		{
			name:    "Empty Password",
			request: VerifyDatabasePassword{KeyUID: "keyuid", Password: ""},
			wantErr: "Password",
		},
		{
			name:    "Valid Request",
			request: VerifyDatabasePassword{KeyUID: "keyuid", Password: "password"},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr != "" {
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
