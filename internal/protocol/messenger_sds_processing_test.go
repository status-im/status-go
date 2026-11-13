package protocol

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/waku-org/sds-go-bindings/sds"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/requests"
	v1protocol "github.com/status-im/status-go/internal/protocol/v1"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
)

func TestMessengerSDSProcessingSuite(t *testing.T) {
	suite.Run(t, new(MessengerSDSProcessingSuite))
}

type MessengerSDSProcessingSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerSDSProcessingSuite) sdsCommunity() (*communities.Community, *Chat) {
	response, err := s.m.CreateCommunity(&requests.CreateCommunity{
		Membership: protobuf.CommunityPermissions_AUTO_ACCEPT,
		Name:       "SDS community", Color: "#ffffff", Description: "SDS validation",
	}, true)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	return response.Communities()[0], response.Chats()[0]
}

func (s *MessengerSDSProcessingSuite) sdsApplication(message proto.Message, messageType protobuf.ApplicationMetadataMessage_Type, key *ecdsa.PrivateKey) []byte {
	body, err := proto.Marshal(message)
	s.Require().NoError(err)
	application, err := v1protocol.WrapIntoAppLayerMessage(body, messageType, key)
	s.Require().NoError(err)
	return application
}

func (s *MessengerSDSProcessingSuite) TestSDSCommunityPostingPermissions() {
	community, chat := s.sdsCommunity()
	_, err := community.UpsertTokenPermission(&protobuf.CommunityTokenPermission{
		Id: "posting", Type: protobuf.CommunityTokenPermission_CAN_VIEW_AND_POST_CHANNEL,
		ChatIds: []string{chat.ID},
	})
	s.Require().NoError(err)
	viewer, err := crypto.GenerateKey()
	s.Require().NoError(err)
	member, err := crypto.GenerateKey()
	s.Require().NoError(err)
	banned, err := crypto.GenerateKey()
	s.Require().NoError(err)
	outsider, err := crypto.GenerateKey()
	s.Require().NoError(err)
	for _, key := range []*ecdsa.PrivateKey{viewer, member, banned} {
		_, err = community.AddMember(&key.PublicKey, nil, 0)
		s.Require().NoError(err)
		role := protobuf.CommunityMember_CHANNEL_ROLE_POSTER
		if key == viewer {
			role = protobuf.CommunityMember_CHANNEL_ROLE_VIEWER
		}
		_, err = community.AddMemberToChat(chat.CommunityChatID(), &key.PublicKey, nil, role)
		s.Require().NoError(err)
	}
	community.Description().Chats[chat.CommunityChatID()].ViewersCanPostReactions = true
	_, err = community.BanUserFromCommunity(&banned.PublicKey, &protobuf.CommunityBanInfo{})
	s.Require().NoError(err)
	s.Require().NoError(s.m.communitiesManager.SaveCommunity(community))
	channelID := reliability.BuildChannelID(community.IDString(), community.UniversalChatID())
	validator := communitySDSValidator{messenger: s.m}
	for _, message := range []struct {
		kind protobuf.ApplicationMetadataMessage_Type
		body proto.Message
	}{
		{protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, &protobuf.ChatMessage{ChatId: chat.ID, MessageType: protobuf.MessageType_COMMUNITY_CHAT}},
		{protobuf.ApplicationMetadataMessage_EMOJI_REACTION, &protobuf.EmojiReaction{ChatId: chat.ID, MessageType: protobuf.MessageType_COMMUNITY_CHAT}},
		{protobuf.ApplicationMetadataMessage_PIN_MESSAGE, &protobuf.PinMessage{ChatId: chat.ID, MessageType: protobuf.MessageType_COMMUNITY_CHAT}},
		{protobuf.ApplicationMetadataMessage_EDIT_MESSAGE, &protobuf.EditMessage{ChatId: chat.ID, MessageType: protobuf.MessageType_COMMUNITY_CHAT}},
		{protobuf.ApplicationMetadataMessage_DELETE_MESSAGE, &protobuf.DeleteMessage{ChatId: chat.ID, MessageType: protobuf.MessageType_COMMUNITY_CHAT}},
		{protobuf.ApplicationMetadataMessage_THREAD_METADATA, &protobuf.ThreadMetadata{ChatId: chat.ID, MessageType: protobuf.MessageType_COMMUNITY_CHAT}},
	} {
		for _, sender := range []struct {
			name    string
			key     *ecdsa.PrivateKey
			allowed bool
		}{
			{"owner", s.m.identity, true},
			{"poster", member, message.kind != protobuf.ApplicationMetadataMessage_PIN_MESSAGE},
			{"non-member", outsider, false},
			{"banned", banned, false},
			{"viewer", viewer, message.kind == protobuf.ApplicationMetadataMessage_EMOJI_REACTION || message.kind == protobuf.ApplicationMetadataMessage_THREAD_METADATA},
		} {
			s.Run(message.kind.String()+"/"+sender.name, func() {
				allowed, err := validator.Validate(sds.ValidationMessage{
					ChannelID: channelID,
					Content:   s.sdsApplication(message.body, message.kind, sender.key),
					// Posting permission must use the application signer, not
					// the relay's identity or untrusted SDS sender ID.
					ValidationContext: crypto.FromECDSAPub(&member.PublicKey),
					SenderID:          crypto.PubkeyToHex(&member.PublicKey),
				})
				s.Require().NoError(err)
				s.Require().Equal(sender.allowed, allowed)
			})
		}
	}
	application := s.sdsApplication(&protobuf.ChatMessage{
		ChatId: chat.ID, MessageType: protobuf.MessageType_COMMUNITY_CHAT,
	}, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, member)
	allowed, err := validator.Validate(sds.ValidationMessage{ChannelID: "wrong-channel", Content: application})
	s.Require().NoError(err)
	s.Require().False(allowed)
	allowed, err = validator.Validate(sds.ValidationMessage{ChannelID: channelID, Content: s.sdsApplication(
		&protobuf.ContactUpdate{}, protobuf.ApplicationMetadataMessage_CONTACT_UPDATE, member,
	)})
	s.Require().NoError(err)
	s.Require().False(allowed, "a different application type must not bypass the community gate")
	allowed, err = validator.Validate(sds.ValidationMessage{ChannelID: channelID, Content: s.sdsApplication(
		&protobuf.ChatMessage{ChatId: chat.ID, MessageType: protobuf.MessageType_PUBLIC_GROUP},
		protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, member,
	)})
	s.Require().NoError(err)
	s.Require().False(allowed, "a non-community message type must not bypass the community gate")
	allowed, err = validator.Validate(sds.ValidationMessage{ChannelID: channelID, Content: application})
	s.Require().NoError(err)
	s.Require().True(allowed, "repair forwarding does not require a transport signer for application messages")
	envelope, err := protobuf.Unmarshal(application)
	s.Require().NoError(err)
	envelope.Signature = nil
	unsigned, err := proto.Marshal(envelope)
	s.Require().NoError(err)
	_, err = validator.Validate(sds.ValidationMessage{ChannelID: channelID, Content: unsigned})
	s.Require().Error(err)
	envelope.Signature = []byte{1}
	invalidSignature, err := proto.Marshal(envelope)
	s.Require().NoError(err)
	_, err = validator.Validate(sds.ValidationMessage{ChannelID: channelID, Content: invalidSignature})
	s.Require().Error(err)
	_, err = validator.Validate(sds.ValidationMessage{ChannelID: channelID, Content: []byte{0xff}})
	s.Require().Error(err)
}

