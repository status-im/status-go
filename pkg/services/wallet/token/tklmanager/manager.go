// Package tklmanager adapts the C token catalogue to the wallet read interface.
package tklmanager

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/nim-token-lists/go/tkl"

	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"
)

var ErrRefreshUnavailable = errors.New("token catalogue refresh is not integrated")

// Loader returns the host's persisted state with the bundled and stored list
// bodies. Bodies are only borrowed while the catalogue loads.
type Loader func(context.Context) (tkl.Bootstrap, []tkl.ListBody, error)

type identity struct {
	chain   uint64
	address common.Address
}

// ChainAddress identifies one token in a batch lookup.
type ChainAddress struct {
	ChainID uint64
	Address common.Address
}

// Manager owns one C handle, which is the only token index: queries call it
// directly and return caller-owned values. Writes are serialized by mu.
type Manager struct {
	mu     sync.Mutex
	handle *tkl.Handle
	// reader is the handle once a catalogue is loaded, nil before and after.
	reader atomic.Pointer[tkl.Handle]
	// aliases mirrors the policy's native aliases; change both via setPolicy.
	aliases          atomic.Pointer[map[identity]identity]
	load             Loader
	refreshIO        *refreshRuntime
	started          bool
	loaded           bool
	pendingChains    []uint64
	hasPendingChains bool
	notify           chan struct{}
}

var _ types.Catalogue = (*Manager)(nil)

func New(config tkl.Config, load Loader, options ...RefreshOptions) (*Manager, error) {
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
	m := &Manager{handle: h, load: load, refreshIO: refresh}
	m.storeAliases(config.Policy)
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
	m.started, m.notify = true, notify
	if err := m.startRefresh(ctx, autoRefresh); err != nil {
		return err
	}
	return nil
}

// ensureLoaded runs under mu and is also used by configuration writes before Start.
func (m *Manager) ensureLoaded(ctx context.Context) error {
	if !m.loaded {
		bootstrap, bodies, err := m.load(ctx)
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err = m.handle.LoadStored(bootstrap, bodies); err != nil {
			return err
		}
		m.loaded = true
	}
	if m.hasPendingChains {
		if _, err := m.handle.SetChains(m.pendingChains); err != nil {
			return err
		}
		m.hasPendingChains = false
	}
	m.reader.Store(m.handle)
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
	m.reader.Store(nil)
	err := m.handle.Destroy()
	m.handle = nil
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
	before := m.handle.Revision()
	change, err := m.handle.SetChains(chains)
	if err != nil {
		return err
	}
	if m.handle.Revision() != before {
		m.notifyChange(change)
	}
	return nil
}

// setPolicy replaces the catalogue policy; it is called with mu held.
func (m *Manager) setPolicy(policy tkl.Policy) error {
	if _, err := m.handle.SetPolicy(policy); err != nil {
		return err
	}
	m.storeAliases(policy)
	return nil
}

// storeAliases keeps the native aliases the core applies, so batch results
// can be matched to the requested identities.
func (m *Manager) storeAliases(policy tkl.Policy) {
	skipped := make(map[string]bool, len(policy.SkippedKeys))
	for _, key := range policy.SkippedKeys {
		skipped[strings.ToLower(key)] = true
	}
	aliases := make(map[identity]identity, len(policy.NativeAliases))
	for _, alias := range policy.NativeAliases {
		address := common.HexToAddress(alias.Address)
		if !skipped[types.TokenKey(alias.ChainID, address)] {
			aliases[identity{alias.ChainID, address}] = identity{alias.ChainID, common.Address{}}
		}
	}
	m.aliases.Store(&aliases)
}

func convertToken(t tkl.Token) *types.Token {
	return &types.Token{ChainID: t.ChainID, Address: common.HexToAddress(t.Address), CrossChainID: t.CrossChainID, Decimals: uint(t.Decimals), Name: t.Name, Symbol: t.Symbol, LogoURI: t.LogoURI, CustomToken: t.Custom}
}

func convertTokens(page tkl.Page[tkl.Token], err error) []*types.Token {
	if err != nil {
		return nil
	}
	out := make([]*types.Token, len(page.Items))
	for i, t := range page.Items {
		out[i] = convertToken(t)
	}
	return out
}

