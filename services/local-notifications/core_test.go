package localnotifications

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/t/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

func createWalletDb(t *testing.T, db *sql.DB) (*wallet.Database, func()) {
	return wallet.NewDB(db, 1777), func() {
		require.NoError(t, db.Close())
	}
}

func TestServiceStartStop(t *testing.T) {
	db, stop := setupAppTestDb(t)
	defer stop()

	s := NewService(db, 1777)
	require.NoError(t, s.Start(nil))
	require.Equal(t, true, s.IsStarted())

	require.NoError(t, s.Stop())
	require.Equal(t, false, s.IsStarted())
}

func TestWalletSubscription(t *testing.T) {
	db, stop := setupAppTestDb(t)
	defer stop()

	feed := &event.Feed{}
	s := NewService(db, 1777)
	require.NoError(t, s.Start(nil))
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

	walletDb, stop := createWalletDb(t, db)
	defer stop()

	s := NewService(db, 1777)
	require.NoError(t, s.Start(nil))
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

	header := &wallet.DBHeader{
		Number:  big.NewInt(1),
		Hash:    common.Hash{1},
		Address: common.Address{1},
	}
	tx := types.NewTransaction(1, common.Address{1}, nil, 10, big.NewInt(10), nil)
	receipt := types.NewReceipt(nil, false, 100)
	receipt.Logs = []*types.Log{}
	transfers := []wallet.Transfer{
		{
			ID:          common.Hash{1},
			Type:        wallet.TransferType("eth"),
			BlockHash:   header.Hash,
			BlockNumber: header.Number,
			Transaction: tx,
			Receipt:     receipt,
			Address:     header.Address,
		},
	}
	require.NoError(t, walletDb.ProcessBlocks(header.Address, big.NewInt(1), big.NewInt(1), []*wallet.DBHeader{header}))
	require.NoError(t, walletDb.ProcessTranfers(transfers, []*wallet.DBHeader{}))

	feed.Send(wallet.Event{
		Type:        wallet.EventMaxKnownBlock,
		BlockNumber: big.NewInt(0),
		Accounts:    []common.Address{header.Address},
	})

	feed.Send(wallet.Event{
		Type:        wallet.EventNewBlock,
		BlockNumber: header.Number,
		Accounts:    []common.Address{header.Address},
	})

	require.NoError(t, utils.Eventually(func() error {
		if signalEvent == nil {
			return fmt.Errorf("Signal was not handled")
		}
		notification := struct {
			Type  string
			Event Notification
		}{}

		require.NoError(t, json.Unmarshal(signalEvent, &notification))

		if notification.Type != "local-notifications" {
			return fmt.Errorf("Wrong signal was sent")
		}
		if notification.Event.Body.To != header.Address {
			return fmt.Errorf("Transaction to address is wrong")
		}
		return nil
	}, 2*time.Second, 100*time.Millisecond))

	require.NoError(t, s.Stop())
}
