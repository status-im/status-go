//go:build tkl

package tklmanager

import (
	"context"
	"time"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"
	"github.com/status-im/nim-token-lists/go/tkl"

	"github.com/status-im/status-go/internal/panics"
)

const (
	shadowMaxBytes   = 32 << 20
	shadowMaxSources = 128
	shadowMaxTokens  = 100000
	shadowMaxSamples = 64
)

// ShadowSnapshot owns a revision-consistent copy for a read-only observer.
// Skipped is a bounded reason code, never source content. An observer must honor
// cancellation and must not reenter the manager; Stop drains it before returning.
type ShadowSnapshot struct {
	Config      tkl.Config
	Contents    []tkl.ListContent
	Customs     []tkl.Token
	Tokens      []types.Token
	Revision    uint64
	Sequence    int
	Coalesced   int
	MirrorBuild time.Duration
	Skipped     string
}

// All state except the queue consumer is protected by Manager.mu.
type shadowState struct {
	config      tkl.Config
	contents    map[string]tkl.ListContent
	customs     map[string]tkl.Token
	queue       chan ShadowSnapshot
	started     time.Time
	bytes       int
	submissions int
	coalesced   int
	disabled    bool
}

func cloneShadowConfig(c tkl.Config) tkl.Config {
	c.Chains = append([]uint64(nil), c.Chains...)
	c.InitialLists = cloneShadowContents(c.InitialLists)
	c.Policy.SkippedKeys = append([]string(nil), c.Policy.SkippedKeys...)
	c.Policy.NativeAliases = append([]tkl.Identity(nil), c.Policy.NativeAliases...)
	c.Policy.NativeTokens = append([]tkl.Token(nil), c.Policy.NativeTokens...)
	return c
}

func cloneShadowContents(rows []tkl.ListContent) []tkl.ListContent {
	result := append([]tkl.ListContent(nil), rows...)
	for i := range result {
		if result[i].Failure != nil {
			diagnostic := *result[i].Failure
			result[i].Failure = &diagnostic
		}
	}
	return result
}

func newShadow(config tkl.Config, r *refreshRuntime) *shadowState {
	s := &shadowState{config: cloneShadowConfig(config), contents: make(map[string]tkl.ListContent), customs: make(map[string]tkl.Token), queue: make(chan ShadowSnapshot, 1), started: time.Now()}
	s.bytes = len(config.EmbeddedRegistry)
	for _, row := range config.InitialLists {
		s.bytes += shadowContentBytes(row)
	}
	r.workers.Add(1)
	go func() {
		defer panics.LogOnPanic()
		defer r.workers.Done()
		ctx, cancel := context.WithDeadline(r.ctx, s.started.Add(24*time.Hour))
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				if r.ctx.Err() == nil {
					r.options.OnShadow(ctx, ShadowSnapshot{Skipped: "window_expired"})
				}
				return
			case snapshot := <-s.queue:
				if ctx.Err() != nil {
					return
				}
				r.options.OnShadow(ctx, snapshot)
			}
		}
	}()
	return s
}

func (s *shadowState) enqueue(snapshot ShadowSnapshot) {
	select {
	case <-s.queue:
		s.coalesced++
	default:
	}
	snapshot.Coalesced = s.coalesced
	s.queue <- snapshot
}

func (s *shadowState) disable(reason string) {
	s.disabled = true
	s.config = tkl.Config{}
	s.contents = nil
	s.customs = nil
	s.enqueue(ShadowSnapshot{Skipped: reason, Sequence: s.submissions})
}

func shadowContentBytes(row tkl.ListContent) int {
	return len(row.Body) + len(row.ID) + len(row.Source) + len(row.ETag) + len(row.FetchedTimestamp) + 128
}
func shadowTokenBytes(row tkl.Token) int {
	return len(row.Address) + len(row.Name) + len(row.Symbol) + len(row.LogoURI) + len(row.CrossChainID) + 128
}

func (s *shadowState) withinBudget() bool {
	if s.disabled {
		return false
	}
	if time.Since(s.started) >= 24*time.Hour {
		s.disable("window_expired")
		return false
	}
	if s.submissions >= shadowMaxSamples {
		s.disable("sample_limit")
		return false
	}
	if len(s.contents)+len(s.config.InitialLists) > shadowMaxSources || len(s.customs) > shadowMaxTokens {
		s.disable("input_count_limit")
		return false
	}
	if s.bytes > shadowMaxBytes {
		s.disable("input_byte_limit")
		return false
	}
	return true
}

func (m *Manager) shadowLoaded(bootstrap tkl.Bootstrap) {
	s := m.shadow
	if s == nil || !s.withinBudget() {
		return
	}
	m.shadowWrites(bootstrap.Contents)
	if s.disabled {
		return
	}
	for _, row := range bootstrap.Customs {
		key := convertToken(row).Key()
		if old, ok := s.customs[key]; ok {
			s.bytes -= shadowTokenBytes(old)
		}
		s.customs[key] = row
		s.bytes += shadowTokenBytes(row)
		if !s.withinBudget() {
			return
		}
	}
}

func (m *Manager) shadowWrites(rows []tkl.ListContent) {
	s := m.shadow
	if s == nil || !s.withinBudget() {
		return
	}
	for _, row := range rows {
		if old, ok := s.contents[row.ID]; ok {
			s.bytes -= shadowContentBytes(old)
		}
		s.contents[row.ID] = cloneShadowContents([]tkl.ListContent{row})[0]
		s.bytes += shadowContentBytes(row)
		if !s.withinBudget() {
			return
		}
	}
}

func (m *Manager) shadowCustom(key string, row *tkl.Token) {
	s := m.shadow
	if s == nil || !s.withinBudget() {
		return
	}
	if old, ok := s.customs[key]; ok {
		s.bytes -= shadowTokenBytes(old)
	}
	if row == nil {
		delete(s.customs, key)
	} else {
		s.customs[key] = *row
		s.bytes += shadowTokenBytes(*row)
	}
	m.captureShadow()
}

func (m *Manager) captureShadow() {
	s := m.shadow
	if s == nil || !m.started || !s.withinBudget() {
		return
	}
	current := m.mirror.Load()
	if current == nil {
		return
	}
	if len(current.tokens) > shadowMaxTokens {
		s.disable("token_limit")
		return
	}
	next := ShadowSnapshot{Config: cloneShadowConfig(s.config), Revision: current.revision, MirrorBuild: current.buildTime, Sequence: s.submissions + 1}
	for _, row := range s.contents {
		next.Contents = append(next.Contents, cloneShadowContents([]tkl.ListContent{row})[0])
	}
	for _, row := range s.customs {
		next.Customs = append(next.Customs, row)
	}
	next.Tokens = make([]types.Token, len(current.tokens))
	for i, row := range current.tokens {
		next.Tokens[i] = *row
	}
	s.submissions++
	s.enqueue(next)
}
