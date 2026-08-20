package statusgo

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brianvoe/gofakeit/v6"

	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/internal/signal"
)

func TestSetSignalHandler(t *testing.T) {
	// Setup
	var data string
	signal.SetHandler(func(b []byte) {
		data = string(b)
	})
	t.Cleanup(signal.ResetHandler)

	// Test data
	testAccount := &multiaccounts.Account{Name: "test"}
	testSettings := &settings.Settings{KeyUID: "0x1"}
	testEnsUsernames := json.RawMessage(`{"test": "test"}`)

	// Action
	signal.SendLoggedIn(testAccount, testSettings, testEnsUsernames, nil)

	// Assertions
	require.Contains(t, data, `"key-uid":"0x1"`, "Signal should contain the correct KeyUID")
	require.Contains(t, data, `"name":"test"`, "Signal should contain the correct account name")
	require.Contains(t, data, `"ensUsernames":{"test":"test"}`, "Signal should contain the correct ENS usernames")
}

func TestIntendedPanic(t *testing.T) {
	message := gofakeit.LetterN(5)
	require.PanicsWithError(t, message, func() {
		IntendedPanic(message)
	})
}

func TestDeleteMultiaccountResponseFormat(t *testing.T) {
	// Test that the function returns a proper JSON response even with invalid input
	tests := []struct {
		name        string
		requestJSON string
	}{
		{
			name:        "invalid JSON",
			requestJSON: `{"keyUID": invalid}`,
		},
		{
			name: "missing required field",
			requestJSON: `{
				"keyUID": ""
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the actual function - it will fail internally but should return valid JSON
			result := deleteMultiaccount(tt.requestJSON)

			// Verify it returns valid JSON
			var response APIResponse
			err := json.Unmarshal([]byte(result), &response)
			require.NoError(t, err, "Response should be valid JSON")
			require.NotEmpty(t, response.Error, "Response should contain an error message")
		})
	}
}

func TestDeleteMultiaccountRequestValidation(t *testing.T) {
	// Test the request validation logic in isolation
	tests := []struct {
		name    string
		keyUID  string
		keyDir  string
		wantErr bool
	}{
		{
			name:    "valid request",
			keyUID:  "0xabcd1234",
			keyDir:  "/path/to/keystore",
			wantErr: false,
		},
		{
			name:    "empty keyUID",
			keyUID:  "",
			keyDir:  "/path/to/keystore",
			wantErr: true,
		},
		{
			name:    "empty keyStoreDir",
			keyUID:  "0xabcd1234",
			keyDir:  "",
			wantErr: true,
		},
		{
			name:    "both empty",
			keyUID:  "",
			keyDir:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := requests.DeleteMultiaccount{
				KeyUID:      tt.keyUID,
				KeyStoreDir: tt.keyDir,
			}

			err := request.Validate()
			if tt.wantErr {
				require.Error(t, err, "Validate() should return an error")
			} else {
				require.NoError(t, err, "Validate() should not return an error")
			}
		})
	}
}

func parseAPIResponse(t *testing.T, raw string) APIResponse {
	var response APIResponse
	err := json.Unmarshal([]byte(raw), &response)
	require.NoError(t, err, "binding must always return a valid JSON envelope")
	return response
}

func TestRestoreAccountAndLoginMalformedJSONReturnsErrorEnvelope(t *testing.T) {
	response := parseAPIResponse(t, restoreAccountAndLogin(`{"mnemonic": not-json}`))
	require.NotEmpty(t, response.Error, "malformed request JSON should surface an unmarshal error in the envelope, not start a restore")
}

func TestSetProfileLogMaxBackupsMalformedJSONReturnsErrorEnvelope(t *testing.T) {
	response := parseAPIResponse(t, SetProfileLogMaxBackups(`{"maxLogBackups": not-json}`))
	require.NotEmpty(t, response.Error, "malformed request JSON should surface an unmarshal error in the envelope, not touch the backend")
}

func TestRestoreAccountAndLoginKeycardRequestRejectsMnemonicSet(t *testing.T) {
	requestJSON := `{"mnemonic":"some seed words","keycard":{"keyUID":"0x1","whisperPrivateKey":"0xabc"}}`
	response := parseAPIResponse(t, restoreAccountAndLogin(requestJSON))
	require.Equal(t, requests.ErrRestoreKeycardAccountMnemonicSet.Error(), response.Error,
		"a request with keycard set must be routed to keycard validation, which forbids a mnemonic")
}

func TestRestoreAccountAndLoginKeycardRequestRejectsMissingWhisperPrivateKey(t *testing.T) {
	requestJSON := `{"keycard":{"keyUID":"0x1"}}`
	response := parseAPIResponse(t, restoreAccountAndLogin(requestJSON))
	require.Equal(t, requests.ErrRestoreKeycardAccountNoWhisperPrivateKey.Error(), response.Error,
		"keycard restore without a whisper private key must be rejected before any restore starts")
}

func TestRestoreAccountAndLoginRegularRequestRejectsMissingMnemonic(t *testing.T) {
	response := parseAPIResponse(t, restoreAccountAndLogin(`{}`))
	require.Equal(t, requests.ErrRestoreRegularAccountMnemonicMissing.Error(), response.Error,
		"a request without keycard must be routed to regular validation, which requires a mnemonic")
}

func TestConvertToKeycardAccountV2MalformedJSONReturnsErrorEnvelope(t *testing.T) {
	response := parseAPIResponse(t, convertToKeycardAccountV2(`{"keycardUID": not-json}`))
	require.NotEmpty(t, response.Error, "malformed request JSON should surface an unmarshal error in the envelope, not reach the backend")
}

func TestConvertToKeycardAccountRequestJSONFieldContract(t *testing.T) {
	requestJSON := `{"keycardUID":"kc-uid","oldPassword":"old","newPassword":"new","account":{"key-uid":"0x1"}}`
	var request requests.ConvertToKeycardAccount
	require.NoError(t, json.Unmarshal([]byte(requestJSON), &request), "documented client JSON must unmarshal")
	require.Equal(t, "kc-uid", request.KeycardUID, "keycardUID field name is the mobile client contract")
	require.Equal(t, "old", request.OldPassword, "oldPassword field name is the mobile client contract")
	require.Equal(t, "new", request.NewPassword, "newPassword field name is the mobile client contract")
	require.Equal(t, "0x1", request.Account.KeyUID, "account.key-uid field name is the mobile client contract")
}
