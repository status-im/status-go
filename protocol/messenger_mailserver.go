package protocol

import (
	"fmt"
	"sort"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/waku-org/go-waku/waku/v2/api/history"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/connection"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/services/mailservers"
)

const (
	initialStoreNodeRequestPageSize = 4
	defaultStoreNodeRequestPageSize = 50

	// tolerance is how many seconds of potentially out-of-order messages we want to fetch
	tolerance uint32 = 60

	oneDayDuration   = 24 * time.Hour
	oneMonthDuration = 31 * oneDayDuration

	backoffByUserAction = 0 * time.Second
)

var ErrNoFiltersForChat = errors.New("no filter registered for given chat")

func (m *Messenger) shouldSync() (bool, error) {
	if m.transport.WakuVersion() != 2 {
		return false, nil
	}

	// TODO (pablo) support community store node as well
	if m.transport.GetActiveStorenode() == "" || !m.Online() {
		return false, nil
	}

	useMailserver, err := m.settings.CanUseMailservers()
	if err != nil {
		m.logger.Error("failed to get use mailservers", zap.Error(err))
		return false, err
	}

	return useMailserver, nil
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
		defer gocommon.LogOnPanic()
		peerID := m.getCommunityStorenode(chat.CommunityID)
		_, err = m.performStorenodeTask(func() (*MessengerResponse, error) {
			response, err := m.syncChatWithFilters(peerID, chat.ID)

			if err != nil {
				m.logger.Error("failed to sync chat", zap.Error(err))
				return nil, err
			}

			if m.config.messengerSignalsHandler != nil {
				m.config.messengerSignalsHandler.MessengerResponse(response)
			}
			return response, nil
		}, history.WithPeerID(peerID))
		if err != nil {
			m.logger.Error("failed to perform mailserver request", zap.Error(err))
		}
	}()
	return true, nil
}

func (m *Messenger) performStorenodeTask(task func() (*MessengerResponse, error), opts ...history.StorenodeTaskOption) (*MessengerResponse, error) {
	responseCh := make(chan *MessengerResponse, 1)
	err := m.transport.PerformStorenodeTask(func() error {
		r, err := task()
		if err != nil {
			return err
		}

		select {
		case <-m.ctx.Done():
			return m.ctx.Err()
		case responseCh <- r:
			return nil
		}
	}, opts...)
	if err != nil {
		return nil, err
	}

	select {
	case r := <-responseCh:
		if r != nil {
			return r, nil
		}
		return nil, errors.New("no response available")
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	}
}

