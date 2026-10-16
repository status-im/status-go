package protocol

import (
	"fmt"
	"sort"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"

	"github.com/status-im/status-go/internal/connection"
	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/pkg/services/mailservers"
)

const (
	initialStoreNodeRequestPageSize = 4
	defaultStoreNodeRequestPageSize = 50

	// tolerance is how many seconds of potentially out-of-order messages we want to fetch
	tolerance uint32 = 60

	oneDayDuration   = 24 * time.Hour
	oneMonthDuration = 31 * oneDayDuration

	// historicSyncMinInterval is the minimum spacing between two historic
	// syncs. The historic-sync worker enforces it by delaying the pending sync,
	// never by dropping triggers (see startHistoricSyncWorker).
	historicSyncMinInterval = 20 * time.Second

	// historicSyncRetryInterval and historicSyncRetryTimeout bound the
	// historic-sync worker's per-sync retry. On login the libp2p host is
	// freshly recreated — and right after a connectivity change the store node
	// may not be dialable yet — so first attempts can fail.
	historicSyncRetryInterval = 5 * time.Second
	historicSyncRetryTimeout  = 3 * time.Minute
)

var ErrNoFiltersForChat = errors.New("no filter registered for given chat")

func (m *Messenger) shouldSync() (bool, error) {
	if !m.started || !m.Online() {
		return false, nil
	}

	useMailserver, err := m.settings.CanUseMailservers()
	if err != nil {
		m.logger.Error("failed to get use mailservers", zap.Error(err))
		return false, err
	}
	if !useMailserver {
		return false, nil
	}

	// The waku node always has the fleet's store nodes configured, so a
	// mailserver is available whenever mailservers are enabled.
	return true, nil
}

func (m *Messenger) scheduleSyncChat(chat *Chat) (bool, error) {
	shouldSync, err := m.shouldSync()
	if err != nil {
		m.logger.Error("failed to get should sync", zap.Error(err))
		return false, err
	}

	if !shouldSync {
		return false, nil
	}

	go func() {
		defer panics.LogOnPanic()
		response, err := m.syncChatWithFilters(chat.ID)
		if err != nil {
			m.logger.Error("failed to sync chat", zap.Error(err))
			return
		}

		if m.config.messengerSignalsHandler != nil {
			m.config.messengerSignalsHandler.MessengerResponse(response)
		}
	}()
	return true, nil
}

func (m *Messenger) scheduleSyncFilters(filters messagingtypes.ChatFilters) (bool, error) {
	shouldSync, err := m.shouldSync()
	if err != nil {
		m.logger.Error("failed to get shouldSync", zap.Error(err))
		return false, err
	}

	if !shouldSync {
		return false, nil
	}

	go func() {
		defer panics.LogOnPanic()
		response, err := m.syncFilters(filters)
		if err != nil {
			m.logger.Error("failed to sync filter", zap.Error(err))
			return
		}

		if m.config.messengerSignalsHandler != nil {
			m.config.messengerSignalsHandler.MessengerResponse(response)
		}
	}()
	return true, nil
}

func (m *Messenger) calculateMailserverTo() time.Time {
	return time.Unix(0, int64(time.Duration(m.GetCurrentTimeInMillis())*time.Millisecond))
}

func (m *Messenger) calculateMailserverTimeBounds(duration time.Duration) (time.Time, time.Time) {
	now := time.Unix(0, int64(time.Duration(m.GetCurrentTimeInMillis())*time.Millisecond))
	to := now
	from := now.Add(-duration)
	return from, to
}

func (m *Messenger) filtersForChat(chatID string) (messagingtypes.ChatFilters, error) {
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return nil, ErrChatNotFound
	}
	var filters []*messagingtypes.ChatFilter

	if chat.OneToOne() {
		// We sync our own topic and any eventual negotiated
		publicKeys := []string{crypto.PubkeyToHex(&m.identity.PublicKey), chatID}

		filters = m.messaging.ChatFiltersByIdentities(publicKeys)

	} else if chat.PrivateGroupChat() {
		var publicKeys []string
		for _, m := range chat.Members {
			publicKeys = append(publicKeys, m.ID)
		}

		filters = m.messaging.ChatFiltersByIdentities(publicKeys)

	} else {
		filter := m.messaging.ChatFilterByChatID(chatID)
		if filter == nil {
			return nil, ErrNoFiltersForChat
		}
		filters = []*messagingtypes.ChatFilter{filter}
	}

	return filters, nil
}

