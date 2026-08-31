package requests

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoginValidateDEK(t *testing.T) {
	validDEK := "8a9f7d2b6c1e4f3a5d8b9c0e2f4a6b8d0c2e4f6a8b0d2c4e6f8a0b2d4c6e8f0a"

	testCases := []struct {
		name    string
		request Login
		err     error
	}{
		{"valid DEK", Login{KeyUID: "0x1234", DEK: validDEK}, nil},
		{"DEK too short", Login{KeyUID: "0x1234", DEK: "abcdef"}, ErrLoginInvalidDEK},
		{"DEK not hex", Login{KeyUID: "0x1234", DEK: "zz" + validDEK[2:]}, ErrLoginInvalidDEK},
		{"DEK with 0x prefix", Login{KeyUID: "0x1234", DEK: "0x" + validDEK}, ErrLoginInvalidDEK},
		{"DEK with password", Login{KeyUID: "0x1234", DEK: validDEK, Password: "pass"}, ErrLoginDEKMutuallyExclusive},
		{"DEK with mnemonic", Login{KeyUID: "0x1234", DEK: validDEK, Mnemonic: "some words"}, ErrLoginDEKMutuallyExclusive},
		{"DEK with keycard key", Login{KeyUID: "0x1234", DEK: validDEK,
			KeycardWhisperPrivateKey: "1111111111111111111111111111111111111111111111111111111111111111"}, ErrLoginDEKMutuallyExclusive},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.request.Validate()
			if tc.err == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.err)
			}
		})
	}
}

func TestLoginValidateNormalizesDEKCase(t *testing.T) {
	lowercaseDEK := "8a9f7d2b6c1e4f3a5d8b9c0e2f4a6b8d0c2e4f6a8b0d2c4e6f8a0b2d4c6e8f0a"

	request := Login{KeyUID: "0x1234", DEK: strings.ToUpper(lowercaseDEK)}
	require.NoError(t, request.Validate())
	require.Equal(t, lowercaseDEK, request.DEK)
}

func TestLoginUnmarshalLogFilePath(t *testing.T) {
	var request Login
	err := json.Unmarshal([]byte(`{
		"keyUid": "0x1234",
		"password": "pass",
		"logFilePath": "/data/logs"
	}`), &request)
	require.NoError(t, err)
	require.Equal(t, "/data/logs", request.LogFilePath)
	require.NoError(t, request.Validate())
}
