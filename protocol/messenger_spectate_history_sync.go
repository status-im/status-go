package protocol

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/waku-org/go-waku/waku/v2/api/history"

	gocommon "github.com/status-im/status-go/common"
	types2 "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/services/mailservers"
)

// spectatedCommunityInitialSyncPeriod bounds the history backfill triggered when a
// user spectates a community (issue #21470-hf).
//
// A spectator holds no community decryption keys, and every channel in a community
// rides ONE universal content topic, so a full default-period (9-day) backfill pulls
// the entire community's traffic as undecryptable payloads. Device measurement
// (Samsung S21FE, fresh account spectating the Status community): ~2-3GB ingested in
// ~11min at ~330% service CPU, GC-bound. Because all channels share one content
// topic, per-channel scoping is impossible — the WINDOW is the only byte-cutting
// lever, so spectate is bounded to 24h.
const spectatedCommunityInitialSyncPeriod = 24 * time.Hour

// spectatedCommunitySyncFrom returns the store-node "from" timestamp (unix seconds)
// for a spectator's scoped backfill: now minus the spectate window.
func spectatedCommunitySyncFrom(nowUnixSeconds uint32) uint32 {
	return nowUnixSeconds - uint32(spectatedCommunityInitialSyncPeriod/time.Second)
}

// communityInitialHistorySync decides how a community's FIRST history backfill is
// scoped. Spectators get the tight 24h window (deeper history is undecryptable heat);
// joiners keep today's behavior — a full default-period backfill — because they hold
// keys and legitimately want the history.
func communityInitialHistorySync(spectated bool) (scoped bool, window time.Duration) {
	if spectated {
		return true, spectatedCommunityInitialSyncPeriod
	}
	return false, 0
}

// mailserverTopicRef identifies a store-node topic (its pubsub + content topic).
type mailserverTopicRef struct {
	pubsubTopic  string
	contentTopic string
}

// mailserverTopicKey is the map key syncFiltersFrom uses to look topics up; the seed
// path must match it so a seeded topic is recognised as "already tracked".
func mailserverTopicKey(pubsubTopic, contentTopic string) string {
	return fmt.Sprintf("%s-%s", pubsubTopic, contentTopic)
}

// communityHistorySeedTopics returns the watermark rows to insert so a spectator's
// first backfill of the given topics is bounded to lastRequest.
//
// syncFiltersFrom ignores its lastRequest argument for topics not yet present in the
// mailserver DB (it hardcodes the default sync period for fresh topics), so without a
// pre-seeded watermark a spectate would refetch the full default period. Topics
// already tracked are skipped — AddTopics is INSERT-OR-REPLACE, and rewinding a good
// watermark would refetch history we already have. Each topic is seeded at most once.
func communityHistorySeedTopics(refs []mailserverTopicRef, existing map[string]struct{}, lastRequest int) []mailservers.MailserverTopic {
	seeded := make(map[string]struct{}, len(refs))
	var seeds []mailservers.MailserverTopic
	for _, ref := range refs {
		key := mailserverTopicKey(ref.pubsubTopic, ref.contentTopic)
		if _, ok := existing[key]; ok {
			continue
		}
		if _, ok := seeded[key]; ok {
			continue
		}
		seeded[key] = struct{}{}
		seeds = append(seeds, mailservers.MailserverTopic{
			PubsubTopic:  ref.pubsubTopic,
			ContentTopic: ref.contentTopic,
			LastRequest:  lastRequest,
		})
	}
	return seeds
}

// communitySyncFilters returns the transport filters that carry a community's
// traffic: its single universal content topic plus the community-level filters. It
// resolves the community's DefaultFilters chat ids to the live filters created when
// the community was initialised. Every channel rides the one universal (community-id)
// content topic, so this set covers all of the community's store-node bytes.
func (m *Messenger) communitySyncFilters(community *communities.Community) types2.ChatFilters {
	var filters types2.ChatFilters
	seen := make(map[string]struct{})
	for _, spec := range m.DefaultFilters(community) {
		filter := m.messaging.ChatFilterByChatID(spec.ChatID)
		if filter == nil {
			continue
		}
		if _, ok := seen[filter.ChatID()]; ok {
			continue
		}
		seen[filter.ChatID()] = struct{}{}
		filters = append(filters, filter)
	}
	return filters
}

// seedCommunityHistoryWatermarks pre-seeds the mailserver watermark for each of the
// given filters' topics that is not already tracked, so the spectate backfill's first
// fetch is bounded to lastRequest instead of the full default sync period.
func (m *Messenger) seedCommunityHistoryWatermarks(filters types2.ChatFilters, lastRequest int) error {
	if m.mailserversDatabase == nil {
		return nil
	}

	existingTopics, err := m.mailserversDatabase.Topics()
	if err != nil {
		return err
	}
	existing := make(map[string]struct{}, len(existingTopics))
	for _, t := range existingTopics {
		existing[mailserverTopicKey(t.PubsubTopic, t.ContentTopic)] = struct{}{}
	}

	refs := make([]mailserverTopicRef, 0, len(filters))
	for _, filter := range filters {
		refs = append(refs, mailserverTopicRef{
			pubsubTopic:  filter.PubsubTopic(),
			contentTopic: filter.ContentTopic().String(),
		})
	}

	seeds := communityHistorySeedTopics(refs, existing, lastRequest)
	if len(seeds) == 0 {
		return nil
	}
	return m.mailserversDatabase.AddTopics(seeds)
}