func (m *Messenger) topicsForChat(chatID string) (string, []messagingtypes.ContentTopic, error) {
	filters, err := m.filtersForChat(chatID)
	if err != nil {
		return "", nil, err
	}

	var contentTopics []messagingtypes.ContentTopic

	for _, filter := range filters {
		contentTopics = append(contentTopics, filter.ContentTopic())
	}

	return filters[0].PubsubTopic(), contentTopics, nil
}

func (m *Messenger) syncChatWithFilters(chatID string) (*MessengerResponse, error) {
	filters, err := m.filtersForChat(chatID)
	if err != nil {
		return nil, err
	}

	return m.syncFilters(filters)
}

func (m *Messenger) defaultSyncPeriodFromNow() (uint32, error) {
	defaultSyncPeriod, err := m.settings.GetDefaultSyncPeriod()
	if err != nil {
		return 0, err
	}
	return uint32(m.getTimesource().GetCurrentTime()/1000) - defaultSyncPeriod, nil
}

// capToDefaultSyncPeriod caps the sync period to the default
func (m *Messenger) capToDefaultSyncPeriod(period uint32) (uint32, error) {
	d, err := m.defaultSyncPeriodFromNow()
	if err != nil {
		return 0, err
	}
	if d > period {
		return d, nil
	}
	return period - tolerance, nil
}

