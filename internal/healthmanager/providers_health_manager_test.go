package healthmanager

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/internal/healthmanager/aggregator"
	"github.com/status-im/status-go/internal/healthmanager/rpcstatus"
)

type ProvidersHealthManagerSuite struct {
	suite.Suite
	phm *ProvidersHealthManager
}

// newTestProvidersHealthManager builds a ProvidersHealthManager with a short Down debounce for tests.
func newTestProvidersHealthManager(chainID uint64) *ProvidersHealthManager {
	phm := NewProvidersHealthManager(chainID)
	phm.downDebounce = 20 * time.Millisecond
	return phm
}

// SetupTest initializes the ProvidersHealthManager before each test
func (s *ProvidersHealthManagerSuite) SetupTest() {
	s.phm = NewProvidersHealthManager(1)
	s.phm.downDebounce = 20 * time.Millisecond
}

// Helper method to update providers and wait for a notification on the given channel
func (s *ProvidersHealthManagerSuite) updateAndWait(ch <-chan struct{}, statuses []rpcstatus.RpcProviderCallStatus, expectedChainStatus rpcstatus.StatusType, timeout time.Duration) {
	s.phm.Update(context.Background(), statuses)

	select {
	case <-ch:
		// Received notification
	case <-time.After(timeout):
		s.Fail("Timeout waiting for chain status update")
	}

	s.assertChainStatus(expectedChainStatus)
}

// Helper method to update providers and wait for a notification on the given channel
func (s *ProvidersHealthManagerSuite) updateAndExpectNoNotification(ch <-chan struct{}, statuses []rpcstatus.RpcProviderCallStatus, expectedChainStatus rpcstatus.StatusType, timeout time.Duration) {
	s.phm.Update(context.Background(), statuses)

	select {
	case <-ch:
		s.Fail("Unexpected status update")
	case <-time.After(timeout):
		// No notification as expected
	}

	s.assertChainStatus(expectedChainStatus)
}

// Helper method to assert the current chain status
func (s *ProvidersHealthManagerSuite) assertChainStatus(expected rpcstatus.StatusType) {
	actual := s.phm.Status().Status
	s.Equal(expected, actual, fmt.Sprintf("Expected chain status to be %s", expected))
}

func (s *ProvidersHealthManagerSuite) TestInitialStatus() {
	s.assertChainStatus(rpcstatus.StatusUnknown)
}

func (s *ProvidersHealthManagerSuite) TestUpdateProviderStatuses() {
	ch := s.phm.Subscribe()
	defer s.phm.Unsubscribe(ch)

	now := time.Now()
	duration1 := 100 * time.Millisecond
	duration2 := 200 * time.Millisecond

	s.updateAndWait(ch, []rpcstatus.RpcProviderCallStatus{
		{
			Name:      "Provider1",
			Timestamp: now,
			Err:       nil,
			StartTime: now.Add(-duration1),
		},
		{
			Name:      "Provider2",
			Timestamp: now,
			Err:       context.DeadlineExceeded,
			StartTime: now.Add(-duration2),
		},
	}, rpcstatus.StatusUp, time.Second)

	statusMap := s.phm.GetStatuses()
	s.Len(statusMap, 2, "Expected 2 provider statuses")
	s.Equal(rpcstatus.StatusUp, statusMap["Provider1"].Status, "Expected Provider1 status to be Up")
	s.Equal(rpcstatus.StatusDown, statusMap["Provider2"].Status, "Expected Provider2 status to be Down")

	// Verify metrics for Provider1
	s.Equal(duration1, statusMap["Provider1"].TotalDuration, "Expected Provider1 TotalDuration to match")
	s.Equal(int64(1), statusMap["Provider1"].TotalRequests, "Expected Provider1 TotalRequests to be 1")
	s.Equal(int64(0), statusMap["Provider1"].TotalTimeoutCount, "Expected Provider1 TotalTimeoutCount to be 0")

	// Verify metrics for Provider2
	s.Equal(duration2, statusMap["Provider2"].TotalDuration, "Expected Provider2 TotalDuration to match")
	s.Equal(int64(1), statusMap["Provider2"].TotalRequests, "Expected Provider2 TotalRequests to be 1")
	s.Equal(int64(1), statusMap["Provider2"].TotalTimeoutCount, "Expected Provider2 TotalTimeoutCount to be 1")

	// Update with additional metrics
	laterTime := now.Add(1 * time.Minute)
	duration3 := 150 * time.Millisecond

	s.updateAndExpectNoNotification(ch, []rpcstatus.RpcProviderCallStatus{
		{
			Name:      "Provider1",
			Timestamp: laterTime,
			Err:       nil,
			StartTime: laterTime.Add(-duration3),
		},
	}, rpcstatus.StatusUp, 100*time.Millisecond)

	// Verify accumulated metrics for Provider1
	statusMap = s.phm.GetStatuses()
	s.Equal(duration1+duration3, statusMap["Provider1"].TotalDuration, "Expected Provider1 TotalDuration to accumulate")
	s.Equal(int64(2), statusMap["Provider1"].TotalRequests, "Expected Provider1 TotalRequests to be 2")
	s.Equal(int64(0), statusMap["Provider1"].TotalTimeoutCount, "Expected Provider1 TotalTimeoutCount to remain 0")
}

