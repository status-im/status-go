package protocol

import (
	"context"
	"errors"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

var ErrInvalidEditOrDeleteAuthor = errors.New("sender is not the author of the message")
var ErrInvalidDeleteTypeAuthor = errors.New("message type cannot be deleted")
var ErrInvalidEditContentType = errors.New("only text messages can be replaced")

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
