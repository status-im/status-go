package adapters

import (
	"errors"

	"github.com/status-im/status-console-client/protocol/v1"
	"github.com/status-im/status-go/services/shhext"
	whisper "github.com/status-im/whisper/whisperv6"
	"golang.org/x/crypto/sha3"
)

func createShhextRequestMessagesParam(enode, mailSymKeyID string, options protocol.RequestOptions) (shhext.MessagesRequest, error) {
	req := shhext.MessagesRequest{
		MailServerPeer: enode,
		From:           uint32(options.From),  // TODO: change to int in status-go
		To:             uint32(options.To),    // TODO: change to int in status-go
		Limit:          uint32(options.Limit), // TODO: change to int in status-go
		SymKeyID:       mailSymKeyID,
	}

	for _, chatOpts := range options.Chats {
		topic, err := topicForChatOptions(chatOpts)
		if err != nil {
			return req, err
		}
		req.Topics = append(req.Topics, topic)
	}

	return req, nil
}

func topicForChatOptions(options protocol.ChatOptions) (whisper.TopicType, error) {
	if options.ChatName != "" {
		return ToTopic(options.ChatName)
	}

	return whisper.TopicType{}, errors.New("invalid options")
}

// ToTopic returns a Whisper topic for a channel name.
func ToTopic(name string) (whisper.TopicType, error) {
	hash := sha3.NewLegacyKeccak256()
	if _, err := hash.Write([]byte(name)); err != nil {
		return whisper.TopicType{}, err
	}
	return whisper.BytesToTopic(hash.Sum(nil)), nil
}