func (s *ProvidersHealthManagerSuite) TestChainStatusUpdatesOnce() {
	ch := s.phm.Subscribe()
	defer s.phm.Unsubscribe(ch)
	s.assertChainStatus(rpcstatus.StatusUnknown)

	// Update providers to Down
	statuses := []rpcstatus.RpcProviderCallStatus{
		{Name: "Provider1", Timestamp: time.Now(), Err: errors.New("error")},
		{Name: "Provider2", Timestamp: time.Now(), Err: nil},
	}
	s.updateAndWait(ch, statuses, rpcstatus.StatusUp, time.Second)
	s.updateAndExpectNoNotification(ch, statuses, rpcstatus.StatusUp, 10*time.Millisecond)
}

func (s *ProvidersHealthManagerSuite) TestSubscribeReceivesOnlyOnChange() {
	ch := s.phm.Subscribe()
	defer s.phm.Unsubscribe(ch)

	// Update provider to Up and wait for notification
	upStatuses := []rpcstatus.RpcProviderCallStatus{
		{Name: "Provider1", Timestamp: time.Now(), Err: nil},
	}
	s.updateAndWait(ch, upStatuses, rpcstatus.StatusUp, time.Second)

	// Update provider to Down and wait for notification
	downStatuses := []rpcstatus.RpcProviderCallStatus{
		{Name: "Provider1", Timestamp: time.Now(), Err: errors.New("some critical error")},
	}
	s.updateAndWait(ch, downStatuses, rpcstatus.StatusDown, time.Second)

	s.updateAndExpectNoNotification(ch, downStatuses, rpcstatus.StatusDown, 10*time.Millisecond)
}

func (s *ProvidersHealthManagerSuite) TestConcurrency() {
	var wg sync.WaitGroup
	providerCount := 1000

	s.phm.Update(context.Background(), []rpcstatus.RpcProviderCallStatus{
		{Name: "ProviderUp", Timestamp: time.Now(), Err: nil},
	})

	ctx := context.Background()
	for i := 0; i < providerCount-1; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			providerName := fmt.Sprintf("Provider%d", i)
			var err error
			if i%2 == 0 {
				err = errors.New("error")
			}
			s.phm.Update(ctx, []rpcstatus.RpcProviderCallStatus{
				{Name: providerName, Timestamp: time.Now(), Err: err},
			})
		}(i)
	}
	wg.Wait()

	statuses := s.phm.GetStatuses()
	s.Len(statuses, providerCount, "Expected 1000 provider statuses")

	chainStatus := s.phm.Status().Status
	s.Equal(chainStatus, rpcstatus.StatusUp, "Expected chain status to be either Up or Down")
}

