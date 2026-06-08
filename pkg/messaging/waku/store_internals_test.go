package wakuv2

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/waku-org/go-waku/waku/v2/protocol"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"

	types "github.com/status-im/status-go/pkg/messaging/waku/types"
)

// --- fakes ---

type fakeRequestor struct {
	mu       sync.Mutex
	requests []*storepb.StoreQueryRequest
	respond  func(req *storepb.StoreQueryRequest, callIdx int) ([]*storepb.WakuMessageKeyValue, []byte, error)
}

func (f *fakeRequestor) query(_ context.Context, _ peer.AddrInfo, req *storepb.StoreQueryRequest) ([]*storepb.WakuMessageKeyValue, []byte, error) {
	f.mu.Lock()
	idx := len(f.requests)
	f.requests = append(f.requests, req)
	f.mu.Unlock()
	return f.respond(req, idx)
}

func (f *fakeRequestor) calls() []*storepb.StoreQueryRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*storepb.StoreQueryRequest, len(f.requests))
	copy(out, f.requests)
	return out
}

// recordingProcessor satisfies envelopeProcessor. The tests return empty message
// sets, so OnEnvelope is never invoked; it exists only to construct a pager.
type recordingProcessor struct{}

func (p *recordingProcessor) OnEnvelope(*protocol.Envelope, bool) error    { return nil }
func (p *recordingProcessor) OnRequestFailed([]byte, peer.AddrInfo, error) {}

func newPager(r storeRequestor) *storePager {
	return newStorePager(r, &recordingProcessor{}, zap.NewNop())
}

func emptyResponse(_ *storepb.StoreQueryRequest, _ int) ([]*storepb.WakuMessageKeyValue, []byte, error) {
	return nil, nil, nil
}

// --- selector ---

