package shhext

import (
	"fmt"
	"testing"
	"time"

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
			&MessagesRequest{From: yesterday, To: now},
		},
		{
			&MessagesRequest{From: 1, To: 0},
			&MessagesRequest{From: uint32(1), To: now},
		},
		{
			&MessagesRequest{From: 0, To: yesterday},
			&MessagesRequest{From: daysAgo(tnow, 2), To: yesterday},
		},
		// 100 - 1 day would be invalid, so we set From to 0
		{
			&MessagesRequest{From: 0, To: 100},
			&MessagesRequest{From: 0, To: 100},
		},
	}

	for i, s := range scenarios {
		t.Run(fmt.Sprintf("Scenario %d", i), func(t *testing.T) {
			s.given.setDefaults(tnow)
			require.Equal(t, s.expected, s.given)
		})
	}
}
