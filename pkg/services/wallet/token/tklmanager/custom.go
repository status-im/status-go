//go:build tkl

package tklmanager

import (
	"context"
	"errors"
	"maps"
	"slices"
	"time"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"
	"github.com/status-im/nim-token-lists/go/tkl"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/logutils"
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
	return m.mutateCustom(ctx, row.Key(), &token, func() (tkl.Mutation, error) {
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
	return m.mutateCustom(ctx, key, nil, func() (tkl.Mutation, error) { return m.handle.CustomValidateDelete(key) }, func(ctx context.Context, _ tkl.Mutation) error { return persist(ctx) })
}

func (m *Manager) mutateCustom(ctx context.Context, key string, row *tkl.Token, prepare func() (tkl.Mutation, error), persist func(context.Context, tkl.Mutation) error) error {
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
		m.shadowCustom(key, row)
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
	if err = m.publishCustom(mutation); err != nil {
		// Retain the durable operation, not its SQL callback. Readers and writers
		// retry publication before using the catalogue; SQL must not run twice.
		m.customResync.Store(&mutation)
		logutils.ZapLogger().Error("catalogue publish failed after custom write", zap.Error(err))
	}
	m.shadowCustom(key, row)
	// Custom SQL writes did not emit token-list refresh events in the SDK.
	return nil
}

func (m *Manager) publishCustom(mutation tkl.Mutation) error {
	change, err := m.handle.CustomCommit(mutation.ID)
	if err != nil {
		return err
	}
	return m.updateCustomMirror(mutation.Key, change.Revision)
}

// resyncCustom runs under mu. Replay only the already-durable operation. Calling
// LoadStored again would fail with AlreadyLoaded on the existing native handle.
func (m *Manager) resyncCustom() error {
	pending := m.customResync.Load()
	if pending == nil {
		return nil
	}
	var mutation tkl.Mutation
	var err error
	if pending.Kind == "UpsertCustom" {
		mutation, err = m.handle.CustomValidateUpsert(pending.Token)
	} else {
		mutation, err = m.handle.CustomValidateDelete(pending.Key)
	}
	if errors.Is(err, tkl.NotFound) {
		// A delete may already have committed before publication failed.
		err = m.rebuild()
	} else if err == nil {
		defer func() { _, _ = m.handle.CustomAbort(mutation.ID) }()
		err = m.publishCustom(mutation)
	}
	if err == nil {
		m.customResync.Store(nil)
		m.captureShadow()
	}
	return err
}

// updateCustomMirror transfers just the affected token across the ABI. The
// snapshot's containers are copied so concurrent readers keep an immutable view.
func (m *Manager) updateCustomMirror(key string, revision uint64) error {
	var started time.Time
	if m.shadow != nil {
		started = time.Now()
	}
	old := m.mirror.Load()
	// Bootstrap, recovery, aliases and non-default priority can change ordering
	// beyond this key. Preserve the core's complete ordering in those cases.
	if old == nil || old.revision+1 != revision || old.aliases[key] != "" ||
		(m.policy.Priority != "" && m.policy.Priority != "StatusPriority") {
		return m.rebuild()
	}
	page, err := m.handle.GetByKeys([]string{key})
	if err != nil {
		return err
	}
	if page.Revision != revision || page.Total != len(page.Items) || len(page.Items) > 1 {
		return errors.New("inconsistent custom token snapshot")
	}
	chain, address, ok := types.ChainAndAddressFromTokenKey(key)
	if !ok {
		return errors.New("invalid custom token key")
	}
	var token *types.Token
	if len(page.Items) == 1 {
		token = convertToken(page.Items[0])
	}
	next := *old
	next.revision, next.lists = revision, nil
	next.byKey = maps.Clone(old.byKey)
	next.byAddress = maps.Clone(old.byAddress)
	next.byChain = maps.Clone(old.byChain)
	id := identity{chain, address}
	if token == nil {
		delete(next.byKey, key)
		delete(next.byAddress, id)
	} else {
		next.byKey[key], next.byAddress[id] = token, token
	}
	next.tokens = replaceCustomToken(old.tokens, id, token)
	next.byChain[chain] = replaceCustomToken(old.byChain[chain], id, token)
	if len(next.byChain[chain]) == 0 {
		delete(next.byChain, chain)
	}
	if !started.IsZero() {
		next.buildTime = time.Since(started)
	}
	m.mirror.Store(&next)
	return nil
}

func replaceCustomToken(tokens []*types.Token, id identity, replacement *types.Token) []*types.Token {
	result := slices.Clone(tokens)
	for i, token := range result {
		if token.ChainID == id.chain && token.Address == id.address {
			if replacement == nil {
				return slices.Delete(result, i, i+1)
			}
			result[i] = replacement
			return result
		}
	}
	if replacement != nil {
		result = append(result, replacement)
	}
	return result
}
