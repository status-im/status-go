package protobuf

import (
	"github.com/golang/protobuf/proto"
)

//go:generate protoc --go_out=. ./chat_message.proto ./application_metadata_message.proto ./membership_update_message.proto ./command.proto ./contact.proto ./pairing.proto ./push_notifications.proto ./emoji_reaction.proto ./enums.proto ./group_chat_invitation.proto ./chat_identity.proto ./communities.proto

func Unmarshal(payload []byte) (*ApplicationMetadataMessage, error) {
	var message ApplicationMetadataMessage
	err := proto.Unmarshal(payload, &message)
	if err != nil {
		return nil, err
	}

	return &message, nil
}
