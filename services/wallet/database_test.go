package wallet

import (
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/appdatabase"
)

func setupTestDB(t *testing.T) (*Database, func()) {
	tmpfile, err := ioutil.TempFile("", "wallet-tests-")
	require.NoError(t, err)
	db, err := appdatabase.InitializeDB(tmpfile.Name(), "wallet-tests")
	require.NoError(t, err)
	return NewDB(db, 1777), func() {
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
		Time:       1,
	}
	require.NoError(t, db.SaveHeaders([]*types.Header{header}, common.Address{1}))
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

func TestDBProcessBlocks(t *testing.T) {
	db, stop := setupTestDB(t)
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
	lastBlock := &LastKnownBlock{
		Number:  to,
		Balance: big.NewInt(0),
		Nonce:   &nonce,
	}
	require.NoError(t, db.ProcessBlocks(common.Address{1}, from, lastBlock, blocks))
	t.Log(db.GetLastBlockByAddress(common.Address{1}, 40))
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
	require.NoError(t, db.SaveTranfers(address, transfers, []*big.Int{big.NewInt(1), big.NewInt(2)}))
}

func TestDBProcessTransfer(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	header := &DBHeader{
		Number:  big.NewInt(1),
		Hash:    common.Hash{1},
		Address: common.Address{1},
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
			Address:     common.Address{1},
		},
	}
	nonce := int64(0)
	lastBlock := &LastKnownBlock{
		Number:  big.NewInt(0),
		Balance: big.NewInt(0),
		Nonce:   &nonce,
	}
	require.NoError(t, db.ProcessBlocks(common.Address{1}, big.NewInt(1), lastBlock, []*DBHeader{header}))
	require.NoError(t, db.ProcessTranfers(transfers, []*DBHeader{}))
}

func TestDBReorgTransfers(t *testing.T) {
	db, stop := setupTestDB(t)
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
	lastBlock := &LastKnownBlock{
		Number:  original.Number,
		Balance: big.NewInt(0),
		Nonce:   &nonce,
	}
	require.NoError(t, db.ProcessBlocks(original.Address, original.Number, lastBlock, []*DBHeader{original}))
	require.NoError(t, db.ProcessTranfers([]Transfer{
		{ethTransfer, common.Hash{1}, *originalTX.To(), original.Number, original.Hash, 100, originalTX, true, 1777, common.Address{1}, rcpt, nil},
	}, []*DBHeader{}))
	nonce = int64(0)
	lastBlock = &LastKnownBlock{
		Number:  replaced.Number,
		Balance: big.NewInt(0),
		Nonce:   &nonce,
	}
	require.NoError(t, db.ProcessBlocks(replaced.Address, replaced.Number, lastBlock, []*DBHeader{replaced}))
	require.NoError(t, db.ProcessTranfers([]Transfer{
		{ethTransfer, common.Hash{2}, *replacedTX.To(), replaced.Number, replaced.Hash, 100, replacedTX, true, 1777, common.Address{1}, rcpt, nil},
	}, []*DBHeader{original}))

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
	lastBlock := &LastKnownBlock{
		Number:  headers[len(headers)-1].Number,
		Balance: big.NewInt(0),
		Nonce:   &nonce,
	}
	require.NoError(t, db.ProcessBlocks(headers[0].Address, headers[0].Number, lastBlock, headers))
	require.NoError(t, db.ProcessTranfers(transfers, []*DBHeader{}))
	rst, err := db.GetTransfers(big.NewInt(7), nil)
	require.NoError(t, err)
	require.Len(t, rst, 3)

	rst, err = db.GetTransfers(big.NewInt(2), big.NewInt(5))
	require.NoError(t, err)
	require.Len(t, rst, 4)

}

func TestCustomTokens(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	rst, err := db.GetCustomTokens()
	require.NoError(t, err)
	require.Nil(t, rst)

	token := Token{
		Address:  common.Address{1},
		Name:     "Zilliqa",
		Symbol:   "ZIL",
		Decimals: 12,
		Color:    "#fa6565",
	}

	err = db.AddCustomToken(token)
	require.NoError(t, err)

	rst, err = db.GetCustomTokens()
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, token, *rst[0])

	err = db.DeleteCustomToken(token.Address)
	require.NoError(t, err)

	rst, err = db.GetCustomTokens()
	require.NoError(t, err)
	require.Equal(t, 0, len(rst))
}

func TestPendingTransactions(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	trx := PendingTransaction{
		TransactionHash: common.Hash{1},
		BlockNumber:     big.NewInt(1),
		From:            common.Address{1},
		To:              common.Address{2},
		Type:            RegisterENS,
		Data:            "someuser.stateofus.eth",
	}

	rst, err := db.GetPendingTransactions()
	require.NoError(t, err)
	require.Nil(t, rst)

	rst, err = db.GetPendingOutboundTransactionsByAddress(trx.From)
	require.NoError(t, err)
	require.Nil(t, rst)

	err = db.StorePendingTransaction(trx)
	require.NoError(t, err)

	rst, err = db.GetPendingOutboundTransactionsByAddress(trx.From)
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, trx, *rst[0])

	rst, err = db.GetPendingTransactions()
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, trx, *rst[0])

	rst, err = db.GetPendingOutboundTransactionsByAddress(common.Address{2})
	require.NoError(t, err)
	require.Nil(t, rst)

	err = db.DeletePendingTransaction(trx.TransactionHash)
	require.NoError(t, err)

	rst, err = db.GetPendingOutboundTransactionsByAddress(trx.From)
	require.NoError(t, err)
	require.Equal(t, 0, len(rst))

	rst, err = db.GetPendingTransactions()
	require.NoError(t, err)
	require.Equal(t, 0, len(rst))

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

func TestUpsertRange(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	r := &BlocksRange{
		from: big.NewInt(0),
		to:   big.NewInt(10),
	}
	nonce := uint64(199)
	balance := big.NewInt(7657)
	account := common.Address{2}

	err := db.UpsertRange(account, db.network, r.from, r.to, balance, nonce)
	require.NoError(t, err)

	block, err := db.GetLastKnownBlockByAddress(account)
	require.NoError(t, err)

	require.Equal(t, 0, block.Number.Cmp(r.to))
	require.Equal(t, 0, block.Balance.Cmp(balance))
	require.Equal(t, nonce, uint64(*block.Nonce))
}
