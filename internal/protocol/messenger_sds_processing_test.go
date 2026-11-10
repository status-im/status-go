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
	"github.com/status-im/status-go/internal/protocol/requests"
	v1protocol "github.com/status-im/status-go/internal/protocol/v1"
	"github.com/status-im/status-go/pkg/messaging"
	"github.com/status-im/status-go/pkg/messaging/layers/transport"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
)

func TestMessengerSDSProcessingSuite(t *testing.T) {
	suite.Run(t, new(MessengerSDSProcessingSuite))
}

type MessengerSDSProcessingSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerSDSProcessingSuite) TestSDSCommunityMessagesRequirePostingPermission() {
	response, err := s.m.CreateCommunity(&requests.CreateCommunity{
		Name:        "SDS permission test",
		Description: "Community for SDS permission tests",
		Color:       "#ffffff",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}, true)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	chat := response.Chats()[0]
	community, err := s.m.communitiesManager.GetByIDString(chat.CommunityID)
	s.Require().NoError(err)
	channelID := chat.CommunityID + ":" + community.UniversalChatID()
	envelopeTopic := messagingtypes.BytesToContentTopic(messaging.ToContentTopic(community.UniversalChatID()))

	unauthorizedKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	for _, test := range []struct {
		name        string
		messageType protobuf.ApplicationMetadataMessage_Type
		payload     proto.Message
	}{
		{
			name:        "chat message",
			messageType: protobuf.ApplicationMetadataMessage_CHAT_MESSAGE,
			payload: &protobuf.ChatMessage{
				ChatId:      chat.ID,
				MessageType: protobuf.MessageType_COMMUNITY_CHAT,
			},
		},
		{
			name:        "emoji reaction",
			messageType: protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
			payload: &protobuf.EmojiReaction{
				ChatId:      chat.ID,
				MessageType: protobuf.MessageType_COMMUNITY_CHAT,
			},
		},
		{
			name:        "pin message",
			messageType: protobuf.ApplicationMetadataMessage_PIN_MESSAGE,
			payload: &protobuf.PinMessage{
				ChatId:      chat.ID,
				MessageType: protobuf.MessageType_COMMUNITY_CHAT,
			},
		},
		{
			name:        "thread metadata",
			messageType: protobuf.ApplicationMetadataMessage_THREAD_METADATA,
			payload: &protobuf.ThreadMetadata{
				ChatId:      chat.ID,
				MessageType: protobuf.MessageType_COMMUNITY_CHAT,
			},
		},
		{
			name:        "edit message",
			messageType: protobuf.ApplicationMetadataMessage_EDIT_MESSAGE,
			payload: &protobuf.EditMessage{
				ChatId:      chat.ID,
				MessageType: protobuf.MessageType_COMMUNITY_CHAT,
			},
		},
		{
			name:        "delete message",
			messageType: protobuf.ApplicationMetadataMessage_DELETE_MESSAGE,
			payload: &protobuf.DeleteMessage{
				ChatId:      chat.ID,
				MessageType: protobuf.MessageType_COMMUNITY_CHAT,
			},
		},
	} {
		s.Run(test.name, func() {
			body, err := proto.Marshal(test.payload)
			s.Require().NoError(err)

			unauthorizedMessage, err := v1protocol.WrapIntoAppLayerMessage(body, test.messageType, unauthorizedKey)
			s.Require().NoError(err)
			disposition, err := s.m.validateSDSCommunityMessage(unauthorizedMessage, channelID, &unauthorizedKey.PublicKey, envelopeTopic)
			s.Require().NoError(err)
			s.Require().Equal(messagingtypes.SDSMessageDispositionReject, disposition, "non-members must be rejected before SDS processing")

			authorizedMessage, err := v1protocol.WrapIntoAppLayerMessage(body, test.messageType, s.m.identity)
			s.Require().NoError(err)
			disposition, err = s.m.validateSDSCommunityMessage(authorizedMessage, channelID, &s.m.identity.PublicKey, envelopeTopic)
			s.Require().NoError(err)
			s.Require().Equal(messagingtypes.SDSMessageDispositionProcess, disposition, "community owner messages should continue through SDS")
		})
	}

	s.Run("member signature cannot be copied into an outsider envelope", func() {
		body, err := proto.Marshal(&protobuf.ChatMessage{
			ChatId:      chat.ID,
			MessageType: protobuf.MessageType_COMMUNITY_CHAT,
		})
		s.Require().NoError(err)
		application, err := v1protocol.WrapIntoAppLayerMessage(body, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, s.m.identity)
		s.Require().NoError(err)
		disposition, err := s.m.validateSDSCommunityMessage(application, channelID, &unauthorizedKey.PublicKey, envelopeTopic)
		s.Require().NoError(err)
		s.Require().Equal(messagingtypes.SDSMessageDispositionReject, disposition)
	})

	s.Run("non-community claimed type cannot bypass community target authorization", func() {
		body, err := proto.Marshal(&protobuf.ChatMessage{
			ChatId:      chat.ID,
			MessageType: protobuf.MessageType_ONE_TO_ONE,
		})
		s.Require().NoError(err)
		application, err := v1protocol.WrapIntoAppLayerMessage(body, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, unauthorizedKey)
		s.Require().NoError(err)
		disposition, err := s.m.validateSDSCommunityMessage(application, channelID, &unauthorizedKey.PublicKey, envelopeTopic)
		s.Require().NoError(err)
		s.Require().Equal(messagingtypes.SDSMessageDispositionReject, disposition)
	})

	s.Run("unsupported application type cannot bypass community target authorization", func() {
		application, err := v1protocol.WrapIntoAppLayerMessage(
			[]byte("unknown"),
			protobuf.ApplicationMetadataMessage_Type(200),
			unauthorizedKey,
		)
		s.Require().NoError(err)
		disposition, err := s.m.validateSDSCommunityMessage(application, channelID, &unauthorizedKey.PublicKey, envelopeTopic)
		s.Require().NoError(err)
		s.Require().Equal(messagingtypes.SDSMessageDispositionReject, disposition)
	})

	s.Run("non-community channel cannot bypass validation on a community topic", func() {
		application, err := v1protocol.WrapIntoAppLayerMessage(
			[]byte("unknown"),
			protobuf.ApplicationMetadataMessage_Type(200),
			unauthorizedKey,
		)
		s.Require().NoError(err)
		disposition, err := s.m.validateSDSCommunityMessage(application, "unrelated-channel", &unauthorizedKey.PublicKey, envelopeTopic)
		s.Require().NoError(err)
		s.Require().Equal(messagingtypes.SDSMessageDispositionReject, disposition)
	})

	s.Run("public chat SDS messages retain normal reliability processing", func() {
		body, err := proto.Marshal(&protobuf.ChatMessage{
			ChatId:      "public-chat",
			MessageType: protobuf.MessageType_PUBLIC_GROUP,
		})
		s.Require().NoError(err)
		application, err := v1protocol.WrapIntoAppLayerMessage(
			body,
			protobuf.ApplicationMetadataMessage_CHAT_MESSAGE,
			unauthorizedKey,
		)
		s.Require().NoError(err)
		publicTopic := messagingtypes.BytesToContentTopic(messaging.ToContentTopic("public-chat"))
		disposition, err := s.m.validateSDSCommunityMessage(
			application,
			"public-chat",
			&unauthorizedKey.PublicKey,
			publicTopic,
		)
		s.Require().NoError(err)
		s.Require().Equal(messagingtypes.SDSMessageDispositionProcess, disposition)
	})

	s.Run("bare community channel routes targeted control requests around SDS", func() {
		communityIDBytes, err := cryptotypes.DecodeHex(chat.CommunityID)
		s.Require().NoError(err)
		controlTopic := messagingtypes.BytesToContentTopic(
			messaging.ToContentTopic(transport.PubkeyToHex(community.PublicKey())[2:]),
		)
		body, err := proto.Marshal(&protobuf.CommunityRequestToJoin{CommunityId: communityIDBytes})
		s.Require().NoError(err)
		application, err := v1protocol.WrapIntoAppLayerMessage(
			body,
			protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN,
			unauthorizedKey,
		)
		s.Require().NoError(err)
		disposition, err := s.m.validateSDSCommunityMessage(application, chat.CommunityID, &unauthorizedKey.PublicKey, controlTopic)
		s.Require().NoError(err)
		s.Require().Equal(messagingtypes.SDSMessageDispositionApplicationOnly, disposition,
			"public join requests must reach application validation without processing SDS metadata")

		otherCommunity, err := crypto.GenerateKey()
		s.Require().NoError(err)
		mismatchedBody, err := proto.Marshal(&protobuf.CommunityRequestToJoin{
			CommunityId: crypto.CompressPubkey(&otherCommunity.PublicKey),
		})
		s.Require().NoError(err)
		mismatchedApplication, err := v1protocol.WrapIntoAppLayerMessage(
			mismatchedBody,
			protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN,
			s.m.identity,
		)
		s.Require().NoError(err)
		disposition, err = s.m.validateSDSCommunityMessage(
			mismatchedApplication,
			chat.CommunityID,
			&s.m.identity.PublicKey,
			controlTopic,
		)
		s.Require().NoError(err)
		s.Require().Equal(messagingtypes.SDSMessageDispositionReject, disposition)

		disposition, err = s.m.validateSDSCommunityMessage(application, chat.CommunityID, &unauthorizedKey.PublicKey, envelopeTopic)
		s.Require().NoError(err)
		s.Require().Equal(messagingtypes.SDSMessageDispositionReject, disposition, "control requests must use their bare-channel topic")

		unsupportedControl, err := v1protocol.WrapIntoAppLayerMessage(
			[]byte("unknown"),
			protobuf.ApplicationMetadataMessage_Type(200),
			s.m.identity,
		)
		s.Require().NoError(err)
		disposition, err = s.m.validateSDSCommunityMessage(
			unsupportedControl,
			chat.CommunityID,
			&s.m.identity.PublicKey,
			controlTopic,
		)
		s.Require().NoError(err)
		s.Require().Equal(messagingtypes.SDSMessageDispositionReject, disposition)
	})

	s.Run("mismatched target and topic are rejected", func() {
		body, err := proto.Marshal(&protobuf.ChatMessage{
			ChatId:      chat.ID,
			MessageType: protobuf.MessageType_COMMUNITY_CHAT,
		})
		s.Require().NoError(err)
		application, err := v1protocol.WrapIntoAppLayerMessage(body, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, s.m.identity)
		s.Require().NoError(err)
		disposition, err := s.m.validateSDSCommunityMessage(application, channelID+":other-topic", &s.m.identity.PublicKey, envelopeTopic)
		s.Require().NoError(err)
		s.Require().Equal(messagingtypes.SDSMessageDispositionReject, disposition)
		disposition, err = s.m.validateSDSCommunityMessage(application, channelID, &s.m.identity.PublicKey, messagingtypes.ContentTopic{})
		s.Require().NoError(err)
		s.Require().Equal(messagingtypes.SDSMessageDispositionReject, disposition)
	})

	s.Run("malformed application content remains retryable", func() {
		disposition, err := s.m.validateSDSCommunityMessage([]byte{0xff}, channelID, &s.m.identity.PublicKey, envelopeTopic)
		s.Require().Error(err)
		s.Require().Equal(messagingtypes.SDSMessageDispositionReject, disposition)
		disposition, err = s.m.validateSDSCommunityMessage(nil, channelID, &s.m.identity.PublicKey, envelopeTopic)
		s.Require().Error(err)
		s.Require().Equal(messagingtypes.SDSMessageDispositionReject, disposition)
	})

	s.Run("community events reach handler authorization without SDS processing", func() {
		communityIDBytes, err := cryptotypes.DecodeHex(chat.CommunityID)
		s.Require().NoError(err)
		body, err := proto.Marshal(&protobuf.CommunityEventsMessage{CommunityId: communityIDBytes})
		s.Require().NoError(err)
		application, err := v1protocol.WrapIntoAppLayerMessage(
			body,
			protobuf.ApplicationMetadataMessage_COMMUNITY_EVENTS_MESSAGE,
			unauthorizedKey,
		)
		s.Require().NoError(err)
		communityTopic := messagingtypes.BytesToContentTopic(messaging.ToContentTopic(chat.CommunityID))
		disposition, err := s.m.validateSDSCommunityMessage(
			application,
			chat.CommunityID+":"+chat.CommunityID,
			&unauthorizedKey.PublicKey,
			communityTopic,
		)
		s.Require().NoError(err)
		s.Require().Equal(messagingtypes.SDSMessageDispositionApplicationOnly, disposition,
			"event sender authorization belongs to the application handler, not SDS")

		description, err := community.MarshaledDescription()
		s.Require().NoError(err)
		application, err = v1protocol.WrapIntoAppLayerMessage(
			description,
			protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION,
			unauthorizedKey,
		)
		s.Require().NoError(err)
		disposition, err = s.m.validateSDSCommunityMessage(
			application,
			chat.CommunityID+":"+chat.CommunityID,
			&unauthorizedKey.PublicKey,
			communityTopic,
		)
		s.Require().NoError(err)
		s.Require().Equal(messagingtypes.SDSMessageDispositionApplicationOnly, disposition,
			"description sender authorization belongs to the application handler, not SDS")

		body, err = proto.Marshal(&protobuf.CommunityEncryptionKeysRequest{CommunityId: communityIDBytes})
		s.Require().NoError(err)
		controlTopic := messagingtypes.BytesToContentTopic(
			messaging.ToContentTopic(transport.PubkeyToHex(community.PublicKey())[2:]),
		)
		application, err = v1protocol.WrapIntoAppLayerMessage(
			body,
			protobuf.ApplicationMetadataMessage_COMMUNITY_ENCRYPTION_KEYS_REQUEST,
			unauthorizedKey,
		)
		s.Require().NoError(err)
		disposition, err = s.m.validateSDSCommunityMessage(
			application,
			chat.CommunityID,
			&unauthorizedKey.PublicKey,
			controlTopic,
		)
		s.Require().NoError(err)
		s.Require().Equal(messagingtypes.SDSMessageDispositionApplicationOnly, disposition,
			"restricted control messages must reach their own authorization handlers without SDS metadata")
	})
}

