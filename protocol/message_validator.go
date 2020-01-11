package protocol

import (
	"errors"
	"strconv"
	"strings"

	"github.com/status-im/status-go/protocol/protobuf"
)

func ValidateReceivedPairInstallation(message *protobuf.PairInstallation) error {
	if message.Clock == 0 {
		return errors.New("clock can't be 0")
	}

	if len(strings.TrimSpace(message.Name)) == 0 {
		return errors.New("name can't be empty")
	}

	if len(strings.TrimSpace(message.DeviceType)) == 0 {
		return errors.New("device type can't be empty")
	}

	if len(strings.TrimSpace(message.InstallationId)) == 0 {
		return errors.New("installationId can't be empty")
	}

	return nil
}

func ValidateReceivedSendTransaction(message *protobuf.SendTransaction) error {
	if message.Clock == 0 {
		return errors.New("clock can't be 0")
	}

	if len(strings.TrimSpace(message.TransactionHash)) == 0 {
		return errors.New("transaction hash can't be empty")
	}

	if message.Signature == nil {
		return errors.New("signature can't be nil")
	}

	return nil
}

func ValidateReceivedRequestAddressForTransaction(message *protobuf.RequestAddressForTransaction) error {
	if message.Clock == 0 {
		return errors.New("clock can't be 0")
	}

	if len(strings.TrimSpace(message.Value)) == 0 {
		return errors.New("value can't be empty")
	}

	_, err := strconv.ParseFloat(message.Value, 64)
	if err != nil {
		return err
	}

	return nil
}

func ValidateReceivedRequestTransaction(message *protobuf.RequestTransaction) error {
	if message.Clock == 0 {
		return errors.New("clock can't be 0")
	}

	if len(strings.TrimSpace(message.Value)) == 0 {
		return errors.New("value can't be empty")
	}

	if len(strings.TrimSpace(message.Address)) == 0 {
		return errors.New("address can't be empty")
	}

	_, err := strconv.ParseFloat(message.Value, 64)
	if err != nil {
		return err
	}

	return nil
}

func ValidateReceivedAcceptRequestAddressForTransaction(message *protobuf.AcceptRequestAddressForTransaction) error {
	if message.Clock == 0 {
		return errors.New("clock can't be 0")
	}

	if len(message.Id) == 0 {
		return errors.New("messageID can't be empty")
	}

	if len(strings.TrimSpace(message.Address)) == 0 {
		return errors.New("address can't be empty")
	}

	return nil
}

func ValidateReceivedDeclineRequestAddressForTransaction(message *protobuf.DeclineRequestAddressForTransaction) error {
	if message.Clock == 0 {
		return errors.New("clock can't be 0")
	}

	if len(message.Id) == 0 {
		return errors.New("messageID can't be empty")
	}

	return nil
}

func ValidateReceivedDeclineRequestTransaction(message *protobuf.DeclineRequestTransaction) error {
	if message.Clock == 0 {
		return errors.New("clock can't be 0")
	}

	if len(message.Id) == 0 {
		return errors.New("messageID can't be empty")
	}

	return nil
}

func ValidateReceivedChatMessage(message *protobuf.ChatMessage) error {
	if message.Clock == 0 {
		return errors.New("clock can't be 0")
	}

	if message.Timestamp == 0 {
		return errors.New("timestamp can't be 0")
	}

	if len(strings.TrimSpace(message.Text)) == 0 {
		return errors.New("text can't be empty")
	}

	if len(message.ChatId) == 0 {
		return errors.New("chatId can't be empty")
	}

	if message.ContentType == protobuf.ChatMessage_UNKNOWN_CONTENT_TYPE {
		return errors.New("unknown content type")
	}

	if message.ContentType == protobuf.ChatMessage_TRANSACTION_COMMAND {
		return errors.New("can't receive request address for transaction from others")
	}

	if message.MessageType == protobuf.ChatMessage_UNKNOWN_MESSAGE_TYPE || message.MessageType == protobuf.ChatMessage_SYSTEM_MESSAGE_PRIVATE_GROUP {
		return errors.New("unknown message type")
	}

	if message.ContentType == protobuf.ChatMessage_STICKER {
		if message.Payload == nil {
			return errors.New("no sticker content")
		}
		sticker := message.GetSticker()
		if sticker == nil {
			return errors.New("no sticker content")
		}
		if len(sticker.Hash) == 0 {
			return errors.New("sticker hash not set")
		}
	}
	return nil
}
