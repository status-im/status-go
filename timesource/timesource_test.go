package timesource

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/beevik/ntp"
	"github.com/stretchr/testify/assert"
)

const (
	// clockCompareDelta declares time required between multiple calls to time.Now
	clockCompareDelta = 30 * time.Microsecond
)

type testCase struct {
	description string
	attempts    int
	responses   []queryResponse
	expected    time.Duration
	expectError bool

	// actual attempts are mutable
	mu             sync.Mutex
	actualAttempts int
}

func (tc *testCase) query(string) (*ntp.Response, error) {
	tc.mu.Lock()
	defer func() {
		tc.actualAttempts++
		tc.mu.Unlock()
	}()
	response := &ntp.Response{ClockOffset: tc.responses[tc.actualAttempts].Offset}
	return response, tc.responses[tc.actualAttempts].Error
}

func newTestCases() []*testCase {
	return []*testCase{
		{
			description: "SameResponse",
			attempts:    3,
			responses: []queryResponse{
				{Offset: 10 * time.Second},
				{Offset: 10 * time.Second},
				{Offset: 10 * time.Second},
			},
			expected: 10 * time.Second,
		},
		{
			description: "Median",
			attempts:    3,
			responses: []queryResponse{
				{Offset: 10 * time.Second},
				{Offset: 20 * time.Second},
				{Offset: 30 * time.Second},
			},
			expected: 20 * time.Second,
		},
		{
			description: "EvenMedian",
			attempts:    2,
			responses: []queryResponse{
				{Offset: 10 * time.Second},
				{Offset: 20 * time.Second},
			},
			expected: 15 * time.Second,
		},
		{
			description: "Error",
			attempts:    3,
			responses: []queryResponse{
				{Offset: 10 * time.Second},
				{Error: errors.New("test")},
				{Offset: 30 * time.Second},
			},
			expected:    time.Duration(0),
			expectError: true,
		},
		{
			description: "MultiError",
			attempts:    3,
			responses: []queryResponse{
				{Error: errors.New("test 1")},
				{Error: errors.New("test 2")},
				{Error: errors.New("test 3")},
			},
			expected:    time.Duration(0),
			expectError: true,
		},
	}
}

func TestComputeOffset(t *testing.T) {
	for _, tc := range newTestCases() {
		t.Run(tc.description, func(t *testing.T) {
			offset, err := computeOffset(tc.query, "", tc.attempts)
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
				attempts:  tc.attempts,
				timeQuery: tc.query,
			}
			assert.WithinDuration(t, time.Now(), source.Now(), clockCompareDelta)
			source.updateOffset()
			assert.WithinDuration(t, time.Now().Add(tc.expected), source.Now(), clockCompareDelta)
		})
	}
}
