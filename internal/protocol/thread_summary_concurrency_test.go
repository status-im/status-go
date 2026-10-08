package protocol

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/appdatabase"
	protocolsqlite "github.com/status-im/status-go/internal/protocol/sqlite"
	"github.com/status-im/status-go/internal/testutils"
)

func writeThreadSummaryState(ctx context.Context, db interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}, generation int, rollback bool) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	name := fmt.Sprintf("state-%d", generation)
	if _, err := tx.ExecContext(ctx, "UPDATE threads SET name = ? WHERE thread_id = 'thread'", name); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE user_messages
		SET text = ?, source = ?, deleted = ?, seen = ? WHERE id = 'reply'`,
		name, "author-"+name, generation%2, (generation/2)%2); err != nil {
		return err
	}
	if rollback {
		return tx.Rollback()
	}
	return tx.Commit()
}

func TestThreadSummarySnapshotDuringExternalCommit(t *testing.T) {
	for _, warm := range []bool{false, true} {
		for _, remove := range []bool{false, true} {
			t.Run(fmt.Sprintf("warm_%t/remove_%t", warm, remove), func(t *testing.T) {
				db, teardown, err := testutils.SetupTestSQLDB(appdatabase.DbInitializer{}, "thread-summary-snapshot")
				require.NoError(t, err)
				t.Cleanup(func() { require.NoError(t, teardown()) })
				require.NoError(t, protocolsqlite.Migrate(db))
				db.SetMaxOpenConns(2)
				db.SetMaxIdleConns(2)
				p := newSQLitePersistence(db)
				seedThreadSummaryCache(t, p)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				require.NoError(t, writeThreadSummaryState(ctx, db, 0, false))
				writer, err := db.Conn(ctx)
				require.NoError(t, err)
				defer writer.Close()
				if warm {
					_, err := p.ThreadsWithSummariesByParentMessageIDs(testPublicChatID, []string{"parent"}, 6)
					require.NoError(t, err)
				}

				threads, err := p.readThreadsWithSummaries(6, func(queryer threadSummaryQuerier) ([]*Thread, error) {
					threads, err := p.threadsByParentMessageIDs(queryer, testPublicChatID, []string{"parent"})
					if err != nil {
						return nil, err
					}
					// Force a committed write after the metadata read, while the
					// reader's transaction is still open. No timing assumptions.
					if remove {
						tx, err := writer.BeginTx(ctx, nil)
						if err != nil {
							return nil, err
						}
						defer tx.Rollback() //nolint:errcheck
						if _, err := tx.ExecContext(ctx, "DELETE FROM user_messages"); err != nil {
							return nil, err
						}
						if _, err := tx.ExecContext(ctx, "DELETE FROM threads"); err != nil {
							return nil, err
						}
						if err := tx.Commit(); err != nil {
							return nil, err
						}
					} else if err := writeThreadSummaryState(ctx, writer, 1, false); err != nil {
						return nil, err
					}
					return threads, nil
				})
				require.NoError(t, err)
				require.Len(t, threads, 1)
				require.Equal(t, "state-0", threads[0].Name)
				require.NoError(t, checkThreadSummaryState(threads[0]))

				// The next read must see the commit, not reuse the older snapshot.
				threads, err = p.ThreadsWithSummariesByParentMessageIDs(testPublicChatID, []string{"parent"}, 6)
				require.NoError(t, err)
				if remove {
					require.Empty(t, threads)
				} else {
					require.Len(t, threads, 1)
					require.Equal(t, "state-1", threads[0].Name)
					require.NoError(t, checkThreadSummaryState(threads[0]))
				}
			})
		}
	}
}

func checkThreadSummaryState(thread *Thread) error {
	generation, err := strconv.Atoi(strings.TrimPrefix(thread.Name, "state-"))
	if err != nil {
		return err
	}
	if thread.LastMessage == nil {
		return fmt.Errorf("missing last message for %s", thread.Name)
	}
	if generation%2 == 1 {
		if thread.MessagesCount != 1 || thread.ParticipantsCount != 1 ||
			thread.LastMessage.Text != "Parent" || thread.UnviewedMessagesCount != 0 {
			return fmt.Errorf("mixed deleted-reply snapshot: %+v", thread)
		}
		return nil
	}
	unread := uint(1 - (generation/2)%2)
	if thread.MessagesCount != 2 || thread.ParticipantsCount != 2 ||
		thread.LastMessage.Text != thread.Name || thread.LastMessage.From != "author-"+thread.Name ||
		thread.UnviewedMessagesCount != unread || len(thread.ParticipantsPreviewIDs) != 2 ||
		thread.ParticipantsPreviewIDs[1] != "author-"+thread.Name {
		return fmt.Errorf("mixed visible-reply snapshot: %+v, last=%+v", thread, thread.LastMessage)
	}
	return nil
}

func TestThreadSummaryConcurrentReadersAndWriter(t *testing.T) {
	for _, cached := range []bool{false, true} {
		for _, single := range []bool{false, true} {
			for _, writers := range []int{1, 2} {
				t.Run(fmt.Sprintf("cache_%t/single_%t/writers_%d", cached, single, writers), func(t *testing.T) {
					db, teardown, err := testutils.SetupTestSQLDB(appdatabase.DbInitializer{}, "thread-summary-concurrency")
					require.NoError(t, err)
					t.Cleanup(func() { require.NoError(t, teardown()) })
					require.NoError(t, protocolsqlite.Migrate(db))
					db.SetMaxOpenConns(6)
					db.SetMaxIdleConns(6)
					p := newSQLitePersistence(db)
					if !cached {
						p.threadSummaryCache = nil
					}
					seedThreadSummaryCache(t, p)
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					require.NoError(t, writeThreadSummaryState(ctx, db, 0, false))

					read := func() error {
						var thread *Thread
						var err error
						if single {
							thread, err = p.ThreadWithSummaryByID(testPublicChatID, "thread")
						} else {
							var threads []*Thread
							threads, err = p.ThreadsWithSummariesByParentMessageIDs(testPublicChatID, []string{"parent"}, 6)
							if err == nil {
								if len(threads) != 1 {
									return fmt.Errorf("expected one thread, got %d", len(threads))
								}
								thread = threads[0]
							}
						}
						if err != nil {
							return err
						}
						return checkThreadSummaryState(thread)
					}
					require.NoError(t, read())

					start := make(chan struct{})
					errors := make(chan error, 6)
					var wg sync.WaitGroup
					for writer := 0; writer < writers; writer++ {
						wg.Add(1)
						go func(writer int) {
							defer wg.Done()
							<-start
							for iteration := 1; iteration <= 80; iteration++ {
								generation := writer*80 + iteration
								if err := writeThreadSummaryState(ctx, db, generation, generation%7 == 0); err != nil {
									errors <- err
									return
								}
							}
						}(writer)
					}
					for reader := 0; reader < 4; reader++ {
						wg.Add(1)
						go func() {
							defer wg.Done()
							<-start
							for iteration := 0; iteration < 80; iteration++ {
								if err := read(); err != nil {
									errors <- err
									return
								}
							}
						}()
					}
					close(start)
					wg.Wait()
					close(errors)
					for err := range errors {
						require.NoError(t, err)
					}
					require.NoError(t, read())
					var committedName string
					require.NoError(t, db.QueryRow("SELECT name FROM threads WHERE thread_id = 'thread'").Scan(&committedName))
					thread, err := p.ThreadWithSummaryByID(testPublicChatID, "thread")
					require.NoError(t, err)
					require.Equal(t, committedName, thread.Name)
					require.NoError(t, checkThreadSummaryState(thread))
				})
			}
		}
	}
}

func TestThreadSummarySnapshotReleasesConnectionOnError(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	db.SetMaxOpenConns(1)
	p := newSQLitePersistence(db)
	seedThreadSummaryCache(t, p)
	expected := errors.New("metadata read failed")
	_, err = p.readThreadsWithSummaries(6, func(queryer threadSummaryQuerier) ([]*Thread, error) {
		if _, err := p.threadByID(queryer, testPublicChatID, "thread"); err != nil {
			return nil, err
		}
		return nil, expected
	})
	require.ErrorIs(t, err, expected)
	// A one-connection pool exposes a leaked transaction or connection immediately.
	thread, err := p.ThreadWithSummaryByID(testPublicChatID, "thread")
	require.NoError(t, err)
	require.Equal(t, "Reply", thread.LastMessage.Text)
	require.NoError(t, writeThreadSummaryState(context.Background(), db, 0, false))
}
