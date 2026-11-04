//go:build include_postgres_tests
// +build include_postgres_tests

package utils

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/persistence"
	"github.com/waku-org/go-waku/waku/persistence/postgres"
	"github.com/waku-org/go-waku/waku/persistence/sqlite"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_store/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func TestStore(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T, db *sql.DB, migrationFn func(*sql.DB, *zap.Logger) error)
	}{
		{"testDbStore", testDbStore},
		{"testStoreRetention", testStoreRetention},
		{"testQuery", testQuery},
	}
	for _, driverName := range []string{"postgres", "sqlite"} {
		// all tests are run for each db
		for _, tc := range tests {
			db, migrationFn := getDB(driverName)
			t.Run(driverName+"_"+tc.name, func(t *testing.T) {
				tc.fn(t, db, migrationFn)
			})
		}
	}
}

func getDB(driver string) (*sql.DB, func(*sql.DB, *zap.Logger) error) {
	switch driver {
	case "postgres":
		return postgres.NewMockPgDB(), postgres.Migrations
	case "sqlite":
		return sqlite.NewMockSqliteDB(), sqlite.Migrations
	}
	return nil, nil
}
func testDbStore(t *testing.T, db *sql.DB, migrationFn func(*sql.DB, *zap.Logger) error) {
	store, err := persistence.NewDBStore(prometheus.DefaultRegisterer, utils.Logger(), persistence.WithDB(db), persistence.WithMigrations(migrationFn))
	require.NoError(t, err)

	err = store.Start(context.Background(), timesource.NewDefaultClock())
	require.NoError(t, err)

	res, err := store.GetAll()
	require.NoError(t, err)
	require.Empty(t, res)

	err = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test", utils.GetUnixEpoch()), *utils.GetUnixEpoch(), "test"))
	require.NoError(t, err)

	res, err = store.GetAll()
	require.NoError(t, err)
	require.NotEmpty(t, res)
}

func testStoreRetention(t *testing.T, db *sql.DB, migrationFn func(*sql.DB, *zap.Logger) error) {
	store, err := persistence.NewDBStore(prometheus.DefaultRegisterer, utils.Logger(), persistence.WithDB(db), persistence.WithMigrations(migrationFn), persistence.WithRetentionPolicy(5, 20*time.Second))
	require.NoError(t, err)

	err = store.Start(context.Background(), timesource.NewDefaultClock())
	require.NoError(t, err)

	insertTime := time.Now()
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test1", proto.Int64(insertTime.Add(-70*time.Second).UnixNano())), insertTime.Add(-70*time.Second).UnixNano(), "test"))
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test2", proto.Int64(insertTime.Add(-60*time.Second).UnixNano())), insertTime.Add(-60*time.Second).UnixNano(), "test"))
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test3", proto.Int64(insertTime.Add(-50*time.Second).UnixNano())), insertTime.Add(-50*time.Second).UnixNano(), "test"))
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test4", proto.Int64(insertTime.Add(-40*time.Second).UnixNano())), insertTime.Add(-40*time.Second).UnixNano(), "test"))
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test5", proto.Int64(insertTime.Add(-30*time.Second).UnixNano())), insertTime.Add(-30*time.Second).UnixNano(), "test"))

	dbResults, err := store.GetAll()
	require.NoError(t, err)
	require.Len(t, dbResults, 5)

	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test6", proto.Int64(insertTime.Add(-20*time.Second).UnixNano())), insertTime.Add(-20*time.Second).UnixNano(), "test"))
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test7", proto.Int64(insertTime.Add(-10*time.Second).UnixNano())), insertTime.Add(-10*time.Second).UnixNano(), "test"))

	// This step simulates starting go-waku again from scratch

	store, err = persistence.NewDBStore(prometheus.DefaultRegisterer, utils.Logger(), persistence.WithDB(db), persistence.WithRetentionPolicy(5, 40*time.Second))
	require.NoError(t, err)

	err = store.Start(context.Background(), timesource.NewDefaultClock())
	require.NoError(t, err)

	dbResults, err = store.GetAll()
	require.NoError(t, err)
	require.Len(t, dbResults, 3)
	require.Equal(t, "test5", dbResults[0].Message.ContentTopic)
	require.Equal(t, "test6", dbResults[1].Message.ContentTopic)
	require.Equal(t, "test7", dbResults[2].Message.ContentTopic)
	// checking the number of all the message in the db
	msgCount, err := store.Count()
	require.NoError(t, err)
	require.Equal(t, msgCount, 3)
}