func convertList(list tkl.TokenList) (*types.TokenList, error) {
	for _, part := range []int64{list.Version.Major, list.Version.Minor, list.Version.Patch} {
		if int64(int(part)) != part {
			return nil, fmt.Errorf("list %s version exceeds Go integer range", list.ID)
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
	return value, nil
}

// hexAddress is the lowercase hex form the core accepts, without a checksum.
func hexAddress(address common.Address) string {
	buf := make([]byte, 2, 2+2*common.AddressLength)
	copy(buf, "0x")
	return string(hex.AppendEncode(buf, address[:]))
}

func (m *Manager) UniqueTokens() []*types.Token {
	h := m.reader.Load()
	if h == nil {
		return nil
	}
	return convertTokens(h.GetAll(0, 0))
}

func (m *Manager) GetTokenByChainAddress(chain uint64, address common.Address) (*types.Token, bool) {
	h := m.reader.Load()
	if h == nil {
		return nil, false
	}
	page, err := h.GetByChainAddress(chain, hexAddress(address))
	if err != nil || len(page.Items) == 0 {
		return nil, false
	}
	return convertToken(page.Items[0]), true
}

// GetTokensByChainAddresses looks up many tokens in one call. The result is
// aligned with ids; unknown tokens are nil.
func (m *Manager) GetTokensByChainAddresses(ids []ChainAddress) []*types.Token {
	result := make([]*types.Token, len(ids))
	h := m.reader.Load()
	if h == nil || len(ids) == 0 {
		return result
	}
	aliases := *m.aliases.Load()
	pairs := make([]tkl.Identity, len(ids))
	for i, id := range ids {
		pairs[i] = tkl.Identity{ChainID: id.ChainID, Address: hexAddress(id.Address)}
	}
	page, err := h.GetByChainAddresses(pairs)
	if err != nil {
		return result
	}
	found := make(map[identity]*types.Token, len(page.Items))
	for _, t := range page.Items {
		token := convertToken(t)
		found[identity{token.ChainID, token.Address}] = token
	}
	for i, id := range ids {
		key := identity{id.ChainID, id.Address}
		if canonical, ok := aliases[key]; ok {
			key = canonical
		}
		if token := found[key]; token != nil {
			value := *token
			result[i] = &value
		}
	}
	return result
}

func (m *Manager) GetTokensByChain(chain uint64) []*types.Token {
	return m.GetTokensByChains([]uint64{chain})
}

// GetTokensByChains returns the catalogue tokens of chains in catalogue order.
func (m *Manager) GetTokensByChains(chains []uint64) []*types.Token {
	h := m.reader.Load()
	if h == nil || len(chains) == 0 {
		return nil
	}
	return convertTokens(h.GetByChains(chains, 0, 0))
}

// GetTokensByKeys returns the tokens of keys in request order, skipping unknown
// keys. Keys are parsed like ChainAndAddressFromTokenKey.
func (m *Manager) GetTokensByKeys(keys []string) ([]*types.Token, error) {
	h := m.reader.Load()
	if h == nil {
		return nil, nil
	}
	pairs := make([]tkl.Identity, 0, len(keys))
	for _, key := range keys {
		if chain, address, ok := types.ChainAndAddressFromTokenKey(strings.ToLower(key)); ok {
			pairs = append(pairs, tkl.Identity{ChainID: chain, Address: hexAddress(address)})
		}
	}
	if len(pairs) == 0 {
		return make([]*types.Token, 0), nil
	}
	tokens := convertTokens(h.GetByChainAddresses(pairs))
	if tokens == nil {
		tokens = make([]*types.Token, 0)
	}
	return tokens, nil
}

func (m *Manager) TokenLists() []*types.TokenList {
	h := m.reader.Load()
	if h == nil {
		return nil
	}
	page, err := h.GetLists()
	if err != nil {
		return nil
	}
	result := make([]*types.TokenList, 0, len(page.Items))
	for _, list := range page.Items {
		value, err := convertList(list)
		if err != nil {
			return nil
		}
		result = append(result, value)
	}
	return result
}

func (m *Manager) TokenList(id string) (*types.TokenList, bool) {
	h := m.reader.Load()
	if h == nil {
		return nil, false
	}
	page, err := h.GetList(id)
	if err != nil || len(page.Items) == 0 {
		return nil, false
	}
	value, err := convertList(page.Items[0])
	if err != nil {
		return nil, false
	}
	return value, true
}