func (m *Messenger) updateFiltersPriority(filters messagingtypes.ChatFilters) error {
	for _, filter := range filters {
		chatID := filter.ChatID()
		chat := m.Chat(chatID)
		if chat != nil {
			err := m.messaging.UpdateFilterPriority(chatID, chat.ReadMessagesAtClockValue)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *Messenger) resetFiltersPriority(filters messagingtypes.ChatFilters) error {
	for _, filter := range filters {
		err := m.messaging.UpdateFilterPriority(filter.ChatID(), 0)
		if err != nil {
			return err
		}
	}

	return nil
}

// RequestAllHistoricMessages requests all the historic messages for any topic.
// It keeps aggregating all responses for callers that need the merged payload.
func (m *Messenger) RequestAllHistoricMessages() (*MessengerResponse, error) {
	return m.requestAllHistoricMessages(true)
}

func (m *Messenger) requestAllHistoricMessages(aggregateResponses bool) (*MessengerResponse, error) {
	response, _, err := m.requestAllHistoricMessagesWithOptions(aggregateResponses, syncFiltersOptions{})
	return response, err
}

func (m *Messenger) runAutomaticHistoricSync(request historicSyncRequest) (bool, error) {
	if m.isPaused() {
		return false, nil
	}
	_, executed, err := m.requestAllHistoricMessagesWithOptions(false, syncFiltersOptions{
		Window: &messagingtypes.HistoryReconcileWindow{From: request.From, To: request.To},
	})
	return executed, err
}

func (m *Messenger) requestAllHistoricMessagesWithOptions(aggregateResponses bool, options syncFiltersOptions) (*MessengerResponse, bool, error) {
	shouldSync, err := m.shouldSync()
	if err != nil {
		return nil, false, err
	}

	if !shouldSync {
		return nil, false, nil
	}

	if m.mailserversDatabase == nil {
		return nil, true, nil
	}

	canSync, err := m.canSyncWithStoreNodes()
	if err != nil {
		return nil, false, err
	}
	if !canSync {
		return nil, false, nil
	}

	return m.withHistoricSyncInFlight(func() (*MessengerResponse, error) {
		var allResponses *MessengerResponse
		if aggregateResponses {
			allResponses = &MessengerResponse{}
		}

		filters := m.messaging.ChatFilters()
		err = m.updateFiltersPriority(filters)
		if err != nil {
			return nil, fmt.Errorf("failed to update filters priority: %w", err)
		}
		defer func() {
			err := m.resetFiltersPriority(filters)
			if err != nil {
				m.logger.Error("failed to reset filters priority", zap.Error(err))
			}
		}()

		// Retry and failover are handled per query by the StoreClient (it pins a
		// store node for the whole query and fails over only at query boundaries).
		response, err := m.syncFiltersWithOptions(filters, options)
		if err != nil {
			return nil, err
		}
		if aggregateResponses && response != nil {
			allResponses.AddChats(response.Chats())
			allResponses.AddMessages(response.Messages())
		}
		return allResponses, nil
	})
}

// withHistoricSyncInFlight runs fn unless another historic sync is already in
// progress. Automatic syncs are serialized and spaced by the historic-sync
// worker (startHistoricSyncWorker); this gate protects against a manual
// RequestAllHistoricMessages (RPC) racing the worker.
func (m *Messenger) withHistoricSyncInFlight(fn func() (*MessengerResponse, error)) (*MessengerResponse, bool, error) {
	m.historicSyncMu.Lock()
	if m.historicSyncInFlight {
		m.historicSyncMu.Unlock()
		m.logger.Debug("skip historic sync request (already in progress)")
		return nil, false, nil
	}
	m.historicSyncInFlight = true
	m.historicSyncMu.Unlock()
	defer func() {
		m.historicSyncMu.Lock()
		m.historicSyncInFlight = false
		m.historicSyncMu.Unlock()
		m.notifyHistoricSyncWorker()
	}()

	response, err := fn()
	return response, true, err
}

type syncFiltersOptions struct {
	// ExactFrom preserves the archive-builder behavior: existing topics are
	// queried from this timestamp regardless of their completeness cursor.
	ExactFrom uint32
	// Window bounds automatic reconciliation. A zero From means cursor-based
	// catch-up with only a fixed upper bound (used at startup).
	Window *messagingtypes.HistoryReconcileWindow
}

func applyHistoryWindowFloor(from uint32, initialized bool, window *messagingtypes.HistoryReconcileWindow) uint32 {
	if !initialized || window == nil || window.From.IsZero() {
		return from
	}
	windowFrom := window.From.Unix() - int64(tolerance)
	if windowFrom < 0 {
		windowFrom = 0
	}
	if uint32(windowFrom) > from {
		return uint32(windowFrom)
	}
	return from
}

func getPrioritizedBatches() []int {
	return []int{1, 5, 10}
}

func (m *Messenger) syncFiltersFrom(filters messagingtypes.ChatFilters, lastRequest uint32) (*MessengerResponse, error) {
	return m.syncFiltersWithOptions(filters, syncFiltersOptions{ExactFrom: lastRequest})
}

func (m *Messenger) syncFiltersWithOptions(filters messagingtypes.ChatFilters, options syncFiltersOptions) (*MessengerResponse, error) {
	canSync, err := m.canSyncWithStoreNodes()
	if err != nil {
		return nil, err
	}
	if !canSync {
		return nil, nil
	}

	response := &MessengerResponse{}
	topicInfo, err := m.mailserversDatabase.Topics()
	if err != nil {
		return nil, err
	}

	topicsData := make(map[string]mailservers.MailserverTopic, len(topicInfo))
	for _, topic := range topicInfo {
		topicsData[fmt.Sprintf("%s-%s", topic.PubsubTopic, topic.ContentTopic)] = topic
	}

	batches := make(map[string]map[int]messagingtypes.StoreNodeBatch)

	to := m.calculateMailserverTo()
	if options.Window != nil && !options.Window.To.IsZero() {
		to = options.Window.To
	}
	syncedTopics := make([]mailservers.MailserverTopic, 0, len(filters))

	sort.Slice(filters[:], func(i, j int) bool {
		p1 := filters[i].Priority()
		p2 := filters[j].Priority()
		return p1 > p2
	})
	prioritizedBatches := getPrioritizedBatches()
	currentBatch := 0

	if len(filters) == 0 || filters[0].Priority() == 0 {
		currentBatch = len(prioritizedBatches)
	}
	if options.Window != nil && !options.Window.From.IsZero() {
		// Windowed reconciliation must evaluate every topic's lower and upper
		// bounds independently; priority batches may intentionally combine
		// topics with different cursors.
		currentBatch = len(prioritizedBatches)
	}

	defaultPeriodFromNow, err := m.defaultSyncPeriodFromNow()
	if err != nil {
		return nil, err
	}

	contentTopicsPerPubsubTopic := make(map[string]map[string]*messagingtypes.ChatFilter, len(filters))
	for _, filter := range filters {
		if !filter.IsListening() || filter.IsEphemeral() {
			continue
		}

		contentTopics, ok := contentTopicsPerPubsubTopic[filter.PubsubTopic()]
		if !ok {
			contentTopics = make(map[string]*messagingtypes.ChatFilter)
		}
		contentTopics[filter.ContentTopic().String()] = filter
		contentTopicsPerPubsubTopic[filter.PubsubTopic()] = contentTopics
	}

	communityDescriptionChatIDs, err := m.communityDescriptionChatIDs()
	if err != nil {
		return nil, err
	}
	var communityDescriptionFilters []*messagingtypes.ChatFilter

	for pubsubTopic, contentTopics := range contentTopicsPerPubsubTopic {
		if _, ok := batches[pubsubTopic]; !ok {
			batches[pubsubTopic] = make(map[int]messagingtypes.StoreNodeBatch)
		}

		for _, filter := range contentTopics {
			var chatID string
			// If the filter has an identity, we use it as a chatID, otherwise is a public chat/community chat filter
			if len(filter.Identity()) != 0 {
				chatID = filter.Identity()
			} else {
				chatID = filter.ChatID()
			}

			// The community description content topic republishes the full
			// community description (including every member) many times per day.
			// Only the most recent description is ever used, so sweeping this
			// topic over the whole sync window downloads gigabytes of stale
			// duplicates. Skip it here and fetch only the newest description
			// (single page, newest-first) below. See status-im/status-app#21498.
			if _, isCommunityDescription := communityDescriptionChatIDs[chatID]; isCommunityDescription {
				communityDescriptionFilters = append(communityDescriptionFilters, filter)
				continue
			}

			topicData, ok := topicsData[fmt.Sprintf("%s-%s", filter.PubsubTopic(), filter.ContentTopic())]
			topicExists := ok
			var capToDefaultSyncPeriod = true
			if !ok {
				topicData = mailservers.MailserverTopic{
					PubsubTopic:  filter.PubsubTopic(),
					ContentTopic: filter.ContentTopic().String(),
					LastRequest:  int(defaultPeriodFromNow),
				}
			} else if options.ExactFrom != 0 {
				topicData.LastRequest = int(options.ExactFrom)
				capToDefaultSyncPeriod = false
			}

			batchID := topicData.LastRequest

			if currentBatch < len(prioritizedBatches) {
				batch, ok := batches[pubsubTopic][currentBatch]
				if ok {
					prevTopicData, ok := topicsData[batch.PubsubTopic+batch.Topics[0].String()]
					if (!ok && topicData.LastRequest != int(defaultPeriodFromNow)) ||
						(ok && prevTopicData.LastRequest != topicData.LastRequest) {
						currentBatch++
					}
				}
				if currentBatch < len(prioritizedBatches) {
					batchID = currentBatch
					currentBatchCap := prioritizedBatches[currentBatch] - 1
					if currentBatchCap == 0 {
						currentBatch++
					} else {
						prioritizedBatches[currentBatch] = currentBatchCap
					}
				}
			}

			batch, ok := batches[pubsubTopic][batchID]
			if !ok {
				from := uint32(topicData.LastRequest)
				if capToDefaultSyncPeriod {
					from, err = m.capToDefaultSyncPeriod(uint32(topicData.LastRequest))
					if err != nil {
						return nil, err
					}
				}
				// Only initialized, existing topics may be narrowed to an
				// unreliable window. New/reset topics still need their default
				// initial history regardless of the current window.
				from = applyHistoryWindowFloor(
					from,
					topicExists && topicData.LastRequest > 0,
					options.Window,
				)
				if int64(from) >= to.Unix() {
					continue
				}
				batch = messagingtypes.StoreNodeBatch{From: time.Unix(int64(from), 0), To: to}
			}

			batch.ChatIDs = append(batch.ChatIDs, chatID)
			batch.PubsubTopic = pubsubTopic
			batch.Topics = append(batch.Topics, filter.ContentTopic())
			batches[pubsubTopic][batchID] = batch

			// Set last request to the new `to`
			if topicData.LastRequest < int(to.Unix()) {
				topicData.LastRequest = int(to.Unix())
			}
			syncedTopics = append(syncedTopics, topicData)
		}
	}

	batchedPubsubTopics := 0
	for _, topicBatches := range batches {
		if len(topicBatches) > 0 {
			batchedPubsubTopics++
		}
	}
	if batchedPubsubTopics > 0 && m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestStarted(batchedPubsubTopics)
	}

	for pubsubTopic := range batches {
		batchKeys := maps.Keys(batches[pubsubTopic])
		sort.Ints(batchKeys)
		for _, k := range batchKeys {
			err := m.processMailserverBatch(batches[pubsubTopic][k])
			if err != nil {
				m.logger.Error("error syncing topics", zap.Error(err))
				return nil, err
			}
		}
	}

	m.logger.Debug("topics synced")
	if batchedPubsubTopics > 0 && m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestCompleted()
	}

	if len(syncedTopics) > 0 {
		err = m.mailserversDatabase.AddTopics(syncedTopics)
		if err != nil {
			return nil, err
		}
	}

	messagesToBeSaved := make([]*common.Message, 0, len(syncedTopics))
	for _, batches := range batches {
		for _, batch := range batches {
			for _, id := range batch.ChatIDs {
				chat, ok := m.allChats.Load(id)
				if !ok || !chat.Active || chat.Timeline() || chat.ProfileUpdates() {
					continue
				}
				gap, err := m.calculateGapForChat(chat, uint32(batch.From.Unix()))
				if err != nil {
					return nil, err
				}
				if chat.SyncedFrom == 0 || chat.SyncedFrom > uint32(batch.From.Unix()) {
					chat.SyncedFrom = uint32(batch.From.Unix())
				}

				chat.SyncedTo = uint32(to.Unix())

				err = m.persistence.SetSyncTimestamps(chat.SyncedFrom, chat.SyncedTo, chat.ID)
				if err != nil {
					return nil, err
				}

				response.AddChat(chat)
				if gap != nil {
					response.AddMessage(gap)
					messagesToBeSaved = append(messagesToBeSaved, gap)
				}
			}
		}
	}

	if len(messagesToBeSaved) > 0 {
		err := m.persistence.SaveMessages(messagesToBeSaved)
		if err != nil {
			return nil, err
		}
	}

	m.fetchLatestCommunityDescriptions(communityDescriptionFilters)

	return response, nil
}

