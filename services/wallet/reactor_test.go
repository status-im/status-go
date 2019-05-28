package wallet

import (
	"context"
	"errors"
	"math/big"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type headers []*types.Header

func (h headers) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	for _, item := range h {
		if item.Hash() == hash {
			return item, nil
		}
	}
	return nil, errors.New("not found")
}

func (h headers) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	for _, item := range h {
		if item.Number.Cmp(number) == 0 {
			return item, nil
		}
	}
	return nil, errors.New("not found")
}

func TestReactorReorgOnNewBlock(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	original := genHeadersChain(5, 1)
	require.NoError(t, db.SaveHeaders(original))
	// rewrite parents ater 2nd block
	reorg := make(headers, 5)
	copy(reorg, original[:2])
	for i := 2; i < len(reorg); i++ {
		reorg[i] = &types.Header{
			Number:     big.NewInt(int64(i)),
			Difficulty: big.NewInt(2), // actual difficulty is not important. using it to change a hash
			Time:       big.NewInt(1),
			ParentHash: reorg[i-1].Hash(),
		}
	}
	reactor := Reactor{
		client: reorg,
		db:     db,
	}
	latest := &types.Header{
		Number:     big.NewInt(5),
		Difficulty: big.NewInt(2),
		Time:       big.NewInt(1),
		ParentHash: reorg[len(reorg)-1].Hash(),
	}
	previous := original[len(original)-1]
	added, removed, err := reactor.onNewBlock(context.TODO(), previous, latest)
	require.NoError(t, err)
	require.Len(t, added, 4)
	require.Len(t, removed, 3)

	sort.Slice(removed, func(i, j int) bool {
		return removed[i].Number.Cmp(removed[j].Number) < 1
	})
	for i, h := range original[2:] {
		require.Equal(t, h.Hash(), removed[i].Hash())
	}

	expected := make([]*types.Header, 4)
	copy(expected, reorg[2:])
	expected[3] = latest
	sort.Slice(added, func(i, j int) bool {
		return added[i].Number.Cmp(added[j].Number) < 1
	})
	for i, h := range expected {
		require.Equal(t, h.Hash(), added[i].Hash())
	}
}

func TestReactorReorgAllKnownHeaders(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	original := genHeadersChain(2, 1)
	reorg := make(headers, 2)
	copy(reorg, genHeadersChain(2, 2))
	latest := &types.Header{
		Number:     big.NewInt(2),
		Difficulty: big.NewInt(2),
		Time:       big.NewInt(1),
		ParentHash: reorg[len(reorg)-1].Hash(),
	}
	require.NoError(t, db.SaveHeaders(original))
	reactor := Reactor{
		client: reorg,
		db:     db,
	}
	added, removed, err := reactor.onNewBlock(context.TODO(), original[len(original)-1], latest)
	require.NoError(t, err)
	require.Len(t, added, 3)
	require.Len(t, removed, 2)
}

func genHeadersChain(size, difficulty int) []*types.Header {
	rst := make([]*types.Header, size)
	for i := 0; i < size; i++ {
		rst[i] = &types.Header{
			Number:     big.NewInt(int64(i)),
			Difficulty: big.NewInt(int64(difficulty)),
			Time:       big.NewInt(1),
		}
		if i != 0 {
			rst[i].ParentHash = rst[i-1].Hash()
		}
	}
	return rst
}