// TestPanicWithNilAggregator tests that Update handles nil aggregator without panicking
// (refs https://github.com/status-im/status-go/issues/6462)
func (s *ProvidersHealthManagerSuite) TestPanicWithNilAggregator() {
	// Create ProvidersHealthManager with nil aggregator for testing
	phm := &ProvidersHealthManager{
		chainID:             1,
		aggregator:          nil, // Explicitly set to nil
		subscriptionManager: NewSubscriptionManager(),
	}

	// Test data
	callStatuses := []rpcstatus.RpcProviderCallStatus{
		{Name: "Provider1", Timestamp: time.Now(), Err: nil},
	}

	// Call should not panic due to our nil check
	s.NotPanics(func() {
		phm.Update(context.Background(), callStatuses)
	}, "Update should not panic with nil aggregator thanks to nil check")
}

// TestPanicWithNilSubscriptionManager tests that emitChainStatus handles nil subscriptionManager without panicking
// (refs https://github.com/status-im/status-go/issues/6462)
func (s *ProvidersHealthManagerSuite) TestPanicWithNilSubscriptionManager() {
	// Create ProvidersHealthManager with nil subscriptionManager for testing
	phm := &ProvidersHealthManager{
		chainID:             1,
		aggregator:          aggregator.NewAggregator("1"),
		subscriptionManager: nil, // Explicitly set to nil
	}

	// Call should not panic due to our nil check
	s.NotPanics(func() {
		phm.emitChainStatus(context.Background())
	}, "emitChainStatus should not panic with nil subscriptionManager thanks to nil check")
}

// TestReset verifies that Reset sets lastStatus to nil and creates a new aggregator
func (s *ProvidersHealthManagerSuite) TestReset() {
	// First, update the providers to establish a known state
	ch := s.phm.Subscribe()
	defer s.phm.Unsubscribe(ch)

	// Update provider to Up and wait for notification
	upStatuses := []rpcstatus.RpcProviderCallStatus{
		{Name: "Provider1", Timestamp: time.Now(), Err: nil},
	}
	s.updateAndWait(ch, upStatuses, rpcstatus.StatusUp, time.Second)

	// Verify providers are in statuses
	statusesBeforeReset := s.phm.GetStatuses()
	s.Len(statusesBeforeReset, 1, "Expected 1 provider status before reset")
	s.Contains(statusesBeforeReset, "Provider1", "Expected Provider1 in statuses before reset")

	// Capture the aggregator and lastStatus references before reset
	originalAggregator := s.phm.aggregator

	// Access the lastStatus field directly
	s.NotNil(s.phm.lastStatus, "lastStatus should not be nil before reset")

	// Reset the providers health manager
	s.phm.Reset()

	// Verify lastStatus is nil
	s.Nil(s.phm.lastStatus, "lastStatus should be nil after reset")

	// Verify aggregator was recreated (different instance)
	s.NotSame(originalAggregator, s.phm.aggregator, "Aggregator should be a new instance after reset")

	// Verify statuses are cleared
	statusesAfterReset := s.phm.GetStatuses()
	s.Empty(statusesAfterReset, "Expected no provider statuses after reset")

	// Verify chain ID remains the same
	s.Equal(uint64(1), s.phm.ChainID(), "Chain ID should remain unchanged after reset")
}

// expectNotification fails if no notification arrives on ch within timeout.
func (s *ProvidersHealthManagerSuite) expectNotification(ch <-chan struct{}, timeout time.Duration) {
	select {
	case <-ch:
	case <-time.After(timeout):
		s.Fail("Timeout waiting for chain status notification")
	}
}

// expectNoNotification fails if any notification arrives on ch within timeout.
func (s *ProvidersHealthManagerSuite) expectNoNotification(ch <-chan struct{}, timeout time.Duration) {
	select {
	case <-ch:
		s.Fail("Unexpected chain status notification")
	case <-time.After(timeout):
	}
}

