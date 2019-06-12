package chatapi

import (
	"context"
	"fmt"

	"github.com/status-im/status-console-client/protocol/client"
	"github.com/status-im/status-console-client/protocol/v1"
	"github.com/status-im/status-go/signal"
)

type PrivateAPI struct {
	s *Service
}

func NewPrivateAPI(s *Service) *PrivateAPI {
	return &PrivateAPI{
		s: s,
	}
}

func (api *PrivateAPI) Chats() (ChatsResponse, error) {
	var result ChatsResponse

	contacts, err := api.s.messenger.Contacts()
	if err != nil {
		return result, fmt.Errorf("failed to load contacts: %v", err)
	}

	contactToChat := make(map[client.Contact][]*protocol.Message)

	for _, c := range contacts {
		messages, err := api.s.messenger.Messages(c, 0)
		if err != nil {
			return result, fmt.Errorf("failed to get messages: %v", err)
		}
		contactToChat[c] = messages
	}

	result = mapToChatsResponse(contactToChat)

	return result, nil
}

func mapToChatsResponse(m map[client.Contact][]*protocol.Message) ChatsResponse {
	var (
		unread int
		result ChatsResponse
		chats  = make(map[string]ChatView)
	)

	for c, messages := range m {
		unreadMessages := countUnread(messages)
		lastMessage := messages[len(messages)-1]

		switch c.Type {
		case client.ContactPublicRoom:
			chats[c.Name] = ChatView{
				ID:                     c.Name,
				Name:                   c.Name,
				ColorHex:               "#abcabc",
				LastMessageContentType: lastMessage.ContentT,
				LastMessageContent: map[string]string{
					"text": lastMessage.Text,
				},
				UnreadMessagesCount: unreadMessages,
				IsGroupChat:         true,
				IsPublic:            true,
			}
		case client.ContactPrivate:
			chats[c.Name] = ChatView{
				ID:                     client.EncodePublicKeyAsString(c.PublicKey),
				Name:                   c.Name,
				ColorHex:               "#abcabc",
				LastMessageContentType: lastMessage.ContentT,
				LastMessageContent: map[string]string{
					"text": lastMessage.Text,
				},
				UnreadMessagesCount: unreadMessages,
				IsGroupChat:         false,
				IsPublic:            false,
			}
		}

		unread += unreadMessages
	}

	result.UnreadMessagesCount = unread
	result.Chats = chats

	return result
}

func countUnread(messages []*protocol.Message) int {
	var counter int
	for _, m := range messages {
		if m.Unread() {
			counter++
		}
	}
	return counter
}

func (api *PrivateAPI) JoinPublicChat(name string) error {
	c := client.CreateContactPublicRoom(name, client.ContactAdded)

	if err := api.s.messenger.Join(context.Background(), c); err != nil {
		return err
	}

	api.sendChatsUpdatedSignal(name)

	return nil
}

func (api *PrivateAPI) StartOneOnOneChat(recipient string) error {
	c, err := client.CreateContactPrivate(recipient, recipient, client.ContactAdded)
	if err != nil {
		return err
	}

	if err := api.s.messenger.Join(context.Background(), c); err != nil {
		return err
	}

	api.sendChatsUpdatedSignal(recipient)

	return nil
}

func (api *PrivateAPI) RemoveChat(id string) error {
	contacts, err := api.s.messenger.Contacts()
	if err != nil {
		return err
	}

	for _, c := range contacts {
		if c.Name != id {
			continue
		}

		if err := api.s.messenger.Leave(c); err != nil {
			return err
		}
		if err := api.s.messenger.RemoveContact(c); err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("chat with id '%s' not found", id)
}

func (api *PrivateAPI) sendChatsUpdatedSignal(name string) {
	signal.SendChatsDidChangeEvent(name)
}
