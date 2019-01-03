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
	"github.com/syndtr/goleveldb/leveldb/util"
)

func TestCleaner(t *testing.T) {
	now := time.Now()
	server := setupTestServer(t)
	cleaner := NewCleanerWithDB(server.db)
	defer server.Close()

	archiveEnvelope(t, now.Add(-10*time.Second), server)
	archiveEnvelope(t, now.Add(-3*time.Second), server)
	archiveEnvelope(t, now.Add(-1*time.Second), server)

	testMessagesCount(t, 3, server)

	testPrune(t, now.Add(-5*time.Second), 2, cleaner, server)
	testPrune(t, now.Add(-2*time.Second), 1, cleaner, server)
	testPrune(t, now, 0, cleaner, server)
}

func benchmarkCleanerPrune(b *testing.B, messages int, batchSize int) {
	t := &testing.T{}
	now := time.Now()
	sentTime := now.Add(-10 * time.Second)
	server := setupTestServer(t)
	defer server.Close()

	cleaner := NewCleanerWithDB(server.db)
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
	s.db, _ = leveldb.Open(storage.NewMemStorage(), nil)
	s.pow = powRequirement
	return &s
}

func archiveEnvelope(t *testing.T, sentTime time.Time, server *WMailServer) *whisper.Envelope {
	env, err := generateEnvelope(sentTime)
	require.NoError(t, err)
	server.Archive(env)

	return env
}

func testPrune(t *testing.T, u time.Time, expected int, c *Cleaner, s *WMailServer) {
	upper := uint32(u.Unix())
	_, err := c.Prune(0, upper)
	require.NoError(t, err)

	count := countMessages(t, s.db)
	require.Equal(t, expected, count)
}

func testMessagesCount(t *testing.T, expected int, s *WMailServer) {
	count := countMessages(t, s.db)
	require.Equal(t, expected, count, fmt.Sprintf("expected %d message, got: %d", expected, count))
}

func countMessages(t *testing.T, db dbImpl) int {
	var (
		count int
		zero  common.Hash
	)

	now := time.Now()
	kl := NewDBKey(uint32(0), zero)
	ku := NewDBKey(uint32(now.Unix()), zero)
	i := db.NewIterator(&util.Range{Start: kl.raw, Limit: ku.raw}, nil)
	defer i.Release()

	for i.Next() {
		var env whisper.Envelope
		err := rlp.DecodeBytes(i.Value(), &env)
		if err != nil {
			t.Fatal(err)
		}

		count++
	}

	return count
}
