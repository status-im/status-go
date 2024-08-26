package protocol

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/protocol/v1"
	"go.uber.org/zap"
	"time"
)

func TestMessengerEmojiSuite(t *testing.T) {
	suite.Run(t, new(MessengerEmojiSuite))
}

type MessengerEmojiSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerEmojiSuite) handleRawMessage(m *Messenger, sender *Messenger, rawMessage *common.RawMessage) *MessengerResponse {
	wrappedMessage, err := protocol.WrapMessageV1(rawMessage.Payload, rawMessage.MessageType, sender.identity)
	s.Require().NoError(err)

	messageID := protocol.MessageID(sender.IdentityPublicKey(), wrappedMessage)

	statusMessage := &protocol.StatusMessage{
		ApplicationLayer: protocol.ApplicationLayer{
			Payload:   rawMessage.Payload,
			ID:        protocol.MessageID(sender.IdentityPublicKey(), wrappedMessage),
			SigPubKey: sender.IdentityPublicKey(),
			Type:      rawMessage.MessageType,
		},
		TransportLayer: protocol.TransportLayer{
			Message: &types.Message{
				Timestamp: uint32(sender.getTimesource().GetCurrentTime()),
			},
		},
	}

	messageState := m.buildMessageState()
	messageState.CurrentMessageState = &CurrentMessageState{
		MessageID:        messageID.String(),
		WhisperTimestamp: sender.getTimesource().GetCurrentTime(), // uint64(statusMessage.TransportLayer.Message.Timestamp) * 1000,
		Contact:          sender.selfContact,
		PublicKey:        statusMessage.SigPubKey(),
		StatusMessage:    statusMessage,
	}
	err = m.dispatchToHandler(messageState, statusMessage.ApplicationLayer.Payload, statusMessage, transport.Filter{}, false)
	s.Require().NoError(err)

	response, err := m.saveDataAndPrepareResponse(messageState)
	s.Require().NoError(err)

	return response
}

func (s *MessengerEmojiSuite) TestSendEmoji() {
	start := time.Now()

	alice := s.m
	alice.account = &multiaccounts.Account{KeyUID: "0xdeadbeef"}
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	bob, err := newMessengerWithKey(s.shh, key, s.logger, nil)
	s.Require().NoError(err)
	defer func() {
		s.logger.Debug("<<< 5", zap.Duration("duration", time.Since(start)))
		TearDownMessenger(&s.Suite, bob)
		s.logger.Debug("<<< 6", zap.Duration("duration", time.Since(start)))
	}()

	s.logger.Debug("<<< 1", zap.Duration("duration", time.Since(start)))

	chatID := statusChatID

	chat := CreatePublicChat(chatID, alice.transport)

	err = alice.SaveChat(chat)
	s.Require().NoError(err)

	_, err = alice.Join(chat)
	s.Require().NoError(err)

	err = bob.SaveChat(chat)
	s.Require().NoError(err)

	_, err = bob.Join(chat)
	s.Require().NoError(err)

	// Send chat message from alice to bob

	aliceDispatchedMessages := make([]common.RawMessage, 0)
	alice.dispatchMessageTestCallback = func(msg common.RawMessage) {
		aliceDispatchedMessages = append(aliceDispatchedMessages, msg)
	}

	message := buildTestMessage(*chat)
	_, err = alice.SendChatMessage(context.Background(), message)
	s.NoError(err)

	//bob.handle
	s.Require().Len(aliceDispatchedMessages, 1)
	rawMessage := aliceDispatchedMessages[0]
	response := s.handleRawMessage(bob, alice, &rawMessage)

	s.logger.Debug("<<< 2", zap.Duration("duration", time.Since(start)))

	//// Wait for message to arrive to bob
	//response, err := WaitOnMessengerResponse(
	//	bob,
	//	func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
	//	"no messages",
	//)
	//s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)

	messageID := response.Messages()[0].ID

	// Respond with an emoji, donald trump style

	bobDispatchedMessages := make([]common.RawMessage, 0)
	bob.dispatchMessageTestCallback = func(msg common.RawMessage) {
		bobDispatchedMessages = append(bobDispatchedMessages, msg)
	}

	response, err = bob.SendEmojiReaction(context.Background(), chat.ID, messageID, protobuf.EmojiReaction_SAD)
	s.Require().NoError(err)
	s.Require().Len(response.EmojiReactions(), 1)

	s.Require().Len(bobDispatchedMessages, 1)
	rawMessage = bobDispatchedMessages[0]
	bobDispatchedMessages = make([]common.RawMessage, 0)

	s.Require().Equal(chatID, rawMessage.LocalChatID)
	s.Require().Equal(common.ResendTypeNone, rawMessage.ResendType)
	s.Require().Equal(protobuf.ApplicationMetadataMessage_EMOJI_REACTION, rawMessage.MessageType)

	response = s.handleRawMessage(alice, bob, &rawMessage)
	s.Require().Len(response.EmojiReactions(), 1)

	//// Wait for the emoji to arrive to alice
	//response, err = WaitOnMessengerResponse(
	//	alice,
	//	func(r *MessengerResponse) bool { return len(r.EmojiReactions()) == 1 },
	//	"no emoji",
	//)
	//s.Require().NoError(err)

	emojiID := response.EmojiReactions()[0].ID()
	s.Require().Len(response.EmojiReactions(), 1)
	s.Require().Equal(response.EmojiReactions()[0].ID(), emojiID)
	s.Require().Equal(response.EmojiReactions()[0].Type, protobuf.EmojiReaction_SAD)

	s.logger.Debug("<<< 3", zap.Duration("duration", time.Since(start)))

	// Retract the emoji
	response, err = bob.SendEmojiReactionRetraction(context.Background(), emojiID)
	s.Require().NoError(err)
	s.Require().Len(response.EmojiReactions(), 1)
	s.Require().True(response.EmojiReactions()[0].Retracted)

	s.Require().Len(bobDispatchedMessages, 1)
	rawMessage = bobDispatchedMessages[0]
	response = s.handleRawMessage(alice, bob, &rawMessage)
	//// Wait for the emoji to arrive to alice
	//response, err = WaitOnMessengerResponse(
	//	alice,
	//	func(r *MessengerResponse) bool { return len(r.EmojiReactions()) == 1 },
	//	"no emoji",
	//)
	//s.Require().NoError(err)

	s.Require().Len(response.EmojiReactions(), 1)
	s.Require().Equal(response.EmojiReactions()[0].ID(), emojiID)
	s.Require().Equal(response.EmojiReactions()[0].Type, protobuf.EmojiReaction_SAD)
	s.Require().True(response.EmojiReactions()[0].Retracted)

	s.logger.Debug("<<< 4", zap.Duration("duration", time.Since(start)))
}

