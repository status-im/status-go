package statusgo

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brianvoe/gofakeit/v6"

	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/signal"
)

type testSignalHandler struct {
	receivedSignal string
}

func (t *testSignalHandler) HandleSignal(data string) {
	t.receivedSignal = data
}

func TestSetMobileSignalHandler(t *testing.T) {
	// Setup
	handler := &testSignalHandler{}
	SetMobileSignalHandler(handler)
	t.Cleanup(signal.ResetMobileSignalHandler)

	// Test data
	testAccount := &multiaccounts.Account{Name: "test"}
	testSettings := &settings.Settings{KeyUID: "0x1"}
	testEnsUsernames := json.RawMessage(`{"test": "test"}`)

	// Action
	signal.SendLoggedIn(testAccount, testSettings, testEnsUsernames, nil)

	// Assertions
	require.Contains(t, handler.receivedSignal, `"key-uid":"0x1"`, "Signal should contain the correct KeyUID")
	require.Contains(t, handler.receivedSignal, `"name":"test"`, "Signal should contain the correct account name")
	require.Contains(t, handler.receivedSignal, `"ensUsernames":{"test":"test"}`, "Signal should contain the correct ENS usernames")
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
