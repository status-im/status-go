package storagestats

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/walletdb"
	protocolsqlite "github.com/status-im/status-go/internal/protocol/sqlite"
	"github.com/status-im/status-go/internal/testutils"
)

// collectedAt pins "now" so that every day count in the assertions is exact.
var collectedAt = time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)

func setupAppDB(t *testing.T) *sql.DB {
	db, cleanup, err := testutils.SetupTestSQLDB(appdatabase.DbInitializer{}, "storage-stats-app-")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, cleanup()) })

	// The protocol tables (chats, messages, contacts, ...) live in the app
	// database file but are migrated by the messenger, not by appdatabase.
	require.NoError(t, protocolsqlite.Migrate(db))
	return db
}

func setupWalletDB(t *testing.T) *sql.DB {
	db, cleanup, err := testutils.SetupTestSQLDB(walletdb.DbInitializer{}, "storage-stats-wallet-")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, cleanup()) })
	return db
}

func exec(t *testing.T, db *sql.DB, query string, args ...interface{}) {
	t.Helper()
	_, err := db.Exec(query, args...)
	require.NoError(t, err, query)
}

// seedAccount writes an account whose shape is known exactly, so that the
// assertions below can be literal numbers rather than recomputed formulas.
func seedAccount(t *testing.T, appDB, walletDB *sql.DB) {
	daysAgoSeconds := func(days int) int64 { return collectedAt.AddDate(0, 0, -days).Unix() }
	// The same text format sqlite's datetime('now') produces, but pinned to
	// collectedAt so the assertions do not drift with wall-clock time.
	daysAgoDatetime := func(days int) string {
		return collectedAt.AddDate(0, 0, -days).UTC().Format("2006-01-02 15:04:05")
	}
	daysAgoMillis := func(days int) int64 { return collectedAt.AddDate(0, 0, -days).UnixMilli() }

	// 11 chats: 3 one-to-one, 2 public, 1 group, 4 community channels and one
	// deprecated profile chat that must land in "other".
	chatTypes := []int{1, 1, 1, 2, 2, 3, 6, 6, 6, 6, 4}
	// Three chats carry a sync position; the rest have never synced.
	syncedToDaysAgo := map[int]int{0: 10, 1: 60, 2: 20}
	for i, chatType := range chatTypes {
		syncedTo := int64(0)
		if days, ok := syncedToDaysAgo[i]; ok {
			syncedTo = daysAgoSeconds(days)
		}
		exec(t, appDB,
			`INSERT INTO chats (id, name, type, active, timestamp, synced_to) VALUES (?, ?, ?, 1, ?, ?)`,
			fmt.Sprintf("chat-%d", i), fmt.Sprintf("chat %d", i), chatType, daysAgoMillis(400), syncedTo)
	}

	// 200 messages over three chats: 5, 15 and 180. The remaining eight chats
	// hold none, which is exactly the case a GROUP BY cannot see.
	messageID := 0
	insertMessages := func(chatID string, count int, ageDays int) {
		for i := 0; i < count; i++ {
			messageID++
			exec(t, appDB, `
				INSERT INTO user_messages
					(id, whisper_timestamp, source, text, content_type, timestamp, chat_id, local_chat_id, clock_value, seen)
				VALUES (?, ?, 'source', 'text', 1, ?, ?, ?, ?, 1)`,
				fmt.Sprintf("msg-%d", messageID), daysAgoMillis(ageDays), daysAgoMillis(ageDays),
				chatID, chatID, messageID)
		}
	}
	// 20 messages in the last 7 days, 20 more inside 30 days, the rest older,
	// and the oldest message exactly 370 days back.
	insertMessages("chat-0", 5, 3)
	insertMessages("chat-1", 15, 3)
	insertMessages("chat-2", 20, 20)
	insertMessages("chat-2", 159, 200)
	insertMessages("chat-2", 1, 370)

	for i := 0; i < 5; i++ {
		// Two of the five contacts are mutual: request sent and received.
		requestState, remoteState := 0, 0
		if i < 2 {
			requestState, remoteState = contactRequestStateSent, contactRequestStateReceived
		}
		exec(t, appDB, `
			INSERT INTO contacts (id, address, name, alias, identicon, photo, tribute_to_talk,
				contact_request_state, contact_request_remote_state)
			VALUES (?, '', '', '', '', '', '', ?, ?)`,
			fmt.Sprintf("contact-%d", i), requestState, remoteState)
	}

	for i := 0; i < 7; i++ {
		read := 1
		if i < 3 {
			read = 0
		}
		exec(t, appDB, `
			INSERT INTO activity_center_notifications (id, timestamp, notification_type, read)
			VALUES (?, ?, 1, ?)`, fmt.Sprintf("notification-%d", i), daysAgoMillis(1), read)
	}

	for i := 0; i < 2; i++ {
		exec(t, appDB, `INSERT INTO communities_communities (id, private_key, description) VALUES (?, '', '')`,
			[]byte(fmt.Sprintf("community-%d", i)))
	}

	// Wallet accounts live in the app database; the oldest one dates the
	// profile. created_at must be seeded the way production writes it -
	// datetime('now') text, not unix seconds - or the test cannot catch a
	// profile-age query that only works on numbers.
	exec(t, appDB, `INSERT INTO keypairs (key_uid, name, type) VALUES ('key-uid', 'profile', 'profile')`)
	accounts := []struct {
		chat    int
		removed int
		ageDays int
	}{
		{chat: 1, removed: 0, ageDays: 400}, // the chat account, created with the profile
		{chat: 0, removed: 0, ageDays: 390},
		{chat: 0, removed: 0, ageDays: 380},
		{chat: 0, removed: 1, ageDays: 370}, // soft-deleted, must not be counted
	}
	for i, a := range accounts {
		exec(t, appDB, `
			INSERT INTO keypairs_accounts (address, key_uid, pubkey, path, name, color, emoji,
				wallet, chat, removed, hidden, operable, created_at, updated_at)
			VALUES (?, 'key-uid', '', '', '', '', '', 0, ?, ?, 0, 'fully', ?, ?)`,
			fmt.Sprintf("0xaddress-%d", i), a.chat, a.removed,
			daysAgoDatetime(a.ageDays), daysAgoDatetime(0))
	}

	// The transport and encryption migration trees share the app database file.
	for i := 0; i < 2; i++ {
		exec(t, appDB, `
			INSERT INTO installations (identity, installation_id, timestamp, enabled)
			VALUES (?, ?, ?, 1)`, []byte("identity"), fmt.Sprintf("installation-%d", i), daysAgoSeconds(1))
	}
	for i := 0; i < 6; i++ {
		exec(t, appDB, `INSERT INTO mailserver_topics (topic, chat_ids) VALUES (?, '')`,
			fmt.Sprintf("0x0000000%d", i))
	}

	for i := 0; i < 4; i++ {
		exec(t, walletDB, `
			INSERT INTO collectibles_ownership_cache (chain_id, contract_address, token_id, owner_address)
			VALUES (1, ?, ?, '0xowner')`, fmt.Sprintf("0xcontract-%d", i), []byte{byte(i)})
	}
	for i := 0; i < 9; i++ {
		exec(t, walletDB, `
			INSERT INTO token_balances (user_address, token_address, balance, chain_id)
			VALUES (?, '0xtoken', '1', ?)`, "0xuser", i)
	}
	for i := 0; i < 3; i++ {
		exec(t, walletDB, `
			INSERT INTO tokens (address, network_id, name, symbol, decimals)
			VALUES (?, 1, 'Token', ?, 18)`, fmt.Sprintf("0xcustom-%d", i), fmt.Sprintf("TKN%d", i))
	}
	for i := 0; i < 12; i++ {
		exec(t, walletDB, `
			INSERT INTO fetched_alchemy_transfers (transfer, chain_id, address)
			VALUES (?, 1, ?)`, fmt.Sprintf(`{"hash":"0x%d"}`, i), []byte("0xaddress"))
	}
	for i := 0; i < 2; i++ {
		exec(t, walletDB, `
			INSERT INTO saved_addresses (address, name, is_test) VALUES (?, ?, 0)`,
			fmt.Sprintf("0xsaved-%d", i), fmt.Sprintf("saved %d", i))
	}
}

