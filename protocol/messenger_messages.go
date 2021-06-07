package protocol

import (
	"context"
	"errors"
	"fmt"

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
	fmt.Println("IDDD", request.ID.String())
	message, err := m.persistence.MessageByID(request.ID.String())
	if err != nil {
		return nil, err
	}

	sender, err := message.GetSenderPubKey()
	if err != nil {
		return nil, err
	}

	if !sender.Equal(&m.identity.PublicKey) {
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

	message.Text = request.Text
	message.EditedAt = clock

	err = message.PrepareContent(common.PubkeyToHex(&m.identity.PublicKey))
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessages([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	editMessage := &EditMessage{}

	editMessage.Text = request.Text
	editMessage.ChatId = message.ChatId
	editMessage.MessageId = request.ID.String()
	editMessage.Clock = clock

	encodedMessage, err := m.encodeChatEntity(chat, editMessage)
	if err != nil {
		return nil, err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_EDIT_MESSAGE,
		ResendAutomatically: true,
	}
	_, err = m.dispatchMessage(ctx, rawMessage)
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	response.AddMessage(message)

	return response, nil
}
