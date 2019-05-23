package wallet

import (
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*Database, func()) {
	tmpfile, err := ioutil.TempFile("", "wallet-tests-")
	require.NoError(t, err)
	db, err := InitializeDB(tmpfile.Name())
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func TestGetHeaderByNumber(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	header := &types.Header{
		Number:     big.NewInt(10),
		Difficulty: big.NewInt(1),
		Time:       big.NewInt(1),
	}
	require.NoError(t, db.SaveHeader(header))
	rst, err := db.GetHeaderByNumber(header.Number)
	require.NoError(t, err)
	require.Equal(t, header.Hash(), rst.Hash())
}

func TestHeaderExists(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	header := &types.Header{
		Number:     big.NewInt(10),
		Difficulty: big.NewInt(1),
		Time:       big.NewInt(1),
	}
	require.NoError(t, db.SaveHeader(header))
	rst, err := db.HeaderExists(header.Hash())
	require.NoError(t, err)
	require.True(t, rst)
}

func TestHeaderDoesntExist(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	rst, err := db.HeaderExists(common.Hash{1})
	require.NoError(t, err)
	require.False(t, rst)
}

func TestLastHeader(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	template := types.Header{
		Difficulty: big.NewInt(1),
		Time:       big.NewInt(1),
	}
	first := template
	first.Number = big.NewInt(10)
	second := template
	second.Number = big.NewInt(11)
	require.NoError(t, db.SaveHeader(&second))
	require.NoError(t, db.SaveHeader(&first))

	rst, err := db.LastHeader()
	require.NoError(t, err)
	require.Equal(t, second.Hash(), rst.Hash())
}

func TestProcessTransfer(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	header := &types.Header{
		Number:     big.NewInt(1),
		Difficulty: big.NewInt(1),
		Time:       big.NewInt(1),
	}
	tx := types.NewTransaction(1, common.Address{1}, nil, 10, big.NewInt(10), nil)
	transfers := []Transfer{
		{
			Type:        ethTransfer,
			Header:      header,
			Transaction: tx,
			Receipt:     types.NewReceipt(nil, false, 100),
		},
	}
	require.NoError(t, db.ProcessTranfers(transfers, []*types.Header{header}, nil))
}

func TestReorgTransfers(t *testing.T) {
	require.FailNow(t, "not implemented")
}

func TestGetTransfersFromBlock(t *testing.T) {
	require.FailNow(t, "not implemented")
}
