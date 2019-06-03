package wallet

import (
	"bytes"
	"context"
	"fmt"
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
	batch, err := iter.Next()
	require.NoError(t, err)
	require.Len(t, batch, 6)
	batch, err = iter.Next()
	require.NoError(t, err)
	require.Len(t, batch, 5)
	require.True(t, iter.Finished())
}

type balancesFixture []*big.Int

func (b balancesFixture) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	lth := len(b)
	index := int(blockNumber.Int64())
	if index > lth {
		return big.NewInt(0), nil
	}
	return b[index], nil
}

func (b balancesFixture) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return &types.Header{}, nil
}

func (b balancesFixture) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return &types.Header{}, nil
}

type batchDownloaderStub struct{}

func (b batchDownloaderStub) GetTransfersInRange(ctx context.Context, from, to *big.Int) ([]Transfer, error) {
	return nil, nil
}

type comparison struct {
	Low  *big.Int
	High *big.Int
}

type comparisons []comparison

func (c comparisons) Verbose() string {
	return comparisonsToString(c)
}

func comparisonsToString(rst []comparison) string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "[]comparison{\n")
	for i := range rst {
		fmt.Fprintf(buf, "{Low: big.NewInt(%v), High: big.NewInt(%v)},\n", rst[i].Low, rst[i].High)
	}
	fmt.Fprintf(buf, "}")
	return buf.String()
}

func TestBinaryIterativeDownloader(t *testing.T) {
	type testCase struct {
		desc      string
		low, high *big.Int
		balances  balancesFixture
		expected  []comparison
	}
	for _, tc := range []testCase{
		{
			desc:     "BalancesAreZero",
			low:      big.NewInt(0),
			high:     big.NewInt(3),
			balances: balancesFixture([]*big.Int{big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0)}),
			expected: comparisons{
				{Low: big.NewInt(0), High: big.NewInt(3)},
			},
		},
		{
			desc:     "RightmostTransfers",
			low:      big.NewInt(0),
			high:     big.NewInt(3),
			balances: balancesFixture([]*big.Int{big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(10)}),
			expected: comparisons{
				{Low: big.NewInt(0), High: big.NewInt(3)},
				{Low: big.NewInt(1), High: big.NewInt(3)},
				{Low: big.NewInt(2), High: big.NewInt(3)},
				{Low: big.NewInt(1), High: big.NewInt(2)},
				{Low: big.NewInt(0), High: big.NewInt(1)},
			},
		},
		{
			desc:     "ChangesEveryIteration",
			low:      big.NewInt(0),
			high:     big.NewInt(3),
			balances: balancesFixture([]*big.Int{big.NewInt(0), big.NewInt(1), big.NewInt(2), big.NewInt(3)}),
			expected: comparisons{
				{Low: big.NewInt(0), High: big.NewInt(3)},
				{Low: big.NewInt(1), High: big.NewInt(3)},
				{Low: big.NewInt(2), High: big.NewInt(3)},
				{Low: big.NewInt(1), High: big.NewInt(2)},
				{Low: big.NewInt(0), High: big.NewInt(1)},
			},
		},
		{
			desc: "TransferInTheMiddle",
			low:  big.NewInt(0),
			high: big.NewInt(40),
			balances: func() balancesFixture {
				fixture := make(balancesFixture, 41)
				change := big.NewInt(77)
				for i := range fixture {
					if i > 21 {
						fixture[i] = change
					} else {
						fixture[i] = zero
					}
				}
				return fixture
			}(),
			expected: comparisons{
				{Low: big.NewInt(0), High: big.NewInt(40)},
				{Low: big.NewInt(20), High: big.NewInt(40)},
				{Low: big.NewInt(30), High: big.NewInt(40)},
				{Low: big.NewInt(14), High: big.NewInt(30)},
				{Low: big.NewInt(22), High: big.NewInt(30)},
				{Low: big.NewInt(10), High: big.NewInt(22)},
				{Low: big.NewInt(16), High: big.NewInt(22)},
				{Low: big.NewInt(19), High: big.NewInt(22)},
				{Low: big.NewInt(20), High: big.NewInt(22)},
				{Low: big.NewInt(21), High: big.NewInt(22)},
				{Low: big.NewInt(10), High: big.NewInt(21)},
				{Low: big.NewInt(4), High: big.NewInt(10)},
				{Low: big.NewInt(1), High: big.NewInt(4)},
				{Low: big.NewInt(0), High: big.NewInt(1)},
			},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			downloader := &BinaryIterativeDownloader{
				low:        tc.low,
				high:       tc.high,
				client:     tc.balances,
				downloader: batchDownloaderStub{},
			}
			rst := comparisons{}
			for !downloader.Finished() {
				rst = append(rst, comparison{downloader.low, downloader.high})
				if len(rst) > len(tc.expected) {
					require.FailNowf(t, "more comparisons then expected", "expected: %s\ngot: %s\n", tc.expected, rst)
				}
				_, err := downloader.Next()
				require.NoError(t, err)
			}
			require.Equal(t, len(tc.expected), len(rst))
			for i := range rst {
				require.Equal(t, tc.expected[i].Low, rst[i].Low)
				require.Equal(t, tc.expected[i].High, rst[i].High)
			}
		})
	}
}