func testQuery(t *testing.T, db *sql.DB, migrationFn func(*sql.DB, *zap.Logger) error) {
	store, err := persistence.NewDBStore(prometheus.DefaultRegisterer, utils.Logger(), persistence.WithDB(db), persistence.WithMigrations(migrationFn), persistence.WithRetentionPolicy(5, 20*time.Second))
	require.NoError(t, err)

	insertTime := time.Now()
	//////////////////////////////////
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test1", proto.Int64(insertTime.Add(-40*time.Second).UnixNano())), insertTime.Add(-10*time.Second).UnixNano(), "test"))
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test2", proto.Int64(insertTime.Add(-30*time.Second).UnixNano())), insertTime.Add(-10*time.Second).UnixNano(), "test"))
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test3", proto.Int64(insertTime.Add(-20*time.Second).UnixNano())), insertTime.Add(-10*time.Second).UnixNano(), "test"))
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test3", proto.Int64(insertTime.Add(-20*time.Second).UnixNano())), insertTime.Add(-10*time.Second).UnixNano(), "test2"))
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test4", proto.Int64(insertTime.Add(-10*time.Second).UnixNano())), insertTime.Add(-10*time.Second).UnixNano(), "test"))

	//  Range [startTime-endTime]
	// Check: matching ContentTopics and pubsubTopic, and ts of msg in range
	// this filters test1,test2 contentTopics. test3 matches list of contentTopic but its not within the time range
	cursor, msgs, err := store.Query(&pb.HistoryQuery{
		PubsubTopic: "test",
		ContentFilters: []*pb.ContentFilter{
			{ContentTopic: "test1"},
			{ContentTopic: "test2"},
			{ContentTopic: "test3"},
		},
		PagingInfo: &pb.PagingInfo{PageSize: 10},
		StartTime:  proto.Int64(insertTime.Add(-40 * time.Second).UnixNano()),
		EndTime:    proto.Int64(insertTime.Add(-20 * time.Second).UnixNano()),
	})
	require.NoError(t, err)
	require.Len(t, msgs, 3)

	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test5", proto.Int64(insertTime.UnixNano())), insertTime.Add(-10*time.Second).UnixNano(), "test"))

	// Range [cursor-endTime]
	// Check:  matching ContentTopic,pubsubTopic, pageSize
	// cursor has last message id of test2 which is now-30second
	// endTime is now+1sec
	// matched messages are both test4,test5, but the len of returned result depends on pageSize
	var cursor2 *pb.Index
	for _, pageSize := range []int{1, 2} {
		cursorLocal, msgs, err := store.Query(&pb.HistoryQuery{
			PubsubTopic: "test",
			ContentFilters: []*pb.ContentFilter{
				{ContentTopic: "test4"},
				{ContentTopic: "test5"},
			},
			PagingInfo: &pb.PagingInfo{Cursor: cursor, PageSize: uint64(pageSize), Direction: pb.PagingInfo_FORWARD},
			EndTime:    proto.Int64(insertTime.Add(1 * time.Second).UnixNano()),
		})
		require.NoError(t, err)
		require.Len(t, msgs, pageSize) // due to pageSize
		require.Equal(t, msgs[0].Message.ContentTopic, "test4")
		if len(msgs) > 1 {
			require.Equal(t, msgs[1].Message.ContentTopic, "test5")
		}
		cursor2 = cursorLocal
	}

	// range [startTime-cursor(test5_ContentTopic_Msg)],
	// Check: backend range with cursor excludes test1 ContentTopic,  matching ContentTopic
	// check backward pagination
	_, msgs, err = store.Query(&pb.HistoryQuery{
		PubsubTopic: "test",
		ContentFilters: []*pb.ContentFilter{
			{ContentTopic: "test1"}, // contentTopic test1 is out of the range
			{ContentTopic: "test2"},
			{ContentTopic: "test3"},
			{ContentTopic: "test4"},
		},
		PagingInfo: &pb.PagingInfo{Cursor: cursor2, PageSize: 4, Direction: pb.PagingInfo_BACKWARD},
		StartTime:  proto.Int64(insertTime.Add(-39 * time.Second).UnixNano()),
	})
	require.NoError(t, err)
	require.Len(t, msgs, 3) // due to pageSize
	// Check:this also makes returned messages are sorted ascending
	for ind, msg := range msgs {
		require.Equal(t, msg.Message.ContentTopic, fmt.Sprintf("test%d", ind+2)) // test2,test3,test4
	}

	// checking most recent timestamp in db
	timestamp, err := store.MostRecentTimestamp()
	require.NoError(t, err)
	require.Equal(t, timestamp, insertTime.UnixNano())
}