func (s *MessengerEmojiSuite) TestEmojiPrivateGroup() {
	bob := s.m
	alice := s.newMessenger()
	defer TearDownMessenger(&s.Suite, alice)
	response, err := bob.CreateGroupChatWithMembers(context.Background(), "test", []string{})
	s.NoError(err)

	s.Require().NoError(makeMutualContact(bob, &alice.identity.PublicKey))

	chat := response.Chats()[0]
	members := []string{types.EncodeHex(crypto.FromECDSAPub(&alice.identity.PublicKey))}
	_, err = bob.AddMembersToGroupChat(context.Background(), chat.ID, members)
	s.NoError(err)

	// Retrieve their messages so that the chat is created
	_, err = WaitOnMessengerResponse(
		alice,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"chat invitation not received",
	)
	s.Require().NoError(err)

	_, err = alice.ConfirmJoiningGroup(context.Background(), chat.ID)
	s.NoError(err)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"no joining group event received",
	)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(*chat)
	_, err = bob.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		alice,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no message received",
	)
	s.Require().NoError(err)
	messageID := response.Messages()[0].ID

	_, err = bob.SendEmojiReaction(context.Background(), chat.ID, messageID, protobuf.EmojiReaction_SAD)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		alice,
		func(r *MessengerResponse) bool { return len(r.EmojiReactions()) == 1 },
		"no emoji reaction received",
	)
	s.Require().NoError(err)
}

func (s *MessengerEmojiSuite) TestCompressedKeyReturnedWithEmoji() {
	emojiReaction := NewEmojiReaction()
	id, err := crypto.GenerateKey()
	s.Require().NoError(err)

	emojiReaction.From = common.PubkeyToHex(&id.PublicKey)
	emojiReaction.LocalChatID = testPublicChatID
	encodedReaction, err := json.Marshal(emojiReaction)
	s.Require().NoError(err)

	// Check that compressedKey and emojiHash exists
	s.Require().True(strings.Contains(string(encodedReaction), "compressedKey\":\"zQ"))
	s.Require().True(strings.Contains(string(encodedReaction), "emojiHash"))
}
