package transfer

import (
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/appdatabase"
)

func setupTestTransferDB(t *testing.T) (*Block, func()) {
	tmpfile, err := ioutil.TempFile("", "wallet-transfer-tests-")
	require.NoError(t, err)
	db, err := appdatabase.InitializeDB(tmpfile.Name(), "wallet-tests")
	require.NoError(t, err)
	return &Block{db}, func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func TestInsertRange(t *testing.T) {
	b, stop := setupTestTransferDB(t)
	defer stop()

	r := &BlocksRange{
		from: big.NewInt(0),
		to:   big.NewInt(10),
	}
	nonce := uint64(199)
	balance := big.NewInt(7657)
	account := common.Address{2}

	err := b.insertRange(777, account, r.from, r.to, balance, nonce)
	require.NoError(t, err)

	block, err := b.GetLastKnownBlockByAddress(777, account)
	require.NoError(t, err)

	require.Equal(t, 0, block.Number.Cmp(r.to))
	require.Equal(t, 0, block.Balance.Cmp(balance))
	require.Equal(t, nonce, uint64(*block.Nonce))
}

func TestGetNewRanges(t *testing.T) {
	ranges := []*BlocksRange{
		&BlocksRange{
			from: big.NewInt(0),
			to:   big.NewInt(10),
		},
		&BlocksRange{
			from: big.NewInt(10),
			to:   big.NewInt(20),
		},
	}

	n, d := getNewRanges(ranges)
	require.Equal(t, 1, len(n))
	newRange := n[0]
	require.Equal(t, int64(0), newRange.from.Int64())
	require.Equal(t, int64(20), newRange.to.Int64())
	require.Equal(t, 2, len(d))

	ranges = []*BlocksRange{
		&BlocksRange{
			from: big.NewInt(0),
			to:   big.NewInt(11),
		},
		&BlocksRange{
			from: big.NewInt(10),
			to:   big.NewInt(20),
		},
	}

	n, d = getNewRanges(ranges)
	require.Equal(t, 1, len(n))
	newRange = n[0]
	require.Equal(t, int64(0), newRange.from.Int64())
	require.Equal(t, int64(20), newRange.to.Int64())
	require.Equal(t, 2, len(d))

	ranges = []*BlocksRange{
		&BlocksRange{
			from: big.NewInt(0),
			to:   big.NewInt(20),
		},
		&BlocksRange{
			from: big.NewInt(5),
			to:   big.NewInt(15),
		},
	}

	n, d = getNewRanges(ranges)
	require.Equal(t, 1, len(n))
	newRange = n[0]
	require.Equal(t, int64(0), newRange.from.Int64())
	require.Equal(t, int64(20), newRange.to.Int64())
	require.Equal(t, 2, len(d))

	ranges = []*BlocksRange{
		&BlocksRange{
			from: big.NewInt(5),
			to:   big.NewInt(15),
		},
		&BlocksRange{
			from: big.NewInt(5),
			to:   big.NewInt(20),
		},
	}

	n, d = getNewRanges(ranges)
	require.Equal(t, 1, len(n))
	newRange = n[0]
	require.Equal(t, int64(5), newRange.from.Int64())
	require.Equal(t, int64(20), newRange.to.Int64())
	require.Equal(t, 2, len(d))

	ranges = []*BlocksRange{
		&BlocksRange{
			from: big.NewInt(5),
			to:   big.NewInt(10),
		},
		&BlocksRange{
			from: big.NewInt(15),
			to:   big.NewInt(20),
		},
	}

	n, d = getNewRanges(ranges)
	require.Equal(t, 0, len(n))
	require.Equal(t, 0, len(d))

	ranges = []*BlocksRange{
		&BlocksRange{
			from: big.NewInt(0),
			to:   big.NewInt(10),
		},
		&BlocksRange{
			from: big.NewInt(10),
			to:   big.NewInt(20),
		},
		&BlocksRange{
			from: big.NewInt(30),
			to:   big.NewInt(40),
		},
	}

	n, d = getNewRanges(ranges)
	require.Equal(t, 1, len(n))
	newRange = n[0]
	require.Equal(t, int64(0), newRange.from.Int64())
	require.Equal(t, int64(20), newRange.to.Int64())
	require.Equal(t, 2, len(d))

	ranges = []*BlocksRange{
		&BlocksRange{
			from: big.NewInt(0),
			to:   big.NewInt(10),
		},
		&BlocksRange{
			from: big.NewInt(10),
			to:   big.NewInt(20),
		},
		&BlocksRange{
			from: big.NewInt(30),
			to:   big.NewInt(40),
		},
		&BlocksRange{
			from: big.NewInt(40),
			to:   big.NewInt(50),
		},
	}

	n, d = getNewRanges(ranges)
	require.Equal(t, 2, len(n))
	newRange = n[0]
	require.Equal(t, int64(0), newRange.from.Int64())
	require.Equal(t, int64(20), newRange.to.Int64())
	newRange = n[1]
	require.Equal(t, int64(30), newRange.from.Int64())
	require.Equal(t, int64(50), newRange.to.Int64())
	require.Equal(t, 4, len(d))
}
