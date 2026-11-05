package reliability

import (
	"database/sql"
	"testing"

	"github.com/status-im/status-go/internal/testutils"
	reliabilitymigrations "github.com/status-im/status-go/pkg/messaging/layers/reliability/migrations"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/sds-go-bindings/sds"
)

const sdsStateMigration = "1793010000_add_sds_channel_state.up.sql"

func newSDSStateTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer(nil))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	migration, err := reliabilitymigrations.Asset(sdsStateMigration)
	require.NoError(t, err)
	_, err = db.Exec(string(migration))
	require.NoError(t, err)
	return db
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
