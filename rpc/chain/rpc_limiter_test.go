package chain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func setupTest() (*InMemRequestsStorage, RequestLimiter) {
	storage := NewInMemRequestsStorage()
	rl := NewRequestLimiter(storage)
	return storage, rl
}

func TestSetMaxRequests(t *testing.T) {
	storage, rl := setupTest()

	// Define test inputs
	tag := "testTag"
	maxRequests := 10
	interval := time.Second

	// Call the SetMaxRequests method
	err := rl.SetMaxRequests(tag, maxRequests, interval)
	require.NoError(t, err)

	// Verify that the data was saved to storage correctly
	data, err := storage.Get(tag)
	require.NoError(t, err)
	require.Equal(t, tag, data.Tag)
	require.Equal(t, interval, data.Period)
	require.Equal(t, maxRequests, data.MaxReqs)
	require.Equal(t, 0, data.NumReqs)
}

func TestGetMaxRequests(t *testing.T) {
	storage, rl := setupTest()

	data := RequestData{
		Tag:     "testTag",
		Period:  time.Second,
		MaxReqs: 10,
		NumReqs: 1,
	}
	// Define test inputs
	storage.Set(data)

	// Call the GetMaxRequests method
	ret, err := rl.GetMaxRequests(data.Tag)
	require.NoError(t, err)

	// Verify the returned data
	require.Equal(t, data, ret)
}

func TestIsLimitReachedWithinPeriod(t *testing.T) {
	storage, rl := setupTest()

	// Define test inputs
	tag := "testTag"
	maxRequests := 10
	interval := time.Second

	// Set up the storage with test data
	data := RequestData{
		Tag:       tag,
		Period:    interval,
		CreatedAt: time.Now(),
		MaxReqs:   maxRequests,
	}
	storage.Set(data)

	// Call the IsLimitReached method
	for i := 0; i < maxRequests; i++ {
		limitReached, err := rl.IsLimitReached(tag)
		require.NoError(t, err)

		// Verify the result
		require.False(t, limitReached)
	}

	// Call the IsLimitReached method again
	limitReached, err := rl.IsLimitReached(tag)
	require.NoError(t, err)
	require.True(t, limitReached)
}

func TestIsLimitReachedWhenPeriodPassed(t *testing.T) {
	storage, rl := setupTest()

	// Define test inputs
	tag := "testTag"
	maxRequests := 10
	interval := time.Second

	// Set up the storage with test data
	data := RequestData{
		Tag:       tag,
		Period:    interval,
		CreatedAt: time.Now().Add(-interval),
		MaxReqs:   maxRequests,
		NumReqs:   maxRequests,
	}
	storage.Set(data)

	// Call the IsLimitReached method
	limitReached, err := rl.IsLimitReached(tag)
	require.NoError(t, err)

	// Verify the result
	require.False(t, limitReached)
}
