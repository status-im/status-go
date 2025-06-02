package requests

import (
	"testing"

	"github.com/status-im/status-go/common"
)

func TestInitializeApplication_Validate(t *testing.T) {
	tests := []struct {
		name    string
		app     InitializeApplication
		wantErr bool
		errType error
	}{
		{
			name: "Valid with minimum required fields",
			app: InitializeApplication{
				DataDir: "/valid/path",
			},
			wantErr: false,
		},
		{
			name:    "Missing DataDir",
			app:     InitializeApplication{},
			wantErr: true,
			errType: ErrInitializeApplicationInvalidDataDir,
		},
		{
			name: "Invalid LogLevel",
			app: InitializeApplication{
				DataDir:  "/valid/path",
				LogLevel: "INVALID",
			},
			wantErr: true,
		},
		{
			name: "Valid LogLevel - ERROR",
			app: InitializeApplication{
				DataDir:  "/valid/path",
				LogLevel: "ERROR",
			},
			wantErr: false,
		},
		{
			name: "Valid LogLevel - WARN",
			app: InitializeApplication{
				DataDir:  "/valid/path",
				LogLevel: "WARN",
			},
			wantErr: false,
		},
		{
			name: "Valid LogLevel - INFO",
			app: InitializeApplication{
				DataDir:  "/valid/path",
				LogLevel: "INFO",
			},
			wantErr: false,
		},
		{
			name: "Valid LogLevel - DEBUG",
			app: InitializeApplication{
				DataDir:  "/valid/path",
				LogLevel: "DEBUG",
			},
			wantErr: false,
		},
		{
			name: "Valid LogLevel - TRACE",
			app: InitializeApplication{
				DataDir:  "/valid/path",
				LogLevel: "TRACE",
			},
			wantErr: false,
		},
		{
			name: "All valid log levels",
			app: InitializeApplication{
				DataDir:  "/valid/path",
				LogLevel: "ERROR",
			},
			wantErr: false,
		},
		{
			name: "Full configuration with valid values",
			app: InitializeApplication{
				DataDir:              "/valid/path",
				MixpanelAppID:        "app-id",
				MixpanelToken:        "token",
				MediaServerEnableTLS: common.Ptr(true),
				SentryDSN:            "sentry-dsn",
				LogDir:               "/logs/path",
				LogEnabled:           true,
				LogLevel:             "INFO",
				APILoggingEnabled:    true,
				MetricsEnabled:       true,
				MetricsAddress:       "localhost:8545",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.app.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.errType != nil && err != tt.errType {
				t.Errorf("Validate() error type = %v, want %v", err, tt.errType)
			}
		})
	}
}
