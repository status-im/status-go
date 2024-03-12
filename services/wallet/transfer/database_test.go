package transfer

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/services/wallet/bigint"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

func setupTestDB(t *testing.T) (*Database, *BlockDAO, func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return NewDB(db), &BlockDAO{db}, func() {
		require.NoError(t, db.Close())
	}
}

func TestDBSaveBlocks(t *testing.T) {
	db, _, stop := setupTestDB(t)
	defer stop()
	address := common.Address{1}
	blocks := []*DBHeader{
		{
			Number:  big.NewInt(1),
			Hash:    common.Hash{1},
			Address: address,
		},
		{
			Number:  big.NewInt(2),
			Hash:    common.Hash{2},
			Address: address,
		}}
	require.NoError(t, db.SaveBlocks(777, blocks))
	transfers := []Transfer{
		{
			ID:          common.Hash{1},
			Type:        w_common.EthTransfer,
			BlockHash:   common.Hash{2},
			BlockNumber: big.NewInt(1),
			Address:     address,
			Timestamp:   123,
			From:        address,
		},
	}
	tx, err := db.client.BeginTx(context.Background(), nil)
	require.NoError(t, err)

	require.NoError(t, saveTransfersMarkBlocksLoaded(tx, 777, address, transfers, []*big.Int{big.NewInt(1), big.NewInt(2)}))
	require.NoError(t, tx.Commit())
}

func TestDBSaveTransfers(t *testing.T) {
	db, _, stop := setupTestDB(t)
	defer stop()
	address := common.Address{1}
	header := &DBHeader{
		Number:  big.NewInt(1),
		Hash:    common.Hash{1},
		Address: address,
	}
	tx := types.NewTransaction(1, address, nil, 10, big.NewInt(10), nil)
	transfers := []Transfer{
		{
			ID:                 common.Hash{1},
			Type:               w_common.EthTransfer,
			BlockHash:          header.Hash,
			BlockNumber:        header.Number,
			Transaction:        tx,
			Receipt:            types.NewReceipt(nil, false, 100),
			Address:            address,
			MultiTransactionID: 0,
		},
	}
	require.NoError(t, db.SaveBlocks(777, []*DBHeader{header}))
	require.NoError(t, saveTransfersMarkBlocksLoaded(db.client, 777, address, transfers, []*big.Int{header.Number}))
}

func TestDBGetTransfersFromBlock(t *testing.T) {
	db, _, stop := setupTestDB(t)
	defer stop()
	headers := []*DBHeader{}
	transfers := []Transfer{}
	address := common.Address{1}
	blockNumbers := []*big.Int{}
	for i := 1; i < 10; i++ {
		header := &DBHeader{
			Number:  big.NewInt(int64(i)),
			Hash:    common.Hash{byte(i)},
			Address: address,
		}
		headers = append(headers, header)
		blockNumbers = append(blockNumbers, header.Number)
		tx := types.NewTransaction(uint64(i), address, nil, 10, big.NewInt(10), nil)
		receipt := types.NewReceipt(nil, false, 100)
		receipt.Logs = []*types.Log{}
		transfer := Transfer{
			ID:          tx.Hash(),
			Type:        w_common.EthTransfer,
			BlockNumber: header.Number,
			BlockHash:   header.Hash,
			Transaction: tx,
			Receipt:     receipt,
			Address:     address,
		}
		transfers = append(transfers, transfer)
	}
	require.NoError(t, db.SaveBlocks(777, headers))
	require.NoError(t, saveTransfersMarkBlocksLoaded(db.client, 777, address, transfers, blockNumbers))
	rst, err := db.GetTransfers(777, big.NewInt(7), nil)
	require.NoError(t, err)
	require.Len(t, rst, 1)
}

func TestGetTransfersForIdentities(t *testing.T) {
	db, _, stop := setupTestDB(t)
	defer stop()

	trs, _, _ := GenerateTestTransfers(t, db.client, 1, 4)
	for i := range trs {
		InsertTestTransfer(t, db.client, trs[i].To, &trs[i])
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
	require.Equal(t, uint64(trs[1].ChainID), entries[0].NetworkID)
	require.Equal(t, uint64(trs[3].ChainID), entries[1].NetworkID)
	require.Equal(t, w_common.MultiTransactionIDType(0), entries[0].MultiTransactionID)
	require.Equal(t, w_common.MultiTransactionIDType(0), entries[1].MultiTransactionID)
}

func TestGetLatestCollectibleTransfer(t *testing.T) {
	db, _, stop := setupTestDB(t)
	defer stop()

	trs, _, _ := GenerateTestTransfers(t, db.client, 1, len(TestCollectibles))

	collectible := TestCollectibles[0]
	collectibleID := thirdparty.CollectibleUniqueID{
		ContractID: thirdparty.ContractID{
			ChainID: collectible.ChainID,
			Address: collectible.TokenAddress,
		},
		TokenID: &bigint.BigInt{Int: collectible.TokenID},
	}
	firstTr := trs[0]
	lastTr := firstTr

	// ExtraTrs is a sequence of send+receive of the same collectible
	extraTrs, _, _ := GenerateTestTransfers(t, db.client, len(trs)+1, 2)
	for i := range extraTrs {
		if i%2 == 0 {
			extraTrs[i].From = firstTr.To
			extraTrs[i].To = firstTr.From
		} else {
			extraTrs[i].From = firstTr.From
			extraTrs[i].To = firstTr.To
		}
		extraTrs[i].ChainID = collectible.ChainID
	}

	for i := range trs {
		collectibleData := TestCollectibles[i]
		trs[i].ChainID = collectibleData.ChainID
		InsertTestTransferWithOptions(t, db.client, trs[i].To, &trs[i], &TestTransferOptions{
			TokenAddress: collectibleData.TokenAddress,
			TokenID:      collectibleData.TokenID,
		})
	}

	foundTx, err := db.GetLatestCollectibleTransfer(lastTr.To, collectibleID)
	require.NoError(t, err)
	require.NotEmpty(t, foundTx)
	require.Equal(t, lastTr.Hash, foundTx.ID)

	for i := range extraTrs {
		InsertTestTransferWithOptions(t, db.client, firstTr.To, &extraTrs[i], &TestTransferOptions{
			TokenAddress: collectible.TokenAddress,
			TokenID:      collectible.TokenID,
		})
	}

	lastTr = extraTrs[len(extraTrs)-1]

	foundTx, err = db.GetLatestCollectibleTransfer(lastTr.To, collectibleID)
	require.NoError(t, err)
	require.NotEmpty(t, foundTx)
	require.Equal(t, lastTr.Hash, foundTx.ID)
}
