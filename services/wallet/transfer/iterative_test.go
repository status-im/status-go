package transfer

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type transfersFixture []Transfer

func (f transfersFixture) GetHeadersInRange(ctx context.Context, from, to *big.Int) ([]*DBHeader, error) {
	rst := []*DBHeader{}
	for _, t := range f {
		if t.BlockNumber.Cmp(from) >= 0 && t.BlockNumber.Cmp(to) <= 0 {
			rst = append(rst, &DBHeader{Number: t.BlockNumber})
		}
	}
	return rst, nil
}

func TestIterFinished(t *testing.T) {
	iterator := IterativeDownloader{
		from: big.NewInt(10),
		to:   big.NewInt(10),
	}
	require.True(t, iterator.Finished())
}

func TestIterNotFinished(t *testing.T) {
	iterator := IterativeDownloader{
		from: big.NewInt(2),
		to:   big.NewInt(5),
	}
	require.False(t, iterator.Finished())
}

func TestIterRevert(t *testing.T) {
	iterator := IterativeDownloader{
		from:     big.NewInt(12),
		to:       big.NewInt(12),
		previous: big.NewInt(9),
	}
	require.True(t, iterator.Finished())
	iterator.Revert()
	require.False(t, iterator.Finished())
}

func TestIterProgress(t *testing.T) {
	var (
		chain     headers = genHeadersChain(10, 1)
		transfers         = make(transfersFixture, 10)
	)
	for i := range transfers {
		transfers[i] = Transfer{
			BlockNumber: chain[i].Number,
			BlockHash:   chain[i].Hash(),
		}
	}
	iter := &IterativeDownloader{
		client:     chain,
		downloader: transfers,
		batchSize:  big.NewInt(5),
		from:       big.NewInt(0),
		to:         big.NewInt(9),
	}
	batch, _, _, err := iter.Next(context.TODO())
	require.NoError(t, err)
	require.Len(t, batch, 6)
	batch, _, _, err = iter.Next(context.TODO())
	require.NoError(t, err)
	require.Len(t, batch, 5)
	require.True(t, iter.Finished())
}

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

func (h headers) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	return nil, errors.New("not implemented")
}

func genHeadersChain(size, difficulty int) []*types.Header {
	rst := make([]*types.Header, size)
	for i := 0; i < size; i++ {
		rst[i] = &types.Header{
			Number:     big.NewInt(int64(i)),
			Difficulty: big.NewInt(int64(difficulty)),
			Time:       1,
		}
		if i != 0 {
			rst[i].ParentHash = rst[i-1].Hash()
		}
	}
	return rst
}
