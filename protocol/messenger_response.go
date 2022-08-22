package protocol

import (
	"encoding/json"

	"github.com/status-im/status-go/services/browsers"

	"github.com/status-im/status-go/appmetrics"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/discord"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/verification"
	localnotifications "github.com/status-im/status-go/services/local-notifications"
	"github.com/status-im/status-go/services/mailservers"
)

type RemovedMessage struct {
	ChatID    string `json:"chatId"`
	MessageID string `json:"messageId"`
}

type ClearedHistory struct {
	ChatID    string `json:"chatId"`
	ClearedAt uint64 `json:"clearedAt"`
}

type MessengerResponse struct {
	Contacts                      []*Contact
	Installations                 []*multidevice.Installation
	EmojiReactions                []*EmojiReaction
	Invitations                   []*GroupChatInvitation
	CommunityChanges              []*communities.CommunityChanges
	RequestsToJoinCommunity       []*communities.RequestToJoin
	AnonymousMetrics              []*appmetrics.AppMetric
	Mailservers                   []mailservers.Mailserver
	Bookmarks                     []*browsers.Bookmark
	Settings                      []*settings.SyncSettingField
	IdentityImages                []images.IdentityImage
	Accounts                      []*accounts.Account
	VerificationRequests          []*verification.Request
	DiscordCategories             []*discord.Category
	DiscordChannels               []*discord.Channel
	DiscordOldestMessageTimestamp int

	// notifications a list of notifications derived from messenger events
	// that are useful to notify the user about
	notifications               map[string]*localnotifications.Notification
	chats                       map[string]*Chat
	removedChats                map[string]bool
	removedMessages             map[string]*RemovedMessage
	communities                 map[string]*communities.Community
	communitiesSettings         map[string]*communities.CommunitySettings
	activityCenterNotifications map[string]*ActivityCenterNotification
	messages                    map[string]*common.Message
	pinMessages                 map[string]*common.PinMessage
	currentStatus               *UserStatus
	statusUpdates               map[string]UserStatus
	clearedHistories            map[string]*ClearedHistory
	trustStatus                 map[string]verification.TrustStatus
}

func (r *MessengerResponse) MarshalJSON() ([]byte, error) {
	responseItem := struct {
		Chats                   []*Chat                             `json:"chats,omitempty"`
		RemovedChats            []string                            `json:"removedChats,omitempty"`
		RemovedMessages         []*RemovedMessage                   `json:"removedMessages,omitempty"`
		Messages                []*common.Message                   `json:"messages,omitempty"`
		Contacts                []*Contact                          `json:"contacts,omitempty"`
		Installations           []*multidevice.Installation         `json:"installations,omitempty"`
		PinMessages             []*common.PinMessage                `json:"pinMessages,omitempty"`
		EmojiReactions          []*EmojiReaction                    `json:"emojiReactions,omitempty"`
		Invitations             []*GroupChatInvitation              `json:"invitations,omitempty"`
		CommunityChanges        []*communities.CommunityChanges     `json:"communityChanges,omitempty"`
		RequestsToJoinCommunity []*communities.RequestToJoin        `json:"requestsToJoinCommunity,omitempty"`
		Mailservers             []mailservers.Mailserver            `json:"mailservers,omitempty"`
		Bookmarks               []*browsers.Bookmark                `json:"bookmarks,omitempty"`
		ClearedHistories        []*ClearedHistory                   `json:"clearedHistories,omitempty"`
		VerificationRequests    []*verification.Request             `json:"verificationRequests,omitempty"`
		TrustStatus             map[string]verification.TrustStatus `json:"trustStatus,omitempty"`
		// Notifications a list of notifications derived from messenger events
		// that are useful to notify the user about
		Notifications                 []*localnotifications.Notification `json:"notifications"`
		Communities                   []*communities.Community           `json:"communities,omitempty"`
		CommunitiesSettings           []*communities.CommunitySettings   `json:"communitiesSettings,omitempty"`
		ActivityCenterNotifications   []*ActivityCenterNotification      `json:"activityCenterNotifications,omitempty"`
		CurrentStatus                 *UserStatus                        `json:"currentStatus,omitempty"`
		StatusUpdates                 []UserStatus                       `json:"statusUpdates,omitempty"`
		Settings                      []*settings.SyncSettingField       `json:"settings,omitempty"`
		IdentityImages                []images.IdentityImage             `json:"identityImages,omitempty"`
		Accounts                      []*accounts.Account                `json:"accounts,omitempty"`
		DiscordCategories             []*discord.Category                `json:"discordCategories,omitempty"`
		DiscordChannels               []*discord.Channel                 `json:"discordChannels,omitempty"`
		DiscordOldestMessageTimestamp int                                `json:"discordOldestMessageTimestamp"`
	}{
		Contacts:                r.Contacts,
		Installations:           r.Installations,
		EmojiReactions:          r.EmojiReactions,
		Invitations:             r.Invitations,
		CommunityChanges:        r.CommunityChanges,
		RequestsToJoinCommunity: r.RequestsToJoinCommunity,
		Mailservers:             r.Mailservers,
		Bookmarks:               r.Bookmarks,
		CurrentStatus:           r.currentStatus,
		Settings:                r.Settings,
		IdentityImages:          r.IdentityImages,
		Accounts:                r.Accounts,
		VerificationRequests:    r.VerificationRequests,

		Messages:                      r.Messages(),
		Notifications:                 r.Notifications(),
		Chats:                         r.Chats(),
		Communities:                   r.Communities(),
		CommunitiesSettings:           r.CommunitiesSettings(),
		RemovedChats:                  r.RemovedChats(),
		RemovedMessages:               r.RemovedMessages(),
		ClearedHistories:              r.ClearedHistories(),
		ActivityCenterNotifications:   r.ActivityCenterNotifications(),
		PinMessages:                   r.PinMessages(),
		StatusUpdates:                 r.StatusUpdates(),
		DiscordCategories:             r.DiscordCategories,
		DiscordChannels:               r.DiscordChannels,
		DiscordOldestMessageTimestamp: r.DiscordOldestMessageTimestamp,
	}

	responseItem.TrustStatus = r.TrustStatus()
	return json.Marshal(responseItem)
}

