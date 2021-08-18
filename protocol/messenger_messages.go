package protocol

import (
	"context"
	"errors"
	"fmt"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

var ErrInvalidEditOrDeleteAuthor = errors.New("sender is not the author of the message")
var ErrInvalidDeleteTypeAuthor = errors.New("message type cannot be deleted")
var ErrInvalidEditContentType = errors.New("only text messages can be replaced")

func (m *Messenger) MessageByID(id string) (*common.Message, error) {
	return m.persistence.MessageByID(id)
}

func (m *Messenger) MessagesExist(ids []string) (map[string]bool, error) {
	return m.persistence.MessagesExist(ids)
}

func (m *Messenger) EditMessage(ctx context.Context, request *requests.EditMessage) (*MessengerResponse, error) {
	err := request.Validate()
	if err != nil {
		return nil, err
	}
	message, err := m.persistence.MessageByID(request.ID.String())
	if err != nil {
		return nil, err
	}

	if message.From != common.PubkeyToHex(&m.identity.PublicKey) {
		return nil, ErrInvalidEditOrDeleteAuthor
	}

	if message.ContentType != protobuf.ChatMessage_TEXT_PLAIN {
		return nil, ErrInvalidEditContentType
	}

	// A valid added chat is required.
	chat, ok := m.allChats.Load(message.ChatId)
	if !ok {
		return nil, errors.New("Chat not found")
	}

	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	editMessage := &EditMessage{}

	editMessage.Text = request.Text
	editMessage.ChatId = message.ChatId
	editMessage.MessageId = request.ID.String()
	editMessage.Clock = clock

	err = m.applyEditMessage(&editMessage.EditMessage, message)
	if err != nil {
		return nil, err
	}

	encodedMessage, err := m.encodeChatEntity(chat, editMessage)
	if err != nil {
		return nil, err
	}

	rawMessage := common.RawMessage{
		LocalChatID:          chat.ID,
		Payload:              encodedMessage,
		MessageType:          protobuf.ApplicationMetadataMessage_EDIT_MESSAGE,
		SkipGroupMessageWrap: true,
		ResendAutomatically:  true,
	}
	_, err = m.dispatchMessage(ctx, rawMessage)
	if err != nil {
		return nil, err
	}

	if chat.LastMessage != nil && chat.LastMessage.ID == message.ID {
		chat.LastMessage = message
		err := m.saveChat(chat)
		if err != nil {
			return nil, err
		}
	}

	response := &MessengerResponse{}
	response.AddMessage(message)
	response.AddChat(chat)

	return response, nil
}

func (m *Messenger) DeleteMessageAndSend(ctx context.Context, messageID string) (*MessengerResponse, error) {
	message, err := m.persistence.MessageByID(messageID)
	if err != nil {
		return nil, err
	}

	if message.From != common.PubkeyToHex(&m.identity.PublicKey) {
		return nil, ErrInvalidEditOrDeleteAuthor
	}

	// A valid added chat is required.
	chat, ok := m.allChats.Load(message.ChatId)
	if !ok {
		return nil, errors.New("Chat not found")
	}

	// Only certain types of messages can be deleted
	if message.ContentType != protobuf.ChatMessage_TEXT_PLAIN &&
		message.ContentType != protobuf.ChatMessage_STICKER &&
		message.ContentType != protobuf.ChatMessage_EMOJI &&
		message.ContentType != protobuf.ChatMessage_IMAGE &&
		message.ContentType != protobuf.ChatMessage_AUDIO {
		return nil, ErrInvalidDeleteTypeAuthor
	}

	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	deleteMessage := &DeleteMessage{}

	deleteMessage.ChatId = message.ChatId
	deleteMessage.MessageId = messageID
	deleteMessage.Clock = clock

	encodedMessage, err := m.encodeChatEntity(chat, deleteMessage)

	if err != nil {
		return nil, err
	}

	rawMessage := common.RawMessage{
		LocalChatID:          chat.ID,
		Payload:              encodedMessage,
		MessageType:          protobuf.ApplicationMetadataMessage_DELETE_MESSAGE,
		SkipGroupMessageWrap: true,
		ResendAutomatically:  true,
	}
	_, err = m.dispatchMessage(ctx, rawMessage)
	if err != nil {
		return nil, err
	}

	message.Deleted = true
	err = m.persistence.SaveMessages([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	err = m.persistence.HideMessage(messageID)
	if err != nil {
		return nil, err
	}

	if chat.LastMessage != nil && chat.LastMessage.ID == message.ID {
		if err := m.updateLastMessage(chat); err != nil {
			return nil, err
		}
	}

	response := &MessengerResponse{}
	response.AddMessage(message)
	response.AddRemovedMessage(&RemovedMessage{MessageID: messageID, ChatID: chat.ID})
	response.AddChat(chat)

	return response, nil
}

func (m *Messenger) applyEditMessage(editMessage *protobuf.EditMessage, message *common.Message) error {
	if err := ValidateText(editMessage.Text); err != nil {
		return err
	}
	message.Text = editMessage.Text
	message.EditedAt = editMessage.Clock

	// Save original message as edit so we can retrieve history
	if message.EditedAt == 0 {
		originalEdit := EditMessage{}
		originalEdit.Clock = message.Clock
		originalEdit.LocalChatID = message.LocalChatID
		originalEdit.MessageId = message.ID
		originalEdit.Text = message.Text
		originalEdit.From = message.From
		err := m.persistence.SaveEdit(originalEdit)
		if err != nil {
			return err
		}
	}

	err := message.PrepareContent(common.PubkeyToHex(&m.identity.PublicKey))
	if err != nil {
		return err
	}

	return m.persistence.SaveMessages([]*common.Message{message})
}

func (m *Messenger) applyDeleteMessage(messageDeletes []*DeleteMessage, message *common.Message) error {
	if messageDeletes[0].From != message.From {
		return ErrInvalidEditOrDeleteAuthor
	}

	message.Deleted = true

	err := message.PrepareContent(common.PubkeyToHex(&m.identity.PublicKey))
	if err != nil {
		return err
	}

	err = m.persistence.SaveMessages([]*common.Message{message})
	if err != nil {
		return err
	}

	return m.persistence.HideMessage(message.ID)
}

func (m *Messenger) MessagesByChatID(request *requests.MessagesByChatID) ([]*common.Message, string, error) {

	if err := request.Validate(); err != nil {
		return nil, "", err
	}

	chatID := request.ChatID
	cursor := request.Cursor
	limit := request.Limit
	direction := request.Direction

	chat, err := m.persistence.Chat(chatID)
	if err != nil {
		return nil, "", err
	}

	if chat.Timeline() {
		var chatIDs = []string{"@" + contactIDFromPublicKey(&m.identity.PublicKey)}
		contacts, err := m.persistence.Contacts()
		if err != nil {
			return nil, "", err
		}
		for _, contact := range contacts {
			if contact.Added {
				chatIDs = append(chatIDs, "@"+contact.ID)
			}
		}
		return m.persistence.MessageByChatIDs(chatIDs, cursor, limit, direction)
	}
	return m.persistence.MessageByChatID(chatID, cursor, limit, direction)
}

// DEPRECATED: use MessagesByChatID
func (m *Messenger) MessageByChatID(chatID, cursor string, limit int) ([]*common.Message, string, error) {
	chat, err := m.persistence.Chat(chatID)
	if err != nil {
		return nil, "", err
	}

	if chat.Timeline() {
		var chatIDs = []string{"@" + contactIDFromPublicKey(&m.identity.PublicKey)}
		m.allContacts.Range(func(contactID string, contact *Contact) (shouldContinue bool) {
			if contact.Added {
				chatIDs = append(chatIDs, "@"+contact.ID)
			}
			return true
		})
		return m.persistence.MessageByChatIDs(chatIDs, cursor, limit, requests.OrderingDirectionDesc)
	}
	return m.persistence.MessageByChatID(chatID, cursor, limit, requests.OrderingDirectionDesc)

}

func (m *Messenger) AllMessageByChatIDWhichMatchTerm(chatID string, searchTerm string, caseSensitive bool) ([]*common.Message, error) {
	_, err := m.persistence.Chat(chatID)
	if err != nil {
		return nil, err
	}

	return m.persistence.AllMessageByChatIDWhichMatchTerm(chatID, searchTerm, caseSensitive)
}

func (m *Messenger) AllMessagesFromChatsAndCommunitiesWhichMatchTerm(communityIds []string, chatIds []string, searchTerm string, caseSensitive bool) ([]*common.Message, error) {
	return m.persistence.AllMessagesFromChatsAndCommunitiesWhichMatchTerm(communityIds, chatIds, searchTerm, caseSensitive)
}

func (m *Messenger) SaveMessages(messages []*common.Message) error {
	return m.persistence.SaveMessages(messages)
}

func (m *Messenger) DeleteMessage(id string) error {
	return m.persistence.DeleteMessage(id)
}

func (m *Messenger) DeleteMessagesByChatID(id string) error {
	return m.persistence.DeleteMessagesByChatID(id)
}

// MarkMessagesSeen marks messages with `ids` as seen in the chat `chatID`.
// It returns the number of affected messages or error. If there is an error,
// the number of affected messages is always zero.
func (m *Messenger) MarkMessagesSeen(chatID string, ids []string) (uint64, uint64, error) {
	count, countWithMentions, err := m.persistence.MarkMessagesSeen(chatID, ids)
	if err != nil {
		return 0, 0, err
	}
	chat, err := m.persistence.Chat(chatID)
	if err != nil {
		return 0, 0, err
	}
	m.allChats.Store(chatID, chat)
	return count, countWithMentions, nil
}

func (m *Messenger) MarkAllRead(chatID string) error {
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return errors.New("chat not found")
	}

	err := m.persistence.MarkAllRead(chatID)
	if err != nil {
		return err
	}

	chat.UnviewedMessagesCount = 0
	chat.UnviewedMentionsCount = 0
	// TODO(samyoul) remove storing of an updated reference pointer?
	m.allChats.Store(chat.ID, chat)
	return nil
}

func (m *Messenger) MarkAllReadInCommunity(communityID string) ([]string, error) {
	chatIDs, err := m.persistence.AllChatIDsByCommunity(communityID)
	if err != nil {
		return nil, err
	}

	err = m.persistence.MarkAllReadMultiple(chatIDs)
	if err != nil {
		return nil, err
	}

	for _, chatID := range chatIDs {
		chat, ok := m.allChats.Load(chatID)

		if ok {
			chat.UnviewedMessagesCount = 0
			chat.UnviewedMentionsCount = 0
			m.allChats.Store(chat.ID, chat)
		} else {
			err = errors.New(fmt.Sprintf("chat with chatID %s not found", chatID))
		}
	}

	return chatIDs, err
}
