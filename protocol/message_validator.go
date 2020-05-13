package protocol

import (
	"errors"
	"strconv"
	"strings"

	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/v1"
)

// maxWhisperDrift is how many milliseconds we allow the clock value to differ
// from whisperTimestamp
const maxWhisperFutureDriftMs uint64 = 120000

func validateClockValue(clock uint64, whisperTimestamp uint64) error {
	if clock == 0 {
		return errors.New("clock can't be 0")
	}

	if clock > whisperTimestamp && clock-whisperTimestamp > maxWhisperFutureDriftMs {
		return errors.New("clock value too high")
	}

	return nil
}

func ValidateMembershipUpdateMessage(message *protocol.MembershipUpdateMessage, timeNowMs uint64) error {

	for _, e := range message.Events {
		if err := validateClockValue(e.ClockValue, timeNowMs); err != nil {
			return err
		}

	}
	return nil
}

func ValidateReceivedPairInstallation(message *protobuf.PairInstallation, whisperTimestamp uint64) error {
	if err := validateClockValue(message.Clock, whisperTimestamp); err != nil {
		return err
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

func ValidateReceivedSendTransaction(message *protobuf.SendTransaction, whisperTimestamp uint64) error {
	if err := validateClockValue(message.Clock, whisperTimestamp); err != nil {
		return err
	}

	if len(strings.TrimSpace(message.TransactionHash)) == 0 {
		return errors.New("transaction hash can't be empty")
	}

	if message.Signature == nil {
		return errors.New("signature can't be nil")
	}

	return nil
}

func ValidateReceivedRequestAddressForTransaction(message *protobuf.RequestAddressForTransaction, whisperTimestamp uint64) error {
	if err := validateClockValue(message.Clock, whisperTimestamp); err != nil {
		return err
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

func ValidateReceivedRequestTransaction(message *protobuf.RequestTransaction, whisperTimestamp uint64) error {
	if err := validateClockValue(message.Clock, whisperTimestamp); err != nil {
		return err
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

func ValidateReceivedAcceptRequestAddressForTransaction(message *protobuf.AcceptRequestAddressForTransaction, whisperTimestamp uint64) error {
	if err := validateClockValue(message.Clock, whisperTimestamp); err != nil {
		return err
	}

	if len(message.Id) == 0 {
		return errors.New("messageID can't be empty")
	}

	if len(strings.TrimSpace(message.Address)) == 0 {
		return errors.New("address can't be empty")
	}

	return nil
}

func ValidateReceivedDeclineRequestAddressForTransaction(message *protobuf.DeclineRequestAddressForTransaction, whisperTimestamp uint64) error {
	if err := validateClockValue(message.Clock, whisperTimestamp); err != nil {
		return err
	}

	if len(message.Id) == 0 {
		return errors.New("messageID can't be empty")
	}

	return nil
}

func ValidateReceivedDeclineRequestTransaction(message *protobuf.DeclineRequestTransaction, whisperTimestamp uint64) error {
	if err := validateClockValue(message.Clock, whisperTimestamp); err != nil {
		return err
	}

	if len(message.Id) == 0 {
		return errors.New("messageID can't be empty")
	}

	return nil
}

func ValidateReceivedChatMessage(message *protobuf.ChatMessage, whisperTimestamp uint64) error {
	if err := validateClockValue(message.Clock, whisperTimestamp); err != nil {
		return err
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

	if message.ContentType == protobuf.ChatMessage_IMAGE {
		if message.Payload == nil {
			return errors.New("no image content")
		}
		image := message.GetImage()
		if image == nil {
			return errors.New("no image content")
		}
		if len(image.Payload) == 0 {
			return errors.New("image payload empty")
		}

		if image.Type == protobuf.ImageMessage_UNKNOWN_IMAGE_TYPE {
			return errors.New("image type unknown")
		}
	}

	return nil
}