func (m *Messenger) syncFilters(filters messagingtypes.ChatFilters) (*MessengerResponse, error) {
	return m.syncFiltersWithOptions(filters, syncFiltersOptions{})
}

// communityDescriptionChatIDs returns chat IDs for joined and spectated communities.
// We fetch only the latest description for each one instead of all old copies.
func (m *Messenger) communityDescriptionChatIDs() (map[string]struct{}, error) {
	communities, err := m.communitiesManager.JoinedOrSpectated()
	if err != nil {
		return nil, err
	}
	chatIDs := make(map[string]struct{}, len(communities))
	for _, community := range communities {
		chatIDs[community.IDString()] = struct{}{}
	}
	return chatIDs, nil
}

// fetchLatestCommunityDescriptions gets the latest description for each
// community filter and stops after the first page of results.
// This avoids downloading many older copies.
func (m *Messenger) fetchLatestCommunityDescriptions(filters []*messagingtypes.ChatFilter) {
	if len(filters) == 0 {
		return
	}

	from, to := m.calculateMailserverTimeBounds(oneMonthDuration)
	if !from.Before(to) {
		return
	}
	stopAfterFirstPage := func(int) (bool, uint64) {
		return false, 0
	}

	for _, filter := range filters {
		batch := messagingtypes.StoreNodeBatch{
			From:        from,
			To:          to,
			PubsubTopic: filter.PubsubTopic(),
			Topics:      []messagingtypes.ContentTopic{filter.ContentTopic()},
			ChatIDs:     []string{filter.ChatID()},
		}
		err := m.processMailserverBatchWithOptions(batch, 1, stopAfterFirstPage, false)
		if err != nil {
			m.logger.Error("failed to fetch latest community description",
				zap.String("chatID", filter.ChatID()),
				zap.Error(err))
		}
	}
}