func (m *Messenger) scheduleSyncFilters(filters []*transport.Filter) (bool, error) {
	shouldSync, err := m.shouldSync()
	if err != nil {
		m.logger.Error("failed to get shouldSync", zap.Error(err))
		return false, err
	}

	if !shouldSync {
		return false, nil
	}

	go func() {
		defer gocommon.LogOnPanic()
		// split filters by community store node so we can request the filters to the correct mailserver
		filtersByMs := m.SplitFiltersByStoreNode(filters)
		for communityID, filtersForMs := range filtersByMs {
			peerID := m.getCommunityStorenode(communityID)
			_, err := m.performStorenodeTask(func() (*MessengerResponse, error) {
				response, err := m.syncFilters(peerID, filtersForMs)

				if err != nil {
					m.logger.Error("failed to sync filter", zap.Error(err))
					return nil, err
				}

				if m.config.messengerSignalsHandler != nil {
					m.config.messengerSignalsHandler.MessengerResponse(response)
				}
				return response, nil
			}, history.WithPeerID(peerID))
			if err != nil {
				m.logger.Error("failed to perform mailserver request", zap.Error(err))
			}
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

func (m *Messenger) filtersForChat(chatID string) ([]*transport.Filter, error) {
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return nil, ErrChatNotFound
	}
	var filters []*transport.Filter

	if chat.OneToOne() {
		// We sync our own topic and any eventual negotiated
		publicKeys := []string{common.PubkeyToHex(&m.identity.PublicKey), chatID}

		filters = m.transport.FiltersByIdentities(publicKeys)

	} else if chat.PrivateGroupChat() {
		var publicKeys []string
		for _, m := range chat.Members {
			publicKeys = append(publicKeys, m.ID)
		}

		filters = m.transport.FiltersByIdentities(publicKeys)

	} else {
		filter := m.transport.FilterByChatID(chatID)
		if filter == nil {
			return nil, ErrNoFiltersForChat
		}
		filters = []*transport.Filter{filter}
	}

	return filters, nil
}

func (m *Messenger) topicsForChat(chatID string) (string, []types.TopicType, error) {
	filters, err := m.filtersForChat(chatID)
	if err != nil {
		return "", nil, err
	}

	var contentTopics []types.TopicType

	for _, filter := range filters {
		contentTopics = append(contentTopics, filter.ContentTopic)
	}

	return filters[0].PubsubTopic, contentTopics, nil
}

func (m *Messenger) syncChatWithFilters(peerID peer.ID, chatID string) (*MessengerResponse, error) {
	filters, err := m.filtersForChat(chatID)
	if err != nil {
		return nil, err
	}

	return m.syncFilters(peerID, filters)
}

func (m *Messenger) syncBackup() error {

	filter := m.transport.PersonalTopicFilter()
	if filter == nil {
		return errors.New("personal topic filter not loaded")
	}
	canSync, err := m.canSyncWithStoreNodes()
	if err != nil {
		return err
	}
	if !canSync {
		return nil
	}

	from, to := m.calculateMailserverTimeBounds(oneMonthDuration)

	batch := types.MailserverBatch{From: from, To: to, Topics: []types.TopicType{filter.ContentTopic}}
	ms := m.getCommunityStorenode(filter.ChatID)
	err = m.processMailserverBatch(ms, batch)
	if err != nil {
		return err
	}
	return m.settings.SetBackupFetched(true)
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

func (m *Messenger) updateFiltersPriority(filters []*transport.Filter) {
	for _, filter := range filters {
		chatID := filter.ChatID
		chat := m.Chat(chatID)
		if chat != nil {
			filter.Priority = chat.ReadMessagesAtClockValue
		}
	}
}

func (m *Messenger) resetFiltersPriority(filters []*transport.Filter) {
	for _, filter := range filters {
		filter.Priority = 0
	}
}

func (m *Messenger) SplitFiltersByStoreNode(filters []*transport.Filter) map[string][]*transport.Filter {
	// split filters by community store node so we can request the filters to the correct mailserver
	filtersByMs := make(map[string][]*transport.Filter, len(filters))
	for _, f := range filters {
		communityID := "" // none by default
		if chat, ok := m.allChats.Load(f.ChatID); ok && chat.CommunityChat() && m.communityStorenodes.HasStorenodeSetup(chat.CommunityID) {
			communityID = chat.CommunityID
		}
		if _, exists := filtersByMs[communityID]; !exists {
			filtersByMs[communityID] = make([]*transport.Filter, 0, len(filters))
		}
		filtersByMs[communityID] = append(filtersByMs[communityID], f)
	}
	return filtersByMs
}

// RequestAllHistoricMessages requests all the historic messages for any topic
func (m *Messenger) RequestAllHistoricMessages(forceFetchingBackup, withRetries bool) (*MessengerResponse, error) {
	shouldSync, err := m.shouldSync()
	if err != nil {
		return nil, err
	}

	if !shouldSync {
		return nil, nil
	}

	backupFetched, err := m.settings.BackupFetched()
	if err != nil {
		return nil, err
	}

	if m.mailserversDatabase == nil {
		return nil, nil
	}

	if forceFetchingBackup || !backupFetched {
		m.logger.Info("fetching backup")
		err := m.syncBackup()
		if err != nil {
			return nil, err
		}
		m.logger.Info("backup fetched")
	}

	filters := m.transport.Filters()
	m.updateFiltersPriority(filters)
	defer m.resetFiltersPriority(filters)

	filtersByMs := m.SplitFiltersByStoreNode(filters)
	allResponses := &MessengerResponse{}
	for communityID, filtersForMs := range filtersByMs {
		peerID := m.getCommunityStorenode(communityID)
		if withRetries {
			response, err := m.performStorenodeTask(func() (*MessengerResponse, error) {
				return m.syncFilters(peerID, filtersForMs)
			}, history.WithPeerID(peerID))
			if err != nil {
				return nil, err
			}
			if response != nil {
				allResponses.AddChats(response.Chats())
				allResponses.AddMessages(response.Messages())
			}
			continue
		}
		response, err := m.syncFilters(peerID, filtersForMs)
		if err != nil {
			return nil, err
		}
		if response != nil {
			allResponses.AddChats(response.Chats())
			allResponses.AddMessages(response.Messages())
		}
	}
	return allResponses, nil
}

const missingMessageCheckPeriod = 30 * time.Second

func (m *Messenger) checkForMissingMessagesLoop() {
	defer gocommon.LogOnPanic()

	if m.transport.WakuVersion() != 2 {
		return
	}

	t := time.NewTicker(missingMessageCheckPeriod)
	defer t.Stop()

	mailserverAvailableSignal := m.transport.OnStorenodeAvailable()

	for {
		select {
		case <-m.quit:
			return

		// Wait for mailserver available, also triggered on mailserver change
		case <-mailserverAvailableSignal:

		case <-t.C:

		}

		filters := m.transport.Filters()
		filtersByMs := m.SplitFiltersByStoreNode(filters)
		for communityID, filtersForMs := range filtersByMs {
			peerID := m.getCommunityStorenode(communityID)
			if peerID == "" {
				continue
			}

			m.transport.SetCriteriaForMissingMessageVerification(peerID, filtersForMs)
		}
	}
}

func getPrioritizedBatches() []int {
	return []int{1, 5, 10}
}

func (m *Messenger) syncFiltersFrom(peerID peer.ID, filters []*transport.Filter, lastRequest uint32) (*MessengerResponse, error) {
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

	topicsData := make(map[string]mailservers.MailserverTopic)
	for _, topic := range topicInfo {
		topicsData[fmt.Sprintf("%s-%s", topic.PubsubTopic, topic.ContentTopic)] = topic
	}

	batches := make(map[string]map[int]types.MailserverBatch)

	to := m.calculateMailserverTo()
	var syncedTopics []mailservers.MailserverTopic

	sort.Slice(filters[:], func(i, j int) bool {
		p1 := filters[i].Priority
		p2 := filters[j].Priority
		return p1 > p2
	})
	prioritizedBatches := getPrioritizedBatches()
	currentBatch := 0

	if len(filters) == 0 || filters[0].Priority == 0 {
		currentBatch = len(prioritizedBatches)
	}

	defaultPeriodFromNow, err := m.defaultSyncPeriodFromNow()
	if err != nil {
		return nil, err
	}

	contentTopicsPerPubsubTopic := make(map[string]map[string]*transport.Filter)
	for _, filter := range filters {
		if !filter.Listen || filter.Ephemeral {
			continue
		}

		contentTopics, ok := contentTopicsPerPubsubTopic[filter.PubsubTopic]
		if !ok {
			contentTopics = make(map[string]*transport.Filter)
		}
		contentTopics[filter.ContentTopic.String()] = filter
		contentTopicsPerPubsubTopic[filter.PubsubTopic] = contentTopics
	}

	for pubsubTopic, contentTopics := range contentTopicsPerPubsubTopic {
		if _, ok := batches[pubsubTopic]; !ok {
			batches[pubsubTopic] = make(map[int]types.MailserverBatch)
		}

		for _, filter := range contentTopics {
			var chatID string
			// If the filter has an identity, we use it as a chatID, otherwise is a public chat/community chat filter
			if len(filter.Identity) != 0 {
				chatID = filter.Identity
			} else {
				chatID = filter.ChatID
			}

			topicData, ok := topicsData[fmt.Sprintf("%s-%s", filter.PubsubTopic, filter.ContentTopic)]
			var capToDefaultSyncPeriod = true
			if !ok {
				if lastRequest == 0 {
					lastRequest = defaultPeriodFromNow
				}
				topicData = mailservers.MailserverTopic{
					PubsubTopic:  filter.PubsubTopic,
					ContentTopic: filter.ContentTopic.String(),
					LastRequest:  int(defaultPeriodFromNow),
				}
			} else if lastRequest != 0 {
				topicData.LastRequest = int(lastRequest)
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
				batch = types.MailserverBatch{From: time.Unix(int64(from), 0), To: to}
			}

			batch.ChatIDs = append(batch.ChatIDs, chatID)
			batch.PubsubTopic = pubsubTopic
			batch.Topics = append(batch.Topics, filter.ContentTopic)
			batches[pubsubTopic][batchID] = batch

			// Set last request to the new `to`
			topicData.LastRequest = int(to.Unix())
			syncedTopics = append(syncedTopics, topicData)
		}
	}

	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestStarted(len(batches))
	}

	var batches24h []types.MailserverBatch
	for pubsubTopic := range batches {
		batchKeys := make([]int, 0, len(batches[pubsubTopic]))
		for k := range batches[pubsubTopic] {
			batchKeys = append(batchKeys, k)
		}
		sort.Ints(batchKeys)

		keysToIterate := append([]int{}, batchKeys...)
		for {
			// For all batches
			var tmpKeysToIterate []int
			for _, k := range keysToIterate {
				batch := batches[pubsubTopic][k]

				dayBatch := types.MailserverBatch{
					To:          batch.To,
					Cursor:      batch.Cursor,
					PubsubTopic: batch.PubsubTopic,
					Topics:      batch.Topics,
					ChatIDs:     batch.ChatIDs,
				}

				from := batch.To.Add(-oneDayDuration)
				if from.After(batch.From) {
					dayBatch.From = from
					batches24h = append(batches24h, dayBatch)

					// Replace og batch with new dates
					batch.To = from
					batches[pubsubTopic][k] = batch
					tmpKeysToIterate = append(tmpKeysToIterate, k)
				} else {
					batches24h = append(batches24h, batch)
				}
			}

			if len(tmpKeysToIterate) == 0 {
				break
			}
			keysToIterate = tmpKeysToIterate
		}
	}

	for _, batch := range batches24h {
		err := m.processMailserverBatch(peerID, batch)
		if err != nil {
			m.logger.Error("error syncing topics", zap.Error(err))
			return nil, err
		}
	}

	m.logger.Debug("topics synced")
	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestCompleted()
	}

	err = m.mailserversDatabase.AddTopics(syncedTopics)
	if err != nil {
		return nil, err
	}

	var messagesToBeSaved []*common.Message
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
	return response, nil
}

func (m *Messenger) syncFilters(peerID peer.ID, filters []*transport.Filter) (*MessengerResponse, error) {
	return m.syncFiltersFrom(peerID, filters, 0)
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
		From:             common.PubkeyToHex(&m.identity.PublicKey),
		WhisperTimestamp: timestamp,
		LocalChatID:      chat.ID,
		Seen:             true,
		ID:               types.EncodeHex(crypto.Keccak256([]byte(fmt.Sprintf("%s-%d-%d", chat.ID, chat.SyncedTo, from)))),
	}

	return message, m.persistence.SaveMessages([]*common.Message{message})
}

