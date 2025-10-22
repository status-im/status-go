package statusgo

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/brianvoe/gofakeit/v6"

	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/settings"
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

func TestEscapeString(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	const string_2 = `C:\Users\anast\status-desktop\`
	const string_3 = `C:\\Users\\anast\\status-desktop\\`
	logger.Info("test log", zap.String("string_2", string_2), zap.String("string_3", string_3))
}
