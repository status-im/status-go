package protocol

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/contacts"
	"github.com/status-im/status-go/protocol/protobuf"
)

func TestMessengerEmojiSuite(t *testing.T) {
	suite.Run(t, new(MessengerEmojiSuite))
}

type MessengerEmojiSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerEmojiSuite) TestSendEmoji() {
	alice := s.m
	alice.account = &multiaccounts.Account{KeyUID: "0xdeadbeef"}

	bob := s.newMessenger()
	defer TearDownMessenger(&s.Suite, bob)

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

	messageID := response.Messages()[0].ID

	// Respond with an emoji
	response, err = bob.SendEmojiReaction(context.Background(), chat.ID, messageID, protobuf.EmojiReaction_SAD, "")
	s.Require().NoError(err)
	s.Require().Len(response.EmojiReactions(), 1)

	emojiID := response.EmojiReactions()[0].ID()

	// Try sending a non-emoji reaction
	_, err = bob.SendEmojiReaction(context.Background(), chat.ID, messageID, protobuf.EmojiReaction_UNKNOWN_EMOJI_REACTION_TYPE, "xD")
	s.Require().Error(err)

	// Wait for the emoji to arrive to alice
	response, err = WaitOnMessengerResponse(
		alice,
		func(r *MessengerResponse) bool { return len(r.EmojiReactions()) == 1 },
		"no emoji",
	)
	s.Require().NoError(err)

	s.Require().Len(response.EmojiReactions(), 1)
	s.Require().Equal(response.EmojiReactions()[0].ID(), emojiID)
	s.Require().Equal(response.EmojiReactions()[0].Emoji, "1f622")

	// Retract the emoji
	response, err = bob.SendEmojiReactionRetraction(context.Background(), emojiID)
	s.Require().NoError(err)
	s.Require().Len(response.EmojiReactions(), 1)
	s.Require().True(response.EmojiReactions()[0].Retracted)

	// Wait for the emoji to arrive to alice
	response, err = WaitOnMessengerResponse(
		alice,
		func(r *MessengerResponse) bool { return len(r.EmojiReactions()) == 1 },
		"no emoji",
	)
	s.Require().NoError(err)

	s.Require().Len(response.EmojiReactions(), 1)
	s.Require().Equal(response.EmojiReactions()[0].ID(), emojiID)
	s.Require().True(response.EmojiReactions()[0].Retracted)
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

	_, err = bob.SendEmojiReaction(context.Background(), chat.ID, messageID, protobuf.EmojiReaction_SAD, "")
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

	emojiReaction.From = crypto.PubkeyToHex(&id.PublicKey)
	emojiReaction.LocalChatID = testPublicChatID
	encodedReaction, err := json.Marshal(emojiReaction)
	s.Require().NoError(err)

	// Check that compressedKey and emojiHash exists
	s.Require().True(strings.Contains(string(encodedReaction), "compressedKey\":\"zQ"))
	s.Require().True(strings.Contains(string(encodedReaction), "emojiHash"))
}

func (s *MessengerEmojiSuite) TestMaxEmojiReactionsPerMessage() {
	alice := s.m
	alice.account = &multiaccounts.Account{KeyUID: "0xdeadbeef"}

	bob := s.newMessenger()
	defer TearDownMessenger(&s.Suite, bob)

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

	messageID := response.Messages()[0].ID

	// Respond with the max amount of emojis
	emojis := []string{
		"1f600",
		"1f913",
		"1f604",
		"1f601",
		"1f606",
		"1f605",
		"1f923",
		"1f602",
		"1f979",
		"1f642",
		"1f643",
		"1f609",
		"1f60a",
		"1f607",
		"1f970",
		"1f60d",
		"1f929",
		"1f618",
		"1f617",
		"263a-fe0f",
	}
	for _, emoji := range emojis {
		response, err = bob.SendEmojiReaction(context.Background(), chat.ID, messageID, protobuf.EmojiReaction_UNKNOWN_EMOJI_REACTION_TYPE, emoji)
		s.Require().NoError(err)
		s.Require().Len(response.EmojiReactions(), 1)
	}

	// Try sending one more (should fail)
	_, err = bob.SendEmojiReaction(context.Background(), chat.ID, messageID, protobuf.EmojiReaction_UNKNOWN_EMOJI_REACTION_TYPE, "1f60b")
	s.Require().Error(err)
	s.Require().Equal(ErrTooManyEmojiReactionsForMessage, err)

	messageState := bob.buildMessageState()
	messageState.CurrentMessageState = &CurrentMessageState{
		Contact: &contacts.Contact{
			ID: crypto.PubkeyToHex(&alice.identity.PublicKey),
		},
	}

	// Try handling a new emoji (like if it was sent by Alice) (should fail)
	err = bob.HandleEmojiReaction(
		context.Background(),
		messageState,
		&protobuf.EmojiReaction{
			MessageId:   messageID,
			ChatId:      chat.ID,
			Emoji:       "1f60b",
			Type:        protobuf.EmojiReaction_UNKNOWN_EMOJI_REACTION_TYPE,
			Clock:       123,
			MessageType: protobuf.MessageType_PUBLIC_GROUP,
		},
		&common.StatusMessage{},
	)
	s.Require().Error(err)
	s.Require().Equal(ErrTooManyEmojiReactionsForMessage, err)

	// Sending an existing emoji should work
	response, err = alice.SendEmojiReaction(context.Background(), chat.ID, messageID, protobuf.EmojiReaction_UNKNOWN_EMOJI_REACTION_TYPE, "1f600")
	s.Require().NoError(err)
	s.Require().Len(response.EmojiReactions(), 1)

	_, err = WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool { return len(r.EmojiReactions()) == 1 },
		"no emoji",
	)
	s.Require().NoError(err)

}
