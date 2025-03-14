package timesource

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/beevik/ntp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// clockCompareDelta declares time required between multiple calls to time.Now
	clockCompareDelta = 100 * time.Microsecond
)

// we don't user real servers for tests, but logic depends on
// actual number of involved NTP servers.
var mockedServers = []string{"ntp1", "ntp2", "ntp3", "ntp4"}

type testCase struct {
	description     string
	servers         []string
	allowedFailures int
	responses       []queryResponse
	expected        time.Duration
	expectError     bool

	// actual attempts are mutable
	mu             sync.Mutex
	actualAttempts int
}

func (tc *testCase) query(string, ntp.QueryOptions) (*ntp.Response, error) {
	tc.mu.Lock()
	defer func() {
		tc.actualAttempts++
		tc.mu.Unlock()
	}()
	response := &ntp.Response{
		ClockOffset: tc.responses[tc.actualAttempts].Offset,
		Stratum:     1,
	}
	return response, tc.responses[tc.actualAttempts].Error
}

func newTestCases() []*testCase {
	return []*testCase{
		{
			description: "SameResponse",
			servers:     mockedServers,
			responses: []queryResponse{
				{Offset: 10 * time.Second},
				{Offset: 10 * time.Second},
				{Offset: 10 * time.Second},
				{Offset: 10 * time.Second},
			},
			expected: 10 * time.Second,
		},
		{
			description: "Median",
			servers:     mockedServers,
			responses: []queryResponse{
				{Offset: 10 * time.Second},
				{Offset: 20 * time.Second},
				{Offset: 20 * time.Second},
				{Offset: 30 * time.Second},
			},
			expected: 20 * time.Second,
		},
		{
			description: "EvenMedian",
			servers:     mockedServers[:2],
			responses: []queryResponse{
				{Offset: 10 * time.Second},
				{Offset: 20 * time.Second},
			},
			expected: 15 * time.Second,
		},
		{
			description: "Error",
			servers:     mockedServers,
			responses: []queryResponse{
				{Offset: 10 * time.Second},
				{Error: errors.New("test")},
				{Offset: 30 * time.Second},
				{Offset: 30 * time.Second},
			},
			expected:    time.Duration(0),
			expectError: true,
		},
		{
			description: "MultiError",
			servers:     mockedServers,
			responses: []queryResponse{
				{Error: errors.New("test 1")},
				{Error: errors.New("test 2")},
				{Error: errors.New("test 3")},
				{Error: errors.New("test 3")},
			},
			expected:    time.Duration(0),
			expectError: true,
		},
		{
			description:     "TolerableError",
			servers:         mockedServers,
			allowedFailures: 1,
			responses: []queryResponse{
				{Offset: 10 * time.Second},
				{Error: errors.New("test")},
				{Offset: 20 * time.Second},
				{Offset: 30 * time.Second},
			},
			expected: 20 * time.Second,
		},
		{
			description:     "NonTolerableError",
			servers:         mockedServers,
			allowedFailures: 1,
			responses: []queryResponse{
				{Offset: 10 * time.Second},
				{Error: errors.New("test")},
				{Error: errors.New("test")},
				{Error: errors.New("test")},
			},
			expected:    time.Duration(0),
			expectError: true,
		},
		{
			description:     "AllFailed",
			servers:         mockedServers,
			allowedFailures: 4,
			responses: []queryResponse{
				{Error: errors.New("test")},
				{Error: errors.New("test")},
				{Error: errors.New("test")},
				{Error: errors.New("test")},
			},
			expected:    time.Duration(0),
			expectError: true,
		},
		{
			description:     "HalfTolerable",
			servers:         mockedServers,
			allowedFailures: 2,
			responses: []queryResponse{
				{Offset: 10 * time.Second},
				{Offset: 20 * time.Second},
				{Error: errors.New("test")},
				{Error: errors.New("test")},
			},
			expected: 15 * time.Second,
		},
	}
}

func TestComputeOffset(t *testing.T) {
	for _, tc := range newTestCases() {
		t.Run(tc.description, func(t *testing.T) {
			offset, err := computeOffset(tc.query, tc.servers, tc.allowedFailures)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expected, offset)
		})
	}
}

