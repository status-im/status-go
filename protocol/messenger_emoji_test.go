package protocol

import (
	"context"
	"crypto/ecdsa"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerEmojiSuite(t *testing.T) {
	suite.Run(t, new(MessengerEmojiSuite))
}

type MessengerEmojiSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger

	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh types.Waku

	tmpFiles []*os.File // files to clean up
	logger   *zap.Logger
}

func (s *MessengerEmojiSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start(nil))

	s.m = s.newMessenger(s.shh)
	s.privateKey = s.m.identity
}

func (s *MessengerEmojiSuite) newMessengerWithKey(shh types.Waku, privateKey *ecdsa.PrivateKey) *Messenger {
	tmpFile, err := ioutil.TempFile("", "")
	s.Require().NoError(err)

	options := []Option{
		WithCustomLogger(s.logger),
		WithMessagesPersistenceEnabled(),
		WithDatabaseConfig(tmpFile.Name(), "some-key"),
		WithDatasync(),
	}
	installationID := uuid.New().String()
	m, err := NewMessenger(
		privateKey,
		&testNode{shh: shh},
		installationID,
		options...,
	)
	s.Require().NoError(err)

	err = m.Init()
	s.Require().NoError(err)

	s.tmpFiles = append(s.tmpFiles, tmpFile)

	return m
}

func (s *MessengerEmojiSuite) newMessenger(shh types.Waku) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return s.newMessengerWithKey(s.shh, privateKey)
}

func (s *MessengerEmojiSuite) TestSendEmoji() {
	alice := s.m
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	bob := s.newMessengerWithKey(s.shh, key)

	chatID := "status"

	chat := CreatePublicChat(chatID, alice.transport)

	err = alice.SaveChat(&chat)
	s.Require().NoError(err)

	err = alice.Join(chat)
	s.Require().NoError(err)

	err = bob.SaveChat(&chat)
	s.Require().NoError(err)

	err = bob.Join(chat)
	s.Require().NoError(err)

	// Send chat message from bob to alice

	message := buildTestMessage(chat)
	_, err = alice.SendChatMessage(context.Background(), message)
	s.NoError(err)

	// Wait for message to arrive to bob
	response, err := WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool { return len(r.Messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().Len(response.Messages, 1)

	messageID := response.Messages[0].ID

	// Respond with an emoji, donald trump style

	response, err = bob.SendEmojiReaction(context.Background(), chat.ID, messageID, protobuf.EmojiReaction_SAD)
	s.Require().NoError(err)
	s.Require().Len(response.EmojiReactions, 1)

	emojiID := response.EmojiReactions[0].ID()

	// Wait for the emoji to arrive to alice
	response, err = WaitOnMessengerResponse(
		alice,
		func(r *MessengerResponse) bool { return len(r.EmojiReactions) > 0 },
		"no emoji",
	)
	s.Require().NoError(err)

	s.Require().Len(response.EmojiReactions, 1)
	s.Require().Equal(response.EmojiReactions[0].ID(), emojiID)
	s.Require().Equal(response.EmojiReactions[0].Type, protobuf.EmojiReaction_SAD)

	// Retract the emoji
	response, err = bob.SendEmojiReactionRetraction(context.Background(), emojiID)
	s.Require().NoError(err)
	s.Require().Len(response.EmojiReactions, 1)
	s.Require().True(response.EmojiReactions[0].Retracted)

	// Wait for the emoji to arrive to alice
	response, err = WaitOnMessengerResponse(
		alice,
		func(r *MessengerResponse) bool { return len(r.EmojiReactions) > 0 },
		"no emoji",
	)
	s.Require().NoError(err)

	s.Require().Len(response.EmojiReactions, 1)
	s.Require().Equal(response.EmojiReactions[0].ID(), emojiID)
	s.Require().Equal(response.EmojiReactions[0].Type, protobuf.EmojiReaction_SAD)
	s.Require().True(response.EmojiReactions[0].Retracted)
}
