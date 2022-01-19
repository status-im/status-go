package protocol

import (
	"context"
	"fmt"
	"time"

	"github.com/pborman/uuid"
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
var mailserverRequestTimeout = 45 * time.Second
var oneMonthInSeconds uint32 = 31 * 24 * 60 * 60

var ErrNoFiltersForChat = errors.New("no filter registered for given chat")

func (m *Messenger) shouldSync() (bool, error) {
	if m.mailserver == nil || !m.online() {
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
		response, err := m.syncChat(chat.ID)

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

func (m *Messenger) scheduleSyncFilter(filter *transport.Filter) {
	_, err := m.scheduleSyncFilters([]*transport.Filter{filter})
	if err != nil {
		m.logger.Error("failed to schedule syncing filters", zap.Error(err))
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
	// hardcoding 31 days just for testing purposes
	// defaultSyncPeriod, err := m.settings.GetDefaultSyncPeriod()
	defaultSyncPeriod := uint32(31 * 86400)
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

// RequestAllHistoricMessages requests all the historic messages for any topic
func (m *Messenger) RequestAllHistoricMessages() error {
	shouldSync, err := m.shouldSync()
	if err != nil {
		return err
	}

	if !shouldSync {
		return nil
	}

	backupFetched, err := m.settings.BackupFetched()
	if err != nil {
		return err
	}

	if !backupFetched {
		m.logger.Info("fetching backup")
		err := m.syncBackup()
		if err != nil {
			return err
		}
		m.logger.Info("backup fetched")
	}

	go func() {
		m.syncFilters(m.transport.Filters())
	}()

	return nil
}

func (m *Messenger) syncFilters(filters []*transport.Filter) (*MessengerResponse, error) {
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
		if !ok {
			lastRequest, err := m.defaultSyncPeriodFromNow()
			if err != nil {
				return nil, err
			}
			topicData = mailservers.MailserverTopic{
				Topic:       filter.Topic.String(),
				LastRequest: int(lastRequest),
			}
		}

		batch, ok := batches[topicData.LastRequest]
		if !ok {
			from, err := m.capToDefaultSyncPeriod(uint32(topicData.LastRequest))
			if err != nil {
				return nil, err
			}

			batch = MailserverBatch{From: from, To: to}
		}

		batch.ChatIDs = append(batch.ChatIDs, chatID)
		batch.Topics = append(batch.Topics, filter.Topic)
		batches[topicData.LastRequest] = batch
		// Set last request to the new `to`
		topicData.LastRequest = int(to)
		syncedTopics = append(syncedTopics, topicData)
	}

	requestID := uuid.NewRandom().String()

	m.logger.Debug("syncing topics", zap.Any("batches", batches), zap.Any("requestId", requestID))

	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestStarted(requestID, len(batches))
	}

	i := 0
	for _, batch := range batches {
		i++
		err := m.processMailserverBatch(batch)
		if err != nil {
			m.logger.Error("error syncing topics", zap.Any("requestId", requestID), zap.Error(err))
			if m.config.messengerSignalsHandler != nil {
				m.config.messengerSignalsHandler.HistoryRequestFailed(requestID, err)
			}
			return nil, err
		}

		if m.config.messengerSignalsHandler != nil {
			m.config.messengerSignalsHandler.HistoryRequestBatchProcessed(requestID, i, len(batches))
		}
	}

	m.logger.Debug("topics synced")
	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestCompleted(requestID)
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

	// TODO: should these messages be returned as signals instead?
	return response, nil
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

func (m *Messenger) processMailserverBatch(batch MailserverBatch) error {
	fmt.Println("======================================================")
	fmt.Println("Processing batch, ", batch.From, batch.To, batch.Topics)

	to := batch.To

	for {
		from := to - 86400
		if from < batch.From {
			from = batch.From
		}

		fmt.Println("- Requesting from: ", from, " to: ", to)

		var topicStrings []string
		for _, t := range batch.Topics {
			topicStrings = append(topicStrings, t.String())
		}
		logger := m.logger.With(zap.Any("chatIDs", batch.ChatIDs), zap.String("fromString", time.Unix(int64(from), 0).Format(time.RFC3339)), zap.String("toString", time.Unix(int64(to), 0).Format(time.RFC3339)), zap.Any("topic", topicStrings), zap.Int64("from", int64(from)), zap.Int64("to", int64(to)))
		logger.Info("syncing topic")
		ctx, cancel := context.WithTimeout(context.Background(), mailserverRequestTimeout)
		defer cancel()

		cursor, storeCursor, err := m.transport.SendMessagesRequestForTopics(ctx, m.mailserver, from, to, nil, nil, batch.Topics, true)
		if err != nil {
			return err
		}

		for len(cursor) != 0 || storeCursor != nil {
			logger.Info("retrieved cursor", zap.String("cursor", types.EncodeHex(cursor)))
			ctx, cancel := context.WithTimeout(context.Background(), mailserverRequestTimeout)
			defer cancel()

			cursor, storeCursor, err = m.transport.SendMessagesRequestForTopics(ctx, m.mailserver, from, to, cursor, storeCursor, batch.Topics, true)
			if err != nil {
				return err
			}
		}
		logger.Info("waiting until message processed")
		m.waitUntilP2PMessagesProcessed()
		logger.Info("synced topic")

		to -= 86400
		if to < batch.From {
			break // We're done!
		}
	}
	fmt.Println("Batch processed!")
	return nil
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
	if m.mailserver == nil {
		return nil, nil, errors.New("no mailserver selected")
	}

	return m.transport.SendMessagesRequestForFilter(ctx, m.mailserver, from, to, cursor, previousStoreCursor, filter, waitForResponse)
}

func (m *Messenger) SyncChatFromSyncedFrom(chatID string) (uint32, error) {
	topics, err := m.topicsForChat(chatID)
	if err != nil {
		return 0, nil
	}
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return 0, ErrChatNotFound
	}

	defaultSyncPeriod, err := m.settings.GetDefaultSyncPeriod()
	if err != nil {
		return 0, err
	}
	batch := MailserverBatch{
		ChatIDs: []string{chatID},
		To:      chat.SyncedFrom,
		From:    chat.SyncedFrom - defaultSyncPeriod,
		Topics:  topics,
	}

	requestID := uuid.NewRandom().String()

	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestStarted(requestID, 1)
	}

	err = m.processMailserverBatch(batch)
	if err != nil {
		if m.config.messengerSignalsHandler != nil {
			m.config.messengerSignalsHandler.HistoryRequestFailed(requestID, err)
		}
		return 0, err
	}

	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestBatchProcessed(requestID, 1, 1)
		m.config.messengerSignalsHandler.HistoryRequestCompleted(requestID)
	}

	if chat.SyncedFrom == 0 || chat.SyncedFrom > batch.From {
		chat.SyncedFrom = batch.From
	}

	m.logger.Debug("setting sync timestamps", zap.Int64("from", int64(batch.From)), zap.Int64("to", int64(chat.SyncedTo)), zap.String("chatID", chatID))

	err = m.persistence.SetSyncTimestamps(batch.From, chat.SyncedTo, chat.ID)
	if err != nil {
		return 0, err
	}

	return batch.From, nil
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

	requestID := uuid.NewRandom().String()

	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestStarted(requestID, 1)
	}

	err = m.processMailserverBatch(batch)
	if err != nil {
		if m.config.messengerSignalsHandler != nil {
			m.config.messengerSignalsHandler.HistoryRequestFailed(requestID, err)
		}
		return err
	}

	if m.config.messengerSignalsHandler != nil {
		m.config.messengerSignalsHandler.HistoryRequestBatchProcessed(requestID, 1, 1)
		m.config.messengerSignalsHandler.HistoryRequestCompleted(requestID)
	}

	return m.persistence.DeleteMessages(messageIDs)
}

func (m *Messenger) waitUntilP2PMessagesProcessed() {

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

func (m *Messenger) RemoveFilters(filters []*transport.Filter) error {
	return m.transport.RemoveFilters(filters)
}

func (m *Messenger) ConnectionChanged(state connection.State) {
	if !m.connectionState.Offline && state.Offline {
		m.sender.StopDatasync()
	}

	if m.connectionState.Offline && !state.Offline {
		m.sender.StartDatasync()
	}

	m.connectionState = state
}
