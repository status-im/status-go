package tklmanager

import (
	"context"
	"errors"

	"github.com/status-im/nim-token-lists/go/tkl"

	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"
)

// UpsertCustom validates in the core before invoking persist. Persistence must
// honor ctx, be atomic, and must not reenter this manager. The writer lock covers
// preparation through publication so no other writer can invalidate durable SQL.
func (m *Manager) UpsertCustom(ctx context.Context, row *types.Token, persist func(context.Context, *types.Token) error) error {
	if row == nil || row.Decimals > 255 || persist == nil {
		return tkl.InvalidArgument
	}
	token := tkl.Token{ChainID: row.ChainID, Address: row.Address.Hex(), Name: row.Name, Symbol: row.Symbol, Decimals: uint8(row.Decimals), Custom: true}
	// The existing SQL schema stores these fields only. Do not publish metadata
	// that would be lost on the next bootstrap.
	return m.mutateCustom(ctx, func() (tkl.Mutation, error) {
		mutation, err := m.handle.CustomValidateUpsert(token)
		if errors.Is(err, tkl.ValidationFailed) {
			// Discovery stores metadata even when it is not catalogue-eligible.
			// Prepare removal of any old eligible version, so the same SQL/commit
			// transaction cannot leave a stale token visible after an update.
			return m.handle.CustomValidateDelete(row.Key())
		}
		return mutation, err
	}, func(ctx context.Context, _ tkl.Mutation) error {
		return persist(ctx, convertToken(token))
	})
}

// DeleteCustom preserves SQL's idempotent delete, including rows the core skipped
// during bootstrap (invalid customs and community-only tokens).
func (m *Manager) DeleteCustom(ctx context.Context, key string, persist func(context.Context) error) error {
	if persist == nil {
		return tkl.InvalidArgument
	}
	return m.mutateCustom(ctx, func() (tkl.Mutation, error) { return m.handle.CustomValidateDelete(key) }, func(ctx context.Context, _ tkl.Mutation) error { return persist(ctx) })
}

func (m *Manager) mutateCustom(ctx context.Context, prepare func() (tkl.Mutation, error), persist func(context.Context, tkl.Mutation) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.handle == nil || (m.refreshIO != nil && m.refreshIO.closing) {
		return tkl.Closed
	}
	if m.refreshIO != nil {
		callCtx, cancel := context.WithCancel(ctx)
		detach := context.AfterFunc(m.refreshIO.ctx, cancel)
		defer detach()
		defer cancel()
		if m.refreshIO.ctx.Err() != nil {
			return tkl.Closed
		}
		ctx = callCtx
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := m.ensureLoaded(ctx); err != nil {
		return err
	}
	mutation, err := prepare()
	if errors.Is(err, tkl.NotFound) {
		if err := persist(ctx, tkl.Mutation{}); err != nil {
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}
	defer func() { _, _ = m.handle.CustomAbort(mutation.ID) }()
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = persist(ctx, mutation); err != nil {
		return err
	}
	// Once SQL succeeds we must commit even if cancellation has just arrived.
	change, err := m.handle.CustomCommit(mutation.ID)
	if err != nil {
		return err
	}
	if m.started && change.Kind != "NoChange" {
		m.notifyChange(change)
	}
	return nil
}
