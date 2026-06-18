package wakuv2

import (
	"context"
	"errors"
	"testing"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/waku-org/go-waku/waku/v2/api/publish"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

type fakePeerProvider struct{ info peer.AddrInfo }

func (f fakePeerProvider) GetActiveStorenodePeerInfo() peer.AddrInfo { return f.info }

type fakeTime struct{ now int64 }

func (f *fakeTime) Now() time.Time { return time.Unix(f.now, 0) }

type fakeVerifier struct {
	existing []pb.MessageHash
	err      error
	calls    int
}

func (f *fakeVerifier) MessageHashesExist(_ context.Context, _ []byte, _ peer.AddrInfo, _ uint64, _ []pb.MessageHash) ([]pb.MessageHash, error) {
	f.calls++
	return f.existing, f.err
}

func newTestCheck(verifier publish.StorenodeMessageVerifier, peerInfo peer.AddrInfo, now int64, stored, expired chan gethcommon.Hash) *MessageSentCheck {
	return NewMessageSentCheck(context.Background(), verifier, fakePeerProvider{info: peerInfo}, &fakeTime{now: now}, stored, expired, zap.NewNop())
}

func existing(hashes ...gethcommon.Hash) []pb.MessageHash {
	out := make([]pb.MessageHash, len(hashes))
	for i, h := range hashes {
		out[i] = pb.ToMessageHash(h.Bytes())
	}
	return out
}

func TestMessageSentCheckAddDelete(t *testing.T) {
	msc := newTestCheck(&fakeVerifier{}, peer.AddrInfo{ID: "p"}, 1000, nil, nil)

	h1, h2, h3 := gethcommon.HexToHash("0x1"), gethcommon.HexToHash("0x2"), gethcommon.HexToHash("0x3")
	msc.Add("topic-a", h1, 100)
	msc.Add("topic-a", h2, 100)
	msc.Add("topic-b", h3, 100)
	require.Len(t, msc.messageIDs["topic-a"], 2)
	require.Len(t, msc.messageIDs["topic-b"], 1)

	msc.DeleteByMessageIDs([]gethcommon.Hash{h1})
	require.Len(t, msc.messageIDs["topic-a"], 1)

	// removing the last id in a topic drops the topic entry entirely
	msc.DeleteByMessageIDs([]gethcommon.Hash{h3})
	_, ok := msc.messageIDs["topic-b"]
	require.False(t, ok)
}

func TestMessageSentCheckQueryAcksStoredAndExpiresMissing(t *testing.T) {
	stored := make(chan gethcommon.Hash, 4)
	expired := make(chan gethcommon.Hash, 4)

	h1 := gethcommon.HexToHash("0x01") // present in the store -> acked
	h2 := gethcommon.HexToHash("0x02") // missing and past expiry -> expired

	verifier := &fakeVerifier{existing: existing(h1)}
	// now=1000, expiry = relayTime + 10; relayTime=980 -> 1000 > 990 -> expired
	msc := newTestCheck(verifier, peer.AddrInfo{ID: "store"}, 1000, stored, expired)

	processed := msc.messageHashBasedQuery(context.Background(), []gethcommon.Hash{h1, h2}, []uint32{980, 980}, "pubsub")

	require.ElementsMatch(t, []gethcommon.Hash{h1, h2}, processed)
	require.Equal(t, 1, verifier.calls)
	require.Equal(t, h1, <-stored)
	require.Equal(t, h2, <-expired)
	require.Empty(t, stored)
	require.Empty(t, expired)
}

func TestMessageSentCheckQueryKeepsRecentMissing(t *testing.T) {
	stored := make(chan gethcommon.Hash, 4)
	expired := make(chan gethcommon.Hash, 4)

	h := gethcommon.HexToHash("0x05") // missing but not yet past expiry -> retried later

	// now=1000, relayTime=995 -> 1000 > 1005 is false -> not expired
	msc := newTestCheck(&fakeVerifier{existing: nil}, peer.AddrInfo{ID: "store"}, 1000, stored, expired)

	processed := msc.messageHashBasedQuery(context.Background(), []gethcommon.Hash{h}, []uint32{995}, "pubsub")

	require.Empty(t, processed)
	require.Empty(t, stored)
	require.Empty(t, expired)
}

func TestMessageSentCheckQueryNoPeer(t *testing.T) {
	verifier := &fakeVerifier{existing: existing(gethcommon.HexToHash("0x1"))}
	msc := newTestCheck(verifier, peer.AddrInfo{}, 1000, make(chan gethcommon.Hash, 1), make(chan gethcommon.Hash, 1))

	processed := msc.messageHashBasedQuery(context.Background(), []gethcommon.Hash{gethcommon.HexToHash("0x1")}, []uint32{980}, "pubsub")

	require.Empty(t, processed)
	require.Equal(t, 0, verifier.calls) // no peer -> no store query attempted
}

func TestMessageSentCheckQueryVerifierError(t *testing.T) {
	msc := newTestCheck(&fakeVerifier{err: errors.New("store down")}, peer.AddrInfo{ID: "store"}, 1000,
		make(chan gethcommon.Hash, 1), make(chan gethcommon.Hash, 1))

	processed := msc.messageHashBasedQuery(context.Background(), []gethcommon.Hash{gethcommon.HexToHash("0x1")}, []uint32{980}, "pubsub")
	require.Empty(t, processed)
}
