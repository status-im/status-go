//go:build tkl

// Package tklmanager adapts the C token catalogue to the wallet read interface.
package tklmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/nim-token-lists/go/tkl"

	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"
)

var ErrRefreshUnavailable = errors.New("token catalogue refresh is not integrated")

type identity struct {
	chain   uint64
	address common.Address
}

type snapshot struct {
	revision       uint64
	tokens         []*types.Token
	byKey          map[string]*types.Token
	byAddress      map[identity]*types.Token
	byChain        map[uint64][]*types.Token
	aliases        map[string]string
	addressAliases map[identity]identity
	// Lazy list cache; accessed only with Manager.mu held.
	lists     []*types.TokenList
	buildTime time.Duration
}

// Manager owns one C handle. Writes, mirror rebuilds and lazy list reads are
// serialized; hot token reads only load an immutable Go index. All reads return
// caller-owned values.
type Manager struct {
	mu        sync.Mutex
	handle    *tkl.Handle
	load      func(context.Context) (tkl.Bootstrap, error)
	mirror    atomic.Pointer[snapshot]
	refreshIO *refreshRuntime
	shadow    *shadowState
	// Policy changes must go through this facade under mu, updating both the
	// core and these inputs before rebuilding/publishing a new snapshot.
	policy           tkl.Policy
	started          bool
	loaded           bool
	pendingChains    []uint64
	hasPendingChains bool
	notify           chan struct{}
}

var _ types.Catalogue = (*Manager)(nil)

func New(config tkl.Config, load func(context.Context) (tkl.Bootstrap, error), options ...RefreshOptions) (*Manager, error) {
	if load == nil {
		return nil, errors.New("bootstrap loader is required")
	}
	if len(options) > 1 {
		return nil, errors.New("only one refresh configuration is allowed")
	}
	var refresh *refreshRuntime
	if len(options) == 1 {
		var err error
		refresh, err = newRefreshRuntime(options[0])
		if err != nil {
			return nil, err
		}
	}
	h, err := tkl.Create(config)
	if err != nil {
		return nil, err
	}
	policy := config.Policy
	policy.SkippedKeys = append([]string(nil), policy.SkippedKeys...)
	policy.NativeAliases = append([]tkl.Identity(nil), policy.NativeAliases...)
	policy.NativeTokens = append([]tkl.Token(nil), policy.NativeTokens...)
	m := &Manager{handle: h, load: load, policy: policy, refreshIO: refresh}
	if refresh != nil && refresh.options.OnShadow != nil {
		m.shadow = newShadow(config, refresh)
	}
	return m, nil
}