func (m *Messenger) canSyncWithStoreNodes() (bool, error) {
	if m.featureFlags.StoreNodesDisabled {
		return false, nil
	}
	if m.connectionState.IsExpensive() {
		return m.settings.CanSyncOnMobileNetwork()
	}

	return true, nil
}

func (m *Messenger) DisableStoreNodes() {
	m.featureFlags.StoreNodesDisabled = true
}

func (m *Messenger) processMailserverBatch(peerID peer.ID, batch types.MailserverBatch) error {
	canSync, err := m.canSyncWithStoreNodes()
	if err != nil {
		return err
	}
	if !canSync {
		return nil
	}

	return m.transport.ProcessMailserverBatch(m.ctx, batch, peerID, defaultStoreNodeRequestPageSize, nil, false)
}

func (m *Messenger) processMailserverBatchWithOptions(peerID peer.ID, batch types.MailserverBatch, pageLimit uint64, shouldProcessNextPage func(int) (bool, uint64), processEnvelopes bool) error {
	canSync, err := m.canSyncWithStoreNodes()
	if err != nil {
		return err
	}
	if !canSync {
		return nil
	}

	return m.transport.ProcessMailserverBatch(m.ctx, batch, peerID, pageLimit, shouldProcessNextPage, processEnvelopes)
}

