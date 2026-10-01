package ownership

import (
	"context"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/status-im/status-go/pkg/pubsub"
	w_common "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
)

// pagedFetcher serves a fixed ownership list page by page. onPage runs before
// each page is returned, which is when a reader may look at the storage.
type pagedFetcher struct {
	pages  [][]thirdparty.CollectibleIDBalance
	onPage func(page int)
}

func (f *pagedFetcher) FetchCollectibleOwnershipByOwner(_ context.Context, _ w_common.ChainID, _ common.Address, cursor string, _ int, _ string) (*thirdparty.CollectibleOwnershipContainer, error) {
	page := 0
	if cursor != thirdparty.FetchFromStartCursor {
		var err error
		page, err = strconv.Atoi(cursor)
		if err != nil {
			return nil, err
		}
	}
	if f.onPage != nil {
		f.onPage(page)
	}
	next := thirdparty.FetchFromStartCursor
	if page+1 < len(f.pages) {
		next = strconv.Itoa(page + 1)
	}
	return &thirdparty.CollectibleOwnershipContainer{
		Items:      f.pages[page],
		NextCursor: next,
		Provider:   "test",
	}, nil
}

func cachedCount(t *testing.T, oDB *OwnershipDB, chainID w_common.ChainID, owner common.Address) int {
	t.Helper()
	ids, err := oDB.GetOwnedCollectibles([]w_common.ChainID{chainID}, []common.Address{owner}, 0, 1000)
	require.NoError(t, err)
	return len(ids)
}

// The UI reads the owned list from the cache while a load runs. During an
// initial (paged) load the pages not fetched yet must stay in the cache,
// otherwise the UI sees a truncated list and diffs its model against it.
func TestInitialLoadKeepsCachedOwnershipBetweenPages(t *testing.T) {
	oDB, cleanup := setupOwnershipDBTest(t)
	defer cleanup()

	chainID := w_common.ChainID(1)
	owner := common.HexToAddress("0x1234")
	owned := generateTestCollectibles(chainID, 0, 6)

	// A previous initial load that never completed left the cache populated
	// without a completed-load timestamp, so the next load pages again.
	_, _, _, err := oDB.Update(chainID, owner, owned, InvalidTimestamp)
	require.NoError(t, err)
	require.Equal(t, 6, cachedCount(t, oDB, chainID, owner))

	cachedBeforePage := map[int]int{}
	fetcher := &pagedFetcher{
		pages: [][]thirdparty.CollectibleIDBalance{owned[:3], owned[3:]},
		onPage: func(page int) {
			cachedBeforePage[page] = cachedCount(t, oDB, chainID, owner)
		},
	}

	loader := NewLoader(chainID, owner, fetcher, oDB, pubsub.NewPublisher(), LoaderParams{FetchLimit: 3}, zaptest.NewLogger(t))
	_, err = loader.Load(context.Background())
	require.NoError(t, err)

	require.Equal(t, 6, cachedBeforePage[0])
	require.Equal(t, 6, cachedBeforePage[1], "a partial page must not evict collectibles that are still to be paged")
	require.Equal(t, 6, cachedCount(t, oDB, chainID, owner))

	ts, err := oDB.GetOwnershipUpdateTimestamp(owner, chainID)
	require.NoError(t, err)
	require.NotEqual(t, InvalidTimestamp, ts, "a completed load records its timestamp")
}
