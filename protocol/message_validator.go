package protocol

import (
	"errors"
	"strings"

	"github.com/status-im/status-go/protocol/protobuf"
)

func ValidateReceivedChatMessage(message *protobuf.ChatMessage) error {
	if message.Clock == 0 {
		return errors.New("Clock can't be 0")
	}

	if message.Timestamp == 0 {
		return errors.New("Timestamp can't be 0")
	}

	if len(strings.TrimSpace(message.Text)) == 0 {
		return errors.New("Text can't be empty")
	}

	if len(message.ChatId) == 0 {
		return errors.New("ChatId can't be empty")
	}

	if message.ContentType == protobuf.ChatMessage_UNKNOWN_CONTENT_TYPE {
		return errors.New("Unknown content type")
	}

	if message.MessageType == protobuf.ChatMessage_UNKNOWN_MESSAGE_TYPE || message.MessageType == protobuf.ChatMessage_SYSTEM_MESSAGE_PRIVATE_GROUP {
		return errors.New("Unknown message type")
	}

	if message.ContentType == protobuf.ChatMessage_STICKER {
		if message.Payload == nil {
			return errors.New("No sticker content")
		}
		sticker := message.GetSticker()
		if sticker == nil {
			return errors.New("No sticker content")
		}
		if len(sticker.Hash) == 0 {
			return errors.New("Sticker hash not set")
		}
	}
	return nil
}
