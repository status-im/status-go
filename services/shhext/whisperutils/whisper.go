package whisperutils

import (
	"github.com/ethereum/go-ethereum/crypto"
	whisper "github.com/status-im/whisper/whisperv6"
)

var discoveryTopic = "contact-discovery"
var DiscoveryTopicBytes = ToTopic(discoveryTopic)

func ToTopic(s string) whisper.TopicType {
	return whisper.BytesToTopic(crypto.Keccak256([]byte(s)))
}