func TestCollectProducesTheExpectedProfile(t *testing.T) {
	appDB := setupAppDB(t)
	walletDB := setupWalletDB(t)
	seedAccount(t, appDB, walletDB)

	profile, err := Collect(context.Background(), appDB, walletDB, collectedAt, zap.NewNop(), nil)
	require.NoError(t, err)

	require.Equal(t, ProfileVersion, profile.ProfileVersion)

	m := profile.Messaging
	require.Equal(t, int64(200), m.MessagesTotal)
	require.Equal(t, ChatCounts{OneToOne: 3, Public: 2, Group: 1, CommunityChannels: 4, Other: 1}, m.Chats)
	require.Equal(t, int64(2), m.Communities)
	require.Equal(t, ContactCounts{Total: 5, Mutual: 2}, m.Contacts)
	require.Equal(t, ActivityCenter{Total: 7, Unread: 3}, m.ActivityCenter)
	require.Equal(t, int64(370), m.OldestMessageDays)
	require.Equal(t, RecentTailPct{D7: 10, D30: 20}, m.RecentTailPct)

	require.Equal(t, Histogram{
		{Label: "0-9", Chats: 9}, // one chat with 5 messages plus the eight empty ones
		{Label: "10-99", Chats: 1},
		{Label: "100-999", Chats: 1},
		{Label: "1000-9999", Chats: 0},
		{Label: "10000+", Chats: 0},
	}, m.PerChatHistogram)

	require.Equal(t, Sync{
		MaxSyncGapDays:    60,
		MedianSyncGapDays: 20,
		// 11 active chats, 3 of which carry a sync position.
		NeverSyncedChats: 8,
		WakuFilters:      6,
		Installations:    2,
	}, profile.Sync)

	// created_at is seeded as datetime text; a query that cannot read that
	// format reports 0, which would read as "profile created today".
	require.Equal(t, int64(400), profile.ProfileAgeDays)

	// Every wallet counter has to be wired to a table that actually exists;
	// a renamed table would otherwise report a silent, plausible zero.
	require.Equal(t, Wallet{
		Accounts:         2,
		Collectibles:     4,
		TokenBalanceRows: 9,
		CustomTokens:     3,
		ActivityRows:     12,
		SavedAddresses:   2,
	}, profile.Wallet)

	require.Positive(t, profile.DB.AppDBBytes)
	require.Positive(t, profile.DB.AppDB.PageCount)
	require.Positive(t, profile.DB.AppDB.PageSize)
	require.Positive(t, profile.DB.AppDB.SchemaVersion)
	require.Contains(t, profile.DB.AppDB.MigrationVersions, "status_go_schema_migrations")
	// The app database file is shared by several migration trees, and their
	// table names both prefix and suffix the marker.
	require.Contains(t, profile.DB.AppDB.MigrationVersions, "status_protocol_go_schema_migrations")
	require.Contains(t, profile.DB.AppDB.MigrationVersions, "status_schema_migrations_transport")
	require.Contains(t, profile.DB.AppDB.MigrationVersions, "mvds_schema_migrations")
	require.Positive(t, profile.DB.WalletDBBytes)

	// Section 2 must cover every table in both files, blob weight included.
	require.Equal(t, int64(200), profile.Tables.AppDB["user_messages"].Rows)
	require.Positive(t, profile.Tables.AppDB["user_messages"].Bytes)
	require.Equal(t, int64(11), profile.Tables.AppDB["chats"].Rows)
	require.Equal(t, int64(4), profile.Tables.WalletDB["collectibles_ownership_cache"].Rows)
	require.Greater(t, len(profile.Tables.AppDB), 50)
	require.Greater(t, len(profile.Tables.WalletDB), 20)
}

