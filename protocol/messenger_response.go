package protocol

import (
	"encoding/json"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/transport"
	localnotifications "github.com/status-im/status-go/services/local-notifications"
	"github.com/status-im/status-go/services/mailservers"
)

type MessengerResponse struct {
	Messages                []*common.Message
	Contacts                []*Contact
	Installations           []*multidevice.Installation
	EmojiReactions          []*EmojiReaction
	Invitations             []*GroupChatInvitation
	CommunityChanges        []*communities.CommunityChanges
	RequestsToJoinCommunity []*communities.RequestToJoin
	Filters                 []*transport.Filter
	RemovedFilters          []*transport.Filter
	Mailservers             []mailservers.Mailserver
	MailserverTopics        []mailservers.MailserverTopic
	MailserverRanges        []mailservers.ChatRequestRange

	// notifications a list of notifications derived from messenger events
	// that are useful to notify the user about
	notifications map[string]*localnotifications.Notification
	chats         map[string]*Chat
	removedChats  map[string]bool
	communities   map[string]*communities.Community
}

func (r *MessengerResponse) MarshalJSON() ([]byte, error) {
	responseItem := struct {
		Chats                   []*Chat                         `json:"chats,omitempty"`
		RemovedChats            []string                        `json:"removedChats,omitempty"`
		Messages                []*common.Message               `json:"messages,omitempty"`
		Contacts                []*Contact                      `json:"contacts,omitempty"`
		Installations           []*multidevice.Installation     `json:"installations,omitempty"`
		EmojiReactions          []*EmojiReaction                `json:"emojiReactions,omitempty"`
		Invitations             []*GroupChatInvitation          `json:"invitations,omitempty"`
		CommunityChanges        []*communities.CommunityChanges `json:"communityChanges,omitempty"`
		RequestsToJoinCommunity []*communities.RequestToJoin    `json:"requestsToJoinCommunity,omitempty"`
		Filters                 []*transport.Filter             `json:"filters,omitempty"`
		RemovedFilters          []*transport.Filter             `json:"removedFilters,omitempty"`
		Mailservers             []mailservers.Mailserver        `json:"mailservers,omitempty"`
		MailserverTopics        []mailservers.MailserverTopic   `json:"mailserverTopics,omitempty"`
		MailserverRanges        []mailservers.ChatRequestRange  `json:"mailserverRanges,omitempty"`
		// Notifications a list of notifications derived from messenger events
		// that are useful to notify the user about
		Notifications []*localnotifications.Notification `json:"notifications"`
		Communities   []*communities.Community           `json:"communities,omitempty"`
	}{
		Messages:                r.Messages,
		Contacts:                r.Contacts,
		Installations:           r.Installations,
		EmojiReactions:          r.EmojiReactions,
		Invitations:             r.Invitations,
		CommunityChanges:        r.CommunityChanges,
		RequestsToJoinCommunity: r.RequestsToJoinCommunity,
		Filters:                 r.Filters,
		RemovedFilters:          r.RemovedFilters,
		Mailservers:             r.Mailservers,
		MailserverTopics:        r.MailserverTopics,
		MailserverRanges:        r.MailserverRanges,
	}

	responseItem.Notifications = r.Notifications()
	responseItem.Chats = r.Chats()
	responseItem.Communities = r.Communities()
	responseItem.RemovedChats = r.RemovedChats()

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

func (r *MessengerResponse) IsEmpty() bool {
	return len(r.chats)+
		len(r.Messages)+
		len(r.Contacts)+
		len(r.Installations)+
		len(r.Invitations)+
		len(r.EmojiReactions)+
		len(r.communities)+
		len(r.CommunityChanges)+
		len(r.Filters)+
		len(r.RemovedFilters)+
		len(r.removedChats)+
		len(r.MailserverTopics)+
		len(r.Mailservers)+
		len(r.MailserverRanges)+
		len(r.notifications)+
		len(r.RequestsToJoinCommunity) == 0
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
		len(response.MailserverTopics)+
		len(response.MailserverRanges)+
		len(response.EmojiReactions)+
		len(response.CommunityChanges) != 0 {
		return ErrNotImplemented
	}

	r.AddChats(response.Chats())
	r.AddRemovedChats(response.RemovedChats())
	r.AddNotifications(response.Notifications())
	r.overrideMessages(response.Messages)
	r.overrideFilters(response.Filters)
	r.overrideRemovedFilters(response.Filters)
	r.AddCommunities(response.Communities())

	return nil
}

// overrideMessages append new messages and override existing ones in response.Messages
func (r *MessengerResponse) overrideMessages(messages []*common.Message) {
	for _, overrideMessage := range messages {
		var found = false
		for idx, chat := range r.Messages {
			if chat.ID == overrideMessage.ID {
				r.Messages[idx] = overrideMessage
				found = true
			}
		}
		if !found {
			r.Messages = append(r.Messages, overrideMessage)
		}
	}
}

// overrideFilters append new filters and override existing ones in response.Filters
func (r *MessengerResponse) overrideFilters(filters []*transport.Filter) {
	for _, overrideFilter := range filters {
		var found = false
		for idx, filter := range r.Filters {
			if filter.FilterID == overrideFilter.FilterID {
				r.Filters[idx] = overrideFilter
				found = true
			}
		}
		if !found {
			r.Filters = append(r.Filters, overrideFilter)
		}
	}
}

// overrideRemovedFilters append removed filters and override existing ones in response.Filters
func (r *MessengerResponse) overrideRemovedFilters(filters []*transport.Filter) {
	for _, overrideFilter := range filters {
		var found = false
		for idx, filter := range r.RemovedFilters {
			if filter.FilterID == overrideFilter.FilterID {
				r.RemovedFilters[idx] = overrideFilter
				found = true
			}
		}
		if !found {
			r.RemovedFilters = append(r.RemovedFilters, overrideFilter)
		}
	}
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
