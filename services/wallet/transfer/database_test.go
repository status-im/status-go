package transfer

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/appdatabase"
)

func setupTestDB(t *testing.T) (*Database, *BlockDAO, func()) {
	db, err := appdatabase.SetupTestMemorySQLDB("wallet-transfer-tests")
	require.NoError(t, err)
	return NewDB(db), &BlockDAO{db}, func() {
		require.NoError(t, db.Close())
	}
}

func TestDBGetHeaderByNumber(t *testing.T) {
	db, _, stop := setupTestDB(t)
	defer stop()
	header := &types.Header{
		Number:     big.NewInt(10),
		Difficulty: big.NewInt(1),
		Time:       1,
	}
	require.NoError(t, db.saveHeaders(777, []*types.Header{header}, common.Address{1}))
	rst, err := db.getHeaderByNumber(777, header.Number)
	require.NoError(t, err)
	require.Equal(t, header.Hash(), rst.Hash)
}

func TestDBGetHeaderByNumberNoRows(t *testing.T) {
	db, _, stop := setupTestDB(t)
	defer stop()
	rst, err := db.getHeaderByNumber(777, big.NewInt(1))
	require.NoError(t, err)
	require.Nil(t, rst)
}

func TestDBProcessBlocks(t *testing.T) {
	db, block, stop := setupTestDB(t)
	defer stop()
	address := common.Address{1}
	from := big.NewInt(0)
	to := big.NewInt(10)
	blocks := []*DBHeader{
		&DBHeader{
			Number: big.NewInt(1),
			Hash:   common.Hash{1},
		},
		&DBHeader{
			Number: big.NewInt(2),
			Hash:   common.Hash{2},
		}}
	t.Log(blocks)
	nonce := int64(0)
	lastBlock := &Block{
		Number:  to,
		Balance: big.NewInt(0),
		Nonce:   &nonce,
	}
	require.NoError(t, db.ProcessBlocks(777, common.Address{1}, from, lastBlock, blocks))
	t.Log(block.GetLastBlockByAddress(777, common.Address{1}, 40))
	transfers := []Transfer{
		{
			ID:          common.Hash{1},
			Type:        ethTransfer,
			BlockHash:   common.Hash{2},
			BlockNumber: big.NewInt(1),
			Address:     common.Address{1},
			Timestamp:   123,
			From:        common.Address{1},
		},
	}
	require.NoError(t, db.SaveTransfersMarkBlocksLoaded(777, address, transfers, []*big.Int{big.NewInt(1), big.NewInt(2)}))
}

func TestDBProcessTransfer(t *testing.T) {
	db, _, stop := setupTestDB(t)
	defer stop()
	header := &DBHeader{
		Number:  big.NewInt(1),
		Hash:    common.Hash{1},
		Address: common.Address{1},
	}
	tx := types.NewTransaction(1, common.Address{1}, nil, 10, big.NewInt(10), nil)
	transfers := []Transfer{
		{
			ID:                 common.Hash{1},
			Type:               ethTransfer,
			BlockHash:          header.Hash,
			BlockNumber:        header.Number,
			Transaction:        tx,
			Receipt:            types.NewReceipt(nil, false, 100),
			Address:            common.Address{1},
			MultiTransactionID: 0,
		},
	}
	nonce := int64(0)
	lastBlock := &Block{
		Number:  big.NewInt(0),
		Balance: big.NewInt(0),
		Nonce:   &nonce,
	}
	require.NoError(t, db.ProcessBlocks(777, common.Address{1}, big.NewInt(1), lastBlock, []*DBHeader{header}))
	require.NoError(t, db.ProcessTransfers(777, transfers, []*DBHeader{}))
}

