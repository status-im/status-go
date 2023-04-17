package protocol

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/connection"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/services/mailservers"
)

// tolerance is how many seconds of potentially out-of-order messages we want to fetch
var tolerance uint32 = 60

var mailserverRequestTimeout = 30 * time.Second
var oneMonthInSeconds uint32 = 31 * 24 * 60 * 60
var mailserverMaxTries uint = 2
var mailserverMaxFailedRequests uint = 2

// maxTopicsPerRequest sets the batch size to limit the number of topics per store query
var maxTopicsPerRequest int = 10

var ErrNoFiltersForChat = errors.New("no filter registered for given chat")

func (m *Messenger) shouldSync() (bool, error) {
	if m.mailserverCycle.activeMailserver == nil || !m.online() {
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

	if !m.connectionState.IsExpensive() {
		return true, nil
	}

	syncingOnMobileNetwork, err := m.settings.CanSyncOnMobileNetwork()
	if err != nil {
		return false, err
	}

	return syncingOnMobileNetwork, nil
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
		_, err := m.performMailserverRequest(func() (*MessengerResponse, error) {
			response, err := m.syncChat(chat.ID)

			if err != nil {
				m.logger.Error("failed to sync chat", zap.Error(err))
				return nil, err
			}

			if m.config.messengerSignalsHandler != nil {
				m.config.messengerSignalsHandler.MessengerResponse(response)
			}
			return response, nil
		})
		if err != nil {
			m.logger.Error("failed to perform mailserver request", zap.Error(err))
		}
	}()
	return true, nil
}

func (m *Messenger) connectToNewMailserverAndWait() error {
	// Handle pinned mailservers
	m.logger.Info("disconnecting mailserver")
	pinnedMailserver, err := m.getPinnedMailserver()
	if err != nil {
		m.logger.Error("could not obtain the pinned mailserver", zap.Error(err))
		return err
	}
	// If pinned mailserver is not nil, no need to disconnect and wait for it to be available
	if pinnedMailserver == nil {
		m.disconnectActiveMailserver()
	}

	return m.findNewMailserver()
}

func (m *Messenger) performMailserverRequest(fn func() (*MessengerResponse, error)) (*MessengerResponse, error) {

	m.mailserverCycle.Lock()
	defer m.mailserverCycle.Unlock()
	var tries uint = 0
	for tries < mailserverMaxTries {
		if !m.isActiveMailserverAvailable() {
			return nil, errors.New("mailserver not available")
		}

		m.logger.Info("trying performing mailserver requests", zap.Uint("try", tries))
		activeMailserver := m.getActiveMailserver()
		// Make sure we are connected to a mailserver
		if activeMailserver == nil {
			return nil, errors.New("mailserver not available")
		}

		// Peform request
		response, err := fn()
		if err == nil {
			// Reset failed requests
			activeMailserver.FailedRequests = 0
			return response, nil
		}

		tries++
		// Increment failed requests
		activeMailserver.FailedRequests++

		// Change mailserver
		if activeMailserver.FailedRequests >= mailserverMaxFailedRequests {
			return nil, errors.New("too many failed requests")
		}
		// Wait a couple of second not to spam
		time.Sleep(2 * time.Second)

	}
	return nil, errors.New("failed to perform mailserver request")
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
		_, err := m.performMailserverRequest(func() (*MessengerResponse, error) {
			response, err := m.syncFilters(filters)

			if err != nil {
				m.logger.Error("failed to sync filter", zap.Error(err))
				return nil, err
			}

			if m.config.messengerSignalsHandler != nil {
				m.config.messengerSignalsHandler.MessengerResponse(response)
			}
			return response, nil
		})
		if err != nil {
			m.logger.Error("failed to perform mailserver request", zap.Error(err))
		}

	}()
	return true, nil
}

