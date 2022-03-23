package localnotifications

import (
	"database/sql"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/t/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

func createWalletDb(t *testing.T, db *sql.DB) (*transfer.Database, func()) {
	return transfer.NewDB(db), func() {
		require.NoError(t, db.Close())
	}
}

func TestServiceStartStop(t *testing.T) {
	db, stop := setupAppTestDb(t)
	defer stop()

	s, err := NewService(db, 1777)
	require.NoError(t, err)
	require.NoError(t, s.Start())
	require.Equal(t, true, s.IsStarted())

	require.NoError(t, s.Stop())
	require.Equal(t, false, s.IsStarted())
}

func TestWalletSubscription(t *testing.T) {
	db, stop := setupAppTestDb(t)
	defer stop()

	feed := &event.Feed{}
	s, err := NewService(db, 1777)
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

	walletDb, stop := createWalletDb(t, db)
	defer stop()

	s, err := NewService(db, 1777)
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
			Type:        transfer.Type("eth"),
			BlockHash:   header.Hash,
			BlockNumber: header.Number,
			Transaction: tx,
			Receipt:     receipt,
			Address:     header.Address,
		},
	}
	nonce := int64(0)
	lastBlock := &transfer.LastKnownBlock{
		Number:  big.NewInt(1),
		Balance: big.NewInt(0),
		Nonce:   &nonce,
	}
	require.NoError(t, walletDb.ProcessBlocks(1777, header.Address, big.NewInt(1), lastBlock, []*transfer.DBHeader{header}))
	require.NoError(t, walletDb.ProcessTranfers(1777, transfers, []*transfer.DBHeader{}))

	feed.Send(transfer.Event{
		Type:        transfer.EventRecentHistoryReady,
		BlockNumber: big.NewInt(0),
		Accounts:    []common.Address{header.Address},
	})

	feed.Send(transfer.Event{
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
