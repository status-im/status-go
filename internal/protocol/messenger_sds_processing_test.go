package protocol

import (
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/waku-org/sds-go-bindings/sds"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	v1protocol "github.com/status-im/status-go/internal/protocol/v1"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
)

func TestMessengerSDSProcessingSuite(t *testing.T) {
	suite.Run(t, new(MessengerSDSProcessingSuite))
}

type MessengerSDSProcessingSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerSDSProcessingSuite) TestNonChatReplayAfterRestart() {
	for _, failure := range []string{"none", "handler", "save", "marker", "legacy", "batch"} {
		s.Run(failure, func() {
			chatID := "sds-processing-" + failure
			chat := CreatePublicChat(chatID, s.m.getTimesource())
			s.Require().NoError(s.m.SaveChat(chat))
			key, err := crypto.GenerateKey()
			s.Require().NoError(err)
			body, err := proto.Marshal(&protobuf.EmojiReaction{
				Clock:     s.m.getTimesource().GetCurrentTime(),
				MessageId: "referenced-message", ChatId: chatID,
				Type:        protobuf.EmojiReaction_SAD,
				MessageType: protobuf.MessageType_PUBLIC_GROUP,
			})
			s.Require().NoError(err)
			application, err := v1protocol.WrapIntoAppLayerMessage(body, protobuf.ApplicationMetadataMessage_EMOJI_REACTION, key)
			s.Require().NoError(err)
			id := messagingtypes.MessageID(&key.PublicKey, application).String()
			sender, err := sds.NewReliabilityManager(zap.NewNop(), nil)
			s.Require().NoError(err)
			wrapped, err := sender.WrapOutgoingMessage(application, sds.MessageID(cryptotypes.EncodeHex(crypto.Keccak256(application))), chatID)
			s.Require().NoError(err)
			s.Require().NoError(sender.Cleanup())
			if failure == "legacy" {
				wrapped = application
			}

			core, logs := observer.New(zap.DebugLevel)
			oldLogger := s.m.logger
			s.m.logger = zap.New(core)
			defer func() { s.m.logger = oldLogger }()
			receiver := s.m
			restart := func() {
				receiver, err = newRunningTestMessenger(s.T(), s.messagingEnv, testMessengerConfig{
					privateKey:  s.m.identity,
					appDatabase: s.m.database,
					logger:      zap.New(core),
				})
				s.Require().NoError(err)
			}
			attempt := 0
			deliver := func() (*MessengerResponse, error) {
				attempt++
				hashSequence := attempt
				if (failure == "handler" || failure == "save" || failure == "marker") && attempt == 2 {
					hashSequence = 1
				}
				envelopes := []*messagingtypes.ReceivedMessage{{
					Sig:     crypto.FromECDSAPub(&key.PublicKey),
					Hash:    crypto.Keccak256([]byte(fmt.Sprintf("%s-%d", chatID, hashSequence))),
					Payload: wrapped,
				}}
				if failure == "batch" {
					envelopes = append(envelopes, &messagingtypes.ReceivedMessage{
						Sig:     crypto.FromECDSAPub(&key.PublicKey),
						Hash:    crypto.Keccak256([]byte(fmt.Sprintf("%s-%d-duplicate", chatID, attempt))),
						Payload: wrapped,
					})
				}
				return receiver.handleRetrievedMessages(map[messagingtypes.ChatFilter][]*messagingtypes.ReceivedMessage{
					{}: envelopes,
				}, false, false)
			}
			if failure == "handler" || failure == "save" || failure == "marker" {
				table, operation := "emoji_reactions", "INSERT"
				if failure == "save" {
					table, operation = "chats", "INSERT"
				} else if failure == "marker" {
					table = "processed_sds_messages"
				}
				_, err := s.m.database.Exec(fmt.Sprintf(
					"CREATE TRIGGER fail_sds_processing BEFORE %s ON %s BEGIN SELECT RAISE(ABORT, 'injected processing failure'); END",
					operation, table))
				s.Require().NoError(err)
				_, deliveryErr := deliver()
				if failure != "handler" {
					s.Require().Error(deliveryErr)
				} else {
					s.Require().NoError(deliveryErr)
				}
				processed, err := s.m.persistence.IsSDSMessageProcessed(id)
				s.Require().NoError(err)
				s.Require().False(processed)
				_, err = s.m.database.Exec("DROP TRIGGER fail_sds_processing")
				s.Require().NoError(err)
				restart()
			}

			response, err := deliver()
			s.Require().NoError(err)
			if failure != "save" && failure != "marker" {
				s.Require().Len(response.EmojiReactions(), 1)
			}
			processed, err := s.m.persistence.IsSDSMessageProcessed(id)
			s.Require().NoError(err)
			s.Require().Equal(failure != "legacy", processed)
			exists, err := s.m.messageExists(id, map[string]bool{})
			s.Require().NoError(err)
			s.Require().False(exists, "the user_messages duplicate lookup must not mask this test")
			handlerCalls := logs.FilterMessage("handling EmojiReaction").Len()
			if failure == "batch" {
				s.Require().Equal(1, handlerCalls)
			}

			restart()
			response, err = deliver()
			s.Require().NoError(err)
			s.Require().Empty(response.EmojiReactions())
			if failure == "legacy" {
				handlerCalls++
			}
			s.Require().Equal(handlerCalls, logs.FilterMessage("handling EmojiReaction").Len())
		})
	}
}