func (m *Messenger) calculateMailserverTo() uint32 {
	return uint32(m.getTimesource().GetCurrentTime() / 1000)
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

func (m *Messenger) topicsForChat(chatID string) ([]types.TopicType, error) {
	filters, err := m.filtersForChat(chatID)
	if err != nil {
		return nil, err
	}

	var topics []types.TopicType

	for _, filter := range filters {
		topics = append(topics, filter.Topic)
	}

	return topics, nil
}

// Assume is a public chat for now
func (m *Messenger) syncChat(chatID string) (*MessengerResponse, error) {
	filters, err := m.filtersForChat(chatID)
	if err != nil {
		return nil, err
	}
	return m.syncFilters(filters)
}

func (m *Messenger) syncBackup() error {

	filter := m.transport.PersonalTopicFilter()
	if filter == nil {
		return errors.New("personal topic filter not loaded")
	}

	to := m.calculateMailserverTo()
	from := uint32(m.getTimesource().GetCurrentTime()/1000) - oneMonthInSeconds
	batch := MailserverBatch{From: from, To: to, Topics: []types.TopicType{filter.Topic}}
	err := m.processMailserverBatch(batch)
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

func (m *Messenger) RequestAllHistoricMessagesWithRetries() (*MessengerResponse, error) {
	return m.performMailserverRequest(m.RequestAllHistoricMessages)
}

// RequestAllHistoricMessages requests all the historic messages for any topic
func (m *Messenger) RequestAllHistoricMessages() (*MessengerResponse, error) {
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

	if !backupFetched {
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
	response, err := m.syncFilters(filters)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func getPrioritizedBatches() []int {
	return []int{1, 5, 10}
}

func (m *Messenger) syncFiltersFrom(filters []*transport.Filter, lastRequest uint32) (*MessengerResponse, error) {
	response := &MessengerResponse{}
	topicInfo, err := m.mailserversDatabase.Topics()
	if err != nil {
		return nil, err
	}

	topicsData := make(map[string]mailservers.MailserverTopic)
	for _, topic := range topicInfo {
		topicsData[topic.Topic] = topic
	}

	batches := make(map[int]MailserverBatch)

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

	for _, filter := range filters {
		if !filter.Listen || filter.Ephemeral {
			continue
		}

		var chatID string
		// If the filter has an identity, we use it as a chatID, otherwise is a public chat/community chat filter
		if len(filter.Identity) != 0 {
			chatID = filter.Identity
		} else {
			chatID = filter.ChatID
		}

		topicData, ok := topicsData[filter.Topic.String()]
		var capToDefaultSyncPeriod = true
		if !ok {
			if lastRequest == 0 {
				lastRequest = defaultPeriodFromNow
			}
			topicData = mailservers.MailserverTopic{
				Topic:       filter.Topic.String(),
				LastRequest: int(defaultPeriodFromNow),
			}
		} else if lastRequest != 0 {
			topicData.LastRequest = int(lastRequest)
			capToDefaultSyncPeriod = false
		}

		batchID := topicData.LastRequest

		if currentBatch < len(prioritizedBatches) {
			batch, ok := batches[currentBatch]
			if ok {
				prevTopicData, ok := topicsData[batch.Topics[0].String()]
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

		batch, ok := batches[batchID]
		if !ok {
			from := uint32(topicData.LastRequest)
			if capToDefaultSyncPeriod {
				from, err = m.capToDefaultSyncPeriod(uint32(topicData.LastRequest))
				if err != nil {
					return nil, err
				}
			}
			batch = MailserverBatch{From: from, To: to}
		}

		batch.ChatIDs = append(batch.ChatIDs, chatID)
		batch.Topics = append(batch.Topics, filter.Topic)
		batches[batchID] = batch
		// Set last request to the new `to`
		topicData.LastRequest = int(to)
		syncedTopics = append(syncedTopics, topicData)
	}

	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestStarted(len(batches))
	}

	batchKeys := make([]int, 0, len(batches))
	for k := range batches {
		batchKeys = append(batchKeys, k)
	}
	sort.Ints(batchKeys)

	var batches24h []MailserverBatch
	keysToIterate := append([]int{}, batchKeys...)
	for {
		// For all batches
		var tmpKeysToIterate []int
		for _, k := range keysToIterate {
			batch := batches[k]

			dayBatch := MailserverBatch{
				To:      batch.To,
				Cursor:  batch.Cursor,
				Topics:  batch.Topics,
				ChatIDs: batch.ChatIDs,
			}

			from := batch.To - 86400
			if from > batch.From {
				dayBatch.From = from
				batches24h = append(batches24h, dayBatch)

				// Replace og batch with new dates
				batch.To = from
				batches[k] = batch
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

	i := 0
	for _, batch := range batches24h {
		i++
		err := m.processMailserverBatch(batch)
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
	for _, batch := range batches {
		for _, id := range batch.ChatIDs {
			chat, ok := m.allChats.Load(id)
			if !ok || !chat.Active || chat.Timeline() || chat.ProfileUpdates() {
				continue
			}
			gap, err := m.calculateGapForChat(chat, batch.From)
			if err != nil {
				return nil, err
			}
			if chat.SyncedFrom == 0 || chat.SyncedFrom > batch.From {
				chat.SyncedFrom = batch.From
			}

			chat.SyncedTo = to

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

	if len(messagesToBeSaved) > 0 {
		err := m.persistence.SaveMessages(messagesToBeSaved)
		if err != nil {
			return nil, err
		}
	}
	return response, nil
}

func (m *Messenger) syncFilters(filters []*transport.Filter) (*MessengerResponse, error) {
	return m.syncFiltersFrom(filters, 0)
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
		ChatMessage: protobuf.ChatMessage{
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

type work struct {
	topics      []types.TopicType
	cursor      []byte
	storeCursor *types.StoreRequestCursor
}

type messageRequester interface {
	SendMessagesRequestForTopics(
		ctx context.Context,
		peerID []byte,
		from, to uint32,
		previousCursor []byte,
		previousStoreCursor *types.StoreRequestCursor,
		topics []types.TopicType,
		waitForResponse bool,
	) (cursor []byte, storeCursor *types.StoreRequestCursor, err error)
}

func processMailserverBatch(ctx context.Context, messageRequester messageRequester, batch MailserverBatch, mailserverID []byte, logger *zap.Logger) error {
	var topicStrings []string
	for _, t := range batch.Topics {
		topicStrings = append(topicStrings, t.String())
	}
	logger = logger.With(zap.Any("chatIDs", batch.ChatIDs), zap.String("fromString", time.Unix(int64(batch.From), 0).Format(time.RFC3339)), zap.String("toString", time.Unix(int64(batch.To), 0).Format(time.RFC3339)), zap.Any("topic", topicStrings), zap.Int64("from", int64(batch.From)), zap.Int64("to", int64(batch.To)))
	logger.Info("syncing topic")

	wg := sync.WaitGroup{}
	workCh := make(chan work, 100)
	errCh := make(chan error)

	go func() {
		defer func() {
			close(errCh)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case w, ok := <-workCh:
				if !ok {
					return
				}

				ctx, cancel := context.WithTimeout(ctx, mailserverRequestTimeout)
				cursor, storeCursor, err := messageRequester.SendMessagesRequestForTopics(ctx, mailserverID, batch.From, batch.To, w.cursor, w.storeCursor, w.topics, true)
				if err != nil {
					logger.Error("failed to send request", zap.Error(err))
					wg.Done()
					cancel()
					errCh <- err
					return
				}

				cancel()

				if len(cursor) != 0 || storeCursor != nil {
					logger.Info("query has cursor", zap.String("cursorV1", types.EncodeHex(cursor)))
					workCh <- work{
						topics:      w.topics,
						cursor:      cursor,
						storeCursor: storeCursor,
					}
				} else {
					wg.Done() // We are done with this topic and its cursor
				}

			}
		}
	}()

	for i := 0; i < len(batch.Topics); i += maxTopicsPerRequest {
		j := i + maxTopicsPerRequest
		if j > len(batch.Topics) {
			j = len(batch.Topics)
		}

		wg.Add(1)
		workCh <- work{
			topics: batch.Topics[i:j],
		}
	}

	wg.Wait()
	close(workCh)

	// NOTE(camellos): Disabling for now, not critical and I'd rather take a bit more time
	// to test it
	//logger.Info("waiting until message processed")
	//m.waitUntilP2PMessagesProcessed()

	result := <-errCh

	logger.Info("synced topic", zap.NamedError("hasError", result))
	return result
}

func (m *Messenger) processMailserverBatch(batch MailserverBatch) error {
	mailserverID, err := m.activeMailserverID()
	if err != nil {
		return err
	}

	return processMailserverBatch(m.ctx, m.transport, batch, mailserverID, m.logger)
}

type MailserverBatch struct {
	From    uint32
	To      uint32
	Cursor  string
	Topics  []types.TopicType
	ChatIDs []string
}

func (m *Messenger) RequestHistoricMessagesForFilter(
	ctx context.Context,
	from, to uint32,
	cursor []byte,
	previousStoreCursor *types.StoreRequestCursor,
	filter *transport.Filter,
	waitForResponse bool,
) ([]byte, *types.StoreRequestCursor, error) {

	activeMailserverID, err := m.activeMailserverID()
	if err != nil {
		return nil, nil, err
	}

	if activeMailserverID == nil {
		m.cycleMailservers()
		activeMailserverID, err = m.activeMailserverID()
		if err != nil {
			return nil, nil, err
		}
		if activeMailserverID == nil {
			return nil, nil, errors.New("no mailserver selected")
		}
	}

	return m.transport.SendMessagesRequestForFilter(ctx, activeMailserverID, from, to, cursor, previousStoreCursor, filter, waitForResponse)
}

func (m *Messenger) SyncChatFromSyncedFrom(chatID string) (uint32, error) {
	var from uint32
	_, err := m.performMailserverRequest(func() (*MessengerResponse, error) {
		topics, err := m.topicsForChat(chatID)
		if err != nil {
			return nil, nil
		}

		chat, ok := m.allChats.Load(chatID)
		if !ok {
			return nil, ErrChatNotFound
		}

		defaultSyncPeriod, err := m.settings.GetDefaultSyncPeriod()
		if err != nil {
			return nil, err
		}
		batch := MailserverBatch{
			ChatIDs: []string{chatID},
			To:      chat.SyncedFrom,
			From:    chat.SyncedFrom - defaultSyncPeriod,
			Topics:  topics,
		}
		if m.config.messengerSignalsHandler != nil {
			m.config.messengerSignalsHandler.HistoryRequestStarted(1)
		}

		err = m.processMailserverBatch(batch)
		if err != nil {
			return nil, err
		}

		if m.config.messengerSignalsHandler != nil {
			m.config.messengerSignalsHandler.HistoryRequestCompleted()
		}
		if chat.SyncedFrom == 0 || chat.SyncedFrom > batch.From {
			chat.SyncedFrom = batch.From
		}

		m.logger.Debug("setting sync timestamps", zap.Int64("from", int64(batch.From)), zap.Int64("to", int64(chat.SyncedTo)), zap.String("chatID", chatID))

		err = m.persistence.SetSyncTimestamps(batch.From, chat.SyncedTo, chat.ID)
		from = batch.From
		return nil, err
	})
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

	_, ok := m.allChats.Load(chatID)
	if !ok {
		return errors.New("chat not existing")
	}

	topics, err := m.topicsForChat(chatID)
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

	batch := MailserverBatch{
		ChatIDs: []string{chatID},
		To:      highestTo,
		From:    lowestFrom,
		Topics:  topics,
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

func (m *Messenger) waitUntilP2PMessagesProcessed() { // nolint: unused

	ticker := time.NewTicker(50 * time.Millisecond)

	for { //nolint: gosimple
		select {
		case <-ticker.C:
			if !m.transport.ProcessingP2PMessages() {
				ticker.Stop()
				return
			}
		}
	}
}

func (m *Messenger) LoadFilters(filters []*transport.Filter) ([]*transport.Filter, error) {
	return m.transport.LoadFilters(filters)
}

func (m *Messenger) ToggleUseMailservers(value bool) error {
	m.mailserverCycle.Lock()
	defer m.mailserverCycle.Unlock()

	err := m.settings.SetUseMailservers(value)
	if err != nil {
		return err
	}

	if value {
		m.cycleMailservers()
		return nil
	}

	m.disconnectActiveMailserver()
	return nil
}

func (m *Messenger) SetPinnedMailservers(mailservers map[string]string) error {
	err := m.settings.SetPinnedMailservers(mailservers)
	if err != nil {
		return err
	}

	m.cycleMailservers()
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
		m.sender.StartDatasync()
	}

	m.connectionState = state
}