func (m *Messenger) calculateGapForChat(chat *Chat, from uint32) (*common.Message, error) {
	// Chat was never synced, no gap necessary
	if chat.SyncedTo == 0 {
		return nil, nil
	}

	// If we filled the gap, nothing to do
	if chat.SyncedTo >= from {
		return nil, nil
	}

	timestamp := m.getTimesource().GetCurrentTime()

	message := &common.Message{
		ChatMessage: &protobuf.ChatMessage{
			ChatId:      chat.ID,
			Text:        "Gap message",
			MessageType: protobuf.MessageType_SYSTEM_MESSAGE_GAP,
			ContentType: protobuf.ChatMessage_SYSTEM_MESSAGE_GAP,
			Clock:       uint64(from) * 1000,
			Timestamp:   timestamp,
		},
		GapParameters: &common.GapParameters{
			From: chat.SyncedTo,
			To:   from,
		},
		From:             crypto.PubkeyToHex(&m.identity.PublicKey),
		WhisperTimestamp: timestamp,
		LocalChatID:      chat.ID,
		Seen:             true,
		ID:               cryptotypes.EncodeHex(crypto.Keccak256([]byte(fmt.Sprintf("%s-%d-%d", chat.ID, chat.SyncedTo, from)))),
	}

	return message, m.persistence.SaveMessages([]*common.Message{message})
}

