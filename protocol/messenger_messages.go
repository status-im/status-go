package protocol

import (
	"context"
	"errors"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

var ErrInvalidEditAuthor = errors.New("sender is not the author of the message")
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
		return nil, ErrInvalidEditAuthor
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
