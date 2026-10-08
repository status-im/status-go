package protocol

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"reflect"
	"sync"
)

const threadSummaryCacheCapacity = 256

type threadSummaryCacheKey struct {
	threadID     string
	previewLimit int
}

type threadSummaryStamp struct {
	connection    interface{}
	totalChanges  int64
	dataVersion   int64
	schemaVersion int64
}

// SQLite's counters are connection-local. Pin the connection while checking
// and reading, and never compare counters from different driver connections.
func readThreadSummaryStamp(conn *sql.Conn) (threadSummaryStamp, error) {
	var stamp threadSummaryStamp
	err := conn.Raw(func(driverConn interface{}) error {
		if typ := reflect.TypeOf(driverConn); typ != nil && typ.Comparable() {
			stamp.connection = driverConn
		}
		return nil
	})
	if err != nil {
		return stamp, err
	}
	err = conn.QueryRowContext(context.Background(), `
		SELECT total_changes(), data_version, schema_version
		FROM pragma_data_version, pragma_schema_version`).Scan(
		&stamp.totalChanges, &stamp.dataVersion, &stamp.schemaVersion)
	return stamp, err
}

type threadSummaryCache struct {
	mu      sync.Mutex
	stamp   threadSummaryStamp
	entries map[threadSummaryCacheKey]Thread
}

// Unread counts and thread metadata are not cached; callers read them live.
func copyThreadSummary(dst, src *Thread) {
	dst.MessagesCount = src.MessagesCount
	dst.ParticipantsCount = src.ParticipantsCount
	dst.ParticipantsPreviewIDs = append([]string{}, src.ParticipantsPreviewIDs...)
	dst.LastMessage = nil
	if src.LastMessage != nil {
		last := *src.LastMessage
		dst.LastMessage = &last
	}
}

func (c *threadSummaryCache) load(stamp threadSummaryStamp, threads []*Thread, previewLimit int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if stamp.connection == nil || c.stamp != stamp {
		return false
	}
	for _, thread := range threads {
		if _, ok := c.entries[threadSummaryCacheKey{thread.ThreadID, previewLimit}]; !ok {
			return false
		}
	}
	for _, thread := range threads {
		summary := c.entries[threadSummaryCacheKey{thread.ThreadID, previewLimit}]
		copyThreadSummary(thread, &summary)
	}
	return true
}

func (c *threadSummaryCache) store(stamp threadSummaryStamp, threads []*Thread, previewLimit int) {
	if stamp.connection == nil || len(threads) > threadSummaryCacheCapacity {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stamp != stamp || len(c.entries)+len(threads) > threadSummaryCacheCapacity {
		c.entries = make(map[threadSummaryCacheKey]Thread)
		c.stamp = stamp
	}
	for _, thread := range threads {
		var summary Thread
		copyThreadSummary(&summary, thread)
		c.entries[threadSummaryCacheKey{thread.ThreadID, previewLimit}] = summary
	}
}

type threadSummaryQuerier interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

// Metadata, unread counts and summaries must all belong to one read snapshot.
func (db sqlitePersistence) readThreadsWithSummaries(previewLimit int,
	readThreads func(threadSummaryQuerier) ([]*Thread, error)) ([]*Thread, error) {
	conn, err := db.db.Conn(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	var stamp threadSummaryStamp
	var stampErr error
	if db.threadSummaryCache != nil {
		stamp, stampErr = readThreadSummaryStamp(conn)
	}
	// The application driver defaults BeginTx to BEGIN IMMEDIATE. This read-only
	// snapshot must not reserve the writer lock; keep all SQL on the pinned conn.
	if _, err := conn.ExecContext(context.Background(), "BEGIN DEFERRED"); err != nil {
		return nil, err
	}
	snapshotOpen := true
	defer func() {
		if snapshotOpen {
			if _, err := conn.ExecContext(context.Background(), "ROLLBACK"); err != nil {
				// Never return a connection with an open transaction to the pool.
				_ = conn.Raw(func(interface{}) error { return driver.ErrBadConn })
			}
		}
	}()
	threads, err := readThreads(conn)
	if err != nil {
		return nil, err
	}
	cacheCurrent := false
	if len(threads) > 0 && db.threadSummaryCache != nil && stampErr == nil {
		// The metadata query establishes the snapshot. A commit between the
		// initial stamp and that query must prevent reuse of an older summary.
		after, err := readThreadSummaryStamp(conn)
		cacheCurrent = err == nil && after == stamp
	}
	cacheHit := cacheCurrent && db.threadSummaryCache.load(stamp, threads, previewLimit)
	if !cacheHit {
		if err := db.queryThreadsSummaries(conn, threads, previewLimit); err != nil {
			return nil, err
		}
	}
	storeSummary := false
	if cacheCurrent && !cacheHit {
		after, err := readThreadSummaryStamp(conn)
		storeSummary = err == nil && after == stamp
	}
	if _, err := conn.ExecContext(context.Background(), "COMMIT"); err != nil {
		return nil, err
	}
	snapshotOpen = false
	if storeSummary {
		db.threadSummaryCache.store(stamp, threads, previewLimit)
	}
	return threads, nil
}