func (m *Messenger) SyncChatFromSyncedFrom(chatID string) (uint32, error) {
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return 0, ErrChatNotFound
	}

	peerID := m.getCommunityStorenode(chat.CommunityID)
	var from uint32
	_, err := m.performStorenodeTask(func() (*MessengerResponse, error) {
		canSync, err := m.canSyncWithStoreNodes()
		if err != nil {
			return nil, err
		}
		if !canSync {
			return nil, nil
		}

		pubsubTopic, topics, err := m.topicsForChat(chatID)
		if err != nil {
			return nil, nil
		}

		defaultSyncPeriod, err := m.settings.GetDefaultSyncPeriod()
		if err != nil {
			return nil, err
		}

		batch := types.MailserverBatch{
			ChatIDs:     []string{chatID},
			To:          time.Unix(int64(chat.SyncedFrom), 0),
			From:        time.Unix(int64(chat.SyncedFrom-defaultSyncPeriod), 0),
			PubsubTopic: pubsubTopic,
			Topics:      topics,
		}
		if m.config.messengerSignalsHandler != nil {
			m.config.messengerSignalsHandler.HistoryRequestStarted(1)
		}

		err = m.processMailserverBatch(peerID, batch)
		if err != nil {
			return nil, err
		}

		if m.config.messengerSignalsHandler != nil {
			m.config.messengerSignalsHandler.HistoryRequestCompleted()
		}
		if chat.SyncedFrom == 0 || chat.SyncedFrom > uint32(batch.From.Unix()) {
			chat.SyncedFrom = uint32(batch.From.Unix())
		}

		m.logger.Debug("setting sync timestamps", zap.Int64("from", batch.From.Unix()), zap.Int64("to", int64(chat.SyncedTo)), zap.String("chatID", chatID))

		err = m.persistence.SetSyncTimestamps(uint32(batch.From.Unix()), chat.SyncedTo, chat.ID)
		from = uint32(batch.From.Unix())
		return nil, err
	}, history.WithPeerID(peerID))
	if err != nil {
		return 0, err
	}

	return from, nil
}

