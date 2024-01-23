package transfer

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/services/accounts/accountsevent"
	"github.com/status-im/status-go/services/wallet/blockchainstate"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

func TestController_watchAccountsChanges(t *testing.T) {
	appDB, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	accountsDB, err := accounts.NewDB(appDB)
	require.NoError(t, err)

	walletDB, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	accountFeed := &event.Feed{}

	bcstate := blockchainstate.NewBlockChainState()
	c := NewTransferController(
		walletDB,
		accountsDB,
		nil, // rpcClient
		accountFeed,
		nil, // transferFeed
		nil, // transactionManager
		nil, // pendingTxManager
		nil, // tokenManager
		nil, // balanceCacher
		bcstate,
	)

	address := common.HexToAddress("0x1234")
	chainID := uint64(777)
	// Insert blocks
	database := NewDB(walletDB)
	err = database.SaveBlocks(chainID, []*DBHeader{
		{
			Number:  big.NewInt(1),
			Hash:    common.Hash{1},
			Network: chainID,
			Address: address,
			Loaded:  false,
		},
	})
	require.NoError(t, err)

	// Insert transfers
	err = saveTransfersMarkBlocksLoaded(walletDB, chainID, address, []Transfer{
		{
			ID:          common.Hash{1},
			BlockHash:   common.Hash{1},
			BlockNumber: big.NewInt(1),
			Address:     address,
			NetworkID:   chainID,
		},
	}, []*big.Int{big.NewInt(1)})
	require.NoError(t, err)

	// Insert block ranges
	blockRangesDAO := &BlockRangeSequentialDAO{walletDB}
	err = blockRangesDAO.upsertRange(chainID, address, newEthTokensBlockRanges())
	require.NoError(t, err)

	ranges, err := blockRangesDAO.getBlockRange(chainID, address)
	require.NoError(t, err)
	require.NotNil(t, ranges)

	ch := make(chan accountsevent.Event)
	// Subscribe for account changes
	accountFeed.Subscribe(ch)

	// Watching accounts must start before sending event.
	// To avoid running goroutine immediately, use any delay.
	go func() {
		time.Sleep(1 * time.Millisecond)

		accountFeed.Send(accountsevent.Event{
			Type:     accountsevent.EventTypeRemoved,
			Accounts: []common.Address{address},
		})
	}()

	c.startAccountWatcher([]uint64{chainID})

	// Wait for event
	<-ch

	// Wait for DB to be cleaned up
	c.accWatcher.Stop()

	// Check that transfers, blocks and block ranges were deleted
	transfers, err := database.GetTransfersByAddress(chainID, address, big.NewInt(2), 1)
	require.NoError(t, err)
	require.Len(t, transfers, 0)

	blocksDAO := &BlockDAO{walletDB}
	block, err := blocksDAO.GetLastBlockByAddress(chainID, address, 1)
	require.NoError(t, err)
	require.Nil(t, block)

	ranges, err = blockRangesDAO.getBlockRange(chainID, address)
	require.NoError(t, err)
	require.Nil(t, ranges.eth)
	require.Nil(t, ranges.tokens)

}

func TestController_cleanupAccountLeftovers(t *testing.T) {
	appDB, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	walletDB, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	accountsDB, err := accounts.NewDB(appDB)
	require.NoError(t, err)

	removedAddr := common.HexToAddress("0x5678")
	existingAddr := types.HexToAddress("0x1234")
	accounts := []*accounts.Account{
		{Address: existingAddr, Chat: false, Wallet: true},
	}
	err = accountsDB.SaveOrUpdateAccounts(accounts, false)
	require.NoError(t, err)

	storedAccs, err := accountsDB.GetWalletAddresses()
	require.NoError(t, err)
	require.Len(t, storedAccs, 1)

	bcstate := blockchainstate.NewBlockChainState()
	c := NewTransferController(
		walletDB,
		accountsDB,
		nil, // rpcClient
		nil, // accountFeed
		nil, // transferFeed
		nil, // transactionManager
		nil, // pendingTxManager
		nil, // tokenManager
		nil, // balanceCacher
		bcstate,
	)
	chainID := uint64(777)
	// Insert blocks
	database := NewDB(walletDB)
	err = database.SaveBlocks(chainID, []*DBHeader{
		{
			Number:  big.NewInt(1),
			Hash:    common.Hash{1},
			Network: chainID,
			Address: removedAddr,
			Loaded:  false,
		},
	})
	require.NoError(t, err)
	err = database.SaveBlocks(chainID, []*DBHeader{
		{
			Number:  big.NewInt(2),
			Hash:    common.Hash{2},
			Network: chainID,
			Address: common.Address(existingAddr),
			Loaded:  false,
		},
	})
	require.NoError(t, err)

	blocksDAO := &BlockDAO{walletDB}
	block, err := blocksDAO.GetLastBlockByAddress(chainID, removedAddr, 1)
	require.NoError(t, err)
	require.NotNil(t, block)
	block, err = blocksDAO.GetLastBlockByAddress(chainID, common.Address(existingAddr), 1)
	require.NoError(t, err)
	require.NotNil(t, block)

	// Insert transfers
	err = saveTransfersMarkBlocksLoaded(walletDB, chainID, removedAddr, []Transfer{
		{
			ID:          common.Hash{1},
			BlockHash:   common.Hash{1},
			BlockNumber: big.NewInt(1),
			Address:     removedAddr,
			NetworkID:   chainID,
		},
	}, []*big.Int{big.NewInt(1)})
	require.NoError(t, err)

	err = saveTransfersMarkBlocksLoaded(walletDB, chainID, common.Address(existingAddr), []Transfer{
		{
			ID:          common.Hash{2},
			BlockHash:   common.Hash{2},
			BlockNumber: big.NewInt(2),
			Address:     common.Address(existingAddr),
			NetworkID:   chainID,
		},
	}, []*big.Int{big.NewInt(2)})
	require.NoError(t, err)

	err = c.cleanupAccountsLeftovers()
	require.NoError(t, err)

	// Check that transfers and blocks of removed account were deleted
	transfers, err := database.GetTransfers(chainID, big.NewInt(1), big.NewInt(2))
	require.NoError(t, err)
	require.Len(t, transfers, 1)
	require.Equal(t, transfers[0].Address, common.Address(existingAddr))

	block, err = blocksDAO.GetLastBlockByAddress(chainID, removedAddr, 1)
	require.NoError(t, err)
	require.Nil(t, block)

	// Make sure that transfers and blocks of existing account were not deleted
	existingBlock, err := blocksDAO.GetLastBlockByAddress(chainID, common.Address(existingAddr), 1)
	require.NoError(t, err)
	require.NotNil(t, existingBlock)
}