func (s *MessengerSDSProcessingSuite) TestNonChatReplayAfterRestart() {
	communityResponse, err := s.m.CreateCommunity(&requests.CreateCommunity{
		Name:        "SDS replay test",
		Description: "Community for SDS replay tests",
		Color:       "#ffffff",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}, true)
	s.Require().NoError(err)
	s.Require().Len(communityResponse.Chats(), 1)
	chat := communityResponse.Chats()[0]
	community, err := s.m.communitiesManager.GetByIDString(chat.CommunityID)
	s.Require().NoError(err)
	authorizedMember, err := crypto.GenerateKey()
	s.Require().NoError(err)
	_, err = community.AddMember(&authorizedMember.PublicKey, nil, community.Clock())
	s.Require().NoError(err)
	s.Require().NoError(s.m.communitiesManager.SaveCommunity(community))
	channelID := chat.CommunityID + ":" + community.UniversalChatID()
	envelopeTopic := messagingtypes.BytesToContentTopic(messaging.ToContentTopic(community.UniversalChatID()))

	for _, failure := range []string{"none", "handler", "save", "marker", "legacy", "batch"} {
		s.Run(failure, func() {
			chatID := chat.ID
			body, err := proto.Marshal(&protobuf.EmojiReaction{
				Clock:     s.m.getTimesource().GetCurrentTime(),
				MessageId: "referenced-message-" + failure, ChatId: chatID,
				Type:        protobuf.EmojiReaction_SAD,
				MessageType: protobuf.MessageType_COMMUNITY_CHAT,
			})
			s.Require().NoError(err)
			application, err := v1protocol.WrapIntoAppLayerMessage(body, protobuf.ApplicationMetadataMessage_EMOJI_REACTION, authorizedMember)
			s.Require().NoError(err)
			disposition, err := s.m.validateSDSCommunityMessage(
				application,
				channelID,
				&authorizedMember.PublicKey,
				envelopeTopic,
			)
			s.Require().NoError(err)
			s.Require().Equal(messagingtypes.SDSMessageDispositionProcess, disposition,
				"replay fixture must pass community authorization before SDS processing")
			id := messagingtypes.MessageID(&authorizedMember.PublicKey, application).String()
			sender, err := sds.NewReliabilityManager(zap.NewNop(), nil)
			s.Require().NoError(err)
			wrapped, err := sender.WrapOutgoingMessage(application, sds.MessageID(cryptotypes.EncodeHex(crypto.Keccak256(application))), channelID)
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
					Sig:     crypto.FromECDSAPub(&authorizedMember.PublicKey),
					Hash:    crypto.Keccak256([]byte(fmt.Sprintf("%s-%d", chatID, hashSequence))),
					Topic:   envelopeTopic,
					Payload: wrapped,
				}}
				if failure == "batch" {
					envelopes = append(envelopes, &messagingtypes.ReceivedMessage{
						Sig:     crypto.FromECDSAPub(&authorizedMember.PublicKey),
						Hash:    crypto.Keccak256([]byte(fmt.Sprintf("%s-%d-duplicate", chatID, attempt))),
						Topic:   envelopeTopic,
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
	s.Require().Equal(2, logs.FilterMessage("failed to decode messages").Len(),
		"the malformed envelope must not be transport-confirmed after its first delivery")
}