func TestNTPTimeSource(t *testing.T) {
	for _, tc := range newTestCases() {
		t.Run(tc.description, func(t *testing.T) {
			source := &NTPTimeSource{
				servers:         tc.servers,
				allowedFailures: tc.allowedFailures,
				timeQuery:       tc.query,
				now:             time.Now,
			}
			assert.WithinDuration(t, time.Now(), source.Now(), clockCompareDelta)
			err := source.updateOffset()
			if tc.expectError {
				assert.Equal(t, errUpdateOffset, err)
			} else {
				assert.NoError(t, err)
			}
			assert.WithinDuration(t, time.Now().Add(tc.expected), source.Now(), clockCompareDelta)
		})
	}
}

func TestRunningPeriodically(t *testing.T) {
	var hits int
	var mu sync.RWMutex
	periods := make([]time.Duration, 0)

	tc := newTestCases()[0]
	fastHits := 3
	slowHits := 1

	t.Run(tc.description, func(t *testing.T) {
		source := &NTPTimeSource{
			servers:           tc.servers,
			allowedFailures:   tc.allowedFailures,
			timeQuery:         tc.query,
			fastNTPSyncPeriod: time.Duration(fastHits*10) * time.Millisecond,
			slowNTPSyncPeriod: time.Duration(slowHits*10) * time.Millisecond,
			now:               time.Now,
		}
		lastCall := time.Now()
		// we're simulating a calls to updateOffset, testing ntp calls happens
		// on NTPTimeSource specified periods (fastNTPSyncPeriod & slowNTPSyncPeriod)
		wg := sync.WaitGroup{}
		wg.Add(1)
		source.runPeriodically(context.TODO(), func() error {
			mu.Lock()
			periods = append(periods, time.Since(lastCall))
			mu.Unlock()
			hits++
			if hits < 3 {
				return errUpdateOffset
			}
			if hits == 6 {
				wg.Done()
			}
			return nil
		}, false)

		wg.Wait()

		mu.Lock()
		require.Len(t, periods, 6)
		defer mu.Unlock()
		prev := 0
		for _, period := range periods[1:3] {
			p := int(period.Seconds() * 100)
			require.True(t, fastHits <= (p-prev))
			prev = p
		}

		for _, period := range periods[3:] {
			p := int(period.Seconds() * 100)
			require.True(t, slowHits <= (p-prev))
			prev = p
		}
	})
}

func TestGetCurrentTimeInMillis(t *testing.T) {
	invokeTimes := 3
	numResponses := len(mockedServers) * invokeTimes
	responseOffset := 10 * time.Second
	tc := &testCase{
		servers:   mockedServers,
		responses: make([]queryResponse, numResponses),
		expected:  responseOffset,
	}
	for i := range tc.responses {
		tc.responses[i] = queryResponse{Offset: responseOffset}
	}

	currentTime := time.Now()
	ts := NTPTimeSource{
		servers:           tc.servers,
		allowedFailures:   tc.allowedFailures,
		timeQuery:         tc.query,
		slowNTPSyncPeriod: SlowNTPSyncPeriod,
		now: func() time.Time {
			return currentTime
		},
	}

	expectedTime := convertToMillis(currentTime.Add(responseOffset))
	n := ts.GetCurrentTimeInMillis()
	require.Equal(t, expectedTime, n)
	// test repeat invoke GetCurrentTimeInMillis
	n = ts.GetCurrentTimeInMillis()
	require.Equal(t, expectedTime, n)
	ts.Stop()

	// test invoke after stop
	n = ts.GetCurrentTimeInMillis()
	require.Equal(t, expectedTime, n)
	ts.Stop()
}

func TestGetCurrentTimeOffline(t *testing.T) {
	// covers https://github.com/status-im/status-desktop/issues/12691
	ts := &NTPTimeSource{
		servers:           defaultServers,
		allowedFailures:   DefaultMaxAllowedFailures,
		fastNTPSyncPeriod: 1 * time.Millisecond,
		slowNTPSyncPeriod: 1 * time.Second,
		timeQuery: func(string, ntp.QueryOptions) (*ntp.Response, error) {
			return nil, errors.New("offline")
		},
		now: time.Now,
	}

	// ensure there is no "panic: sync: negative WaitGroup counter"
	// when GetCurrentTime() is invoked more than once when offline
	_ = ts.GetCurrentTime()
	_ = ts.GetCurrentTime()
}