// The artifact a reporter pastes into a ticket and the data the UI drew must be
// the same profile.
func TestCollectedProfileSurvivesTheJSONRoundTrip(t *testing.T) {
	appDB := setupAppDB(t)
	walletDB := setupWalletDB(t)
	seedAccount(t, appDB, walletDB)

	profile, err := Collect(context.Background(), appDB, walletDB, collectedAt, zap.NewNop(), nil)
	require.NoError(t, err)

	rendered, err := json.Marshal(profile)
	require.NoError(t, err)

	var parsed Profile
	require.NoError(t, json.Unmarshal(rendered, &parsed))
	require.Equal(t, *profile, parsed)
}

func TestCollectReportsDeterministicProgress(t *testing.T) {
	appDB := setupAppDB(t)
	walletDB := setupWalletDB(t)

	var steps []int
	var totals []int
	_, err := Collect(context.Background(), appDB, walletDB, collectedAt, zap.NewNop(),
		func(done, total int) {
			steps = append(steps, done)
			totals = append(totals, total)
		})
	require.NoError(t, err)

	require.NotEmpty(t, steps)
	for i, done := range steps {
		require.Equal(t, i+1, done, "progress must be reported once per step, in order")
		require.Equal(t, totals[0], totals[i], "the total must be known upfront and never move")
	}
	require.Equal(t, len(steps), totals[0])
}

