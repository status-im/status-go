// Package pausabletest checks that pausable services return from Pause and
// Resume quickly, whatever the network does. Clients call both on the UI thread.
package pausabletest

import (
	"testing"
	"time"

	"github.com/status-im/status-go/internal/panics"
)

// Budget is the longest a lifecycle call may take on the UI thread.
const Budget = 250 * time.Millisecond

// RequireWithin runs op and fails the test when it takes longer than budget.
// It gives up waiting after a hard cap so a hung op cannot stall the suite.
// Every call logs a "pausable-latency" line with the measured duration.
func RequireWithin(t testing.TB, service, op string, budget time.Duration, f func() error) {
	t.Helper()
	hardCap := max(10*budget, 3*time.Second)
	start := time.Now()
	done := make(chan error, 1)
	go func() {
		defer panics.LogOnPanic()
		done <- f()
	}()

	select {
	case err := <-done:
		took := time.Since(start)
		t.Logf("pausable-latency service=%s op=%s took=%s budget=%s", service, op, took, budget)
		if err != nil {
			t.Errorf("%s %s: %v", service, op, err)
		}
		if took > budget {
			t.Errorf("%s %s took %s, budget %s", service, op, took, budget)
		}
	case <-time.After(hardCap):
		t.Logf("pausable-latency service=%s op=%s took=>%s budget=%s", service, op, hardCap, budget)
		t.Fatalf("%s %s did not return within %s (budget %s)", service, op, hardCap, budget)
	}
}
