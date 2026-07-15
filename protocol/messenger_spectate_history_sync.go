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

// spectatedCommunityInitialSyncPeriod matches the app-wide default sync period
// so spectators keep the same history horizon as everyone else.
const spectatedCommunityInitialSyncPeriod = 9 * 24 * time.Hour

func spectatedCommunitySyncFrom(nowUnixSeconds uint32) uint32 {
	return nowUnixSeconds - uint32(spectatedCommunityInitialSyncPeriod/time.Second)
}

type mailserverTopicRef struct {
	pubsubTopic  string
	contentTopic string
}

// mailserverTopicKey must match the key format syncFiltersFrom uses, so a
// seeded topic is recognised as already tracked.
func mailserverTopicKey(pubsubTopic, contentTopic string) string {
	return fmt.Sprintf("%s-%s", pubsubTopic, contentTopic)
}

// communityHistorySeedTopics returns watermark rows for topics not yet tracked:
// syncFiltersFrom hardcodes the full default sync period for topics missing
// from the mailserver DB, and already-tracked topics must not be rewound
// (AddTopics is INSERT-OR-REPLACE).
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

// communitySyncFilters resolves the community's DefaultFilters chat ids to live
// transport filters; every channel rides the community's one universal content
// topic, so this set covers all of its store-node traffic.
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

// asyncSyncSpectatedCommunity backfills a freshly-spectated community's history
// over the scoped window, replacing the global all-filters backfill spectating
// previously triggered. The fetch is cancellable per community (leave /
// background); a cancelled batch advances no watermark, so later syncs resume
// where it stopped.
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
					// Cancelled by leave/background — don't retry or penalise the
					// storenode's failure accounting.
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

// communityHistoryFetchRegistry tracks the in-flight per-community backfills so
// leave/background can cancel them. Registrations carry a monotonic token so a
// goroutine finishing on its own removes only the entry it still owns, never a
// fresher registration that replaced it.
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

// register cancels and replaces any fetch already registered for communityID.
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

func (r *communityHistoryFetchRegistry) cancel(communityID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, ok := r.entries[communityID]; ok {
		entry.cancel()
		delete(r.entries, communityID)
	}
}

func (r *communityHistoryFetchRegistry) cancelAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for communityID, entry := range r.entries {
		entry.cancel()
		delete(r.entries, communityID)
	}
}

// finish removes the registration only if token still identifies the current
// entry, so a newer fetch that replaced it stays cancellable.
func (r *communityHistoryFetchRegistry) finish(communityID string, token uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, ok := r.entries[communityID]; ok && entry.token == token {
		delete(r.entries, communityID)
	}
}