func TestSystemTimeChangeDetection(t *testing.T) {
	// Create a controlled time source with fixed time
	currentTime := time.Now()
	const timeJump = 2 * TimeChangeThreshold

	// Track timeQuery calls (which indicates UpdateOffset was called)
	timeQueryCalled := 0

	testOffset := 500 * time.Millisecond

	// Mock time function that returns our controlled time
	mockTimeNow := func() time.Time {
		return currentTime
	}

	// Create a time source with our mocks
	ts := &NTPTimeSource{
		servers:           []string{"test-server"},
		allowedFailures:   0,
		fastNTPSyncPeriod: 1 * time.Hour,
		slowNTPSyncPeriod: 1 * time.Hour,
		now:               mockTimeNow,
	}

	// Set up the timeQuery function to track calls
	ts.timeQuery = func(string, ntp.QueryOptions) (*ntp.Response, error) {
		timeQueryCalled++
		return &ntp.Response{
			ClockOffset: testOffset,
			Stratum:     1,
		}, nil
	}

	// Initialize time tracking fields
	ts.latestOffset = testOffset
	ts.lastMonotonic = currentTime

	// Test case 1: No time change
	// -------------------------------------------------------------------------
	// Reset the counter before this test case
	timeQueryCalled = 0

	time1 := ts.Now()
	assert.Equal(t, currentTime.Add(testOffset), time1,
		"Time should be adjusted by offset with no time change")
	assert.Equal(t, currentTime, ts.lastMonotonic,
		"lastMonotonic should be updated after Now() call")
	assert.Equal(t, 0, timeQueryCalled,
		"UpdateOffset should not be called when no time change is detected")

	// Test case 2: Small time change (below threshold)
	// -------------------------------------------------------------------------
	// Reset the counter before this test case
	timeQueryCalled = 0

	// Advance time by a small amount
	oldTime := currentTime
	currentTime = currentTime.Add(500 * time.Millisecond) // Below TimeChangeThreshold (1s)

	// Set time tracking fields to simulate a small time difference
	ts.lastMonotonic = oldTime

	// Call Now() with small time change
	time2 := ts.Now()
	assert.Equal(t, currentTime.Add(testOffset), time2,
		"Time should be adjusted by offset with small time change")
	assert.Equal(t, oldTime, ts.lastMonotonic,
		"lastMonotonic should not be updated after small time change")
	assert.Equal(t, 0, timeQueryCalled,
		"UpdateOffset should not be called when time change is below threshold")

	// Test case 3: Large backward time change
	// -------------------------------------------------------------------------
	// Reset the counter before this test case
	timeQueryCalled = 0

	// Advance time significantly
	oldTime = currentTime
	currentTime = currentTime.Add(1 * time.Minute)

	// Simulate wall clock being set backward
	ts.lastMonotonic = oldTime.Add(-timeJump)

	// Call Now() which should detect the time change
	time3 := ts.Now()
	assert.Equal(t, currentTime.Add(testOffset), time3,
		"Time should be adjusted by offset after backward time change")
	assert.Equal(t, currentTime, ts.lastMonotonic,
		"lastMonotonic should be updated after backward time change")
	assert.Equal(t, 1, timeQueryCalled,
		"UpdateOffset should be called when backward time change is detected")

	// Test case 4: Large forward time change
	// -------------------------------------------------------------------------
	// Reset the counter before this test case
	timeQueryCalled = 0

	// Advance time significantly
	oldTime = currentTime
	currentTime = currentTime.Add(1 * time.Minute)

	// Simulate wall clock being set forward
	ts.lastMonotonic = oldTime.Add(timeJump)

	// Call Now() which should detect the time change
	time4 := ts.Now()
	assert.Equal(t, currentTime.Add(testOffset), time4,
		"Time should be adjusted by offset after forward time change")
	assert.Equal(t, currentTime, ts.lastMonotonic,
		"lastMonotonic should be updated after forward time change")
	assert.Equal(t, 1, timeQueryCalled,
		"UpdateOffset should be called when forward time change is detected")
}

