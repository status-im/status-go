package protocol

import (
	"encoding/json"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	localnotifications "github.com/status-im/status-go/services/local-notifications"
	"github.com/status-im/status-go/services/mailservers"
)

type MessengerResponse struct {
	Contacts                []*Contact
	Installations           []*multidevice.Installation
	EmojiReactions          []*EmojiReaction
	Invitations             []*GroupChatInvitation
	CommunityChanges        []*communities.CommunityChanges
	RequestsToJoinCommunity []*communities.RequestToJoin
	Mailservers             []mailservers.Mailserver

	// notifications a list of notifications derived from messenger events
	// that are useful to notify the user about
	notifications               map[string]*localnotifications.Notification
	chats                       map[string]*Chat
	removedChats                map[string]bool
	removedMessages             map[string]bool
	communities                 map[string]*communities.Community
	activityCenterNotifications map[string]*ActivityCenterNotification
	messages                    map[string]*common.Message
	pinMessages                 map[string]*common.PinMessage
	currentStatus               *UserStatus
	statusUpdates               map[string]UserStatus
}

func (r *MessengerResponse) MarshalJSON() ([]byte, error) {
	responseItem := struct {
		Chats                   []*Chat                         `json:"chats,omitempty"`
		RemovedChats            []string                        `json:"removedChats,omitempty"`
		RemovedMessages         []string                        `json:"removedMessages,omitempty"`
		Messages                []*common.Message               `json:"messages,omitempty"`
		Contacts                []*Contact                      `json:"contacts,omitempty"`
		Installations           []*multidevice.Installation     `json:"installations,omitempty"`
		PinMessages             []*common.PinMessage            `json:"pinMessages,omitempty"`
		EmojiReactions          []*EmojiReaction                `json:"emojiReactions,omitempty"`
		Invitations             []*GroupChatInvitation          `json:"invitations,omitempty"`
		CommunityChanges        []*communities.CommunityChanges `json:"communityChanges,omitempty"`
		RequestsToJoinCommunity []*communities.RequestToJoin    `json:"requestsToJoinCommunity,omitempty"`
		Mailservers             []mailservers.Mailserver        `json:"mailservers,omitempty"`
		// Notifications a list of notifications derived from messenger events
		// that are useful to notify the user about
		Notifications               []*localnotifications.Notification `json:"notifications"`
		Communities                 []*communities.Community           `json:"communities,omitempty"`
		ActivityCenterNotifications []*ActivityCenterNotification      `json:"activityCenterNotifications,omitempty"`
		CurrentStatus               *UserStatus                        `json:"currentStatus,omitempty"`
		StatusUpdates               []UserStatus                       `json:"statusUpdates,omitempty"`
	}{
		Contacts:                r.Contacts,
		Installations:           r.Installations,
		EmojiReactions:          r.EmojiReactions,
		Invitations:             r.Invitations,
		CommunityChanges:        r.CommunityChanges,
		RequestsToJoinCommunity: r.RequestsToJoinCommunity,
		Mailservers:             r.Mailservers,
		CurrentStatus:           r.currentStatus,
	}

	responseItem.Messages = r.Messages()
	responseItem.Notifications = r.Notifications()
	responseItem.Chats = r.Chats()
	responseItem.Communities = r.Communities()
	responseItem.RemovedChats = r.RemovedChats()
	responseItem.RemovedMessages = r.RemovedMessages()
	responseItem.ActivityCenterNotifications = r.ActivityCenterNotifications()
	responseItem.PinMessages = r.PinMessages()
	responseItem.StatusUpdates = r.StatusUpdates()

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

func (r *MessengerResponse) RemovedMessages() []string {
	var messages []string
	for messageID := range r.removedMessages {
		messages = append(messages, messageID)
	}
	return messages
}

func (r *MessengerResponse) Communities() []*communities.Community {
	var communities []*communities.Community
	for _, c := range r.communities {
		communities = append(communities, c)
	}
	return communities
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
		len(r.Installations)+
		len(r.Invitations)+
		len(r.EmojiReactions)+
		len(r.communities)+
		len(r.CommunityChanges)+
		len(r.removedChats)+
		len(r.removedMessages)+
		len(r.Mailservers)+
		len(r.notifications)+
		len(r.statusUpdates)+
		len(r.activityCenterNotifications)+
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

func (r *MessengerResponse) AddRemovedMessages(messages []string) {
	for _, m := range messages {
		r.AddRemovedMessage(m)
	}
}

func (r *MessengerResponse) AddRemovedMessage(messageID string) {
	if r.removedMessages == nil {
		r.removedMessages = make(map[string]bool)
	}

	r.removedMessages[messageID] = true
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
	r.messages[message.ID] = message
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
