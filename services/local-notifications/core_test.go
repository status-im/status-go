package localnotifications

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/t/utils"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

func createWalletDb(t *testing.T) (*transfer.Database, func()) {
	db, cleanup, err := helpers.SetupTestSQLDB(walletdatabase.DbInitializer{}, "local-notifications-tests-wallet-")
	require.NoError(t, err)
	return transfer.NewDB(db), func() {
		require.NoError(t, cleanup())
	}
}

func TestServiceStartStop(t *testing.T) {
	db, stop := setupAppTestDb(t)
	defer stop()

	walletDb, walletStop := createWalletDb(t)
	defer walletStop()

	s, err := NewService(db, walletDb, 1777)
	require.NoError(t, err)
	require.NoError(t, s.Start())
	require.Equal(t, true, s.IsStarted())

	require.NoError(t, s.Stop())
	require.Equal(t, false, s.IsStarted())
}

func TestWalletSubscription(t *testing.T) {
	db, stop := setupAppTestDb(t)
	defer stop()

	walletDb, walletStop := createWalletDb(t)
	defer walletStop()

	feed := &event.Feed{}
	s, err := NewService(db, walletDb, 1777)
	require.NoError(t, err)
	require.NoError(t, s.Start())
	require.Equal(t, true, s.IsStarted())

	require.NoError(t, s.SubscribeWallet(feed))
	require.Equal(t, true, s.IsWatchingWallet())

	s.StartWalletWatcher()
	require.Equal(t, true, s.IsWatchingWallet())

	s.StopWalletWatcher()
	require.Equal(t, false, s.IsWatchingWallet())

	require.NoError(t, s.Stop())
	require.Equal(t, false, s.IsStarted())
}

func TestTransactionNotification(t *testing.T) {
	db, stop := setupAppTestDb(t)
	defer stop()

	walletDb, walletStop := createWalletDb(t)
	defer walletStop()

	s, err := NewService(db, walletDb, 1777)
	require.NoError(t, err)
	require.NoError(t, s.Start())
	require.Equal(t, true, s.IsStarted())

	var signalEvent []byte

	signalHandler := signal.MobileSignalHandler(func(s []byte) {
		signalEvent = s
	})

	signal.SetMobileSignalHandler(signalHandler)

	feed := &event.Feed{}
	require.NoError(t, s.SubscribeWallet(feed))
	s.WatchingEnabled = true

	s.StartWalletWatcher()

	header := &transfer.DBHeader{
		Number:  big.NewInt(1),
		Hash:    common.Hash{1},
		Address: common.Address{1},
	}
	tx := types.NewTransaction(1, common.Address{1}, nil, 10, big.NewInt(10), nil)
	receipt := types.NewReceipt(nil, false, 100)
	receipt.Logs = []*types.Log{}
	transfers := []transfer.Transfer{
		{
			ID:          common.Hash{1},
			Type:        w_common.Type("eth"),
			BlockHash:   header.Hash,
			BlockNumber: header.Number,
			Transaction: tx,
			Receipt:     receipt,
			Address:     header.Address,
		},
	}
	require.NoError(t, walletDb.SaveBlocks(1777, []*transfer.DBHeader{header}))
	require.NoError(t, transfer.SaveTransfersMarkBlocksLoaded(walletDb, 1777, header.Address, transfers, []*big.Int{header.Number}))

	feed.Send(walletevent.Event{
		Type:     transfer.EventRecentHistoryReady,
		Accounts: []common.Address{header.Address},
	})

	feed.Send(walletevent.Event{
		Type:        transfer.EventNewTransfers,
		BlockNumber: header.Number,
		Accounts:    []common.Address{header.Address},
	})

	require.NoError(t, utils.Eventually(func() error {
		if signalEvent == nil {
			return fmt.Errorf("signal was not handled")
		}
		require.True(t, strings.Contains(string(signalEvent), `"type":"local-notifications"`))
		require.True(t, strings.Contains(string(signalEvent), `"to":"`+header.Address.Hex()))
		return nil
	}, 2*time.Second, 100*time.Millisecond))

	require.NoError(t, s.Stop())
}
