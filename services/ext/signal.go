package ext

import (
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/discord"
	"github.com/status-im/status-go/signal"
)

// EnvelopeSignalHandler sends signals when envelope is sent or expired.
type EnvelopeSignalHandler struct{}

// EnvelopeSent triggered when envelope delivered atleast to 1 peer.
func (h EnvelopeSignalHandler) EnvelopeSent(identifiers [][]byte) {
	signal.SendEnvelopeSent(identifiers)
}

// EnvelopeExpired triggered when envelope is expired but wasn't delivered to any peer.
func (h EnvelopeSignalHandler) EnvelopeExpired(identifiers [][]byte, err error) {
	signal.SendEnvelopeExpired(identifiers, err)
}

// MailServerRequestCompleted triggered when the mailserver sends a message to notify that the request has been completed
func (h EnvelopeSignalHandler) MailServerRequestCompleted(requestID types.Hash, lastEnvelopeHash types.Hash, cursor []byte, err error) {
	signal.SendMailServerRequestCompleted(requestID, lastEnvelopeHash, cursor, err)
}

// MailServerRequestExpired triggered when the mailserver request expires
func (h EnvelopeSignalHandler) MailServerRequestExpired(hash types.Hash) {
	signal.SendMailServerRequestExpired(hash)
}

// PublisherSignalHandler sends signals on protocol events
type PublisherSignalHandler struct{}

func (h PublisherSignalHandler) DecryptMessageFailed(pubKey string) {
	signal.SendDecryptMessageFailed(pubKey)
}

func (h PublisherSignalHandler) BundleAdded(identity string, installationID string) {
	signal.SendBundleAdded(identity, installationID)
}

func (h PublisherSignalHandler) NewMessages(response *protocol.MessengerResponse) {
	signal.SendNewMessages(response)
}

func (h PublisherSignalHandler) Stats(stats types.StatsSummary) {
	signal.SendStats(stats)
}

// MessengerSignalHandler sends signals on messenger events
type MessengerSignalsHandler struct{}

// MessageDelivered passes information that message was delivered
func (m MessengerSignalsHandler) MessageDelivered(chatID string, messageID string) {
	signal.SendMessageDelivered(chatID, messageID)
}

// BackupPerformed passes information that a backup was performed
func (m MessengerSignalsHandler) BackupPerformed(lastBackup uint64) {
	signal.SendBackupPerformed(lastBackup)
}

// MessageDelivered passes info about community that was requested before
func (m MessengerSignalsHandler) CommunityInfoFound(community *communities.Community) {
	signal.SendCommunityInfoFound(community)
}

func (m *MessengerSignalsHandler) MessengerResponse(response *protocol.MessengerResponse) {
	PublisherSignalHandler{}.NewMessages(response)
}

func (m *MessengerSignalsHandler) HistoryRequestStarted(requestID string, numBatches int) {
	signal.SendHistoricMessagesRequestStarted(requestID, numBatches)
}

func (m *MessengerSignalsHandler) HistoryRequestBatchProcessed(requestID string, batchIndex int, numBatches int) {
	signal.SendHistoricMessagesRequestBatchProcessed(requestID, batchIndex, numBatches)
}

func (m *MessengerSignalsHandler) HistoryRequestFailed(requestID string, err error) {
	signal.SendHistoricMessagesRequestFailed(requestID, err)
}

func (m *MessengerSignalsHandler) HistoryRequestCompleted(requestID string) {
	signal.SendHistoricMessagesRequestCompleted(requestID)
}

func (m *MessengerSignalsHandler) HistoryArchivesProtocolEnabled() {
	signal.SendHistoryArchivesProtocolEnabled()
}

func (m *MessengerSignalsHandler) HistoryArchivesProtocolDisabled() {
	signal.SendHistoryArchivesProtocolDisabled()
}

func (m *MessengerSignalsHandler) CreatingHistoryArchives(communityID string) {
	signal.SendCreatingHistoryArchives(communityID)
}

func (m *MessengerSignalsHandler) NoHistoryArchivesCreated(communityID string, from int, to int) {
	signal.SendNoHistoryArchivesCreated(communityID, from, to)
}

func (m *MessengerSignalsHandler) HistoryArchivesCreated(communityID string, from int, to int) {
	signal.SendHistoryArchivesCreated(communityID, from, to)
}

func (m *MessengerSignalsHandler) HistoryArchivesSeeding(communityID string) {
	signal.SendHistoryArchivesSeeding(communityID)
}

func (m *MessengerSignalsHandler) HistoryArchivesUnseeded(communityID string) {
	signal.SendHistoryArchivesUnseeded(communityID)
}

func (m *MessengerSignalsHandler) HistoryArchiveDownloaded(communityID string, from int, to int) {
	signal.SendHistoryArchiveDownloaded(communityID, from, to)
}

func (m *MessengerSignalsHandler) StatusUpdatesTimedOut(statusUpdates *[]protocol.UserStatus) {
	signal.SendStatusUpdatesTimedOut(statusUpdates)
}

func (m *MessengerSignalsHandler) DiscordCategoriesAndChannelsExtracted(categories []*discord.Category, channels []*discord.Channel, oldestMessageTimestamp int64, errors map[string]*discord.ImportError) {
	signal.SendDiscordCategoriesAndChannelsExtracted(categories, channels, oldestMessageTimestamp, errors)
}