func TestCollectSurvivesAWalletDatabaseThatIsNotThere(t *testing.T) {
	appDB := setupAppDB(t)
	seedAccount(t, appDB, setupWalletDB(t))

	profile, err := Collect(context.Background(), appDB, nil, collectedAt, zap.NewNop(), nil)
	require.NoError(t, err)

	require.Equal(t, int64(200), profile.Messaging.MessagesTotal)
	require.Equal(t, Wallet{Accounts: 2}, profile.Wallet,
		"with no wallet database only the app-database-backed counter survives")
	require.Empty(t, profile.Tables.WalletDB)
	require.Equal(t, int64(0), profile.DB.WalletDBBytes)
}

func TestCollectRefusesWithoutAnAppDatabase(t *testing.T) {
	_, err := Collect(context.Background(), nil, nil, collectedAt, zap.NewNop(), nil)
	require.Error(t, err)
}

func TestCollectStopsWhenTheContextIsCancelled(t *testing.T) {
	appDB := setupAppDB(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Collect(ctx, appDB, nil, collectedAt, zap.NewNop(), nil)
	require.ErrorIs(t, err, context.Canceled)
}

// A step that fails must say so. Without this the fields it feeds stay at zero,
// and a zero nobody flagged reads as "this account has none" - which is exactly
// how a table renamed out from under us stays invisible in a ticket.
func TestCollectNamesTheStepsThatFailed(t *testing.T) {
	appDB := setupAppDB(t)
	seedAccount(t, appDB, setupWalletDB(t))

	// The sync step counts installations with `deleted = 0`; removing the
	// column makes that query fail the way a schema change would.
	exec(t, appDB, `ALTER TABLE installations DROP COLUMN deleted`)

	profile, err := Collect(context.Background(), appDB, nil, collectedAt, zap.NewNop(), nil)
	require.NoError(t, err, "one broken step must not cost the whole profile")

	require.Contains(t, profile.Incomplete, "sync")
	require.Equal(t, int64(0), profile.Sync.Installations)
	// Everything else still has to be collected.
	require.Equal(t, int64(200), profile.Messaging.MessagesTotal)
}

func TestCollectReportsNothingIncompleteOnAHealthyAccount(t *testing.T) {
	appDB := setupAppDB(t)
	walletDB := setupWalletDB(t)
	seedAccount(t, appDB, walletDB)

	profile, err := Collect(context.Background(), appDB, walletDB, collectedAt, zap.NewNop(), nil)
	require.NoError(t, err)
	require.Empty(t, profile.Incomplete)
	require.NotNil(t, profile.Incomplete, "the field must always be present, empty or not")
}

func TestHistogramBucketsChatsByMessageCount(t *testing.T) {
	h := newHistogram()
	for _, count := range []int64{0, 5, 9, 10, 99, 100, 999, 1000, 9999, 10_000, 250_000} {
		h.add(count)
	}

	require.Equal(t, Histogram{
		{Label: "0-9", Chats: 3},
		{Label: "10-99", Chats: 2},
		{Label: "100-999", Chats: 2},
		{Label: "1000-9999", Chats: 2},
		{Label: "10000+", Chats: 2},
	}, h)
}

func TestDaysAreRelativeAndNeverNegative(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)

	require.Equal(t, int64(10), daysSinceUnixSeconds(now, now.AddDate(0, 0, -10).Unix()))
	require.Equal(t, int64(0), daysSinceUnixSeconds(now, 0))
	require.Equal(t, int64(0), daysSinceUnixSeconds(now, now.AddDate(0, 0, 5).Unix()),
		"a clock skewed into the future must not produce a negative age")
	require.Equal(t, int64(370), daysSinceUnixMillis(now, now.AddDate(0, 0, -370).UnixMilli()))
}

