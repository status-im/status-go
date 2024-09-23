package requests

import "testing"

func TestVerifyDatabasePasswordV2Validate(t *testing.T) {
	tests := []struct {
		name    string
		request VerifyDatabasePassword
		wantErr error
	}{
		{
			name:    "Empty KeyUID",
			request: VerifyDatabasePassword{KeyUID: "", Password: "password"},
			wantErr: ErrVerifyDatabasePasswordEmptyKeyUID,
		},
		{
			name:    "Empty Password",
			request: VerifyDatabasePassword{KeyUID: "keyuid", Password: ""},
			wantErr: ErrVerifyDatabasePasswordEmptyPassword,
		},
		{
			name:    "Valid Request",
			request: VerifyDatabasePassword{KeyUID: "keyuid", Password: "password"},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if err != tt.wantErr {
				t.Errorf("VerifyDatabasePasswordV2.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
