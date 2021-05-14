package mailserver

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/status-im/status-go/eth-node/types"
	waku "github.com/status-im/status-go/waku/common"
)

func TestCleaner(t *testing.T) {
	now := time.Now()
	server := setupTestServer(t)
	defer server.Close()
	cleaner := newDBCleaner(server.ms.db, time.Hour)

	archiveEnvelope(t, now.Add(-10*time.Second), server)
	archiveEnvelope(t, now.Add(-3*time.Second), server)
	archiveEnvelope(t, now.Add(-1*time.Second), server)

	testMessagesCount(t, 3, server)

	testPrune(t, now.Add(-5*time.Second), 1, cleaner)
	testPrune(t, now.Add(-2*time.Second), 1, cleaner)
	testPrune(t, now, 1, cleaner)

	testMessagesCount(t, 0, server)
}

func TestCleanerSchedule(t *testing.T) {
	now := time.Now()
	server := setupTestServer(t)
	defer server.Close()

	cleaner := newDBCleaner(server.ms.db, time.Hour)
	cleaner.period = time.Millisecond * 10
	cleaner.Start()
	defer cleaner.Stop()

	archiveEnvelope(t, now.Add(-3*time.Hour), server)
	archiveEnvelope(t, now.Add(-2*time.Hour), server)
	archiveEnvelope(t, now.Add(-1*time.Minute), server)

	time.Sleep(time.Millisecond * 50)

	testMessagesCount(t, 1, server)
}

func benchmarkCleanerPrune(b *testing.B, messages int, batchSize int) {
	t := &testing.T{}
	now := time.Now()
	sentTime := now.Add(-10 * time.Second)
	server := setupTestServer(t)
	defer server.Close()

	cleaner := newDBCleaner(server.ms.db, time.Hour)
	cleaner.batchSize = batchSize

	for i := 0; i < messages; i++ {
		archiveEnvelope(t, sentTime, server)
	}

	for i := 0; i < b.N; i++ {
		testPrune(t, now, 0, cleaner)
	}
}

func BenchmarkCleanerPruneM100_000_B100_000(b *testing.B) {
	benchmarkCleanerPrune(b, 100000, 100000)
}

func BenchmarkCleanerPruneM100_000_B10_000(b *testing.B) {
	benchmarkCleanerPrune(b, 100000, 10000)
}

func BenchmarkCleanerPruneM100_000_B1000(b *testing.B) {
	benchmarkCleanerPrune(b, 100000, 1000)
}

func BenchmarkCleanerPruneM100_000_B100(b *testing.B) {
	benchmarkCleanerPrune(b, 100000, 100)
}

func setupTestServer(t *testing.T) *WakuMailServer {
	var s WakuMailServer
	db, _ := leveldb.Open(storage.NewMemStorage(), nil)

	s.ms = &mailServer{
		db: &LevelDB{
			ldb:  db,
			done: make(chan struct{}),
		},
		adapter: &wakuAdapter{},
	}
	s.minRequestPoW = powRequirement
	return &s
}

func archiveEnvelope(t *testing.T, sentTime time.Time, server *WakuMailServer) *waku.Envelope {
	env, err := generateEnvelope(sentTime)
	require.NoError(t, err)
	server.Archive(env)

	return env
}

func testPrune(t *testing.T, u time.Time, expected int, c *dbCleaner) {
	n, err := c.PruneEntriesOlderThan(u)
	require.NoError(t, err)
	require.Equal(t, expected, n)
}

func testMessagesCount(t *testing.T, expected int, s *WakuMailServer) {
	count := countMessages(t, s.ms.db)
	require.Equal(t, expected, count, fmt.Sprintf("expected %d message, got: %d", expected, count))
}

func countMessages(t *testing.T, db DB) int {
	var (
		count      int
		zero       types.Hash
		emptyTopic types.TopicType
	)

	now := time.Now()
	kl := NewDBKey(uint32(0), emptyTopic, zero)
	ku := NewDBKey(uint32(now.Unix()), emptyTopic, zero)

	query := CursorQuery{
		start: kl.raw,
		end:   ku.raw,
	}

	i, _ := db.BuildIterator(query)
	defer func() { _ = i.Release() }()

	for i.Next() {
		var env waku.Envelope
		value, err := i.GetEnvelope(query.bloom)
		if err != nil {
			t.Fatal(err)
		}

		err = rlp.DecodeBytes(value, &env)
		if err != nil {
			t.Fatal(err)
		}

		count++
	}

	return count
}
