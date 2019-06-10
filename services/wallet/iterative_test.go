package wallet

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type transfersFixture []Transfer

func (f transfersFixture) GetTransfersInRange(ctx context.Context, from, to *big.Int) ([]Transfer, error) {
	rst := []Transfer{}
	for _, t := range f {
		if t.BlockNumber.Cmp(from) >= 0 && t.BlockNumber.Cmp(to) <= 0 {
			rst = append(rst, t)
		}
	}
	return rst, nil
}

func TestIterFinished(t *testing.T) {
	iterator := IterativeDownloader{
		from: &DBHeader{Number: big.NewInt(10)},
		to:   &DBHeader{Number: big.NewInt(10)},
	}
	require.True(t, iterator.Finished())
}

func TestIterNotFinished(t *testing.T) {
	iterator := IterativeDownloader{
		from: &DBHeader{Number: big.NewInt(2)},
		to:   &DBHeader{Number: big.NewInt(5)},
	}
	require.False(t, iterator.Finished())
}

func TestIterRevert(t *testing.T) {
	iterator := IterativeDownloader{
		from:     &DBHeader{Number: big.NewInt(12)},
		to:       &DBHeader{Number: big.NewInt(12)},
		previous: &DBHeader{Number: big.NewInt(9)},
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
		from:       &DBHeader{Number: big.NewInt(0)},
		to:         &DBHeader{Number: big.NewInt(9)},
	}
	batch, err := iter.Next(context.TODO())
	require.NoError(t, err)
	require.Len(t, batch, 6)
	batch, err = iter.Next(context.TODO())
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
			Time:       big.NewInt(1),
		}
		if i != 0 {
			rst[i].ParentHash = rst[i-1].Hash()
		}
	}
	return rst
}
