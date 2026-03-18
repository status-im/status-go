package transport

import (
	"context"
	"errors"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"

	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
)

type fakeWakuWithoutHashFetcher struct {
	wakutypes.Waku
	active peer.AddrInfo
}

func (f *fakeWakuWithoutHashFetcher) GetActiveStorenode() peer.AddrInfo {
	return f.active
}

type fakeWakuWithHashFetcher struct {
	wakutypes.Waku
	active            peer.AddrInfo
	receivedStorenode peer.AddrInfo
	receivedHashes    []string
	fetchErr          error
	fetchInvocations  int
}

func (f *fakeWakuWithHashFetcher) GetActiveStorenode() peer.AddrInfo {
	return f.active
}

func (f *fakeWakuWithHashFetcher) FetchMessagesByHashes(ctx context.Context, storenode peer.AddrInfo, messageHashes []string) error {
	f.fetchInvocations++
	f.receivedStorenode = storenode
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

func TestFetchMessagesByHashes_NoActiveStorenode(t *testing.T) {
	waku := &fakeWakuWithHashFetcher{}
	tr := &Transport{waku: waku}

	err := tr.FetchMessagesByHashes(context.Background(), []string{"0x01"})
	require.EqualError(t, err, "no active storenode")
	require.Equal(t, 0, waku.fetchInvocations)
}

func TestFetchMessagesByHashes_BackendDoesNotSupportHashFetch(t *testing.T) {
	waku := &fakeWakuWithoutHashFetcher{
		active: peer.AddrInfo{ID: peer.ID("peer-1")},
	}
	tr := &Transport{waku: waku}

	err := tr.FetchMessagesByHashes(context.Background(), []string{"0x01"})
	require.EqualError(t, err, "waku backend does not support hash-based message fetch")
}

func TestFetchMessagesByHashes_ForwardsToBackend(t *testing.T) {
	active := peer.AddrInfo{ID: peer.ID("peer-1")}
	waku := &fakeWakuWithHashFetcher{active: active}
	tr := &Transport{waku: waku}

	hashes := []string{"0x01", "0x02"}
	err := tr.FetchMessagesByHashes(context.Background(), hashes)
	require.NoError(t, err)
	require.Equal(t, 1, waku.fetchInvocations)
	require.Equal(t, active, waku.receivedStorenode)
	require.Equal(t, hashes, waku.receivedHashes)
}

func TestFetchMessagesByHashes_BackendErrorPropagates(t *testing.T) {
	backendErr := errors.New("fetch failed")
	waku := &fakeWakuWithHashFetcher{
		active:   peer.AddrInfo{ID: peer.ID("peer-1")},
		fetchErr: backendErr,
	}
	tr := &Transport{waku: waku}

	err := tr.FetchMessagesByHashes(context.Background(), []string{"0x01"})
	require.ErrorIs(t, err, backendErr)
	require.Equal(t, 1, waku.fetchInvocations)
}
