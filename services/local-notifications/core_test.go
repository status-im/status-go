package localnotifications

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/wallet"

	"github.com/ethereum/go-ethereum/event"
)

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
	require.Equal(t, false, s.IsWatchingWallet())

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

	s := NewService(db, 1777)
	defer s.Stop()

	feed := &event.Feed{}
	require.NoError(t, s.SubscribeWallet(feed))

	s.StartWalletWatcher()
	feed.Send(wallet.Event{Type: wallet.EventNewBlock, BlockNumber: big.NewInt(21)})
	// TODO: Add test for empty block, and for block with transaction
}