func TestTimeTrackingInitialization(t *testing.T) {
	// Create a fixed time for testing
	fixedTime := time.Now()

	// Create a mock time function that returns our fixed time
	mockTimeNow := func() time.Time {
		return fixedTime
	}

	// Create a mock query function that always succeeds
	mockQuery := func(string, ntp.QueryOptions) (*ntp.Response, error) {
		return &ntp.Response{
			ClockOffset: 100 * time.Millisecond,
			Stratum:     1,
		}, nil
	}

	// Create the time source with our controlled functions
	ts := &NTPTimeSource{
		servers:           mockedServers,
		allowedFailures:   DefaultMaxAllowedFailures,
		fastNTPSyncPeriod: 1 * time.Hour, // Use long periods to avoid actual periodic updates during test
		slowNTPSyncPeriod: 1 * time.Hour,
		timeQuery:         mockQuery,
		now:               mockTimeNow,
	}

	// Verify that time tracking fields are initially zero
	assert.True(t, ts.lastMonotonic.IsZero(), "lastMonotonic should be zero before Start()")

	// Start the time source
	err := ts.Start(context.TODO())
	assert.NoError(t, err, "Start should not return an error")

	defer func() {
		ts.Stop()
	}()

	// Verify that time tracking fields are initialized
	assert.Equal(t, fixedTime, ts.lastMonotonic, "lastMonotonic should be initialized to current time")

	// Verify that the time source is marked as started
	assert.True(t, ts.started, "Time source should be marked as started")
}

func TestTimeChangeDetectionSkippedWhenNotInitialized(t *testing.T) {
	// Create a fixed time for testing
	fixedTime := time.Now()

	var offsetUpdateAttempted bool

	// Create a mock time function that returns our fixed time
	mockTimeNow := func() time.Time {
		return fixedTime
	}

	// Create a mock query function that tracks if it was called
	mockQuery := func(string, ntp.QueryOptions) (*ntp.Response, error) {
		offsetUpdateAttempted = true
		return &ntp.Response{
			ClockOffset: 100 * time.Millisecond,
			Stratum:     1,
		}, nil
	}

	// Create the time source with our controlled functions
	ts := &NTPTimeSource{
		servers:           mockedServers,
		allowedFailures:   DefaultMaxAllowedFailures,
		fastNTPSyncPeriod: 1 * time.Hour,
		slowNTPSyncPeriod: 1 * time.Hour,
		timeQuery:         mockQuery,
		now:               mockTimeNow,
		latestOffset:      50 * time.Millisecond, // Set an initial offset
	}

	// Ensure time tracking fields are zero (not initialized)
	assert.True(t, ts.lastMonotonic.IsZero(), "lastMonotonic should be zero initially")

	// Call Now() which should skip time change detection
	result := ts.Now()

	// Verify that the result is correctly adjusted by the offset
	expectedTime := fixedTime.Add(ts.latestOffset)
	assert.Equal(t, expectedTime, result, "Time should be adjusted by offset")

	// Verify that UpdateOffset was not called
	assert.False(t, offsetUpdateAttempted, "UpdateOffset should not be called when time tracking is not initialized")
}

func TestTimeChangeDetectionWithUpdateFailure(t *testing.T) {
	// Create a controlled time source for testing
	var (
		currentTime = time.Now()
		mockOffset  = 500 * time.Millisecond
		timeJump    = 2 * time.Second // Greater than TimeChangeThreshold (1s)
	)

	// Create a mock time function that we can control
	mockTimeNow := func() time.Time {
		return currentTime
	}

	// Create a mock query function that always fails
	mockQuery := func(string, ntp.QueryOptions) (*ntp.Response, error) {
		return nil, errors.New("network error")
	}

	// Create the time source with our controlled functions
	ts := &NTPTimeSource{
		servers:           mockedServers,
		allowedFailures:   DefaultMaxAllowedFailures,
		fastNTPSyncPeriod: 1 * time.Hour,
		slowNTPSyncPeriod: 1 * time.Hour,
		timeQuery:         mockQuery,
		now:               mockTimeNow,
		latestOffset:      mockOffset, // Set an initial offset
	}

	// Initialize the time tracking fields
	ts.lastMonotonic = currentTime

	// First call to Now() with no time change
	time1 := ts.Now()
	assert.Equal(t, currentTime.Add(mockOffset), time1, "Time should be adjusted by offset")

	// Simulate a time change
	oldTime := currentTime
	currentTime = currentTime.Add(1 * time.Minute) // Advance current time by 1 minute

	// Set lastMonotonic to simulate that more time has passed in reality
	ts.lastMonotonic = oldTime.Add(-timeJump)

	// Call Now() which should detect the time change and attempt to update offset
	time2 := ts.Now()

	// Even though UpdateOffset fails, Now() should still return a time adjusted by the previous offset
	assert.Equal(t, currentTime.Add(mockOffset), time2, "Time should still be adjusted by original offset after failed update")

	// Verify that the time tracking fields are updated even when UpdateOffset fails
	assert.Equal(t, currentTime, ts.lastMonotonic, "lastMonotonic should be updated even when UpdateOffset fails")
}
