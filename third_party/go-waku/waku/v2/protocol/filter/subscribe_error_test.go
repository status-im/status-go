package filter

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

// TestSubscribeError_Error covers two contracts of the typed aggregate error
// returned by WakuFilterLightNode.Subscribe under partial failure:
//
//  1. The string preserves the "subscribe failed" prefix so existing log
//     greps and dashboards keep working.
//  2. The string length is bounded regardless of FailedPeers count. The
//     previous implementation aggregated all content topics with
//     strings.Join, which is unbounded and was observed to panic with
//     `strings: Join output length overflow` under sustained subscribe
//     storms — crashing statusgo. The bounded form prevents that crash.
func TestSubscribeError_Error(t *testing.T) {
	t.Run("empty FailedPeers — well-formed string", func(t *testing.T) {
		e := &SubscribeError{}
		s := e.Error()
		require.NotEmpty(t, s)
		require.Contains(t, s, "subscribe failed")
	})

	t.Run("single peer single topic", func(t *testing.T) {
		e := &SubscribeError{
			FailedPeers: []PeerSubscribeFailure{
				{
					PeerID:        peer.ID("peer-1"),
					ContentTopics: []string{"/waku/1/0xabc/rfc26"},
					Err:           &FilterError{Code: http.StatusTooManyRequests, Message: "rate limited"},
				},
			},
		}
		s := e.Error()
		require.Contains(t, s, "subscribe failed")
		require.Contains(t, s, "1")
	})

	t.Run("bounded length under massive FailedPeers (would-panic-with-Join scenario)", func(t *testing.T) {
		// 10,000 failed peers × 50 long topics each = the kind of unbounded
		// growth that produced the `strings: Join output length overflow`
		// panic in production. Error() must stay bounded.
		const peerCount = 10000
		const topicsPerPeer = 50
		failedPeers := make([]PeerSubscribeFailure, peerCount)
		longTopic := "/" + strings.Repeat("a", 200) + "/topic"
		for i := range failedPeers {
			topics := make([]string, topicsPerPeer)
			for j := range topics {
				topics[j] = fmt.Sprintf("%s/%d-%d", longTopic, i, j)
			}
			failedPeers[i] = PeerSubscribeFailure{
				PeerID:        peer.ID(fmt.Sprintf("peer-%d", i)),
				ContentTopics: topics,
				Err:           &FilterError{Code: 503, Message: "service unavailable"},
			}
		}
		e := &SubscribeError{FailedPeers: failedPeers}
		s := e.Error()
		// 4 KB cap is the contract — generous enough for useful info, far
		// below any plausible strings.Join overflow point.
		require.LessOrEqual(t, len(s), 4096,
			"SubscribeError.Error() must be bounded; got %d bytes (would risk strings.Join overflow)", len(s))
		require.Contains(t, s, "subscribe failed")
	})
}

// TestSubscribeError_HasRateLimitError verifies the predicate that the
// apiSub layer uses to decide whether to enter a longer rate-limit backoff.
// 429 must be detectable both as a direct *FilterError and through a wrapping
// fmt.Errorf("%w") chain — production code occasionally adds context wrappers.
func TestSubscribeError_HasRateLimitError(t *testing.T) {
	t.Run("nil SubscribeError pointer — false (safe predicate)", func(t *testing.T) {
		var e *SubscribeError
		require.False(t, e.HasRateLimitError())
	})

	t.Run("empty FailedPeers — false", func(t *testing.T) {
		e := &SubscribeError{}
		require.False(t, e.HasRateLimitError())
	})

	t.Run("only dial / generic errors — false", func(t *testing.T) {
		e := &SubscribeError{
			FailedPeers: []PeerSubscribeFailure{
				{Err: errors.New("dial backoff")},
				{Err: errors.New("stream reset")},
				{Err: &FilterError{Code: 503, Message: "service unavailable"}},
			},
		}
		require.False(t, e.HasRateLimitError())
	})

	t.Run("single direct 429 — true", func(t *testing.T) {
		e := &SubscribeError{
			FailedPeers: []PeerSubscribeFailure{
				{Err: &FilterError{Code: http.StatusTooManyRequests, Message: "rate limited"}},
			},
		}
		require.True(t, e.HasRateLimitError())
	})

	t.Run("multiple peers, one 429 — true", func(t *testing.T) {
		e := &SubscribeError{
			FailedPeers: []PeerSubscribeFailure{
				{Err: errors.New("stream reset")},
				{Err: &FilterError{Code: 503}},
				{Err: &FilterError{Code: http.StatusTooManyRequests, Message: "rate limited"}},
				{Err: errors.New("dial backoff")},
			},
		}
		require.True(t, e.HasRateLimitError())
	})

	t.Run("429 wrapped via fmt.Errorf %w — true", func(t *testing.T) {
		wrapped := fmt.Errorf("subscribe attempt failed: %w", &FilterError{Code: http.StatusTooManyRequests, Message: "rate limited"})
		e := &SubscribeError{
			FailedPeers: []PeerSubscribeFailure{
				{Err: wrapped},
			},
		}
		require.True(t, e.HasRateLimitError())
	})

	t.Run("nil Err entries are skipped safely", func(t *testing.T) {
		e := &SubscribeError{
			FailedPeers: []PeerSubscribeFailure{
				{Err: nil},
				{Err: &FilterError{Code: http.StatusTooManyRequests, Message: "rate limited"}},
				{Err: nil},
			},
		}
		require.True(t, e.HasRateLimitError())
	})
}
