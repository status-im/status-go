package protocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
)

func TestMessengerReplySuite(t *testing.T) {
	suite.Run(t, new(MessengerReplySuite))
}

type MessengerReplySuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerReplySuite) TestReceiveReply() {
	alice := s.m
	alice.account = &multiaccounts.Account{KeyUID: "0xdeadbeef"}

	bob := s.newMessenger()

	chatID := statusChatID

	chat := CreatePublicChat(chatID, alice.getTimesource())

	err := alice.SaveChat(chat)
	s.Require().NoError(err)

	_, err = alice.Join(chat)
	s.Require().NoError(err)

	err = bob.SaveChat(chat)
	s.Require().NoError(err)

	_, err = bob.Join(chat)
	s.Require().NoError(err)

	// Send chat message from alice to bob
	message := buildTestMessage(*chat)
	_, err = alice.SendChatMessage(context.Background(), message)
	s.NoError(err)

	// Wait for message to arrive to bob
	response, err := WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().Len(response.Messages(), 1)
	s.Require().False(response.Messages()[0].Replied)

	messageID := response.Messages()[0].ID

	// Bob replies to the message
	replyMessage := buildTestMessage(*chat)
	replyMessage.ResponseTo = messageID
	_, err = bob.SendChatMessage(context.Background(), replyMessage)
	s.NoError(err)

	// Wait for reply to arrive to alice
	response, err = WaitOnMessengerResponse(
		alice,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	messages := response.Messages()
	s.Require().Len(messages, 2)
	//* Verify that the reply reponds to the original message
	messageToCheck := messages[0]
	if messageToCheck.ResponseTo == "" || messageToCheck.ResponseTo != messageID {
		// We need to use the second message in the response. They got out of order by accident
		messageToCheck = messages[1]
	}
	s.Require().True(messageToCheck.ResponseTo == messageID)
	// Verify that it's replied
	s.Require().True(messageToCheck.Replied)
}

func (s *MessengerReplySuite) TestSendReplyToNonExistentMessage() {
	alice := s.m

	chatID := statusChatID
	chat := CreatePublicChat(chatID, alice.getTimesource())

	err := alice.SaveChat(chat)
	s.Require().NoError(err)

	_, err = alice.Join(chat)
	s.Require().NoError(err)

	// Reply to a message ID that was never sent or received
	replyMessage := buildTestMessage(*chat)
	replyMessage.ResponseTo = "non-existent-message-id"
	_, err = alice.SendChatMessage(context.Background(), replyMessage)
	s.Require().ErrorIs(err, ErrInvalidResponseTo)
}

func (s *MessengerReplySuite) TestSendRejectsInvalidReplyTargets() {
	chat := CreateOneToOneChat("DM", &s.newMessenger().identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	for _, tt := range []struct {
		name        string
		localChatID string
		contentType protobuf.ChatMessage_ContentType
	}{
		{"other chat", "group-chat", protobuf.ChatMessage_TEXT_PLAIN},
		{"system message", chat.ID, protobuf.ChatMessage_SYSTEM_MESSAGE_PINNED_MESSAGE},
	} {
		s.Run(tt.name, func() {
			target := buildTestMessage(*chat)
			target.ID = tt.name
			target.LocalChatID = tt.localChatID
			target.ContentType = tt.contentType
			s.Require().NoError(s.m.persistence.SaveMessages([]*common.Message{target}))

			reply := buildTestMessage(*chat)
			reply.ResponseTo = target.ID
			_, err := s.m.SendChatMessage(context.Background(), reply)
			s.Require().ErrorIs(err, ErrInvalidResponseTo)
		})
	}
}

func (s *MessengerReplySuite) TestHistoryDoesNotResolveInvalidReplyTargets() {
	chat := CreatePublicChat(statusChatID, s.m.getTimesource())
	for _, tt := range []struct {
		name        string
		localChatID string
		contentType protobuf.ChatMessage_ContentType
	}{
		{"other chat", "other-chat", protobuf.ChatMessage_TEXT_PLAIN},
		{"system message", chat.ID, protobuf.ChatMessage_SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP},
	} {
		s.Run(tt.name, func() {
			target := buildTestMessage(*chat)
			target.ID = tt.name + "-target"
			target.LocalChatID = tt.localChatID
			target.ContentType = tt.contentType

			reply := buildTestMessage(*chat)
			reply.ID = tt.name + "-reply"
			reply.ResponseTo = target.ID
			reply.Replied = true
			s.Require().NoError(s.m.persistence.SaveMessages([]*common.Message{target, reply}))

			messages, err := s.m.pullMessagesAndResponsesFromDB([]*common.Message{reply})
			s.Require().NoError(err)
			s.Require().Len(messages, 1)
			s.Require().Equal(reply.ID, messages[0].ID)
			s.Require().Equal(reply.Text, messages[0].Text)
			s.Require().Nil(messages[0].QuotedMessage)
			s.Require().False(messages[0].Replied)
		})
	}
}

func (s *MessengerReplySuite) TestHistoryResolvesTargetThatArrivedLater() {
	chat := CreatePublicChat(statusChatID, s.m.getTimesource())
	reply := buildTestMessage(*chat)
	reply.ID = "reply"
	reply.ResponseTo = "original"
	s.Require().NoError(s.m.persistence.SaveMessages([]*common.Message{reply}))

	stored, err := s.m.persistence.MessageByID(reply.ID)
	s.Require().NoError(err)
	s.Require().Nil(stored.QuotedMessage)

	target := buildTestMessage(*chat)
	target.ID = reply.ResponseTo
	s.Require().NoError(s.m.persistence.SaveMessages([]*common.Message{target}))

	stored, err = s.m.persistence.MessageByID(reply.ID)
	s.Require().NoError(err)
	s.Require().NotNil(stored.QuotedMessage)
	s.Require().Equal(target.ID, stored.QuotedMessage.ID)
}
