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
	"github.com/waku-org/go-waku/waku/v2/utils"
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
			_, cancel := context.WithCancel(context.Background())
			source := &NTPTimeSource{
				servers:         tc.servers,
				allowedFailures: tc.allowedFailures,
				timeQuery:       tc.query,
				log:             utils.Logger(),
				cancel:          cancel,
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
		ctx, cancel := context.WithCancel(context.Background())
		source := &NTPTimeSource{
			servers:           tc.servers,
			allowedFailures:   tc.allowedFailures,
			timeQuery:         tc.query,
			fastNTPSyncPeriod: time.Duration(fastHits*10) * time.Millisecond,
			slowNTPSyncPeriod: time.Duration(slowHits*10) * time.Millisecond,
			log:               utils.Logger(),
			cancel:            cancel,
		}
		lastCall := time.Now()
		// we're simulating a calls to updateOffset, testing ntp calls happens
		// on NTPTimeSource specified periods (fastNTPSyncPeriod & slowNTPSyncPeriod)
		err := source.runPeriodically(ctx, func() error {
			mu.Lock()
			periods = append(periods, time.Since(lastCall))
			mu.Unlock()
			hits++
			if hits < 3 {
				return errUpdateOffset
			}
			if hits == 6 {
				source.wg.Done()
			}
			return nil
		})

		source.wg.Wait()
		require.NoError(t, err)

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
