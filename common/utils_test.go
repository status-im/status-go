package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGo(t *testing.T) {
	// Test that Go recovers from panic
	paniced := false
	panicErr := "test panic"
	oldDefaultPanicFunc := defaultPanicFunc
	defer func() {
		defaultPanicFunc = oldDefaultPanicFunc
	}()
	defaultPanicFunc = func(err any) {
		require.NotNil(t, err)
		paniced = true
	}
	recovered := make(chan bool, 1)
	Go(func() {
		panic(panicErr)
	}, func(err any) {
		recovered <- true
		if err != panicErr {
			t.Errorf("Expected panic with 'test panic', got %v", err)
		}
	})

	timeout := 5 * time.Second
	select {
	case <-recovered:
		// Panic was recovered successfully
	case <-time.After(timeout):
		t.Error("Go did not recover from panic within the timeout")
	}

	require.True(t, paniced)

	// Test that Go executes normally when no panic occurs
	executed := make(chan bool, 1)
	Go(func() {
		executed <- true
	})

	select {
	case <-executed:
		// Function executed successfully
	case <-time.After(timeout):
		t.Error("Go did not execute the function within the timeout")
	}
}