func (s *MessengerSDSProcessingSuite) TestCommunityDescriptionRetainsReplayException() {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	application, err := v1protocol.WrapIntoAppLayerMessage([]byte{0xff}, protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, key)
	s.Require().NoError(err)
	messages := common.NewStatusMessages([]*messagingtypes.Message{{
		SDSWrapped:      true,
		TransportLayer:  messagingtypes.TransportLayer{Message: &messagingtypes.ReceivedMessage{}},
		EncryptionLayer: messagingtypes.EncryptionLayer{Payload: application},
	}})
	id := cryptotypes.EncodeHex(messages[0].ApplicationLayer.ID)
	s.Require().NoError(s.m.persistence.MarkSDSMessagesProcessed(map[string]bool{id: true}))
	// Malformed protobuf must reach dispatch rather than being suppressed by
	// the marker: descriptions can be retried after keys become available.
	err = s.m.processStatusMessage(nil, s.m.buildMessageState(), messages[0], messagingtypes.ChatFilter{}, false, zap.NewNop())
	s.Require().Error(err)
}

func TestProcessedSDSMessagePersistence(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	persistence := newSQLitePersistence(db)
	processed, err := persistence.IsSDSMessageProcessed("message")
	require.NoError(t, err)
	require.False(t, processed)
	require.NoError(t, persistence.MarkSDSMessagesProcessed(map[string]bool{"message": true}))
	require.NoError(t, persistence.MarkSDSMessagesProcessed(map[string]bool{"message": true}))
	processed, err = newSQLitePersistence(db).IsSDSMessageProcessed("message")
	require.NoError(t, err)
	require.True(t, processed)

	_, err = db.Exec(`CREATE TRIGGER fail_sds_marker BEFORE INSERT ON processed_sds_messages
		BEGIN SELECT RAISE(ABORT, 'injected marker failure'); END`)
	require.NoError(t, err)
	require.Error(t, persistence.MarkSDSMessagesProcessed(map[string]bool{"retryable": true}))
	processed, err = persistence.IsSDSMessageProcessed("retryable")
	require.NoError(t, err)
	require.False(t, processed)
}

func (s *MessengerSDSProcessingSuite) TestInvalidApplicationMessageRemainsRetryable() {
	sender, err := sds.NewReliabilityManager(zap.NewNop(), nil)
	s.Require().NoError(err)
	wrapped, err := sender.WrapOutgoingMessage([]byte{0xff}, "malformed-application", "sds-malformed")
	s.Require().NoError(err)
	s.Require().NoError(sender.Cleanup())
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	core, logs := observer.New(zap.DebugLevel)
	oldLogger := s.m.logger
	s.m.logger = zap.New(core)
	defer func() { s.m.logger = oldLogger }()
	envelope := &messagingtypes.ReceivedMessage{
		Sig:     crypto.FromECDSAPub(&key.PublicKey),
		Hash:    crypto.Keccak256(wrapped),
		Payload: wrapped,
	}
	for i := 0; i < 2; i++ {
		response, err := s.m.handleRetrievedMessages(map[messagingtypes.ChatFilter][]*messagingtypes.ReceivedMessage{
			{}: {envelope},
		}, false, false)
		s.Require().NoError(err)
		s.Require().Empty(response.Messages())
	}
	s.Require().Equal(2, logs.FilterMessage("failed to process application message").Len(),
		"the malformed envelope must not be transport-confirmed after its first delivery")
}