func TestStoreSelectorCandidates(t *testing.T) {
	s := newStoreSelector()
	require.False(t, s.hasStorenodes())
	require.Empty(t, s.candidates())

	nodes := []peer.AddrInfo{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	s.setStorenodes(nodes)
	require.True(t, s.hasStorenodes())
	require.ElementsMatch(t, nodes, s.candidates())
}

// --- chunking ---

func TestChunkContentTopics(t *testing.T) {
	require.Nil(t, chunkContentTopics(nil, 10))

	topics := make([]string, 25)
	for i := range topics {
		topics[i] = string(rune('a' + i))
	}
	chunks := chunkContentTopics(topics, 10)
	require.Len(t, chunks, 3)
	require.Len(t, chunks[0], 10)
	require.Len(t, chunks[1], 10)
	require.Len(t, chunks[2], 5)
}

// --- pager ---

func TestStorePagerCursorPagination(t *testing.T) {
	r := &fakeRequestor{respond: func(_ *storepb.StoreQueryRequest, idx int) ([]*storepb.WakuMessageKeyValue, []byte, error) {
		if idx == 0 {
			return nil, []byte("cursor-1"), nil // first page returns a cursor
		}
		return nil, nil, nil // second page completes the window
	}}
	to := time.Now()
	from := to.Add(-time.Hour)
	require.NoError(t, newPager(r).run(context.Background(), peer.AddrInfo{ID: "n"}, "pubsub", []string{"ct"}, from, to, 10, nil, false))

	calls := r.calls()
	require.Len(t, calls, 2)
	require.Nil(t, calls[0].PaginationCursor)
	require.Equal(t, []byte("cursor-1"), calls[1].PaginationCursor)
}

func TestStorePagerSplitsByWindow(t *testing.T) {
	r := &fakeRequestor{respond: emptyResponse}
	to := time.Now()
	from := to.Add(-50 * time.Hour) // > 2x the 24h window
	require.NoError(t, newPager(r).run(context.Background(), peer.AddrInfo{ID: "n"}, "pubsub", []string{"ct"}, from, to, 10, nil, false))

	calls := r.calls()
	require.Len(t, calls, 3)                               // 24h + 24h + 2h
	require.Equal(t, to.UnixNano(), *calls[0].TimeEnd)     // newest window first
	require.Equal(t, from.UnixNano(), *calls[2].TimeStart) // oldest window reaches from
	for _, c := range calls {
		require.LessOrEqual(t, *c.TimeEnd-*c.TimeStart, maxStoreRequestWindow.Nanoseconds())
	}
}

func TestStorePagerEarlyStop(t *testing.T) {
	r := &fakeRequestor{respond: func(_ *storepb.StoreQueryRequest, _ int) ([]*storepb.WakuMessageKeyValue, []byte, error) {
		return nil, []byte("more"), nil // always offers another page
	}}
	stopAfterFirst := func(int) (bool, uint64) { return false, 0 }
	to := time.Now()
	from := to.Add(-time.Hour)
	require.NoError(t, newPager(r).run(context.Background(), peer.AddrInfo{ID: "n"}, "pubsub", []string{"ct"}, from, to, 10, stopAfterFirst, false))

	require.Len(t, r.calls(), 1) // stopped despite the cursor
}

func TestStorePagerChunksContentTopics(t *testing.T) {
	r := &fakeRequestor{respond: emptyResponse}
	topics := make([]string, 15)
	for i := range topics {
		topics[i] = string(rune('a' + i))
	}
	to := time.Now()
	from := to.Add(-time.Hour)
	require.NoError(t, newPager(r).run(context.Background(), peer.AddrInfo{ID: "n"}, "pubsub", topics, from, to, 10, nil, false))

	calls := r.calls()
	require.Len(t, calls, 2) // 10 + 5
	seen := map[string]struct{}{}
	for _, c := range calls {
		require.LessOrEqual(t, len(c.ContentTopics), maxContentTopicsPerRequest)
		for _, ct := range c.ContentTopics {
			seen[ct] = struct{}{}
		}
	}
	require.Len(t, seen, 15)
}

// --- StoreClient failover ---

func newClient(r storeRequestor, nodes []peer.AddrInfo) *StoreClient {
	sel := newStoreSelector()
	sel.setStorenodes(nodes)
	resolve := func(s string) string {
		if s == "" {
			return "default"
		}
		return s
	}
	return NewStoreClient(sel, newStorePager(r, &recordingProcessor{}, zap.NewNop()), resolve, zap.NewNop())
}

func batch() types.MailserverBatch {
	return types.MailserverBatch{
		From:        time.Now().Add(-time.Hour),
		To:          time.Now(),
		PubsubTopic: "pubsub",
		Topics:      []types.TopicType{{1, 2, 3, 4}},
	}
}

func TestStoreClientFailsOverToNextNode(t *testing.T) {
	r := &fakeRequestor{respond: func(_ *storepb.StoreQueryRequest, idx int) ([]*storepb.WakuMessageKeyValue, []byte, error) {
		if idx == 0 {
			return nil, nil, errors.New("first node down")
		}
		return nil, nil, nil
	}}
	sc := newClient(r, []peer.AddrInfo{{ID: "n1"}, {ID: "n2"}})

	require.NoError(t, sc.Query(context.Background(), batch(), 10, nil, false))
	require.Len(t, r.calls(), 2) // failed over from the first node to the second
}

func TestStoreClientReturnsErrorWhenAllNodesFail(t *testing.T) {
	r := &fakeRequestor{respond: func(_ *storepb.StoreQueryRequest, _ int) ([]*storepb.WakuMessageKeyValue, []byte, error) {
		return nil, nil, errors.New("down")
	}}
	sc := newClient(r, []peer.AddrInfo{{ID: "n1"}, {ID: "n2"}, {ID: "n3"}, {ID: "n4"}})

	require.Error(t, sc.Query(context.Background(), batch(), 10, nil, false))
	require.Len(t, r.calls(), maxStoreQueryAttempts) // bounded number of attempts
}

func TestStoreClientNoStorenodes(t *testing.T) {
	r := &fakeRequestor{respond: emptyResponse}
	sc := newClient(r, nil)

	require.ErrorIs(t, sc.Query(context.Background(), batch(), 10, nil, false), ErrNoStorenodesReachable)
	require.Empty(t, r.calls())
}
