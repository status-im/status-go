package wallet

import (
	"context"
	"math/big"
	"testing"

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
	iterator := IterativeDownloader{known: &DBHeader{Number: big.NewInt(0)}}
	require.True(t, iterator.Finished())
}

func TestIterNotFinished(t *testing.T) {
	iterator := IterativeDownloader{known: &DBHeader{Number: big.NewInt(2)}}
	require.False(t, iterator.Finished())
}

func TestIterRevert(t *testing.T) {
	iterator := IterativeDownloader{
		known:    &DBHeader{Number: big.NewInt(0)},
		previous: &DBHeader{Number: big.NewInt(10)},
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
		known:      &DBHeader{Number: big.NewInt(9)},
	}
	batch, err := iter.Next(context.TODO())
	require.NoError(t, err)
	require.Len(t, batch, 6)
	batch, err = iter.Next(context.TODO())
	require.NoError(t, err)
	require.Len(t, batch, 5)
	require.True(t, iter.Finished())
}