// testDownDebounce is deliberately far longer than the waits the debounce tests perform between
// updates. The debounce only has to outlast the test's own bookkeeping, and on a loaded CI agent a
// 40ms wait can take several times that; with a short debounce the pending Down timer fires in the
// middle of the scenario and the test reports a spurious notification. Claims about *when* a Down
// lands are made by measuring elapsed time instead of sleeping for a hand-tuned fraction of it.
const testDownDebounce = time.Second

// expectNoPendingNotification fails if a notification is already queued. Emission is synchronous
// inside Update, so checking that a given update emitted nothing needs no waiting at all and cannot
// be tripped by a slow scheduler.
func (s *ProvidersHealthManagerSuite) expectNoPendingNotification(ch <-chan struct{}) {
	select {
	case <-ch:
		s.Fail("Unexpected chain status notification")
	default:
	}
}

// expectNotificationNoEarlierThan waits for a notification and fails if it arrived sooner than
// minElapsed after start. Scheduling delays can only push the notification later, which passes.
func (s *ProvidersHealthManagerSuite) expectNotificationNoEarlierThan(ch <-chan struct{}, start time.Time, minElapsed time.Duration) {
	s.expectNotification(ch, minElapsed+5*time.Second)
	s.GreaterOrEqual(time.Since(start), minElapsed, "chain status notification arrived before the debounce elapsed")
}

// expectNoNotificationPast fails if a notification arrives before deadline, plus a margin, so that
// a timer which should have been stopped is observed past the moment it would have fired.
func (s *ProvidersHealthManagerSuite) expectNoNotificationPast(ch <-chan struct{}, deadline time.Time) {
	s.expectNoNotification(ch, time.Until(deadline)+200*time.Millisecond)
}

func downStatus(name string) []rpcstatus.RpcProviderCallStatus {
	return []rpcstatus.RpcProviderCallStatus{{Name: name, Timestamp: time.Now(), Err: errors.New("critical error")}}
}

func upStatus(name string) []rpcstatus.RpcProviderCallStatus {
	return []rpcstatus.RpcProviderCallStatus{{Name: name, Timestamp: time.Now(), Err: nil}}
}

func (s *ProvidersHealthManagerSuite) TestDownEmissionIsDebounced() {
	s.phm.downDebounce = testDownDebounce
	ch := s.phm.Subscribe()
	defer s.phm.Unsubscribe(ch)

	s.phm.Update(context.Background(), upStatus("Provider1"))
	s.expectNotification(ch, time.Second)
	s.assertChainStatus(rpcstatus.StatusUp)

	downAt := time.Now()
	s.phm.Update(context.Background(), downStatus("Provider1"))

	// The aggregated status flips right away; only the notification is deferred.
	s.assertChainStatus(rpcstatus.StatusDown)
	s.expectNoPendingNotification(ch)

	s.expectNotificationNoEarlierThan(ch, downAt, testDownDebounce)
	s.assertChainStatus(rpcstatus.StatusDown)

	// A repeated Down is not a transition, so it arms nothing and emits nothing.
	s.phm.Update(context.Background(), downStatus("Provider1"))
	s.expectNoNotification(ch, 50*time.Millisecond)
}

func (s *ProvidersHealthManagerSuite) TestShortDownDoesNotEmit() {
	s.phm.downDebounce = testDownDebounce
	ch := s.phm.Subscribe()
	defer s.phm.Unsubscribe(ch)

	s.phm.Update(context.Background(), upStatus("Provider1"))
	s.expectNotification(ch, time.Second)

	downAt := time.Now()
	s.phm.Update(context.Background(), downStatus("Provider1"))
	time.Sleep(50 * time.Millisecond)
	s.phm.Update(context.Background(), upStatus("Provider1"))
	s.expectNoPendingNotification(ch)

	// Watch past the moment the pending timer would have fired had the recovery not stopped it.
	s.expectNoNotificationPast(ch, downAt.Add(testDownDebounce))
	s.assertChainStatus(rpcstatus.StatusUp)
}