func (m *Messenger) canSyncWithStoreNodes() (bool, error) {
	if m.isConnectionExpensive() {
		return m.settings.CanSyncOnMobileNetwork()
	}
	return true, nil
}

func (m *Messenger) ConnectionChanged(state connection.State) {
	m.messaging.ConnectionChanged(state)
	m.setConnectionState(state)
}

func (m *Messenger) setConnectionState(state connection.State) {
	m.connectionStateMu.Lock()
	defer m.connectionStateMu.Unlock()

	m.connectionState = state
}

func (m *Messenger) isConnectionExpensive() bool {
	m.connectionStateMu.RLock()
	defer m.connectionStateMu.RUnlock()

	return m.connectionState.IsExpensive()
}

// processMailserverBatch queries the store for a single batch, applying the
// mobile-network gate. The store node is selected internally by the StoreClient
// (no peer argument).
func (m *Messenger) processMailserverBatch(batch messagingtypes.StoreNodeBatch) error {
	canSync, err := m.canSyncWithStoreNodes()
	if err != nil {
		return err
	}
	if !canSync {
		return nil
	}

	return m.messaging.Query(m.ctx, batch, defaultStoreNodeRequestPageSize, nil, false)
}

func (m *Messenger) processMailserverBatchWithOptions(batch messagingtypes.StoreNodeBatch, pageLimit uint64, shouldProcessNextPage func(int) (bool, uint64), processEnvelopes bool) error {
	canSync, err := m.canSyncWithStoreNodes()
	if err != nil {
		return err
	}
	if !canSync {
		return nil
	}

	return m.messaging.Query(m.ctx, batch, pageLimit, shouldProcessNextPage, processEnvelopes)
}