func (m *Manager) Start(ctx context.Context, autoRefresh bool, notify chan struct{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.handle == nil || (m.refreshIO != nil && m.refreshIO.closing) {
		return tkl.Closed
	}
	if autoRefresh && m.refreshIO == nil {
		return ErrRefreshUnavailable
	}
	if m.started {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := m.ensureLoaded(ctx); err != nil {
		return err
	}
	if err := m.rebuild(); err != nil {
		return err
	}
	m.started, m.notify = true, notify
	if err := m.startRefresh(ctx, autoRefresh); err != nil {
		return err
	}
	m.captureShadow()
	return nil
}

// ensureLoaded runs under mu and is also used by configuration writes before Start.
func (m *Manager) ensureLoaded(ctx context.Context) error {
	if !m.loaded {
		bootstrap, err := m.load(ctx)
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err = m.handle.LoadStored(bootstrap); err != nil {
			return err
		}
		m.loaded = true
		m.shadowLoaded(bootstrap)
	}
	if m.hasPendingChains {
		if _, err := m.handle.SetChains(m.pendingChains); err != nil {
			return err
		}
		m.hasPendingChains = false
		if m.shadow != nil && !m.shadow.disabled {
			m.shadow.config.Chains = append([]uint64(nil), m.pendingChains...)
		}
	}
	return nil
}

func (m *Manager) Stop() error {
	// This cancellation function is immutable from construction. Cancel before
	// waiting for the writer lock so context-aware SQL can release that lock.
	if r := m.refreshIO; r != nil {
		r.stop()
	}
	m.mu.Lock()
	if m.handle == nil {
		m.mu.Unlock()
		return nil
	}
	if r := m.refreshIO; r != nil {
		if r.closing {
			m.mu.Unlock()
			<-r.done
			return nil
		}
		r.closing = true
		if r.detachParent != nil {
			r.detachParent()
		}
		if r.cancel != nil {
			r.cancel()
		}
		m.mu.Unlock()
		r.workers.Wait()
		m.mu.Lock()
		defer close(r.done)
	}
	defer m.mu.Unlock()
	err := m.handle.Destroy()
	m.handle = nil
	m.mirror.Store(nil)
	m.shadow = nil
	return err
}

func (m *Manager) SetChains(chains []uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.handle == nil || (m.refreshIO != nil && m.refreshIO.closing) {
		return tkl.Closed
	}
	if !m.started {
		m.pendingChains = append([]uint64(nil), chains...)
		m.hasPendingChains = true
		return nil
	}
	before := m.mirror.Load()
	change, err := m.handle.SetChains(chains)
	if err != nil {
		return err
	}
	if err := m.rebuild(); err != nil {
		return err
	}
	if m.shadow != nil && !m.shadow.disabled {
		m.shadow.config.Chains = append([]uint64(nil), chains...)
	}
	m.captureShadow()
	if after := m.mirror.Load(); before == nil || before.revision != after.revision {
		m.notifyChange(change)
	}
	return nil
}

// rebuild is called with mu held, so every bulk page belongs to one revision.
func (m *Manager) rebuild() error {
	var started time.Time
	if m.shadow != nil {
		started = time.Now()
	}
	revision := m.handle.Revision()
	if old := m.mirror.Load(); old != nil && old.revision == revision {
		return nil
	}
	page, err := m.handle.GetAll(0, 0)
	if err != nil {
		return err
	}
	if page.Revision != revision || len(page.Items) != page.Total {
		return errors.New("inconsistent catalogue snapshot")
	}
	next := &snapshot{revision: revision, tokens: make([]*types.Token, 0, len(page.Items)), byKey: make(map[string]*types.Token), byChain: make(map[uint64][]*types.Token)}
	next.byAddress = make(map[identity]*types.Token, len(page.Items))
	for _, token := range page.Items {
		value := convertToken(token)
		next.tokens = append(next.tokens, value)
		next.byKey[value.Key()] = value
		next.byAddress[identity{value.ChainID, value.Address}] = value
		next.byChain[value.ChainID] = append(next.byChain[value.ChainID], value)
	}
	next.aliases = make(map[string]string)
	next.addressAliases = make(map[identity]identity)
	skipped := make(map[string]bool)
	for _, key := range m.policy.SkippedKeys {
		skipped[strings.ToLower(key)] = true
	}
	for _, alias := range m.policy.NativeAliases {
		key := types.TokenKey(alias.ChainID, common.HexToAddress(alias.Address))
		if !skipped[key] {
			next.aliases[key] = types.TokenKey(alias.ChainID, common.Address{})
			next.addressAliases[identity{alias.ChainID, common.HexToAddress(alias.Address)}] = identity{alias.ChainID, common.Address{}}
		}
	}
	if !started.IsZero() {
		next.buildTime = time.Since(started)
	}
	m.mirror.Store(next)
	return nil
}

// loadLists runs under mu so revision changes and destruction cannot overlap
// the fetch. Failures are not cached: the SDK read interface has no error return,
// so this call reports no lists and a subsequent call can retry.
func (m *Manager) loadLists() (*snapshot, error) {
	s := m.mirror.Load()
	if s == nil || m.handle == nil {
		return nil, nil
	}
	if s.lists != nil {
		return s, nil
	}
	page, err := m.handle.GetLists()
	if err != nil {
		return nil, err
	}
	if page.Revision != s.revision || m.handle.Revision() != s.revision {
		return nil, errors.New("inconsistent list snapshot")
	}
	lists := make([]*types.TokenList, 0, len(page.Items))
	for _, list := range page.Items {
		for _, part := range []int64{list.Version.Major, list.Version.Minor, list.Version.Patch} {
			if int64(int(part)) != part {
				return nil, fmt.Errorf("list %s version exceeds SDK integer range", list.ID)
			}
		}
		value := &types.TokenList{ID: list.ID, Name: list.Name, Timestamp: list.Timestamp, FetchedTimestamp: list.FetchedTimestamp, Source: list.Source, LogoURI: list.LogoURI, Keywords: list.Keywords, Version: types.Version{Major: int(list.Version.Major), Minor: int(list.Version.Minor), Patch: int(list.Version.Patch)}, Tokens: make([]*types.Token, 0, len(list.Tokens))}
		if len(list.Tags) > 0 {
			if err := json.Unmarshal(list.Tags, &value.Tags); err != nil {
				return nil, fmt.Errorf("decode list %s tags: %w", list.ID, err)
			}
		}
		for _, token := range list.Tokens {
			value.Tokens = append(value.Tokens, convertToken(token))
		}
		lists = append(lists, value)
	}
	s.lists = lists
	return s, nil
}

func convertToken(t tkl.Token) *types.Token {
	return &types.Token{ChainID: t.ChainID, Address: common.HexToAddress(t.Address), CrossChainID: t.CrossChainID, Decimals: uint(t.Decimals), Name: t.Name, Symbol: t.Symbol, LogoURI: t.LogoURI, CustomToken: t.Custom}
}
func cloneToken(t *types.Token) *types.Token {
	if t == nil {
		return nil
	}
	value := *t
	return &value
}
func cloneTokens(tokens []*types.Token) []*types.Token {
	out := make([]*types.Token, len(tokens))
	for i, t := range tokens {
		out[i] = cloneToken(t)
	}
	return out
}
func (m *Manager) UniqueTokens() []*types.Token {
	if s := m.mirror.Load(); s != nil {
		return cloneTokens(s.tokens)
	}
	return nil
}
func (m *Manager) GetTokenByChainAddress(chain uint64, address common.Address) (*types.Token, bool) {
	s := m.mirror.Load()
	if s == nil {
		return nil, false
	}
	key := identity{chain, address}
	if canonical, ok := s.addressAliases[key]; ok {
		key = canonical
	}
	token, ok := s.byAddress[key]
	return cloneToken(token), ok
}
func (m *Manager) GetTokensByChain(chain uint64) []*types.Token {
	if s := m.mirror.Load(); s != nil {
		return cloneTokens(s.byChain[chain])
	}
	return nil
}
func (m *Manager) GetTokensByKeys(keys []string) ([]*types.Token, error) {
	s := m.mirror.Load()
	if s == nil {
		return nil, nil
	}
	result := make([]*types.Token, 0, len(keys))
	for _, key := range keys {
		key = strings.ToLower(key)
		if chain, address, ok := types.ChainAndAddressFromTokenKey(key); ok {
			key = types.TokenKey(chain, address)
		}
		if canonical, ok := s.aliases[key]; ok {
			key = canonical
		}
		if token := s.byKey[key]; token != nil {
			result = append(result, cloneToken(token))
		}
	}
	return result, nil
}
func cloneList(list *types.TokenList) *types.TokenList {
	value := *list
	value.Tokens = cloneTokens(list.Tokens)
	value.Keywords = append([]string(nil), list.Keywords...)
	if list.Tags != nil {
		data, _ := json.Marshal(list.Tags)
		value.Tags = nil
		_ = json.Unmarshal(data, &value.Tags)
	}
	return &value
}
func (m *Manager) TokenLists() []*types.TokenList {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, err := m.loadLists()
	if err != nil || s == nil {
		return nil
	}
	result := make([]*types.TokenList, 0, len(s.lists))
	for _, list := range s.lists {
		result = append(result, cloneList(list))
	}
	return result
}
func (m *Manager) TokenList(id string) (*types.TokenList, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, err := m.loadLists(); err == nil && s != nil {
		for _, list := range s.lists {
			if list.ID == id {
				return cloneList(list), true
			}
		}
	}
	return nil, false
}
