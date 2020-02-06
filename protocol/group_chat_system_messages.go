package protocol

import (
	"strings"
	"time"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
	v1protocol "github.com/status-im/status-go/protocol/v1"
)

var defaultSystemMessagesTranslations = map[protobuf.MembershipUpdateEvent_EventType]string{
	protobuf.MembershipUpdateEvent_CHAT_CREATED:   "{{from}} created the group {{name}}",
	protobuf.MembershipUpdateEvent_NAME_CHANGED:   "{{from}} changed the group's name to {{name}}",
	protobuf.MembershipUpdateEvent_MEMBERS_ADDED:  "{{from}} has invited {{members}}",
	protobuf.MembershipUpdateEvent_MEMBER_JOINED:  "{{from}} joined the group",
	protobuf.MembershipUpdateEvent_ADMINS_ADDED:   "{{from}} has made {{members}} admin",
	protobuf.MembershipUpdateEvent_MEMBER_REMOVED: "{{member}} left the group",
	protobuf.MembershipUpdateEvent_ADMIN_REMOVED:  "{{member}} is not admin anymore",
}

func tsprintf(format string, params map[string]string) string {
	for key, val := range params {
		format = strings.Replace(format, "{{"+key+"}}", val, -1)
	}
	return format
}

func eventToSystemMessage(e v1protocol.MembershipUpdateEvent, translations map[protobuf.MembershipUpdateEvent_EventType]string) *Message {
	var text string
	switch e.Type {
	case protobuf.MembershipUpdateEvent_CHAT_CREATED:
		text = tsprintf(translations[protobuf.MembershipUpdateEvent_CHAT_CREATED], map[string]string{"from": "@" + e.From, "name": e.Name})
	case protobuf.MembershipUpdateEvent_NAME_CHANGED:
		text = tsprintf(translations[protobuf.MembershipUpdateEvent_NAME_CHANGED], map[string]string{"from": "@" + e.From, "name": e.Name})
	case protobuf.MembershipUpdateEvent_MEMBERS_ADDED:

		var memberMentions []string
		for _, s := range e.Members {
			memberMentions = append(memberMentions, "@"+s)
		}
		text = tsprintf(translations[protobuf.MembershipUpdateEvent_MEMBERS_ADDED], map[string]string{"from": "@" + e.From, "members": strings.Join(memberMentions, ", ")})
	case protobuf.MembershipUpdateEvent_MEMBER_JOINED:
		text = tsprintf(translations[protobuf.MembershipUpdateEvent_MEMBER_JOINED], map[string]string{"from": "@" + e.From})
	case protobuf.MembershipUpdateEvent_ADMINS_ADDED:
		var memberMentions []string
		for _, s := range e.Members {
			memberMentions = append(memberMentions, "@"+s)
		}
		text = tsprintf(translations[protobuf.MembershipUpdateEvent_ADMINS_ADDED], map[string]string{"from": "@" + e.From, "members": strings.Join(memberMentions, ", ")})
	case protobuf.MembershipUpdateEvent_MEMBER_REMOVED:
		text = tsprintf(translations[protobuf.MembershipUpdateEvent_MEMBER_REMOVED], map[string]string{"member": "@" + e.Members[0]})
	case protobuf.MembershipUpdateEvent_ADMIN_REMOVED:
		text = tsprintf(translations[protobuf.MembershipUpdateEvent_ADMIN_REMOVED], map[string]string{"member": "@" + e.Members[0]})

	}
	timestamp := v1protocol.TimestampInMsFromTime(time.Now())
	message := &Message{
		ChatMessage: protobuf.ChatMessage{
			ChatId:      e.ChatID,
			Text:        text,
			MessageType: protobuf.ChatMessage_SYSTEM_MESSAGE_PRIVATE_GROUP,
			ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			Clock:       e.ClockValue,
			Timestamp:   timestamp,
		},
		From:             e.From,
		WhisperTimestamp: timestamp,
		LocalChatID:      e.ChatID,
		Seen:             true,
		ID:               types.EncodeHex(crypto.Keccak256(e.Signature)),
	}
	_ = message.PrepareContent()
	return message
}

func buildSystemMessages(events []v1protocol.MembershipUpdateEvent, translations map[protobuf.MembershipUpdateEvent_EventType]string) []*Message {
	var messages []*Message

	for _, e := range events {
		messages = append(messages, eventToSystemMessage(e, translations))

	}

	return messages
}