func (s *MessengerSDSProcessingSuite) TestSDSUnauthorizedMetadataDoesNotChangeState() {
	community, chat := s.sdsCommunity()
	channelID := reliability.BuildChannelID(community.IDString(), community.UniversalChatID())
	message := common.NewMessage()
	message.ChatId = chat.ID
	message.Text = "pending acknowledgement"
	message.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	_, err := s.m.SendChatMessage(context.Background(), message)
	s.Require().NoError(err)
	persistence := reliability.NewSQLitePersistence(s.m.database)
	before, err := persistence.LoadChannel(channelID)
	s.Require().NoError(err)
	s.Require().NotEmpty(before.MessageHistory)

	attacker, err := sds.NewReliabilityManager(zap.NewNop(), nil)
	s.Require().NoError(err)
	s.T().Cleanup(func() { s.Require().NoError(attacker.Cleanup()) })
	_, err = attacker.UnwrapReceivedMessage(before.MessageHistory[0])
	s.Require().NoError(err)
	_, err = attacker.WrapOutgoingMessage([]byte("fake dependency"), "fake-dependency", channelID)
	s.Require().NoError(err)
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	application := s.sdsApplication(&protobuf.ChatMessage{
		ChatId: chat.ID, MessageType: protobuf.MessageType_COMMUNITY_CHAT, Text: "unauthorized",
	}, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, key)
	id := cryptotypes.EncodeHex(crypto.Keccak256(application))
	wrapped, err := attacker.WrapOutgoingMessage(application, sds.MessageID(id), channelID)
	s.Require().NoError(err)
	hash := crypto.Keccak256(wrapped)
	response, err := s.m.messaging.HandleReceivedMessages(&messagingtypes.ReceivedMessage{
		Sig: crypto.FromECDSAPub(&key.PublicKey), Hash: hash, Payload: wrapped,
	})
	s.Require().NoError(err)
	s.Require().Empty(response.Messages)
	s.Require().Equal([][]byte{hash}, response.ConsumedEnvelopeHashes)
	after, err := persistence.LoadChannel(channelID)
	s.Require().NoError(err)
	s.Require().Equal(before, after, "rejected history and acknowledgement metadata must not reach SDS")
	store := reliability.NewSQLiteRetrievalHintPersistence(s.m.database)
	received, err := store.IsSDSMessageReceived(channelID, id)
	s.Require().NoError(err)
	s.Require().False(received)
}

