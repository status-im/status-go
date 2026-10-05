//go:build tkl

package token

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/status-im/nim-token-lists/go/tkl"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/internal/signal"
	"github.com/status-im/status-go/pkg/services/wallet/token/tklmanager"
	"github.com/status-im/status-go/pkg/services/wallet/walletevent"
)

func putTKLBatch(ctx context.Context, db *sql.DB, writes []tkl.ListContent) error {
	if len(writes) == 0 {
		return ctx.Err()
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, row := range writes {
		var fetched interface{}
		if row.FetchedAt != 0 {
			fetched = time.Unix(row.FetchedAt, 0).UTC()
		} else if row.FetchedTimestamp != "" {
			stamp, parseErr := time.Parse(time.RFC3339, row.FetchedTimestamp)
			if parseErr != nil {
				return parseErr
			}
			if !stamp.IsZero() {
				fetched = stamp.UTC()
			}
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO token_lists (id,source,etag,tokens_json,fetched) VALUES (?,?,?,?,?)`, row.ID, row.Source, row.ETag, []byte(row.Body), fetched)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// newTKLRefreshManager's callbacks own success timestamps and update events;
// callers must not route its notify channel through the legacy SDK notifier.
func newTKLRefreshManager(mng *Manager, chains []uint64, lastSuccess time.Time, client *http.Client, refreshInterval, checkInterval time.Duration) (*tklmanager.Manager, error) {
	return newTKLReadManager(mng, chains, lastSuccess, tklmanager.RefreshOptions{
		Client: client, RefreshInterval: refreshInterval, CheckInterval: checkInterval,
		Persist: func(ctx context.Context, writes []tkl.ListContent) error {
			return putTKLBatch(ctx, mng.walletDB, writes)
		},
		OnSuccess: mng.settings.SaveLastTokensUpdate,
		OnChange: func(tkl.Change) {
			signal.SendWalletEvent(signal.TokenListsUpdated, nil)
			if mng.walletFeed != nil {
				go func() {
					defer panics.LogOnPanic()
					mng.walletFeed.Send(walletevent.Event{Type: walletevent.EventTokenListsUpdated})
				}()
			}
		},
		OnError: func(err error) { logutils.ZapLogger().Error("Token catalogue refresh failed", zap.Error(err)) },
	})
}
