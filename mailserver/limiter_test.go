package mailserver

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsAllowed(t *testing.T) {
	peerID := "peerID"
	testCases := []struct {
		t               time.Duration
		shouldBeAllowed bool
		db              func() map[string]time.Time
		info            string
	}{
		{
			t:               5 * time.Millisecond,
			shouldBeAllowed: true,
			db: func() map[string]time.Time {
				return make(map[string]time.Time)
			},
			info: "Expecting limiter.isAllowed to allow with an empty db",
		},
		{
			t:               5 * time.Millisecond,
			shouldBeAllowed: true,
			db: func() map[string]time.Time {
				db := make(map[string]time.Time)
				db[peerID] = time.Now().Add(time.Duration(-10) * time.Millisecond)
				return db
			},
			info: "Expecting limiter.isAllowed to allow with an expired peer on its db",
		},
		{
			t:               5 * time.Millisecond,
			shouldBeAllowed: false,
			db: func() map[string]time.Time {
				db := make(map[string]time.Time)
				db[peerID] = time.Now().Add(time.Duration(-1) * time.Millisecond)
				return db
			},
			info: "Expecting limiter.isAllowed to not allow with a non expired peer on its db",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.info, func(*testing.T) {
			l := newRateLimiter(tc.t)
			l.db = tc.db()
			assert.Equal(t, tc.shouldBeAllowed, l.IsAllowed(peerID))
		})
	}
}

func TestRemoveExpiredRateLimits(t *testing.T) {
	peer := "peer"
	l := newRateLimiter(time.Duration(5) * time.Second)
	for i := 0; i < 10; i++ {
		peerID := fmt.Sprintf("%s%d", peer, i)
		l.db[peerID] = time.Now().Add(time.Duration(i*(-2)) * time.Second)
	}

	l.deleteExpired()
	assert.Equal(t, 3, len(l.db))

	for i := 0; i < 3; i++ {
		peerID := fmt.Sprintf("%s%d", peer, i)
		_, ok := l.db[peerID]
		assert.True(t, ok, fmt.Sprintf("Non expired peer '%s' should exist, but it doesn't", peerID))
	}
	for i := 3; i < 10; i++ {
		peerID := fmt.Sprintf("%s%d", peer, i)
		_, ok := l.db[peerID]
		assert.False(t, ok, fmt.Sprintf("Expired peer '%s' should not exist, but it does", peerID))
	}
}

func TestAddingLimts(t *testing.T) {
	peerID := "peerAdding"
	l := newRateLimiter(time.Duration(5) * time.Second)
	pre := time.Now()
	l.Add(peerID)
	post := time.Now()
	assert.True(t, l.db[peerID].After(pre))
	assert.True(t, l.db[peerID].Before(post))
}
