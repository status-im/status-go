package protocol

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/identity/identicon"
)

func extendMessageFromChat(message *Message, chat *Chat, key *ecdsa.PublicKey) error {
	clock, timestamp := chat.NextClockAndTimestamp()

	message.LocalChatID = chat.ID
	message.Clock = clock
	message.Timestamp = timestamp
	message.From = types.EncodeHex(crypto.FromECDSAPub(key))
	message.SigPubKey = key
	message.WhisperTimestamp = timestamp
	message.Seen = true
	message.OutgoingStatus = OutgoingStatusSending

	identicon, err := identicon.GenerateBase64(message.From)
	if err != nil {
		return err
	}

	message.Identicon = identicon

	alias, err := alias.GenerateFromPublicKeyString(message.From)
	if err != nil {
		return err
	}

	message.Alias = alias
	return nil

}
