package chat

import (
	"github.com/ethereum/go-ethereum/crypto"
	whisper "github.com/status-im/whisper/whisperv6"
)

var discoveryTopic = "contact-discovery"
var discoveryTopicBytes = toTopic(discoveryTopic)

var topicSalt = []byte{0x01, 0x02, 0x03, 0x04}

func toTopic(s string) whisper.TopicType {
	return whisper.BytesToTopic(crypto.Keccak256([]byte(s)))
}

func SharedSecretToTopic(secret []byte) whisper.TopicType {
	return whisper.BytesToTopic(crypto.Keccak256(append(secret, topicSalt...)))
}

func defaultWhisperMessage() whisper.NewMessage {
	msg := whisper.NewMessage{}

	msg.TTL = 10
	msg.PowTarget = 0.002
	msg.PowTime = 1

	return msg
}

func PublicMessageToWhisper(rpcMsg SendPublicMessageRPC, payload []byte) whisper.NewMessage {
	msg := defaultWhisperMessage()

	msg.Topic = toTopic(rpcMsg.Chat)

	msg.Payload = payload
	msg.Sig = rpcMsg.Sig

	return msg
}

func DirectMessageToWhisper(rpcMsg SendDirectMessageRPC, payload []byte, sharedSecret []byte) whisper.NewMessage {
	var topicBytes whisper.TopicType
	msg := defaultWhisperMessage()

	if rpcMsg.Chat == "" {
		if sharedSecret != nil {
			topicBytes = SharedSecretToTopic(sharedSecret)
		} else {
			topicBytes = discoveryTopicBytes
			msg.PublicKey = rpcMsg.PubKey
		}
	} else {
		topicBytes = toTopic(rpcMsg.Chat)
		msg.PublicKey = rpcMsg.PubKey
	}

	msg.Topic = topicBytes

	msg.Payload = payload
	msg.Sig = rpcMsg.Sig

	return msg
}
