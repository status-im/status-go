package statusgo

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

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

func TestSwitchToLowMemoryMode(t *testing.T) {
	// Reset the lastGCTime to ensure we can trigger GC
	lastGCTimeLock.Lock()
	lastGCTime = time.Time{}
	lastGCTimeLock.Unlock()

	// First call should succeed
	result := SwitchToLowMemoryMode()

	var response APIResponse
	err := json.Unmarshal([]byte(result), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error != "" {
		t.Errorf("Expected no error on first call, got: %s", response.Error)
	}

	// Second immediate call should be skipped due to rate limiting
	result = SwitchToLowMemoryMode()

	err = json.Unmarshal([]byte(result), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error == "" {
		t.Error("Expected error on second immediate call, but got none")
	}

	// The error should indicate that GC was skipped
	if response.Error != "skipping GC because it was called too recently" {
		t.Errorf("Unexpected error message: %s", response.Error)
	}

	// Test concurrent access
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			SwitchToLowMemoryMode()
		}()
	}
	wg.Wait()

	// After waiting, we should be able to call it again
	lastGCTimeLock.Lock()
	lastGCTime = time.Now().Add(-(gcInterval + time.Second))
	lastGCTimeLock.Unlock()

	result = SwitchToLowMemoryMode()

	err = json.Unmarshal([]byte(result), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error != "" {
		t.Errorf("Expected no error after waiting, got: %s", response.Error)
	}
}

// TestReleaseMemory is a simple test to ensure the function doesn't panic
func TestReleaseMemory(t *testing.T) {
	releaseMemory()
}
