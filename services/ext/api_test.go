package ext

import (
	"fmt"
	"testing"
	"time"

	"github.com/status-im/status-go/eth-node/types"
	wakutypes "github.com/status-im/status-go/waku/types"

	"github.com/stretchr/testify/require"
)

func TestMessagesRequest_setDefaults(t *testing.T) {
	daysAgo := func(now time.Time, days int) uint32 {
		return uint32(now.UTC().Add(-24 * time.Hour * time.Duration(days)).Unix())
	}

	tnow := time.Now()
	now := uint32(tnow.UTC().Unix())
	yesterday := daysAgo(tnow, 1)

	scenarios := []struct {
		given    *MessagesRequest
		expected *MessagesRequest
	}{
		{
			&MessagesRequest{From: 0, To: 0},
			&MessagesRequest{From: yesterday, To: now, Timeout: defaultRequestTimeout},
		},
		{
			&MessagesRequest{From: 1, To: 0},
			&MessagesRequest{From: uint32(1), To: now, Timeout: defaultRequestTimeout},
		},
		{
			&MessagesRequest{From: 0, To: yesterday},
			&MessagesRequest{From: daysAgo(tnow, 2), To: yesterday, Timeout: defaultRequestTimeout},
		},
		// 100 - 1 day would be invalid, so we set From to 0
		{
			&MessagesRequest{From: 0, To: 100},
			&MessagesRequest{From: 0, To: 100, Timeout: defaultRequestTimeout},
		},
		// set Timeout
		{
			&MessagesRequest{From: 0, To: 0, Timeout: 100},
			&MessagesRequest{From: yesterday, To: now, Timeout: 100},
		},
	}

	for i, s := range scenarios {
		t.Run(fmt.Sprintf("Scenario %d", i), func(t *testing.T) {
			s.given.SetDefaults(tnow)
			require.Equal(t, s.expected, s.given)
		})
	}
}

func TestExpiredOrCompleted(t *testing.T) {
	timeout := time.Millisecond
	events := make(chan wakutypes.EnvelopeEvent)
	errors := make(chan error, 1)
	hash := types.Hash{1}
	go func() {
		_, err := WaitForExpiredOrCompleted(hash, events, timeout)
		errors <- err
	}()
	select {
	case <-time.After(time.Second):
		require.FailNow(t, "timed out waiting for waitForExpiredOrCompleted to complete")
	case err := <-errors:
		require.EqualError(t, err, fmt.Sprintf("request %x expired", hash))
	}
}