func (r *MessengerResponse) Chats() []*Chat {
	var chats []*Chat
	for _, chat := range r.chats {
		chats = append(chats, chat)
	}
	return chats
}

func (r *MessengerResponse) RemovedChats() []string {
	var chats []string
	for chatID := range r.removedChats {
		chats = append(chats, chatID)
	}
	return chats
}

func (r *MessengerResponse) RemovedMessages() []*RemovedMessage {
	var messages []*RemovedMessage
	for messageID := range r.removedMessages {
		messages = append(messages, r.removedMessages[messageID])
	}
	return messages
}

func (r *MessengerResponse) ClearedHistories() []*ClearedHistory {
	var clearedHistories []*ClearedHistory
	for chatID := range r.clearedHistories {
		clearedHistories = append(clearedHistories, r.clearedHistories[chatID])
	}
	return clearedHistories
}

func (r *MessengerResponse) Communities() []*communities.Community {
	var communities []*communities.Community
	for _, c := range r.communities {
		communities = append(communities, c)
	}
	return communities
}

func (r *MessengerResponse) CommunitiesSettings() []*communities.CommunitySettings {
	var settings []*communities.CommunitySettings
	for _, s := range r.communitiesSettings {
		settings = append(settings, s)
	}
	return settings
}

func (r *MessengerResponse) Notifications() []*localnotifications.Notification {
	var notifications []*localnotifications.Notification
	for _, n := range r.notifications {
		notifications = append(notifications, n)
	}
	return notifications
}

func (r *MessengerResponse) PinMessages() []*common.PinMessage {
	var pinMessages []*common.PinMessage
	for _, pm := range r.pinMessages {
		pinMessages = append(pinMessages, pm)
	}
	return pinMessages
}

func (r *MessengerResponse) TrustStatus() map[string]verification.TrustStatus {
	if len(r.trustStatus) == 0 {
		return nil
	}

	result := make(map[string]verification.TrustStatus)
	for contactID, trustStatus := range r.trustStatus {
		result[contactID] = trustStatus
	}
	return result
}

func (r *MessengerResponse) StatusUpdates() []UserStatus {
	var userStatus []UserStatus
	for pk, s := range r.statusUpdates {
		s.PublicKey = pk
		userStatus = append(userStatus, s)
	}
	return userStatus
}

func (r *MessengerResponse) IsEmpty() bool {
	return len(r.chats)+
		len(r.messages)+
		len(r.pinMessages)+
		len(r.Contacts)+
		len(r.Bookmarks)+
		len(r.clearedHistories)+
		len(r.Settings)+
		len(r.Installations)+
		len(r.Invitations)+
		len(r.EmojiReactions)+
		len(r.communities)+
		len(r.CommunityChanges)+
		len(r.removedChats)+
		len(r.removedMessages)+
		len(r.Mailservers)+
		len(r.IdentityImages)+
		len(r.Accounts)+
		len(r.notifications)+
		len(r.statusUpdates)+
		len(r.activityCenterNotifications)+
		len(r.trustStatus)+
		len(r.VerificationRequests)+
		len(r.RequestsToJoinCommunity) == 0 &&
		r.currentStatus == nil
}

