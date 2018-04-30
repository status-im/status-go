package timesource

import (
	"testing"
	"time"

	"github.com/beevik/ntp"
	"github.com/stretchr/testify/assert"
)

const (
	// clockCompareDelta declares time required between multiple calls to time.Now
	clockCompareDelta = 5 * time.Microsecond
)

func TestComputeOffset(t *testing.T) {
	// TODO(dshulyak) table tests with more test cases
	// TODO(dshulyak) reduce duplication in test setup
	responses := make([]*ntp.Response, 5)
	for i := range responses {
		responses[i] = &ntp.Response{ClockOffset: time.Duration(1)}
	}
	actualAttempts := 0
	queryMethod := func(string) (*ntp.Response, error) {
		defer func() { actualAttempts++ }()
		return responses[actualAttempts], nil
	}
	offset, err := computeOffset(queryMethod, "", 3)
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(1), offset)
}

func TestNTPTimeSource(t *testing.T) {
	madeOffset := 30 * time.Second
	responses := make([]*ntp.Response, 5)
	for i := range responses {
		responses[i] = &ntp.Response{ClockOffset: madeOffset}
	}
	actualAttempts := 0
	queryMethod := func(string) (*ntp.Response, error) {
		defer func() { actualAttempts++ }()
		return responses[actualAttempts], nil
	}
	source := &NTPTimeSource{
		attempts:    3,
		queryMethod: queryMethod,
	}
	assert.WithinDuration(t, time.Now(), source.Now(), clockCompareDelta)
	source.updateOffset()
	assert.WithinDuration(t, time.Now().Add(madeOffset), source.Now(), clockCompareDelta)
}
