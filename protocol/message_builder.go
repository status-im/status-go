package protocol

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/identity/alias"
)

func extendMessageFromChat(message *common.Message, chat *Chat, key *ecdsa.PublicKey, timesource common.TimeSource) error {
	clock, timestamp := chat.NextClockAndTimestamp(timesource)

	message.LocalChatID = chat.ID
	message.Clock = clock
	message.Timestamp = timestamp
	message.From = types.EncodeHex(crypto.FromECDSAPub(key))
	message.SigPubKey = key
	message.WhisperTimestamp = timestamp
	message.Seen = true
	message.OutgoingStatus = common.OutgoingStatusSending

	alias, err := alias.GenerateFromPublicKeyString(message.From)
	if err != nil {
		return err
	}

	message.Alias = alias
	return nil

}

func extendPinMessageFromChat(message *common.PinMessage, chat *Chat, key *ecdsa.PublicKey, timesource common.TimeSource) error {
	clock, timestamp := chat.NextClockAndTimestamp(timesource)

	message.LocalChatID = chat.ID
	message.Clock = clock
	message.From = types.EncodeHex(crypto.FromECDSAPub(key))
	message.SigPubKey = key
	message.WhisperTimestamp = timestamp

	alias, err := alias.GenerateFromPublicKeyString(message.From)
	if err != nil {
		return err
	}

	message.Alias = alias
	return nil

}