func (m *Messenger) FillGaps(chatID string, messageIDs []string) error {
	messages, err := m.persistence.MessagesByIDs(messageIDs)
	if err != nil {
		return err
	}

	chat, ok := m.allChats.Load(chatID)
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

	batch := types.MailserverBatch{
		ChatIDs:     []string{chatID},
		To:          time.Unix(int64(highestTo), 0),
		From:        time.Unix(int64(lowestFrom), 0),
		PubsubTopic: pubsubTopic,
		Topics:      topics,
	}

	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestStarted(1)
	}

	peerID := m.getCommunityStorenode(chat.CommunityID)
	err = m.processMailserverBatch(peerID, batch)
	if err != nil {
		return err
	}

	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestCompleted()
	}

	return m.persistence.DeleteMessages(messageIDs)
}

func (m *Messenger) LoadFilters(filters []*transport.Filter) ([]*transport.Filter, error) {
	return m.transport.LoadFilters(filters)
}

func (m *Messenger) ToggleUseMailservers(value bool) error {
	err := m.settings.SetUseMailservers(value)
	if err != nil {
		return err
	}

	m.transport.DisconnectActiveStorenode(m.ctx, backoffByUserAction, value)

	return nil
}

func (m *Messenger) SetPinnedMailservers(mailservers map[string]string) error {
	err := m.settings.SetPinnedMailservers(mailservers)
	if err != nil {
		return err
	}

	m.transport.DisconnectActiveStorenode(m.ctx, backoffByUserAction, true)

	return nil
}

func (m *Messenger) RemoveFilters(filters []*transport.Filter) error {
	return m.transport.RemoveFilters(filters)
}

func (m *Messenger) ConnectionChanged(state connection.State) {
	m.transport.ConnectionChanged(state)
	if !m.connectionState.Offline && state.Offline {
		m.sender.StopDatasync()
	}

	if m.connectionState.Offline && !state.Offline {
		err := m.sender.StartDatasync(m.mvdsStatusChangeEvent, m.sendDataSync)
		if err != nil {
			m.logger.Error("failed to start datasync", zap.Error(err))
		}
	}

	m.connectionState = state
}

func (m *Messenger) fetchMessages(chatID string, duration time.Duration) (uint32, error) {
	from, to := m.calculateMailserverTimeBounds(duration)

	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return 0, ErrChatNotFound
	}

	peerID := m.getCommunityStorenode(chat.CommunityID)
	_, err := m.performStorenodeTask(func() (*MessengerResponse, error) {
		canSync, err := m.canSyncWithStoreNodes()
		if err != nil {
			return nil, err
		}
		if !canSync {
			return nil, nil
		}

		m.logger.Debug("fetching messages", zap.String("chatID", chatID), zap.Stringer("peerID", peerID))
		pubsubTopic, topics, err := m.topicsForChat(chatID)
		if err != nil {
			return nil, nil
		}

		batch := types.MailserverBatch{
			ChatIDs:     []string{chatID},
			From:        from,
			To:          to,
			PubsubTopic: pubsubTopic,
			Topics:      topics,
		}
		if m.config.messengerSignalsHandler != nil {
			m.config.messengerSignalsHandler.HistoryRequestStarted(1)
		}

		err = m.processMailserverBatch(peerID, batch)
		if err != nil {
			return nil, err
		}

		if m.config.messengerSignalsHandler != nil {
			m.config.messengerSignalsHandler.HistoryRequestCompleted()
		}
		if chat.SyncedFrom == 0 || chat.SyncedFrom > uint32(batch.From.Second()) {
			chat.SyncedFrom = uint32(batch.From.Second())
		}

		m.logger.Debug("setting sync timestamps", zap.Int64("from", batch.From.Unix()), zap.Int64("to", int64(chat.SyncedTo)), zap.String("chatID", chatID))

		err = m.persistence.SetSyncTimestamps(uint32(batch.From.Unix()), chat.SyncedTo, chat.ID)
		from = batch.From
		return nil, err
	}, history.WithPeerID(peerID))
	if err != nil {
		return 0, err
	}

	return uint32(from.Unix()), nil
}
