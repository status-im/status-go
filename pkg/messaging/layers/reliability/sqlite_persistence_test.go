package reliability

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/sds-go-bindings/sds"

	"github.com/status-im/status-go/internal/testutils"
	reliabilitymigrations "github.com/status-im/status-go/pkg/messaging/layers/reliability/migrations"
	"github.com/status-im/status-go/pkg/messaging/types"
)

const (
	sdsStateMigration          = "1793010000_add_sds_channel_state.up.sql"
	sdsRetrievalHintsMigration = "1793650000_add_sds_retrieval_hints.up.sql"
	sdsOrphanHintsMigration    = "1793650001_remove_orphan_sds_retrieval_hints.up.sql"
)

func newSDSStateTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer(nil))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	for _, migrationName := range []string{sdsStateMigration, sdsRetrievalHintsMigration, "1794330000_add_sds_received_messages.up.sql"} {
		migration, err := reliabilitymigrations.Asset(migrationName)
		require.NoError(t, err)
		_, err = db.Exec(string(migration))
		require.NoError(t, err)
	}
	return db
}

func TestSQLiteReceivedSDSMessagesSurviveHistoryEviction(t *testing.T) {
	db := newSDSStateTestDB(t)
	store := NewSQLiteRetrievalHintPersistence(db)
	history := NewSQLitePersistence(db)
	require.NoError(t, history.UpdateHistory("channel", sds.HistoryUpdate{
		Append: []sds.HistoryMessage{{MessageID: "outgoing", Data: []byte("outgoing")}},
	}))
	require.NoError(t, store.SaveSDSRetrievalHints("channel", "outgoing", [][]byte{[]byte("hash")}))
	received, err := store.IsSDSMessageReceived("channel", "outgoing")
	require.NoError(t, err)
	require.False(t, received, "outgoing retrieval hints do not prove receipt")
	require.NoError(t, store.SaveSDSMessageReceived("channel", "received"))
	require.NoError(t, store.SaveSDSMessageReceived("channel", "received"))
	require.NoError(t, history.UpdateHistory("channel", sds.HistoryUpdate{Evict: []string{"received"}}))
	require.NoError(t, history.DropChannel("channel"))
	received, err = NewSQLiteRetrievalHintPersistence(db).IsSDSMessageReceived("channel", "received")
	require.NoError(t, err)
	require.True(t, received)
	received, err = store.IsSDSMessageReceived("other-channel", "received")
	require.NoError(t, err)
	require.False(t, received)
	require.Error(t, store.SaveSDSMessageReceived("", "received"))
	require.Error(t, store.SaveSDSMessageReceived("channel", ""))
}
func TestSQLiteSDSPersistenceStoresLoadsAndDropsChannelState(t *testing.T) {
	db := newSDSStateTestDB(t)
	persistence := NewSQLitePersistence(db)

	meta := []byte("channel snapshot")
	require.NoError(t, persistence.SaveChannelMeta("channel-a", meta))
	require.NoError(t, persistence.UpdateHistory("channel-a", sds.HistoryUpdate{
		Append: []sds.HistoryMessage{
			{MessageID: "message-b", LamportTimestamp: 2, Data: []byte("b")},
			{MessageID: "message-a", LamportTimestamp: 1, Data: []byte("a")},
		},
	}))

	data, err := persistence.LoadChannel("channel-a")
	require.NoError(t, err)
	require.Equal(t, meta, data.Meta)
	require.Equal(t, [][]byte{[]byte("a"), []byte("b")}, data.MessageHistory)

	require.NoError(t, persistence.UpdateHistory("channel-a", sds.HistoryUpdate{
		Evict: []string{"message-a"},
	}))
	data, err = persistence.LoadChannel("channel-a")
	require.NoError(t, err)
	require.Equal(t, [][]byte{[]byte("b")}, data.MessageHistory)

	require.NoError(t, persistence.DropChannel("channel-a"))
	data, err = persistence.LoadChannel("channel-a")
	require.NoError(t, err)
	require.Empty(t, data.Meta)
	require.Empty(t, data.MessageHistory)
}

func TestSQLiteReceivedSDSMessageTTL(t *testing.T) {
	db := newSDSStateTestDB(t)
	start := time.Unix(1_800_000_000, 0)
	now := start
	store := &SQLitePersistence{db: db, now: func() time.Time { return now }}
	require.Equal(t, 31*24*time.Hour, types.StoreMessageRetention)
	require.NoError(t, store.SaveSDSMessageReceived("channel", "old"))

	now = start.Add(types.StoreMessageRetention - time.Second)
	require.NoError(t, store.SaveSDSMessageReceived("channel", "old"))
	require.NoError(t, store.SaveSDSMessageReceived("channel", "recent"))
	received, err := store.IsSDSMessageReceived("channel", "old")
	require.NoError(t, err)
	require.True(t, received)
	require.NoError(t, store.DeleteExpiredSDSReceivedMessages())

	// Reopening storage preserves the original receipt time, even after a duplicate.
	store = &SQLitePersistence{db: db, now: func() time.Time { return now }}
	now = start.Add(types.StoreMessageRetention)
	received, err = store.IsSDSMessageReceived("channel", "old")
	require.NoError(t, err)
	require.False(t, received, "expiry must apply before periodic deletion")
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM sds_received_messages").Scan(&count))
	require.Equal(t, 2, count)
	require.NoError(t, store.DeleteExpiredSDSReceivedMessages())
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM sds_received_messages").Scan(&count))
	require.Equal(t, 1, count)
	received, err = store.IsSDSMessageReceived("channel", "recent")
	require.NoError(t, err)
	require.True(t, received)
	require.NoError(t, store.DeleteExpiredSDSReceivedMessages())

	now = now.Add(types.StoreMessageRetention)
	require.NoError(t, store.DeleteExpiredSDSReceivedMessages())
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM sds_received_messages").Scan(&count))
	require.Zero(t, count)
}

