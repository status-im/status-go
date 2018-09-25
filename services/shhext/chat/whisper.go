package chat

import (
	"github.com/ethereum/go-ethereum/crypto"
	whisper "github.com/status-im/whisper/whisperv6"
)

var discoveryTopic = "contact-discovery"
var discoveryTopicBytes = toTopic(discoveryTopic)

func toTopic(s string) whisper.TopicType {
	return whisper.BytesToTopic(crypto.Keccak256([]byte(s)))
}

func defaultWhisperMessage() *whisper.NewMessage {
	msg := &whisper.NewMessage{}

	msg.TTL = 10
	msg.PowTarget = 0.002
	msg.PowTime = 1

	return msg
}

func PublicMessageToWhisper(rpcMsg *SendPublicMessageRPC, payload []byte) *whisper.NewMessage {
	msg := defaultWhisperMessage()

	msg.Topic = toTopic(rpcMsg.Chat)

	msg.Payload = payload
	msg.Sig = rpcMsg.Sig

	return msg
}

func DirectMessageToWhisper(rpcMsg *SendDirectMessageRPC, payload []byte) *whisper.NewMessage {

	msg := defaultWhisperMessage()

	msg.Topic = discoveryTopicBytes

	msg.Payload = payload
	msg.Sig = rpcMsg.Sig
	msg.PublicKey = rpcMsg.PubKey

	return msg
}
