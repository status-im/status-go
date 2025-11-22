package statusgo

import (
	"testing"

	"github.com/stretchr/testify/require"

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

	// Action
	signal.SendLoggedIn(testAccount, testSettings, nil)

	// Assertions
	require.Contains(t, handler.receivedSignal, `"key-uid":"0x1"`, "Signal should contain the correct KeyUID")
	require.Contains(t, handler.receivedSignal, `"name":"test"`, "Signal should contain the correct account name")
}

func TestIntendedPanic(t *testing.T) {
	message := gofakeit.LetterN(5)
	require.PanicsWithError(t, message, func() {
		IntendedPanic(message)
	})
}