func TestSQLiteReceivedSDSMessageErrors(t *testing.T) {
	db := newSDSStateTestDB(t)
	store := NewSQLiteRetrievalHintPersistence(db)
	_, err := db.Exec("DROP TABLE sds_received_messages")
	require.NoError(t, err)
	require.Error(t, store.SaveSDSMessageReceived("channel", "message"))
	received, err := store.IsSDSMessageReceived("channel", "message")
	require.Error(t, err)
	require.False(t, received)
	require.Error(t, store.DeleteExpiredSDSReceivedMessages())
}

func TestSQLiteSDSRetrievalHintsFollowHistoryLifecycle(t *testing.T) {
	db := newSDSStateTestDB(t)
	history := NewSQLitePersistence(db)

	require.NoError(t, history.UpdateHistory("channel-a", sds.HistoryUpdate{
		Append: []sds.HistoryMessage{{MessageID: "message-a", Data: []byte("a")}},
	}))
	require.NoError(t, history.UpdateHistory("channel-b", sds.HistoryUpdate{
		Append: []sds.HistoryMessage{{MessageID: "message-a", Data: []byte("a")}},
	}))
	require.NoError(t, history.SaveSDSRetrievalHints("channel-a", "message-a", [][]byte{
		[]byte("hash-a"),
		[]byte("hash-shared"),
	}))
	require.NoError(t, history.SaveSDSRetrievalHints("channel-b", "message-a", [][]byte{
		[]byte("hash-b"),
		[]byte("hash-shared"),
	}))

	restartedStore := NewSQLiteRetrievalHintPersistence(db)
	got, err := restartedStore.LoadSDSRetrievalHints("channel-a", "message-a")
	require.NoError(t, err)
	require.Equal(t, [][]byte{[]byte("hash-a"), []byte("hash-shared")}, got)
	got, err = restartedStore.LoadSDSRetrievalHints("channel-b", "message-a")
	require.NoError(t, err)
	require.Equal(t, [][]byte{[]byte("hash-b"), []byte("hash-shared")}, got)

	require.NoError(t, history.UpdateHistory("channel-a", sds.HistoryUpdate{
		Evict: []string{"message-a"},
	}))
	require.NoError(t, history.SaveSDSRetrievalHints("channel-a", "message-a", [][]byte{[]byte("late-a")}))
	got, err = restartedStore.LoadSDSRetrievalHints("channel-a", "message-a")
	require.NoError(t, err)
	require.Empty(t, got)
	got, err = restartedStore.LoadSDSRetrievalHints("channel-b", "message-a")
	require.NoError(t, err)
	require.Equal(t, [][]byte{[]byte("hash-b"), []byte("hash-shared")}, got)

	require.NoError(t, history.DropChannel("channel-b"))
	require.NoError(t, history.SaveSDSRetrievalHints("channel-b", "message-a", [][]byte{[]byte("late-b")}))
	got, err = restartedStore.LoadSDSRetrievalHints("channel-b", "message-a")
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestSQLiteSDSPendingHintsAreBoundedAndClearedOnDrop(t *testing.T) {
	store := NewSQLitePersistence(newSDSStateTestDB(t))
	hashes := [][]byte{[]byte("envelope")}
	for i := range maxPendingSDSHints + 1 {
		require.NoError(t, store.SaveSDSRetrievalHints("channel", fmt.Sprint(i), hashes))
	}
	require.Len(t, store.pendingHintIndex, maxPendingSDSHints)
	require.Equal(t, maxPendingSDSHints, store.pendingHints.Len())
	require.NoError(t, store.UpdateHistory("channel", sds.HistoryUpdate{
		Append: []sds.HistoryMessage{{MessageID: "0", Data: []byte("oldest")}},
	}))
	got, err := store.LoadSDSRetrievalHints("channel", "0")
	require.NoError(t, err)
	require.Empty(t, got)

	require.NoError(t, store.DropChannel("channel"))
	require.Empty(t, store.pendingHintIndex)
	require.Zero(t, store.pendingHints.Len())
}

func TestSQLiteMigrationRemovesOrphanSDSRetrievalHints(t *testing.T) {
	db := newSDSStateTestDB(t)
	_, err := db.Exec(`
		INSERT INTO sds_channel_history VALUES ('retained-channel', 'shared-id', 0, X'01');
		INSERT INTO sds_retrieval_hints VALUES
			('retained-channel', 'shared-id', X'01'),
			('other-channel', 'shared-id', X'02'),
			('retained-channel', 'absent-id', X'03')`)
	require.NoError(t, err)

	migration, err := reliabilitymigrations.Asset(sdsOrphanHintsMigration)
	require.NoError(t, err)
	_, err = db.Exec(string(migration))
	require.NoError(t, err)

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM sds_retrieval_hints`).Scan(&count))
	require.Equal(t, 1, count)
	got, err := NewSQLitePersistence(db).LoadSDSRetrievalHints("retained-channel", "shared-id")
	require.NoError(t, err)
	require.Equal(t, [][]byte{{1}}, got)
}
