package ext

import (
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/protocol"
	"github.com/status-im/status-go/internal/protocol/backupsync"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/discord"
	"github.com/status-im/status-go/internal/signal"
)

// EnvelopeSignalHandler sends signals when envelope is sent or expired, in the backend
// session it was created in.
type EnvelopeSignalHandler struct {
	session signal.Session
}

// EnvelopeSent triggered when envelope delivered at least to 1 peer.
func (h EnvelopeSignalHandler) EnvelopeSent(identifiers [][]byte) {
	h.session.SendEnvelopeSent(identifiers)
}

// EnvelopeExpired triggered when envelope is expired but wasn't delivered to any peer.
func (h EnvelopeSignalHandler) EnvelopeExpired(identifiers [][]byte, err error) {
	h.session.SendEnvelopeExpired(identifiers, err)
}

// MailServerRequestCompleted triggered when the mailserver sends a message to notify that the request has been completed
func (h EnvelopeSignalHandler) MailServerRequestCompleted(requestID types.Hash, lastEnvelopeHash types.Hash, cursor []byte, err error) {
	h.session.SendMailServerRequestCompleted(requestID, lastEnvelopeHash, cursor, err)
}

// MailServerRequestExpired triggered when the mailserver request expires
func (h EnvelopeSignalHandler) MailServerRequestExpired(hash types.Hash) {
	h.session.SendMailServerRequestExpired(hash)
}

// MessengerSignalsHandler sends signals on messenger events, in the backend session it
// was created in: messenger work still running after a logout must not reach the client
// as the next session's.
type MessengerSignalsHandler struct {
	session signal.Session
}

// MessageDelivered passes information that message was delivered
func (m *MessengerSignalsHandler) MessageDelivered(chatID string, messageID string) {
	m.session.SendMessageDelivered(chatID, messageID)
}

// CommunityInfoFound passes info about community that was requested before
func (m *MessengerSignalsHandler) CommunityInfoFound(community *communities.Community) {
	m.session.SendCommunityInfoFound(community)
}

func (m *MessengerSignalsHandler) MessengerResponse(response *protocol.MessengerResponse) {
	m.session.SendNewMessages(response)
}

func (m *MessengerSignalsHandler) HistoryRequestStarted(numBatches int) {
	m.session.SendHistoricMessagesRequestStarted(numBatches)
}

func (m *MessengerSignalsHandler) HistoryRequestCompleted() {
	m.session.SendHistoricMessagesRequestCompleted()
}

func (m *MessengerSignalsHandler) HistoryArchivesProtocolEnabled() {
	m.session.SendHistoryArchivesProtocolEnabled()
}

func (m *MessengerSignalsHandler) HistoryArchivesProtocolDisabled() {
	m.session.SendHistoryArchivesProtocolDisabled()
}

func (m *MessengerSignalsHandler) CreatingHistoryArchives(communityID string) {
	m.session.SendCreatingHistoryArchives(communityID)
}

func (m *MessengerSignalsHandler) NoHistoryArchivesCreated(communityID string, from int, to int) {
	m.session.SendNoHistoryArchivesCreated(communityID, from, to)
}

func (m *MessengerSignalsHandler) HistoryArchivesCreated(communityID string, from int, to int) {
	m.session.SendHistoryArchivesCreated(communityID, from, to)
}

func (m *MessengerSignalsHandler) HistoryArchivesSeeding(communityID string) {
	m.session.SendHistoryArchivesSeeding(communityID)
}

func (m *MessengerSignalsHandler) HistoryArchivesUnseeded(communityID string) {
	m.session.SendHistoryArchivesUnseeded(communityID)
}

func (m *MessengerSignalsHandler) HistoryArchiveDownloaded(communityID string, from int, to int) {
	m.session.SendHistoryArchiveDownloaded(communityID, from, to)
}

func (m *MessengerSignalsHandler) IndexDownloadCompleted(communityID string, indexCid string) {
	m.session.SendIndexDownloadCompleted(communityID, indexCid)
}

func (m *MessengerSignalsHandler) DownloadingHistoryArchivesStarted(communityID string) {
	m.session.SendDownloadingHistoryArchivesStarted(communityID)
}

func (m *MessengerSignalsHandler) ImportingHistoryArchiveMessages(communityID string) {
	m.session.SendImportingHistoryArchiveMessages(communityID)
}

func (m *MessengerSignalsHandler) DownloadingHistoryArchivesFinished(communityID string) {
	m.session.SendDownloadingHistoryArchivesFinished(communityID)
}

func (m *MessengerSignalsHandler) StatusUpdatesTimedOut(statusUpdates *[]protocol.UserStatus) {
	m.session.SendStatusUpdatesTimedOut(statusUpdates)
}

func (m *MessengerSignalsHandler) DiscordCategoriesAndChannelsExtracted(categories []*discord.Category, channels []*discord.Channel, oldestMessageTimestamp int64, errors map[string]*discord.ImportError) {
	m.session.SendDiscordCategoriesAndChannelsExtracted(categories, channels, oldestMessageTimestamp, errors)
}

func (m *MessengerSignalsHandler) DiscordCommunityImportProgress(importProgress *discord.ImportProgress) {
	m.session.SendDiscordCommunityImportProgress(importProgress)
}

func (m *MessengerSignalsHandler) DiscordChannelImportProgress(importProgress *discord.ImportProgress) {
	m.session.SendDiscordChannelImportProgress(importProgress)
}

func (m *MessengerSignalsHandler) DiscordCommunityImportFinished(id string) {
	m.session.SendDiscordCommunityImportFinished(id)
}

func (m *MessengerSignalsHandler) DiscordChannelImportFinished(communityID string, channelID string) {
	m.session.SendDiscordChannelImportFinished(communityID, channelID)
}

func (m *MessengerSignalsHandler) DiscordCommunityImportCancelled(id string) {
	m.session.SendDiscordCommunityImportCancelled(id)
}

func (m *MessengerSignalsHandler) DiscordCommunityImportCleanedUp(id string) {
	m.session.SendDiscordCommunityImportCleanedUp(id)
}

func (m *MessengerSignalsHandler) DiscordChannelImportCancelled(id string) {
	m.session.SendDiscordChannelImportCancelled(id)
}

func (m *MessengerSignalsHandler) SendBackedUpProfile(response *backupsync.BackedUpDataResponse) {
	m.session.SendBackedUpProfile(response)
}

func (m *MessengerSignalsHandler) SendBackedUpSettings(response *backupsync.BackedUpDataResponse) {
	m.session.SendBackedUpSettings(response)
}

func (m *MessengerSignalsHandler) SendCuratedCommunitiesUpdate(response *communities.KnownCommunitiesResponse) {
	m.session.SendCuratedCommunitiesUpdate(response)
}