func (s *MessengerSDSProcessingSuite) TestSDSLegacyControlChannelCompatibility() {
	community, _ := s.sdsCommunity()
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	application := s.sdsApplication(&protobuf.CommunityRequestToJoin{},
		protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN, key)
	peer, err := sds.NewReliabilityManager(zap.NewNop(), nil)
	s.Require().NoError(err)
	s.T().Cleanup(func() { s.Require().NoError(peer.Cleanup()) })
	for _, channel := range []struct {
		id      string
		allowed bool
	}{
		{community.IDString(), true},
		{reliability.BuildChannelID(community.IDString(), community.UniversalChatID()), false},
		{reliability.BuildChannelID(community.IDString(), "other-topic"), false},
	} {
		s.Run(channel.id, func() {
			wrapped, err := peer.WrapOutgoingMessage(application, sds.MessageID("legacy-join"), channel.id)
			s.Require().NoError(err)
			hash := crypto.Keccak256(wrapped)
			response, err := s.m.messaging.HandleReceivedMessages(&messagingtypes.ReceivedMessage{
				Sig: crypto.FromECDSAPub(&key.PublicKey), Hash: hash, Payload: wrapped,
			})
			s.Require().NoError(err)
			if channel.allowed {
				s.Require().Len(response.Messages, 1)
			} else {
				s.Require().Empty(response.Messages)
				s.Require().Equal([][]byte{hash}, response.ConsumedEnvelopeHashes)
				state, err := reliability.NewSQLitePersistence(s.m.database).LoadChannel(channel.id)
				s.Require().NoError(err)
				s.Require().Empty(state.Meta)
				s.Require().Empty(state.MessageHistory)
			}
		})
	}
}

