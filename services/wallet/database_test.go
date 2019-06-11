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
	db, err := InitializeDB(tmpfile.Name(), "wallet-tests")
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func TestDBGetHeaderByNumber(t *testing.T) {
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
	require.Equal(t, header.Hash(), rst.Hash)
}

func TestDBGetHeaderByNumberNoRows(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	rst, err := db.GetHeaderByNumber(big.NewInt(1))
	require.NoError(t, err)
	require.Nil(t, rst)
}

func TestDBHeaderExists(t *testing.T) {
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

func TestDBHeaderDoesntExist(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	rst, err := db.HeaderExists(common.Hash{1})
	require.NoError(t, err)
	require.False(t, rst)
}

func TestDBProcessTransfer(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	header := &DBHeader{
		Number: big.NewInt(1),
		Hash:   common.Hash{1},
	}
	tx := types.NewTransaction(1, common.Address{1}, nil, 10, big.NewInt(10), nil)
	transfers := []Transfer{
		{
			ID:          common.Hash{1},
			Type:        ethTransfer,
			BlockHash:   header.Hash,
			BlockNumber: header.Number,
			Transaction: tx,
			Receipt:     types.NewReceipt(nil, false, 100),
		},
	}
	require.NoError(t, db.ProcessTranfers(transfers, nil, []*DBHeader{header}, nil, 0))
}

func TestDBReorgTransfers(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	rcpt := types.NewReceipt(nil, false, 100)
	rcpt.Logs = []*types.Log{}
	original := &DBHeader{
		Number: big.NewInt(1),
		Hash:   common.Hash{1},
	}
	replaced := &DBHeader{
		Number: big.NewInt(1),
		Hash:   common.Hash{2},
	}
	originalTX := types.NewTransaction(1, common.Address{1}, nil, 10, big.NewInt(10), nil)
	replacedTX := types.NewTransaction(2, common.Address{1}, nil, 10, big.NewInt(10), nil)
	require.NoError(t, db.ProcessTranfers([]Transfer{
		{ethTransfer, common.Hash{1}, *originalTX.To(), original.Number, original.Hash, originalTX, rcpt},
	}, nil, []*DBHeader{original}, nil, 0))
	require.NoError(t, db.ProcessTranfers([]Transfer{
		{ethTransfer, common.Hash{2}, *replacedTX.To(), replaced.Number, replaced.Hash, replacedTX, rcpt},
	}, nil, []*DBHeader{replaced}, []*DBHeader{original}, 0))

	all, err := db.GetTransfers(big.NewInt(0), nil)
	require.NoError(t, err)
	require.Len(t, all, 1)
	require.Equal(t, replacedTX.Hash(), all[0].Transaction.Hash())
}

func TestDBGetTransfersFromBlock(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	headers := []*DBHeader{}
	transfers := []Transfer{}
	for i := 1; i < 10; i++ {
		header := &DBHeader{
			Number: big.NewInt(int64(i)),
			Hash:   common.Hash{byte(i)},
		}
		headers = append(headers, header)
		tx := types.NewTransaction(uint64(i), common.Address{1}, nil, 10, big.NewInt(10), nil)
		receipt := types.NewReceipt(nil, false, 100)
		receipt.Logs = []*types.Log{}
		transfer := Transfer{
			ID:          tx.Hash(),
			Type:        ethTransfer,
			BlockNumber: header.Number,
			BlockHash:   header.Hash,
			Transaction: tx,
			Receipt:     receipt,
		}
		transfers = append(transfers, transfer)
	}
	require.NoError(t, db.ProcessTranfers(transfers, nil, headers, nil, 0))
	rst, err := db.GetTransfers(big.NewInt(7), nil)
	require.NoError(t, err)
	require.Len(t, rst, 3)

	rst, err = db.GetTransfers(big.NewInt(2), big.NewInt(5))
	require.NoError(t, err)
	require.Len(t, rst, 4)

}

func TestDBLatestSynced(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	address := common.Address{1}
	h1 := &types.Header{
		Number:     big.NewInt(10),
		Difficulty: big.NewInt(1),
		Time:       big.NewInt(1),
	}
	h2 := &types.Header{
		Number:     big.NewInt(9),
		Difficulty: big.NewInt(1),
		Time:       big.NewInt(1),
	}
	require.NoError(t, db.SaveHeader(h1))
	require.NoError(t, db.SaveHeader(h2))
	require.NoError(t, db.SaveSyncedHeader(address, h1, ethSync))
	require.NoError(t, db.SaveSyncedHeader(address, h2, ethSync))

	latest, err := db.GetLatestSynced(address, ethSync)
	require.NoError(t, err)
	require.NotNil(t, latest)
	require.Equal(t, h1.Number, latest.Number)
	require.Equal(t, h1.Hash(), latest.Hash)
}

func TestDBLatestSyncedDoesntExist(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	latest, err := db.GetLatestSynced(common.Address{1}, ethSync)
	require.NoError(t, err)
	require.Nil(t, latest)
}

func TestDBProcessTransfersUpdate(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	address := common.Address{1}
	header := &DBHeader{
		Number: big.NewInt(10),
		Hash:   common.Hash{1},
	}
	transfer := Transfer{
		ID:          common.Hash{1},
		BlockNumber: header.Number,
		BlockHash:   header.Hash,
		Transaction: types.NewTransaction(0, common.Address{}, nil, 0, nil, nil),
		Address:     address,
	}
	require.NoError(t, db.ProcessTranfers([]Transfer{transfer}, []common.Address{address}, []*DBHeader{header}, nil, ethSync))
	require.NoError(t, db.ProcessTranfers([]Transfer{transfer}, []common.Address{address}, []*DBHeader{header}, nil, erc20Sync))

	latest, err := db.GetLatestSynced(address, ethSync|erc20Sync)
	require.NoError(t, err)
	require.Equal(t, header.Hash, latest.Hash)
}

func TestDBLastHeadExist(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	headers := []*DBHeader{
		{Number: big.NewInt(1), Hash: common.Hash{1}, Head: true},
		{Number: big.NewInt(2), Hash: common.Hash{2}, Head: true},
		{Number: big.NewInt(3), Hash: common.Hash{3}, Head: true},
	}
	require.NoError(t, db.ProcessTranfers(nil, nil, headers, nil, 0))
	last, err := db.GetLastHead()
	require.NoError(t, err)
	require.Equal(t, headers[2].Hash, last.Hash)
}

func TestDBLastHeadDoesntExist(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	last, err := db.GetLastHead()
	require.NoError(t, err)
	require.Nil(t, last)
}