func TestDBReorgTransfers(t *testing.T) {
	db, _, stop := setupTestDB(t)
	defer stop()
	rcpt := types.NewReceipt(nil, false, 100)
	rcpt.Logs = []*types.Log{}
	original := &DBHeader{
		Number:  big.NewInt(1),
		Hash:    common.Hash{1},
		Address: common.Address{1},
	}
	replaced := &DBHeader{
		Number:  big.NewInt(1),
		Hash:    common.Hash{2},
		Address: common.Address{1},
	}
	originalTX := types.NewTransaction(1, common.Address{1}, nil, 10, big.NewInt(10), nil)
	replacedTX := types.NewTransaction(2, common.Address{1}, nil, 10, big.NewInt(10), nil)
	nonce := int64(0)
	lastBlock := &Block{
		Number:  original.Number,
		Balance: big.NewInt(0),
		Nonce:   &nonce,
	}
	require.NoError(t, db.ProcessBlocks(777, original.Address, original.Number, lastBlock, []*DBHeader{original}))
	require.NoError(t, db.ProcessTransfers(777, []Transfer{
		{ethTransfer, common.Hash{1}, *originalTX.To(), original.Number, original.Hash, 100, originalTX, true, 1777, common.Address{1}, rcpt, nil, "2100", NoMultiTransactionID},
	}, []*DBHeader{}))
	nonce = int64(0)
	lastBlock = &Block{
		Number:  replaced.Number,
		Balance: big.NewInt(0),
		Nonce:   &nonce,
	}
	require.NoError(t, db.ProcessBlocks(777, replaced.Address, replaced.Number, lastBlock, []*DBHeader{replaced}))
	require.NoError(t, db.ProcessTransfers(777, []Transfer{
		{ethTransfer, common.Hash{2}, *replacedTX.To(), replaced.Number, replaced.Hash, 100, replacedTX, true, 1777, common.Address{1}, rcpt, nil, "2100", NoMultiTransactionID},
	}, []*DBHeader{original}))

	all, err := db.GetTransfers(777, big.NewInt(0), nil)
	require.NoError(t, err)
	require.Len(t, all, 1)
	require.Equal(t, replacedTX.Hash(), all[0].Transaction.Hash())
}

func TestDBGetTransfersFromBlock(t *testing.T) {
	db, _, stop := setupTestDB(t)
	defer stop()
	headers := []*DBHeader{}
	transfers := []Transfer{}
	for i := 1; i < 10; i++ {
		header := &DBHeader{
			Number:  big.NewInt(int64(i)),
			Hash:    common.Hash{byte(i)},
			Address: common.Address{1},
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
			Address:     common.Address{1},
		}
		transfers = append(transfers, transfer)
	}
	nonce := int64(0)
	lastBlock := &Block{
		Number:  headers[len(headers)-1].Number,
		Balance: big.NewInt(0),
		Nonce:   &nonce,
	}
	require.NoError(t, db.ProcessBlocks(777, headers[0].Address, headers[0].Number, lastBlock, headers))
	require.NoError(t, db.ProcessTransfers(777, transfers, []*DBHeader{}))
	rst, err := db.GetTransfers(777, big.NewInt(7), nil)
	require.NoError(t, err)
	require.Len(t, rst, 3)

	rst, err = db.GetTransfers(777, big.NewInt(2), big.NewInt(5))
	require.NoError(t, err)
	require.Len(t, rst, 4)
}

func TestGetTransfersForIdentities(t *testing.T) {
	db, _, stop := setupTestDB(t)
	defer stop()

	trs := GenerateTestTransactions(t, db.client, 1, 4)
	for i := range trs {
		InsertTestTransfer(t, db.client, &trs[i])
	}

	entries, err := db.GetTransfersForIdentities(context.Background(), []TransactionIdentity{
		TransactionIdentity{trs[1].ChainID, trs[1].Hash, trs[1].To},
		TransactionIdentity{trs[3].ChainID, trs[3].Hash, trs[3].To}})
	require.NoError(t, err)
	require.Equal(t, 2, len(entries))
	require.Equal(t, trs[1].Hash, entries[0].ID)
	require.Equal(t, trs[3].Hash, entries[1].ID)
	require.Equal(t, trs[1].From, entries[0].From)
	require.Equal(t, trs[3].From, entries[1].From)
	require.Equal(t, trs[1].To, entries[0].Address)
	require.Equal(t, trs[3].To, entries[1].Address)
	require.Equal(t, big.NewInt(trs[1].BlkNumber), entries[0].BlockNumber)
	require.Equal(t, big.NewInt(trs[3].BlkNumber), entries[1].BlockNumber)
	require.Equal(t, uint64(trs[1].Timestamp), entries[0].Timestamp)
	require.Equal(t, uint64(trs[3].Timestamp), entries[1].Timestamp)
	require.Equal(t, trs[1].ChainID, entries[0].NetworkID)
	require.Equal(t, trs[3].ChainID, entries[1].NetworkID)
	require.Equal(t, MultiTransactionIDType(0), entries[0].MultiTransactionID)
	require.Equal(t, MultiTransactionIDType(0), entries[1].MultiTransactionID)
}