// asyncSyncSpectatedCommunity backfills a freshly-spectated community's history over
// the scoped 24h window (issue #21470-hf), replacing the global all-filters,
// full-period backfill that spectating previously triggered. The fetch runs under a
// per-community cancellable context registered on the messenger, so it stops when the
// user leaves the community or the app is backgrounded.
//
// No message loss: skipped (older or cancelled) history stays on the store node; each
// completed batch advances the per-topic watermark so later syncs resume exactly where
// this one stopped; live relay delivery is unaffected; and the missing-message
// verifier remains a net for anything not covered. A cancelled batch advances no
// watermark, so nothing is marked fetched that was not.
func (m *Messenger) asyncSyncSpectatedCommunity(community *communities.Community) {
	shouldSync, err := m.shouldSync()
	if err != nil {
		m.logger.Error("failed to get shouldSync for spectated community", zap.Error(err))
		return
	}
	if !shouldSync {
		return
	}

	filters := m.communitySyncFilters(community)
	if len(filters) == 0 {
		return
	}

	communityID := community.IDString()
	from := spectatedCommunitySyncFrom(uint32(m.getTimesource().GetCurrentTime() / 1000))

	ctx, cancel := context.WithCancel(m.ctx)
	token := m.communityHistoryFetches.register(communityID, cancel)

	m.shutdownWaitGroup.Add(1)
	go func() {
		defer gocommon.LogOnPanic()
		defer m.shutdownWaitGroup.Done()
		defer cancel()
		defer m.communityHistoryFetches.finish(communityID, token)

		if err := m.seedCommunityHistoryWatermarks(filters, int(from)); err != nil {
			m.logger.Warn("failed to seed spectated community history watermarks",
				zap.String("communityID", communityID), zap.Error(err))
			return
		}

		peerInfo := m.messaging.GetActiveStorenode()
		_, err := m.performStorenodeTask(func() (*MessengerResponse, error) {
			response, err := m.syncFiltersFrom(ctx, peerInfo, filters, from)
			if err != nil {
				if ctx.Err() != nil {
					// Cancelled by leave / background: stop cleanly, without retrying
					// or penalising the storenode's failure accounting.
					m.logger.Debug("spectated community history sync cancelled",
						zap.String("communityID", communityID))
					return nil, nil
				}
				m.logger.Error("failed to sync spectated community history",
					zap.String("communityID", communityID), zap.Error(err))
				return nil, err
			}
			if m.config.messengerSignalsHandler != nil {
				m.config.messengerSignalsHandler.MessengerResponse(response)
			}
			return response, nil
		}, history.WithPeerID(peerInfo.ID))
		if err != nil {
			m.logger.Error("failed to perform spectated community history request",
				zap.String("communityID", communityID), zap.Error(err))
		}
	}()
}

// communityHistoryFetchRegistry tracks the cancellable, per-community spectate
// backfill goroutines so they can be aborted when the user leaves the community or
// the app is backgrounded (issue #21470-hf, part B). On device the backfill was
// measured continuing headless after the UI process died; cancellation stops it.
//
// Each registration is tagged with a monotonic token so a goroutine finishing on its
// own only removes the map entry it still owns, never a fresher registration that
// replaced it (context.CancelFunc values are not comparable, hence the token).
type communityHistoryFetchRegistry struct {
	mu      sync.Mutex
	seq     uint64
	entries map[string]communityHistoryFetchEntry
}

type communityHistoryFetchEntry struct {
	token  uint64
	cancel context.CancelFunc
}

func newCommunityHistoryFetchRegistry() *communityHistoryFetchRegistry {
	return &communityHistoryFetchRegistry{
		entries: make(map[string]communityHistoryFetchEntry),
	}
}

// register records a new in-flight fetch for communityID, cancelling and replacing
// any fetch already registered for it (e.g. a rapid re-spectate). It returns a token
// identifying this registration, to be passed to finish.
func (r *communityHistoryFetchRegistry) register(communityID string, cancel context.CancelFunc) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.entries[communityID]; ok {
		existing.cancel()
	}
	r.seq++
	token := r.seq
	r.entries[communityID] = communityHistoryFetchEntry{token: token, cancel: cancel}
	return token
}

// cancel aborts the in-flight fetch for communityID (leave / unspectate path). It is
// a no-op if none is registered.
func (r *communityHistoryFetchRegistry) cancel(communityID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, ok := r.entries[communityID]; ok {
		entry.cancel()
		delete(r.entries, communityID)
	}
}

// cancelAll aborts every in-flight fetch (app backgrounded).
func (r *communityHistoryFetchRegistry) cancelAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for communityID, entry := range r.entries {
		entry.cancel()
		delete(r.entries, communityID)
	}
}

// finish removes the registration for communityID IFF token still identifies the
// current entry. A goroutine calls this on natural completion; if a newer fetch has
// since replaced it, the entry is left untouched so the newer fetch stays cancellable.
func (r *communityHistoryFetchRegistry) finish(communityID string, token uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, ok := r.entries[communityID]; ok && entry.token == token {
		delete(r.entries, communityID)
	}
}
