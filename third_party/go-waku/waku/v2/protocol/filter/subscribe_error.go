package filter

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
)

// PeerSubscribeFailure captures a single peer's subscribe failure: the peer
// involved, the content topics that were attempted, and the underlying error
// (typically *FilterError, a dial error, or a stream-reset error).
type PeerSubscribeFailure struct {
	PeerID        peer.ID
	ContentTopics []string
	Err           error
}

// SubscribeError is the typed error returned by WakuFilterLightNode.Subscribe
// when one or more peers fail to accept a subscription. It preserves the
// per-peer error (including typed *FilterError) so callers — notably the
// apiSub layer — can react to specific failure codes such as HTTP 429
// (rate-limit).
//
// The previous implementation flattened all per-peer failures into a single
// plain *errors.errorString via fmt.Errorf+strings.Join. That lost the typed
// peer error AND could panic with "strings: Join output length overflow"
// under sustained failure storms (observed in production). SubscribeError
// fixes both problems: the typed peers are preserved, and Error() is
// bounded so it cannot overflow.
type SubscribeError struct {
	FailedPeers []PeerSubscribeFailure
}

// errorMaxBytes bounds the rendered string from Error() so it cannot grow
// unboundedly with FailedPeers and trigger a string-allocation overflow.
const errorMaxBytes = 4096

// Error renders a bounded, log-grep-friendly message. The "subscribe failed"
// prefix is intentional: existing log greps look for "subscriptions failed"
// (the prior wording) and "subscribe failed" both still hit on "failed".
func (s *SubscribeError) Error() string {
	if s == nil {
		return "<nil SubscribeError>"
	}
	var b strings.Builder
	b.Grow(256)
	firstCode := s.firstFilterErrorCode()
	if firstCode != 0 {
		fmt.Fprintf(&b, "subscribe failed for %d peer(s) (first peer code=%d)",
			len(s.FailedPeers), firstCode)
	} else {
		fmt.Fprintf(&b, "subscribe failed for %d peer(s)", len(s.FailedPeers))
	}
	if len(s.FailedPeers) == 0 {
		return b.String()
	}
	b.WriteString(": ")
	// Render at most a small fixed number of peers; truncate the rest. The
	// per-peer rendering uses up to one topic and the err Code/Msg, never
	// the full topic list — so total length is hard-bounded.
	const maxPeersRendered = 3
	rendered := 0
	for i, fp := range s.FailedPeers {
		if rendered >= maxPeersRendered {
			fmt.Fprintf(&b, " (+%d more)", len(s.FailedPeers)-rendered)
			break
		}
		if i > 0 {
			b.WriteString("; ")
		}
		topicPreview := "<none>"
		if len(fp.ContentTopics) > 0 {
			topicPreview = fp.ContentTopics[0]
			if len(fp.ContentTopics) > 1 {
				topicPreview = fmt.Sprintf("%s (+%d more)", topicPreview, len(fp.ContentTopics)-1)
			}
		}
		errMsg := "<nil>"
		if fp.Err != nil {
			errMsg = fp.Err.Error()
		}
		fmt.Fprintf(&b, "peer=%s topic=%s err=%s", fp.PeerID, topicPreview, errMsg)
		rendered++
		// Defensive: if a single peer's error string is pathologically large
		// (unlikely but possible), stop early.
		if b.Len() >= errorMaxBytes {
			break
		}
	}
	out := b.String()
	if len(out) > errorMaxBytes {
		out = out[:errorMaxBytes-3] + "..."
	}
	return out
}

// HasRateLimitError reports whether at least one peer returned HTTP 429
// (filter request rejected due to rate limit). The apiSub layer uses this
// to apply a longer backoff that respects the rate-limit signal instead of
// hammering the peer further.
func (s *SubscribeError) HasRateLimitError() bool {
	if s == nil {
		return false
	}
	var fe *FilterError
	for _, fp := range s.FailedPeers {
		if fp.Err == nil {
			continue
		}
		if errors.As(fp.Err, &fe) && fe.Code == http.StatusTooManyRequests {
			return true
		}
	}
	return false
}

// firstFilterErrorCode returns the Code of the first *FilterError found in
// FailedPeers, or 0 if none. Used by Error() for a quick at-a-glance signal
// in logs (typically the most useful single number when triaging).
func (s *SubscribeError) firstFilterErrorCode() int {
	var fe *FilterError
	for _, fp := range s.FailedPeers {
		if fp.Err == nil {
			continue
		}
		if errors.As(fp.Err, &fe) {
			return fe.Code
		}
	}
	return 0
}
