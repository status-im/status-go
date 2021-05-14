package protocol

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"
	"fmt"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

// SendPinMessage sends the PinMessage to the corresponding chat
func (m *Messenger) SendPinMessage(ctx context.Context, message *common.PinMessage) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.sendPinMessage(ctx, message)
}

func (m *Messenger) sendPinMessage(ctx context.Context, message *common.PinMessage) (*MessengerResponse, error) {
	var response MessengerResponse

	// A valid added chat is required.
	chat, ok := m.allChats.Load(message.ChatId)
	if !ok {
		return nil, errors.New("chat not found")
	}

	err := m.handleStandaloneChatIdentity(chat)
	if err != nil {
		return nil, err
	}

	err = extendPinMessageFromChat(message, chat, &m.identity.PublicKey, m.getTimesource())
	if err != nil {
		return nil, err
	}

	message.ID, err = generatePinMessageID(&m.identity.PublicKey, message, chat)
	if err != nil {
		return nil, err
	}

	encodedMessage, err := m.encodeChatEntity(chat, message)
	if err != nil {
		return nil, err
	}

	rawMessage := common.RawMessage{
		LocalChatID:          chat.ID,
		Payload:              encodedMessage,
		MessageType:          protobuf.ApplicationMetadataMessage_PIN_MESSAGE,
		SkipGroupMessageWrap: true,
		ResendAutomatically:  true,
	}
	_, err = m.dispatchMessage(ctx, rawMessage)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SavePinMessages([]*common.PinMessage{message})
	if err != nil {
		return nil, err
	}

	response.AddPinMessage(message)
	response.AddChat(chat)
	return &response, m.saveChat(chat)
}

func (m *Messenger) PinnedMessageByChatID(chatID, cursor string, limit int) ([]*common.PinnedMessage, string, error) {
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
			if contact.IsAdded() {
				chatIDs = append(chatIDs, "@"+contact.ID)
			}
		}
		return m.persistence.PinnedMessageByChatIDs(chatIDs, cursor, limit)
	}
	return m.persistence.PinnedMessageByChatID(chatID, cursor, limit)
}

func (m *Messenger) SavePinMessages(messages []*common.PinMessage) error {
	return m.persistence.SavePinMessages(messages)
}

func generatePinMessageID(pubKey *ecdsa.PublicKey, pm *common.PinMessage, chat *Chat) (string, error) {
	data := gethcommon.FromHex(pm.MessageId)

	switch {
	case chat.ChatType == ChatTypeOneToOne:
		ourPubKey := crypto.FromECDSAPub(pubKey)
		tmpPubKey, err := chat.PublicKey()
		if err != nil {
			return "", err
		}
		theirPubKey := crypto.FromECDSAPub(tmpPubKey)

		if bytes.Compare(ourPubKey, theirPubKey) < 0 {
			data = append(data, ourPubKey...)   // our key
			data = append(data, theirPubKey...) // their key
		} else {
			data = append(data, theirPubKey...) // their key
			data = append(data, ourPubKey...)   // our key
		}
	default:
		data = append(data, []byte(chat.ID)...)
	}
	id := sha256.Sum256(data)
	idString := fmt.Sprintf("%x", id)

	return idString, nil
}
