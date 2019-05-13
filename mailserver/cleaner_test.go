package mailserver

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

func TestCleaner(t *testing.T) {
	now := time.Now()
	server := setupTestServer(t)
	defer server.Close()
	cleaner := newDBCleaner(server.db, time.Hour)

	archiveEnvelope(t, now.Add(-10*time.Second), server)
	archiveEnvelope(t, now.Add(-3*time.Second), server)
	archiveEnvelope(t, now.Add(-1*time.Second), server)

	testMessagesCount(t, 3, server)

	testPrune(t, now.Add(-5*time.Second), 1, cleaner, server)
	testPrune(t, now.Add(-2*time.Second), 1, cleaner, server)
	testPrune(t, now, 1, cleaner, server)

	testMessagesCount(t, 0, server)
}

func TestCleanerSchedule(t *testing.T) {
	now := time.Now()
	server := setupTestServer(t)
	defer server.Close()

	cleaner := newDBCleaner(server.db, time.Hour)
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

	cleaner := newDBCleaner(server.db, time.Hour)
	cleaner.batchSize = batchSize

	for i := 0; i < messages; i++ {
		archiveEnvelope(t, sentTime, server)
	}

	for i := 0; i < b.N; i++ {
		testPrune(t, now, 0, cleaner, server)
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

func setupTestServer(t *testing.T) *WMailServer {
	var s WMailServer
	db, _ := leveldb.Open(storage.NewMemStorage(), nil)

	s.db = &LevelDB{ldb: db}
	s.pow = powRequirement
	return &s
}

func archiveEnvelope(t *testing.T, sentTime time.Time, server *WMailServer) *whisper.Envelope {
	env, err := generateEnvelope(sentTime)
	require.NoError(t, err)
	server.Archive(env)

	return env
}

func testPrune(t *testing.T, u time.Time, expected int, c *dbCleaner, s *WMailServer) {
	n, err := c.PruneEntriesOlderThan(u)
	require.NoError(t, err)
	require.Equal(t, expected, n)
}

func testMessagesCount(t *testing.T, expected int, s *WMailServer) {
	count := countMessages(t, s.db)
	require.Equal(t, expected, count, fmt.Sprintf("expected %d message, got: %d", expected, count))
}

func countMessages(t *testing.T, db DB) int {
	var (
		count      int
		zero       common.Hash
		emptyTopic whisper.TopicType
	)

	now := time.Now()
	kl := NewDBKey(uint32(0), emptyTopic, zero)
	ku := NewDBKey(uint32(now.Unix()), emptyTopic, zero)

	query := CursorQuery{
		start: kl.raw,
		end:   ku.raw,
	}

	i, _ := db.BuildIterator(query)
	defer i.Release()

	for i.Next() {
		var env whisper.Envelope
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
