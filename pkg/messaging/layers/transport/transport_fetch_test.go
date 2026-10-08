package transport

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
)

type fakeWakuWithoutHashFetcher struct {
	wakutypes.Waku
}

type fakeWakuWithHashFetcher struct {
	wakutypes.Waku
	receivedHashes   []string
	fetchErr         error
	fetchInvocations int
}

type fakeProcessedMessageIDsCache struct {
	receivedIDs []string
	hits        map[string]bool
	err         error
}

func (f *fakeProcessedMessageIDsCache) Clear() error {
	return nil
}

func (f *fakeProcessedMessageIDsCache) Hits(ids []string) (map[string]bool, error) {
	f.receivedIDs = append([]string(nil), ids...)
	return f.hits, f.err
}

func (f *fakeProcessedMessageIDsCache) Add(ids []string, timestamp uint64) error {
	return nil
}

func (f *fakeProcessedMessageIDsCache) Clean(timestamp uint64) error {
	return nil
}

func (f *fakeWakuWithHashFetcher) FetchMessagesByHashes(ctx context.Context, messageHashes []string) error {
	f.fetchInvocations++
	f.receivedHashes = append([]string(nil), messageHashes...)
	return f.fetchErr
}

func TestFetchMessagesByHashes_EmptyHashesNoop(t *testing.T) {
	waku := &fakeWakuWithHashFetcher{}
	tr := &Transport{waku: waku}

	err := tr.FetchMessagesByHashes(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, 0, waku.fetchInvocations)
}

func TestAlreadyProcessed_ForwardsToCache(t *testing.T) {
	hashes := []string{"0x01", "0x02"}
	expectedHits := map[string]bool{"0x02": true}
	cache := &fakeProcessedMessageIDsCache{hits: expectedHits}
	tr := &Transport{cache: cache}

	hits, err := tr.AlreadyProcessed(hashes)
	require.NoError(t, err)
	require.Equal(t, hashes, cache.receivedIDs)
	require.Equal(t, expectedHits, hits)
}

func TestFetchMessagesByHashes_BackendDoesNotSupportHashFetch(t *testing.T) {
	waku := &fakeWakuWithoutHashFetcher{}
	tr := &Transport{waku: waku}

	err := tr.FetchMessagesByHashes(context.Background(), []string{"0x01"})
	require.EqualError(t, err, "waku backend does not support hash-based message fetch")
}

func TestFetchMessagesByHashes_ForwardsToBackend(t *testing.T) {
	waku := &fakeWakuWithHashFetcher{}
	tr := &Transport{waku: waku}

	hashes := []string{"0x01", "0x02"}
	err := tr.FetchMessagesByHashes(context.Background(), hashes)
	require.NoError(t, err)
	require.Equal(t, 1, waku.fetchInvocations)
	require.Equal(t, hashes, waku.receivedHashes)
}

func TestFetchMessagesByHashes_BackendErrorPropagates(t *testing.T) {
	backendErr := errors.New("fetch failed")
	waku := &fakeWakuWithHashFetcher{fetchErr: backendErr}
	tr := &Transport{waku: waku}

	err := tr.FetchMessagesByHashes(context.Background(), []string{"0x01"})
	require.ErrorIs(t, err, backendErr)
	require.Equal(t, 1, waku.fetchInvocations)
}