func (s *ProvidersHealthManagerSuite) TestRecoveryEmitsImmediately() {
	s.phm.downDebounce = 100 * time.Millisecond
	ch := s.phm.Subscribe()
	defer s.phm.Unsubscribe(ch)

	s.phm.Update(context.Background(), upStatus("Provider1"))
	s.expectNotification(ch, time.Second)
	s.phm.Update(context.Background(), downStatus("Provider1"))
	s.expectNotification(ch, time.Second)
	s.assertChainStatus(rpcstatus.StatusDown)

	s.phm.Update(context.Background(), upStatus("Provider1"))
	s.expectNotification(ch, 50*time.Millisecond)
	s.assertChainStatus(rpcstatus.StatusUp)
}

func (s *ProvidersHealthManagerSuite) TestDownDebounceResetsAfterSilentRecovery() {
	// Separates the two Down transitions far enough to tell their timers apart, while staying far
	// below the debounce so a slow agent cannot let the first timer fire before the recovery stops it.
	const silentGap = 200 * time.Millisecond

	s.phm.downDebounce = testDownDebounce
	ch := s.phm.Subscribe()
	defer s.phm.Unsubscribe(ch)

	s.phm.Update(context.Background(), upStatus("Provider1"))
	s.expectNotification(ch, time.Second)

	s.phm.Update(context.Background(), downStatus("Provider1"))
	time.Sleep(silentGap)

	// This recovery does not emit (lastStatus is still Up), but must still stop the previous timer.
	s.phm.Update(context.Background(), upStatus("Provider1"))
	s.expectNoPendingNotification(ch)

	secondDownAt := time.Now()
	s.phm.Update(context.Background(), downStatus("Provider1"))

	// Had the first timer survived the recovery, the Down would land a silentGap earlier than a
	// full debounce after this second transition.
	s.expectNotificationNoEarlierThan(ch, secondDownAt, testDownDebounce)
	s.assertChainStatus(rpcstatus.StatusDown)
}

func (s *ProvidersHealthManagerSuite) TestPauseStopsPendingDownTimer() {
	s.phm.downDebounce = testDownDebounce
	ch := s.phm.Subscribe()
	defer s.phm.Unsubscribe(ch)

	s.phm.Update(context.Background(), upStatus("Provider1"))
	s.expectNotification(ch, time.Second)

	downAt := time.Now()
	s.phm.Update(context.Background(), downStatus("Provider1"))
	s.expectNoPendingNotification(ch)

	s.phm.Pause()
	s.expectNoNotificationPast(ch, downAt.Add(testDownDebounce))
}

func (s *ProvidersHealthManagerSuite) TestResumeCoalescesPausedUpdates() {
	s.phm.downDebounce = testDownDebounce
	ch := s.phm.Subscribe()
	defer s.phm.Unsubscribe(ch)

	s.phm.Update(context.Background(), upStatus("Provider1"))
	s.expectNotification(ch, time.Second)
	s.assertChainStatus(rpcstatus.StatusUp)

	pausedDownAt := time.Now()
	s.phm.Pause()
	s.phm.Update(context.Background(), downStatus("Provider1"))

	// Nothing is armed while paused, so nothing fires even past a full debounce.
	s.expectNoNotificationPast(ch, pausedDownAt.Add(testDownDebounce))

	// Resume should emit one coalesced Down, and only after the debounce.
	resumedAt := time.Now()
	s.phm.Resume()
	s.expectNoPendingNotification(ch)
	s.expectNotificationNoEarlierThan(ch, resumedAt, testDownDebounce)
	s.assertChainStatus(rpcstatus.StatusDown)
}

func TestProvidersHealthManagerSuite(t *testing.T) {
	suite.Run(t, new(ProvidersHealthManagerSuite))
}