func TestPercentAndMedian(t *testing.T) {
	require.Equal(t, int64(0), percent(5, 0))
	require.Equal(t, int64(5), percent(5, 100))
	require.Equal(t, int64(33), percent(1, 3))

	require.Equal(t, int64(0), median(nil))
	require.Equal(t, int64(3), median([]int64{5, 1, 3}))
	require.Equal(t, int64(4), median([]int64{1, 3, 5, 7}))
}

// A chat that has never synced is the #21605 condition, and it has no gap to
// average into max/median - so it has to be counted separately rather than
// disappearing into a healthy-looking 0.
func TestCollectCountsNeverSyncedChatsSeparately(t *testing.T) {
	appDB := setupAppDB(t)
	seedAccount(t, appDB, setupWalletDB(t))

	// Wipe every sync position: the account is now entirely unsynced.
	exec(t, appDB, `UPDATE chats SET synced_to = 0`)

	profile, err := Collect(context.Background(), appDB, nil, collectedAt, zap.NewNop(), nil)
	require.NoError(t, err)

	require.Equal(t, int64(11), profile.Sync.NeverSyncedChats)
	require.Equal(t, int64(0), profile.Sync.MaxSyncGapDays,
		"max/median stay over synced chats only")
	require.Equal(t, int64(0), profile.Sync.MedianSyncGapDays)
	require.NotContains(t, profile.Incomplete, "sync")
}

// A wallet table renamed out from under us must not read as "this account has
// no collectibles": the step has to name itself in Incomplete.
func TestCollectFlagsAMissingWalletTable(t *testing.T) {
	appDB := setupAppDB(t)
	walletDB := setupWalletDB(t)
	seedAccount(t, appDB, walletDB)

	exec(t, walletDB, `ALTER TABLE collectibles_ownership_cache RENAME TO collectibles_ownership_cache_v2`)

	profile, err := Collect(context.Background(), appDB, walletDB, collectedAt, zap.NewNop(), nil)
	require.NoError(t, err, "a missing table must not fail the whole profile")

	require.Contains(t, profile.Incomplete, "wallet")
	require.Equal(t, int64(0), profile.Wallet.Collectibles)
	// The other counters are still collected: partial data beats none.
	require.Equal(t, int64(9), profile.Wallet.TokenBalanceRows)
	require.Equal(t, int64(2), profile.Wallet.SavedAddresses)
	// And the renamed table is still visible in the generic walk.
	require.Equal(t, int64(4), profile.Tables.WalletDB["collectibles_ownership_cache_v2"].Rows)
}