func (m *Messenger) SyncChatFromSyncedFrom(chatID string) (uint32, error) {
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return 0, ErrChatNotFound
	}

	canSync, err := m.canSyncWithStoreNodes()
	if err != nil {
		return 0, err
	}
	if !canSync {
		return 0, nil
	}

	pubsubTopic, topics, err := m.topicsForChat(chatID)
	if err != nil {
		return 0, nil
	}

	defaultSyncPeriod, err := m.settings.GetDefaultSyncPeriod()
	if err != nil {
		return 0, err
	}

	batch := messagingtypes.StoreNodeBatch{
		ChatIDs:     []string{chatID},
		To:          time.Unix(int64(chat.SyncedFrom), 0),
		From:        time.Unix(int64(chat.SyncedFrom-defaultSyncPeriod), 0),
		PubsubTopic: pubsubTopic,
		Topics:      topics,
	}
	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestStarted(1)
	}

	err = m.processMailserverBatch(batch)
	if err != nil {
		return 0, err
	}

	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestCompleted()
	}
	if chat.SyncedFrom == 0 || chat.SyncedFrom > uint32(batch.From.Unix()) {
		chat.SyncedFrom = uint32(batch.From.Unix())
	}

	m.logger.Debug("setting sync timestamps", zap.Int64("from", batch.From.Unix()), zap.Int64("to", int64(chat.SyncedTo)), zap.String("chatID", chatID))

	err = m.persistence.SetSyncTimestamps(uint32(batch.From.Unix()), chat.SyncedTo, chat.ID)
	if err != nil {
		return 0, err
	}

	return uint32(batch.From.Unix()), nil
}

func (m *Messenger) FillGaps(chatID string, messageIDs []string) error {
	messages, err := m.persistence.MessagesByIDs(messageIDs)
	if err != nil {
		return err
	}

	_, ok := m.allChats.Load(chatID)
	if !ok {
		return errors.New("chat not existing")
	}

	pubsubTopic, topics, err := m.topicsForChat(chatID)
	if err != nil {
		return err
	}

	var lowestFrom, highestTo uint32

	for _, message := range messages {
		if message.GapParameters == nil {
			return errors.New("can't sync non-gap message")
		}

		if lowestFrom == 0 || lowestFrom > message.GapParameters.From {
			lowestFrom = message.GapParameters.From
		}

		if highestTo < message.GapParameters.To {
			highestTo = message.GapParameters.To
		}
	}

	batch := messagingtypes.StoreNodeBatch{
		ChatIDs:     []string{chatID},
		To:          time.Unix(int64(highestTo), 0),
		From:        time.Unix(int64(lowestFrom), 0),
		PubsubTopic: pubsubTopic,
		Topics:      topics,
	}

	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestStarted(1)
	}

	err = m.processMailserverBatch(batch)
	if err != nil {
		return err
	}

	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestCompleted()
	}

	return m.persistence.DeleteMessages(messageIDs)
}

func (m *Messenger) ToggleUseMailservers(value bool) error {
	err := m.settings.SetUseMailservers(value)
	if err != nil {
		return err
	}

	return nil
}

func (m *Messenger) RemoveFilters(filters []*messagingtypes.ChatFilter) error {
	return m.messaging.RemoveFilters(filters)
}

func (m *Messenger) fetchMessages(chatID string, duration time.Duration) (uint32, error) {
	from, to := m.calculateMailserverTimeBounds(duration)

	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return 0, ErrChatNotFound
	}

	canSync, err := m.canSyncWithStoreNodes()
	if err != nil {
		return 0, err
	}
	if !canSync {
		return uint32(from.Unix()), nil
	}

	m.logger.Debug("fetching messages", zap.String("chatID", chatID))
	pubsubTopic, topics, err := m.topicsForChat(chatID)
	if err != nil {
		return uint32(from.Unix()), nil
	}

	batch := messagingtypes.StoreNodeBatch{
		ChatIDs:     []string{chatID},
		From:        from,
		To:          to,
		PubsubTopic: pubsubTopic,
		Topics:      topics,
	}
	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestStarted(1)
	}

	err = m.processMailserverBatch(batch)
	if err != nil {
		return 0, err
	}

	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestCompleted()
	}
	if chat.SyncedFrom == 0 || chat.SyncedFrom > uint32(batch.From.Second()) {
		chat.SyncedFrom = uint32(batch.From.Second())
	}

	m.logger.Debug("setting sync timestamps", zap.Int64("from", batch.From.Unix()), zap.Int64("to", int64(chat.SyncedTo)), zap.String("chatID", chatID))

	err = m.persistence.SetSyncTimestamps(uint32(batch.From.Unix()), chat.SyncedTo, chat.ID)
	if err != nil {
		return 0, err
	}

	return uint32(from.Unix()), nil
}