// Merge takes another response and appends the new Chats & new Messages and replaces
// the existing Messages & Chats if they have the same ID
func (r *MessengerResponse) Merge(response *MessengerResponse) error {
	if len(response.Contacts)+
		len(response.Installations)+
		len(response.EmojiReactions)+
		len(response.Invitations)+
		len(response.RequestsToJoinCommunity)+
		len(response.Mailservers)+
		len(response.EmojiReactions)+
		len(response.Bookmarks)+
		len(response.clearedHistories)+
		len(response.VerificationRequests)+
		len(response.trustStatus)+
		len(response.DiscordChannels)+
		len(response.DiscordCategories)+
		len(response.CommunityChanges) != 0 {
		return ErrNotImplemented
	}

	r.AddChats(response.Chats())
	r.AddRemovedChats(response.RemovedChats())
	r.AddRemovedMessages(response.RemovedMessages())
	r.AddNotifications(response.Notifications())
	r.AddMessages(response.Messages())
	r.AddCommunities(response.Communities())
	r.AddPinMessages(response.PinMessages())
	r.AddVerificationRequests(response.VerificationRequests)
	r.AddTrustStatuses(response.trustStatus)
	r.AddActivityCenterNotifications(response.ActivityCenterNotifications())

	return nil
}

func (r *MessengerResponse) AddCommunities(communities []*communities.Community) {
	for _, overrideCommunity := range communities {
		r.AddCommunity(overrideCommunity)
	}
}

func (r *MessengerResponse) AddCommunity(c *communities.Community) {
	if r.communities == nil {
		r.communities = make(map[string]*communities.Community)
	}

	r.communities[c.IDString()] = c
}

func (r *MessengerResponse) AddCommunitySettings(c *communities.CommunitySettings) {
	if r.communitiesSettings == nil {
		r.communitiesSettings = make(map[string]*communities.CommunitySettings)
	}

	r.communitiesSettings[c.CommunityID] = c
}

func (r *MessengerResponse) AddBookmark(bookmark *browsers.Bookmark) {
	r.Bookmarks = append(r.Bookmarks, bookmark)
}

func (r *MessengerResponse) AddBookmarks(bookmarks []*browsers.Bookmark) {
	for _, b := range bookmarks {
		r.AddBookmark(b)
	}
}

func (r *MessengerResponse) AddVerificationRequest(vr *verification.Request) {
	r.VerificationRequests = append(r.VerificationRequests, vr)
}

func (r *MessengerResponse) AddVerificationRequests(vrs []*verification.Request) {
	for _, vr := range vrs {
		r.AddVerificationRequest(vr)
	}
}

func (r *MessengerResponse) AddTrustStatus(contactID string, trustStatus verification.TrustStatus) {
	if r.trustStatus == nil {
		r.trustStatus = make(map[string]verification.TrustStatus)
	}

	r.trustStatus[contactID] = trustStatus
}

func (r *MessengerResponse) AddTrustStatuses(ts map[string]verification.TrustStatus) {
	if r.trustStatus == nil {
		r.trustStatus = make(map[string]verification.TrustStatus)
	}

	for k, v := range ts {
		r.trustStatus[k] = v
	}
}

func (r *MessengerResponse) AddChat(c *Chat) {
	if r.chats == nil {
		r.chats = make(map[string]*Chat)
	}

	r.chats[c.ID] = c
}

func (r *MessengerResponse) AddChats(chats []*Chat) {
	for _, c := range chats {
		r.AddChat(c)
	}
}

func (r *MessengerResponse) AddNotification(n *localnotifications.Notification) {
	if r.notifications == nil {
		r.notifications = make(map[string]*localnotifications.Notification)
	}

	r.notifications[n.ID.String()] = n
}

func (r *MessengerResponse) ClearNotifications() {
	r.notifications = nil
}

func (r *MessengerResponse) AddNotifications(notifications []*localnotifications.Notification) {
	for _, c := range notifications {
		r.AddNotification(c)
	}
}

func (r *MessengerResponse) AddRemovedChats(chats []string) {
	for _, c := range chats {
		r.AddRemovedChat(c)
	}
}

func (r *MessengerResponse) AddRemovedChat(chatID string) {
	if r.removedChats == nil {
		r.removedChats = make(map[string]bool)
	}

	r.removedChats[chatID] = true
}

func (r *MessengerResponse) AddRemovedMessages(messages []*RemovedMessage) {
	for _, m := range messages {
		r.AddRemovedMessage(m)
	}
}

