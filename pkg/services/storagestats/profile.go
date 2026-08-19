// Package storagestats collects an "account load profile": the quantitative
// shape of the logged-in account's databases, safe to paste into a public bug
// report.
//
// The profile must stay publishable, so new fields have to keep these rules:
// time only as relative day counts, no per-entity rows (aggregates and
// histograms only), and no strings other than table names from our own schema.
package storagestats

// ProfileVersion must be bumped when a field's meaning or a histogram bucket
// boundary changes.
const ProfileVersion = 1

// Profile is serialised as JSON: the UI renders it and the reporter pastes the
// same bytes into a ticket.
type Profile struct {
	ProfileVersion int `json:"profileVersion"`

	// ProfileAgeDays is derived from the oldest wallet account.
	ProfileAgeDays int64 `json:"profileAgeDays"`

	// Incomplete names the steps that hit an error, including a step whose
	// table has been renamed away. Fields fed by a listed step may read 0
	// because they could not be measured; fields of unlisted steps are real.
	Incomplete []string `json:"incomplete"`

	Messaging Messaging `json:"messaging"`
	Sync      Sync      `json:"sync"`
	Wallet    Wallet    `json:"wallet"`
	DB        DB        `json:"db"`
	Tables    Tables    `json:"tables"`
}

// Messaging field names mirror the seeder harness knobs, so a profile maps 1:1
// onto a seeder run.
type Messaging struct {
	MessagesTotal     int64          `json:"messagesTotal"`
	Chats             ChatCounts     `json:"chats"`
	Communities       int64          `json:"communities"`
	Contacts          ContactCounts  `json:"contacts"`
	ActivityCenter    ActivityCenter `json:"activityCenter"`
	OldestMessageDays int64          `json:"oldestMessageDays"`
	// RecentTailPct is the share of all messages in the last 7 / 30 days, in
	// whole percent (the seeder's `tailpct` knob).
	RecentTailPct    RecentTailPct `json:"recentTailPct"`
	PerChatHistogram Histogram     `json:"perChatHistogram"`
}

// ChatCounts breaks chats down by protocol chat type; Other catches the
// deprecated profile/timeline types and any type added later.
type ChatCounts struct {
	OneToOne          int64 `json:"oneToOne"`
	Public            int64 `json:"public"`
	Group             int64 `json:"group"`
	CommunityChannels int64 `json:"communityChannels"`
	Other             int64 `json:"other"`
}

// ContactCounts counts rows in the contacts table, not the UI's contact list.
type ContactCounts struct {
	Total  int64 `json:"total"`
	Mutual int64 `json:"mutual"`
}

// ActivityCenter counts all notification rows, dismissed and deleted included.
type ActivityCenter struct {
	Total  int64 `json:"total"`
	Unread int64 `json:"unread"`
}

type RecentTailPct struct {
	D7  int64 `json:"d7"`
	D30 int64 `json:"d30"`
}

// Sync describes how far behind the local mailserver sync state is.
type Sync struct {
	// MaxSyncGapDays and MedianSyncGapDays cover only chats that have synced at
	// least once; NeverSyncedChats counts the active chats they exclude.
	MaxSyncGapDays    int64 `json:"maxSyncGapDays"`
	MedianSyncGapDays int64 `json:"medianSyncGapDays"`
	NeverSyncedChats  int64 `json:"neverSyncedChats"`
	// WakuFilters is the row count of mailserver_topics.
	WakuFilters   int64 `json:"wakuFilters"`
	Installations int64 `json:"installations"`
}

type Wallet struct {
	Accounts         int64 `json:"accounts"`
	Collectibles     int64 `json:"collectibles"`
	TokenBalanceRows int64 `json:"tokenBalanceRows"`
	CustomTokens     int64 `json:"customTokens"`
	ActivityRows     int64 `json:"activityRows"`
	SavedAddresses   int64 `json:"savedAddresses"`
}

// DB is the physical shape of the two database files.
type DB struct {
	AppDBBytes    int64  `json:"appDbBytes"`
	WalletDBBytes int64  `json:"walletDbBytes"`
	AppDB         DBFile `json:"appDb"`
	WalletDB      DBFile `json:"walletDb"`
}

// DBFile describes one sqlite file. MigrationVersions maps each migration table
// in the file to its applied version.
type DBFile struct {
	PageCount         int64            `json:"pageCount"`
	PageSize          int64            `json:"pageSize"`
	FreelistCount     int64            `json:"freelistCount"`
	SchemaVersion     int64            `json:"schemaVersion"`
	MigrationVersions map[string]int64 `json:"migrationVersions"`
}

// TableStats holds one table's row count and Bytes, the summed logical length
// of every column value - not the on-disk page footprint.
type TableStats struct {
	Rows  int64 `json:"rows"`
	Bytes int64 `json:"bytes"`
}

// Tables covers every table in both databases, so a surprise in an unforeseen
// table is still visible.
type Tables struct {
	AppDB    map[string]TableStats `json:"appDb"`
	WalletDB map[string]TableStats `json:"walletDb"`
}

// Bucket is one column of the per-chat histogram.
type Bucket struct {
	Label string `json:"bucket"`
	Chats int64  `json:"chats"`
}

// Histogram keeps its buckets in bucketBounds order.
type Histogram []Bucket
