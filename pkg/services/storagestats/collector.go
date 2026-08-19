package storagestats

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Chat types, mirroring protocol.ChatType. Duplicated because importing
// protocol would pull in the whole messenger and its native waku dependencies.
const (
	chatTypeOneToOne         = 1
	chatTypePublic           = 2
	chatTypePrivateGroupChat = 3
	chatTypeCommunityChat    = 6
)

// Contact request states, mirroring contacts.ContactRequestState.
const (
	contactRequestStateSent     = 2
	contactRequestStateReceived = 3
)

// ProgressFunc is called once per completed step. total is final from the first
// call: the table list is enumerated before any counting starts.
type ProgressFunc func(done, total int)

// migrationTableMarker matches the golang-migrate bookkeeping tables. A file
// holds one per migration tree, and the marker appears as both prefix and
// suffix (status_go_schema_migrations, status_schema_migrations_waku), so
// matching is by substring.
const migrationTableMarker = "schema_migrations"

// mainMigrationTable owns the database file itself and supplies schemaVersion.
const mainMigrationTable = "status_go_" + migrationTableMarker

// Collect walks both databases and builds a Profile. It must stay serial and
// off the caller's thread: COUNT(*) on a sqlcipher table is a full decrypting
// scan, so parallel steps would starve the rest of the process.
func Collect(ctx context.Context, appDB, walletDB *sql.DB, now time.Time, logger *zap.Logger, progress ProgressFunc) (*Profile, error) {
	if appDB == nil {
		return nil, fmt.Errorf("storagestats: no app database, is an account logged in?")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	c := &collector{
		now:    now,
		logger: logger,
		profile: Profile{
			ProfileVersion: ProfileVersion,
			Tables: Tables{
				AppDB:    map[string]TableStats{},
				WalletDB: map[string]TableStats{},
			},
			Incomplete: []string{},
		},
	}
	c.profile.Messaging.PerChatHistogram = newHistogram()
	c.profile.DB.AppDB.MigrationVersions = map[string]int64{}
	c.profile.DB.WalletDB.MigrationVersions = map[string]int64{}

	var err error
	if c.app, err = newScope(ctx, appDB); err != nil {
		return nil, fmt.Errorf("storagestats: enumerating app database tables: %w", err)
	}
	if walletDB != nil {
		if c.wallet, err = newScope(ctx, walletDB); err != nil {
			return nil, fmt.Errorf("storagestats: enumerating wallet database tables: %w", err)
		}
	}

	steps := c.plan()
	total := len(steps)
	for i, step := range steps {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// One unreadable table must not cost the whole profile; Incomplete is
		// what distinguishes the resulting 0 from a genuine 0.
		if err := step.run(ctx); err != nil {
			c.logger.Warn("storage stats step failed",
				zap.String("step", step.name), zap.Error(err))
			c.profile.Incomplete = append(c.profile.Incomplete, step.name)
		}
		if progress != nil {
			progress(i+1, total)
		}
	}

	return &c.profile, nil
}

type collector struct {
	now     time.Time
	logger  *zap.Logger
	app     *scope
	wallet  *scope
	profile Profile

	// chatCount places chats with no messages into the first histogram bucket.
	chatCount int64
}

type step struct {
	name string
	run  func(ctx context.Context) error
}

func (c *collector) plan() []step {
	steps := []step{
		{"messages", c.collectMessages},
		{"chats", c.collectChats},
		{"perChatHistogram", c.collectPerChatHistogram},
		{"communities", c.collectCommunities},
		{"contacts", c.collectContacts},
		{"activityCenter", c.collectActivityCenter},
		{"sync", c.collectSync},
		{"profileAge", c.collectProfileAge},
		{"wallet", c.collectWallet},
		{"appDbPhysical", func(ctx context.Context) error {
			return c.collectPhysical(ctx, c.app, &c.profile.DB.AppDB, &c.profile.DB.AppDBBytes)
		}},
		{"walletDbPhysical", func(ctx context.Context) error {
			return c.collectPhysical(ctx, c.wallet, &c.profile.DB.WalletDB, &c.profile.DB.WalletDBBytes)
		}},
	}

	steps = append(steps, tableSteps(c.app, c.profile.Tables.AppDB)...)
	steps = append(steps, tableSteps(c.wallet, c.profile.Tables.WalletDB)...)
	return steps
}

func tableSteps(s *scope, out map[string]TableStats) []step {
	if s == nil {
		return nil
	}
	steps := make([]step, 0, len(s.names))
	for _, name := range s.names {
		steps = append(steps, step{
			name: name,
			run: func(ctx context.Context) error {
				stats, err := s.tableStats(ctx, name)
				if err != nil {
					return err
				}
				out[name] = stats
				return nil
			},
		})
	}
	return steps
}

// --- curated metrics --------------------------------------------------------

func (c *collector) collectMessages(ctx context.Context) error {
	if !c.app.has("user_messages") {
		return nil
	}
	d7 := c.now.AddDate(0, 0, -7).UnixMilli()
	d30 := c.now.AddDate(0, 0, -30).UnixMilli()

	var total, oldest, tail7, tail30 int64
	err := c.app.db.QueryRowContext(ctx, `
		SELECT COUNT(*),
		       COALESCE(MIN(NULLIF(whisper_timestamp, 0)), 0),
		       COALESCE(SUM(CASE WHEN whisper_timestamp >= ? THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN whisper_timestamp >= ? THEN 1 ELSE 0 END), 0)
		FROM user_messages`, d7, d30).Scan(&total, &oldest, &tail7, &tail30)
	if err != nil {
		return err
	}

	c.profile.Messaging.MessagesTotal = total
	c.profile.Messaging.OldestMessageDays = daysSinceUnixMillis(c.now, oldest)
	c.profile.Messaging.RecentTailPct = RecentTailPct{
		D7:  percent(tail7, total),
		D30: percent(tail30, total),
	}
	return nil
}

func (c *collector) collectChats(ctx context.Context) error {
	if !c.app.has("chats") {
		return nil
	}
	rows, err := c.app.db.QueryContext(ctx, `SELECT type, COUNT(*) FROM chats GROUP BY type`)
	if err != nil {
		return err
	}
	defer rows.Close()

	counts := &c.profile.Messaging.Chats
	for rows.Next() {
		var chatType, count int64
		if err := rows.Scan(&chatType, &count); err != nil {
			return err
		}
		c.chatCount += count
		switch chatType {
		case chatTypeOneToOne:
			counts.OneToOne += count
		case chatTypePublic:
			counts.Public += count
		case chatTypePrivateGroupChat:
			counts.Group += count
		case chatTypeCommunityChat:
			counts.CommunityChannels += count
		default:
			counts.Other += count
		}
	}
	return rows.Err()
}

// collectPerChatHistogram selects counts only, never a chat id, so no
// per-entity data can reach the profile.
func (c *collector) collectPerChatHistogram(ctx context.Context) error {
	if !c.app.has("user_messages") {
		return nil
	}
	rows, err := c.app.db.QueryContext(ctx, `SELECT COUNT(*) FROM user_messages GROUP BY local_chat_id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	histogram := c.profile.Messaging.PerChatHistogram
	var chatsWithMessages int64
	for rows.Next() {
		var count int64
		if err := rows.Scan(&count); err != nil {
			return err
		}
		chatsWithMessages++
		histogram.add(count)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Chats with no messages are absent from the GROUP BY but belong in the
	// distribution.
	if empty := c.chatCount - chatsWithMessages; empty > 0 {
		for i := 0; i < int(empty); i++ {
			histogram.add(0)
		}
	}
	return nil
}

func (c *collector) collectCommunities(ctx context.Context) error {
	count, err := c.app.count(ctx, "communities_communities", "")
	if err != nil {
		return err
	}
	c.profile.Messaging.Communities = count
	return nil
}

func (c *collector) collectContacts(ctx context.Context) error {
	total, err := c.app.count(ctx, "contacts", "")
	if err != nil {
		return err
	}
	// Mutual means request sent and request received; two separate columns.
	mutual, err := c.app.count(ctx, "contacts", fmt.Sprintf(
		"contact_request_state = %d AND contact_request_remote_state = %d",
		contactRequestStateSent, contactRequestStateReceived))
	if err != nil {
		return err
	}
	c.profile.Messaging.Contacts = ContactCounts{Total: total, Mutual: mutual}
	return nil
}

func (c *collector) collectActivityCenter(ctx context.Context) error {
	total, err := c.app.count(ctx, "activity_center_notifications", "")
	if err != nil {
		return err
	}
	unread, err := c.app.count(ctx, "activity_center_notifications", "read = 0")
	if err != nil {
		return err
	}
	c.profile.Messaging.ActivityCenter = ActivityCenter{Total: total, Unread: unread}
	return nil
}

// collectSync measures how far behind the mailserver sync state is.
func (c *collector) collectSync(ctx context.Context) error {
	if c.app.has("chats") {
		rows, err := c.app.db.QueryContext(ctx,
			`SELECT COALESCE(synced_to, 0) FROM chats WHERE active = 1`)
		if err != nil {
			return err
		}
		defer rows.Close()

		var gaps []int64
		var max, neverSynced int64
		for rows.Next() {
			var syncedTo int64
			if err := rows.Scan(&syncedTo); err != nil {
				return err
			}
			// Never-synced chats have no gap to measure; counting them
			// separately keeps max/median comparable across profiles.
			if syncedTo <= 0 {
				neverSynced++
				continue
			}
			gap := daysSinceUnixSeconds(c.now, syncedTo)
			gaps = append(gaps, gap)
			if gap > max {
				max = gap
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		c.profile.Sync.MaxSyncGapDays = max
		c.profile.Sync.MedianSyncGapDays = median(gaps)
		c.profile.Sync.NeverSyncedChats = neverSynced
	}

	// mailserver_topics is the persisted content-topic set.
	filters, err := c.app.count(ctx, "mailserver_topics", "")
	if err != nil {
		return err
	}
	c.profile.Sync.WakuFilters = filters

	installations, err := c.app.count(ctx, "installations", "deleted = 0")
	if err != nil {
		return err
	}
	c.profile.Sync.Installations = installations
	return nil
}

// collectProfileAge dates the profile from its oldest wallet account.
func (c *collector) collectProfileAge(ctx context.Context) error {
	if !c.app.has("keypairs_accounts") {
		return nil
	}
	// created_at is datetime('now') text; the second branch covers older rows
	// written as raw unix seconds.
	var createdAt sql.NullInt64
	err := c.app.db.QueryRowContext(ctx, `
		SELECT MIN(COALESCE(
			CAST(strftime('%s', created_at) AS INTEGER),
			CAST(created_at AS INTEGER)
		)) FROM keypairs_accounts`).Scan(&createdAt)
	if err != nil {
		return err
	}
	if createdAt.Valid {
		c.profile.ProfileAgeDays = daysSinceUnixSeconds(c.now, createdAt.Int64)
	}
	return nil
}

func (c *collector) collectWallet(ctx context.Context) error {
	// Every counter is collected even if an earlier one failed - partial data
	// beats none - but the errors are returned so the step lands in Incomplete.
	var errs []error

	// Wallet accounts live in the app database. The filter matches the one the
	// rest of status-go lists accounts with, excluding chat and removed rows.
	accounts, err := c.app.count(ctx, "keypairs_accounts", "chat = 0 AND removed = 0")
	if err != nil {
		errs = append(errs, err)
	}
	c.profile.Wallet.Accounts = accounts

	for _, m := range []struct {
		table string
		out   *int64
	}{
		{"collectibles_ownership_cache", &c.profile.Wallet.Collectibles},
		{"token_balances", &c.profile.Wallet.TokenBalanceRows},
		{"tokens", &c.profile.Wallet.CustomTokens},
		{"fetched_alchemy_transfers", &c.profile.Wallet.ActivityRows},
		{"saved_addresses", &c.profile.Wallet.SavedAddresses},
	} {
		// A wallet database that has no such table has been renamed or migrated
		// under us; reporting 0 would read as "this account has none".
		if c.wallet != nil && !c.wallet.has(m.table) {
			errs = append(errs, fmt.Errorf("wallet table %s is missing", m.table))
			continue
		}
		count, err := c.wallet.count(ctx, m.table, "")
		if err != nil {
			errs = append(errs, err)
			continue
		}
		*m.out = count
	}
	return errors.Join(errs...)
}

func (c *collector) collectPhysical(ctx context.Context, s *scope, out *DBFile, bytes *int64) error {
	if s == nil {
		return nil
	}
	out.MigrationVersions = map[string]int64{}

	var errs []error
	for _, p := range []struct {
		pragma string
		out    *int64
	}{
		{"page_count", &out.PageCount},
		{"page_size", &out.PageSize},
		{"freelist_count", &out.FreelistCount},
	} {
		var value int64
		if err := s.db.QueryRowContext(ctx, "PRAGMA "+p.pragma).Scan(&value); err != nil {
			errs = append(errs, fmt.Errorf("reading pragma %s: %w", p.pragma, err))
			continue
		}
		*p.out = value
	}
	*bytes = out.PageCount * out.PageSize

	for _, name := range s.names {
		if !strings.Contains(name, migrationTableMarker) {
			continue
		}
		var version sql.NullInt64
		if err := s.db.QueryRowContext(ctx, `SELECT MAX(version) FROM "`+name+`"`).Scan(&version); err != nil {
			errs = append(errs, fmt.Errorf("reading migration version of %s: %w", name, err))
			continue
		}
		if version.Valid {
			out.MigrationVersions[name] = version.Int64
		}
	}
	// The status_go_ table is this file's own tree; the rest share the file.
	out.SchemaVersion = out.MigrationVersions[mainMigrationTable]
	return errors.Join(errs...)
}

// --- generic per-table walk -------------------------------------------------

// scope is one database plus the tables it actually has, so a query against a
// missing table is skipped rather than failed.
type scope struct {
	db     *sql.DB
	names  []string
	lookup map[string]struct{}
}

func newScope(ctx context.Context, db *sql.DB) (*scope, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	s := &scope{db: db, lookup: map[string]struct{}{}}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		s.names = append(s.names, name)
		s.lookup[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Strings(s.names)
	return s, nil
}

func (s *scope) has(name string) bool {
	if s == nil {
		return false
	}
	_, ok := s.lookup[name]
	return ok
}

// count returns COUNT(*) for a table, or 0 when the table is absent.
func (s *scope) count(ctx context.Context, table, where string) (int64, error) {
	if !s.has(table) {
		return 0, nil
	}
	query := `SELECT COUNT(*) FROM "` + table + `"`
	if where != "" {
		query += " WHERE " + where
	}
	var count int64
	if err := s.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("counting %s: %w", table, err)
	}
	return count, nil
}

// tableStats returns row count and summed payload length in a single scan.
// Bytes is a logical size, not the on-disk footprint: our sqlcipher build has
// no dbstat virtual table.
func (s *scope) tableStats(ctx context.Context, table string) (TableStats, error) {
	columns, err := s.columns(ctx, table)
	if err != nil {
		return TableStats{}, err
	}
	if len(columns) == 0 {
		return TableStats{}, nil
	}

	lengths := make([]string, 0, len(columns))
	for _, column := range columns {
		// CAST to BLOB makes LENGTH count bytes, not characters; COALESCE keeps
		// a NULL from voiding the whole sum.
		lengths = append(lengths, `COALESCE(LENGTH(CAST("`+column+`" AS BLOB)), 0)`)
	}

	var stats TableStats
	query := `SELECT COUNT(*), COALESCE(SUM(` + strings.Join(lengths, " + ") + `), 0) FROM "` + table + `"`
	if err := s.db.QueryRowContext(ctx, query).Scan(&stats.Rows, &stats.Bytes); err != nil {
		return TableStats{}, fmt.Errorf("measuring %s: %w", table, err)
	}
	return stats, nil
}

func (s *scope) columns(ctx context.Context, table string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		return nil, fmt.Errorf("listing columns of %s: %w", table, err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}
	return columns, rows.Err()
}

// --- helpers ----------------------------------------------------------------

// bucketBounds are part of the artifact contract: changing them requires
// bumping ProfileVersion.
var bucketBounds = []struct {
	label string
	from  int64
	to    int64 // exclusive; math.MaxInt64 for the open-ended last bucket
}{
	{"0-9", 0, 10},
	{"10-99", 10, 100},
	{"100-999", 100, 1000},
	{"1000-9999", 1000, 10000},
	{"10000+", 10000, math.MaxInt64},
}

// newHistogram returns every bucket, empty, so the artifact's shape does not
// depend on the data.
func newHistogram() Histogram {
	h := make(Histogram, 0, len(bucketBounds))
	for _, b := range bucketBounds {
		h = append(h, Bucket{Label: b.label})
	}
	return h
}

// add places one chat, holding count messages, into its bucket.
func (h Histogram) add(count int64) {
	for i, b := range bucketBounds {
		if count >= b.from && count < b.to {
			h[i].Chats++
			return
		}
	}
}

// daysSince returns whole days elapsed, clamped at 0 for future timestamps.
func daysSince(now time.Time, then time.Time) int64 {
	if then.IsZero() || then.After(now) {
		return 0
	}
	return int64(now.Sub(then).Hours() / 24)
}

func daysSinceUnixSeconds(now time.Time, seconds int64) int64 {
	if seconds <= 0 {
		return 0
	}
	return daysSince(now, time.Unix(seconds, 0))
}

// Message timestamps in the protocol database are milliseconds.
func daysSinceUnixMillis(now time.Time, millis int64) int64 {
	if millis <= 0 {
		return 0
	}
	return daysSince(now, time.UnixMilli(millis))
}

// percent returns part/total in whole percent, 0 when total is 0.
func percent(part, total int64) int64 {
	if total <= 0 {
		return 0
	}
	return int64(math.Round(float64(part) * 100 / float64(total)))
}

// median returns the median of values, which it sorts in place.
func median(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	slices.Sort(values)
	mid := len(values) / 2
	if len(values)%2 == 1 {
		return values[mid]
	}
	return (values[mid-1] + values[mid]) / 2
}