func (r *MessengerResponse) AddRemovedMessage(rm *RemovedMessage) {
	if r.removedMessages == nil {
		r.removedMessages = make(map[string]*RemovedMessage)
	}

	r.removedMessages[rm.MessageID] = rm
	// Remove original message from the map

	if len(r.messages) != 0 && r.messages[rm.MessageID] != nil {
		delete(r.messages, rm.MessageID)
	}
}

func (r *MessengerResponse) AddClearedHistory(ch *ClearedHistory) {
	if r.clearedHistories == nil {
		r.clearedHistories = make(map[string]*ClearedHistory)
	}

	existingClearedHistory, ok := r.clearedHistories[ch.ChatID]
	if !ok || existingClearedHistory.ClearedAt < ch.ClearedAt {
		r.clearedHistories[ch.ChatID] = ch
	}
}

func (r *MessengerResponse) AddActivityCenterNotifications(ns []*ActivityCenterNotification) {
	for _, n := range ns {
		r.AddActivityCenterNotification(n)
	}
}

func (r *MessengerResponse) AddActivityCenterNotification(n *ActivityCenterNotification) {
	if r.activityCenterNotifications == nil {
		r.activityCenterNotifications = make(map[string]*ActivityCenterNotification)
	}

	r.activityCenterNotifications[n.ID.String()] = n
}

func (r *MessengerResponse) RemoveActivityCenterNotification(id string) bool {
	if r.activityCenterNotifications == nil {
		return false
	}

	if _, ok := r.activityCenterNotifications[id]; ok {
		delete(r.activityCenterNotifications, id)
		return true
	}

	return false
}

func (r *MessengerResponse) ActivityCenterNotifications() []*ActivityCenterNotification {
	var ns []*ActivityCenterNotification
	for _, n := range r.activityCenterNotifications {
		ns = append(ns, n)
	}
	return ns
}

func (r *MessengerResponse) AddPinMessage(pm *common.PinMessage) {
	if r.pinMessages == nil {
		r.pinMessages = make(map[string]*common.PinMessage)
	}

	r.pinMessages[pm.ID] = pm
}

func (r *MessengerResponse) AddPinMessages(pms []*common.PinMessage) {
	for _, pm := range pms {
		r.AddPinMessage(pm)
	}
}

func (r *MessengerResponse) SetCurrentStatus(status UserStatus) {
	r.currentStatus = &status
}

func (r *MessengerResponse) AddStatusUpdate(upd UserStatus) {
	if r.statusUpdates == nil {
		r.statusUpdates = make(map[string]UserStatus)
	}

	r.statusUpdates[upd.PublicKey] = upd
}

func (r *MessengerResponse) Messages() []*common.Message {
	var ms []*common.Message
	for _, m := range r.messages {
		ms = append(ms, m)
	}
	return ms
}

func (r *MessengerResponse) AddMessages(ms []*common.Message) {
	for _, m := range ms {
		r.AddMessage(m)
	}
}

func (r *MessengerResponse) AddMessage(message *common.Message) {
	if r.messages == nil {
		r.messages = make(map[string]*common.Message)
	}
	if message.Deleted {
		return
	}
	r.messages[message.ID] = message
}

func (r *MessengerResponse) AddContact(c *Contact) {

	for idx, c1 := range r.Contacts {
		if c1.ID == c.ID {
			r.Contacts[idx] = c
			return
		}
	}

	r.Contacts = append(r.Contacts, c)
}

func (r *MessengerResponse) SetMessages(messages []*common.Message) {
	r.messages = make(map[string]*common.Message)
	r.AddMessages(messages)
}

func (r *MessengerResponse) GetMessage(messageID string) *common.Message {
	if r.messages == nil {
		return nil
	}
	return r.messages[messageID]
}

func (r *MessengerResponse) AddDiscordCategory(dc *discord.Category) {
	for idx, c := range r.DiscordCategories {
		if dc.ID == c.ID {
			r.DiscordCategories[idx] = dc
			return
		}
	}

	r.DiscordCategories = append(r.DiscordCategories, dc)
}

func (r *MessengerResponse) AddDiscordChannel(dc *discord.Channel) {
	for idx, c := range r.DiscordChannels {
		if dc.ID == c.ID {
			r.DiscordChannels[idx] = dc
			return
		}
	}

	r.DiscordChannels = append(r.DiscordChannels, dc)
}

func (r *MessengerResponse) HasDiscordCategory(id string) bool {
	for _, c := range r.DiscordCategories {
		if id == c.ID {
			return true
		}
	}
	return false
}

func (r *MessengerResponse) HasDiscordChannel(id string) bool {
	for _, c := range r.DiscordChannels {
		if id == c.ID {
			return true
		}
	}
	return false
}