func (s *MessengerSDSProcessingSuite) TestSDSCommunityControlMessagesSkipHistory() {
	community, _ := s.sdsCommunity()
	body, err := proto.Marshal(&protobuf.CommunityRequestToJoin{})
	s.Require().NoError(err)
	_, err = s.m.SendMessageToControlNode(context.Background(), community, &common.RawMessage{
		CommunityID: community.ID(),
		MessageType: protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN,
		Payload:     body,
	})
	s.Require().NoError(err)
	state, err := reliability.NewSQLitePersistence(s.m.database).LoadChannel(community.IDString())
	s.Require().NoError(err)
	s.Require().Empty(state.Meta)
	s.Require().Empty(state.MessageHistory)
}

func (s *MessengerSDSProcessingSuite) TestSDSCommunitySyncMembership() {
	community, _ := s.sdsCommunity()
	channelID := reliability.BuildChannelID(community.IDString(), community.UniversalChatID())
	member, err := crypto.GenerateKey()
	s.Require().NoError(err)
	banned, err := crypto.GenerateKey()
	s.Require().NoError(err)
	outsider, err := crypto.GenerateKey()
	s.Require().NoError(err)
	for _, key := range []*ecdsa.PrivateKey{member, banned} {
		_, err = community.AddMember(&key.PublicKey, nil, 0)
		s.Require().NoError(err)
	}
	_, err = community.BanUserFromCommunity(&banned.PublicKey, &protobuf.CommunityBanInfo{})
	s.Require().NoError(err)
	s.Require().NoError(s.m.communitiesManager.SaveCommunity(community))
	// Metadata-only SDS wire message; senderId deliberately claims the owner.
	sync := protowire.AppendString(protowire.AppendTag(nil, 1, protowire.BytesType), "sync")
	sync = protowire.AppendVarint(protowire.AppendTag(sync, 2, protowire.VarintType), 1)
	sync = protowire.AppendString(protowire.AppendTag(sync, 4, protowire.BytesType), channelID)
	sync = protowire.AppendString(protowire.AppendTag(sync, 7, protowire.BytesType), s.m.IdentityPublicKeyString())
	persistence := reliability.NewSQLitePersistence(s.m.database)
	for _, sender := range []struct {
		name    string
		key     *ecdsa.PrivateKey
		allowed bool
	}{
		{"non-member", outsider, false}, {"banned", banned, false},
		{"member-without-posting-role", member, true}, {"owner", s.m.identity, true},
	} {
		s.Run(sender.name, func() {
			before, err := persistence.LoadChannel(channelID)
			s.Require().NoError(err)
			allowed, err := (communitySDSValidator{messenger: s.m}).Validate(sds.ValidationMessage{
				ChannelID: channelID, ValidationContext: crypto.FromECDSAPub(&sender.key.PublicKey),
			})
			s.Require().NoError(err)
			s.Require().Equal(sender.allowed, allowed)
			hash := crypto.Keccak256(append(crypto.FromECDSAPub(&sender.key.PublicKey), sync...))
			response, err := s.m.messaging.HandleReceivedMessages(&messagingtypes.ReceivedMessage{
				Sig: crypto.FromECDSAPub(&sender.key.PublicKey), Hash: hash, Payload: sync,
			})
			s.Require().NoError(err)
			s.Require().Empty(response.Messages)
			s.Require().Equal([][]byte{hash}, response.ConsumedEnvelopeHashes)
			after, err := persistence.LoadChannel(channelID)
			s.Require().NoError(err)
			if sender.allowed {
				s.Require().NotEmpty(after.Meta)
			} else {
				s.Require().Equal(before, after)
			}
		})
	}
	_, err = (communitySDSValidator{messenger: s.m}).Validate(sds.ValidationMessage{ChannelID: channelID})
	s.Require().Error(err, "sync messages require an authenticated transport sender")
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
	s.Require().Equal(2, logs.FilterMessage("failed to decode messages").Len(),
		"the malformed envelope must not be transport-confirmed after its first delivery")
}
