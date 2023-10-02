package protocol

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/mutecomm/go-sqlcipher/v4" // require go-sqlcipher that overrides default implementation
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/deprecation"
	coretypes "github.com/status-im/status-go/eth-node/core/types"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	enstypes "github.com/status-im/status-go/eth-node/types/ens"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/server"
)

const (
	testPK              = "0x0424a68f89ba5fcd5e0640c1e1f591d561fa4125ca4e2a43592bc4123eca10ce064e522c254bb83079ba404327f6eafc01ec90a1444331fe769d3f3a7f90b0dde1"
	testPublicChatID    = "super-chat"
	testContract        = "0x314159265dd8dbb310642f98f50c066173c1259b"
	testValue           = "2000"
	testTransactionHash = "0x412a851ac2ae51cad34a56c8a9cfee55d577ac5e1ac71cf488a2f2093a373799"
	newEnsName          = "new-name"
)

func TestMessengerSuite(t *testing.T) {
	suite.Run(t, new(MessengerSuite))
}

type MessengerSuite struct {
	MessengerBaseTestSuite
}

type testNode struct {
	shh types.Waku
}

func (n *testNode) NewENSVerifier(_ *zap.Logger) enstypes.ENSVerifier {
	panic("not implemented")
}

func (n *testNode) AddPeer(_ string) error {
	panic("not implemented")
}

func (n *testNode) RemovePeer(_ string) error {
	panic("not implemented")
}

func (n *testNode) GetWaku(_ interface{}) (types.Waku, error) {
	return n.shh, nil
}

func (n *testNode) GetWakuV2(_ interface{}) (types.Waku, error) {
	return n.shh, nil
}

func (n *testNode) GetWhisper(_ interface{}) (types.Whisper, error) {
	return nil, nil
}

func (n *testNode) PeersCount() int {
	return 1
}

func (s *MessengerSuite) TestInit() {
	testCases := []struct {
		Name         string
		Prep         func()
		AddedFilters int
	}{
		{
			Name:         "no chats and contacts",
			Prep:         func() {},
			AddedFilters: 3,
		},
		{
			Name: "active public chat",
			Prep: func() {
				publicChat := Chat{
					ChatType: ChatTypePublic,
					ID:       "some-public-chat",
					Active:   true,
				}
				err := s.m.SaveChat(&publicChat)
				s.Require().NoError(err)
			},
			AddedFilters: 1,
		},
		{
			Name: "active one-to-one chat",
			Prep: func() {
				key, err := crypto.GenerateKey()
				s.Require().NoError(err)
				privateChat := Chat{
					ID:       types.EncodeHex(crypto.FromECDSAPub(&key.PublicKey)),
					ChatType: ChatTypeOneToOne,
					Active:   true,
				}
				err = s.m.SaveChat(&privateChat)
				s.Require().NoError(err)
			},
			AddedFilters: 1,
		},
		{
			Name: "active group chat",
			Prep: func() {
				key1, err := crypto.GenerateKey()
				s.Require().NoError(err)
				key2, err := crypto.GenerateKey()
				s.Require().NoError(err)
				groupChat := Chat{
					ID:       "some-id",
					ChatType: ChatTypePrivateGroupChat,
					Active:   true,
					Members: []ChatMember{
						{
							ID: types.EncodeHex(crypto.FromECDSAPub(&key1.PublicKey)),
						},
						{
							ID: types.EncodeHex(crypto.FromECDSAPub(&key2.PublicKey)),
						},
					},
				}
				err = s.m.SaveChat(&groupChat)
				s.Require().NoError(err)
			},
			AddedFilters: 2,
		},
		{
			Name: "inactive chat",
			Prep: func() {
				publicChat := Chat{
					ChatType: ChatTypePublic,
					ID:       "some-public-chat-2",
					Active:   false,
				}
				err := s.m.SaveChat(&publicChat)
				s.Require().NoError(err)
			},
			AddedFilters: 0,
		},
		{
			Name: "added contact",
			Prep: func() {
				key, err := crypto.GenerateKey()
				s.Require().NoError(err)
				_, err = s.m.AddContact(context.Background(), &requests.AddContact{ID: types.EncodeHex(crypto.FromECDSAPub(&key.PublicKey))})
				s.Require().NoError(err)
			},
			AddedFilters: deprecation.AddProfileFiltersCount(1),
		},
	}

	expectedFilters := 0
	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			tc.Prep()
			err := s.m.Init()
			s.Require().NoError(err)
			filters := s.m.transport.Filters()
			expectedFilters += tc.AddedFilters
			s.Equal(deprecation.AddTimelineFiltersCount(expectedFilters), len(filters))
		})
	}
}

func buildAudioMessage(s *MessengerSuite, chat Chat) *common.Message {
	clock, timestamp := chat.NextClockAndTimestamp(&testTimeSource{})
	message := common.NewMessage()
	message.Text = "text-input-message"
	message.ChatId = chat.ID
	message.Clock = clock
	message.Timestamp = timestamp
	message.WhisperTimestamp = clock
	message.LocalChatID = chat.ID
	message.MessageType = protobuf.MessageType_PUBLIC_GROUP
	message.ContentType = protobuf.ChatMessage_AUDIO
	message.Payload = &protobuf.ChatMessage_Audio{
		Audio: &protobuf.AudioMessage{
			Type:    1,
			Payload: []byte("some-payload"),
		},
	}

	return message
}

func buildTestMessage(chat Chat) *common.Message {
	clock, timestamp := chat.NextClockAndTimestamp(&testTimeSource{})
	message := common.NewMessage()
	message.Text = "text-input-message"
	message.ChatId = chat.ID
	message.Clock = clock
	message.Timestamp = timestamp
	message.WhisperTimestamp = clock
	message.LocalChatID = chat.ID
	message.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	switch chat.ChatType {
	case ChatTypePublic, ChatTypeProfile:
		message.MessageType = protobuf.MessageType_PUBLIC_GROUP
	case ChatTypeOneToOne:
		message.MessageType = protobuf.MessageType_ONE_TO_ONE
	case ChatTypePrivateGroupChat:
		message.MessageType = protobuf.MessageType_PRIVATE_GROUP
	}

	return message
}

func buildTestGapMessage(chat Chat) *common.Message {
	clock, timestamp := chat.NextClockAndTimestamp(&testTimeSource{})
	message := common.NewMessage()
	message.ChatId = chat.ID
	message.Clock = clock
	message.Timestamp = timestamp
	message.WhisperTimestamp = clock
	message.LocalChatID = chat.ID
	message.ContentType = protobuf.ChatMessage_SYSTEM_MESSAGE_GAP
	switch chat.ChatType {
	case ChatTypePublic, ChatTypeProfile:
		message.MessageType = protobuf.MessageType_PUBLIC_GROUP
	case ChatTypeOneToOne:
		message.MessageType = protobuf.MessageType_ONE_TO_ONE
	case ChatTypePrivateGroupChat:
		message.MessageType = protobuf.MessageType_PRIVATE_GROUP
	}

	return message
}

func (s *MessengerSuite) TestMarkMessagesSeen() {
	chat := CreatePublicChat("test-chat", s.m.transport)
	chat.UnviewedMessagesCount = 2
	chat.UnviewedMentionsCount = 3
	chat.Highlight = true
	err := s.m.SaveChat(chat)
	s.Require().NoError(err)
	inputMessage1 := buildTestMessage(*chat)
	inputMessage1.ID = "1"
	inputMessage1.Seen = false
	inputMessage1.Text = "hey @" + common.PubkeyToHex(&s.m.identity.PublicKey)
	inputMessage1.Mentioned = true
	inputMessage2 := buildTestMessage(*chat)
	inputMessage2.ID = "2"
	inputMessage2.Text = "hey @" + common.PubkeyToHex(&s.m.identity.PublicKey)
	inputMessage2.Mentioned = true
	inputMessage2.Seen = false

	err = s.m.SaveMessages([]*common.Message{inputMessage1, inputMessage2})
	s.Require().NoError(err)

	count, countWithMentions, err := s.m.MarkMessagesSeen(chat.ID, []string{inputMessage1.ID})
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), count)
	s.Require().Equal(uint64(1), countWithMentions)

	// Make sure that if it's not seen, it does not return a count of 1
	count, countWithMentions, err = s.m.MarkMessagesSeen(chat.ID, []string{inputMessage1.ID})
	s.Require().NoError(err)
	s.Require().Equal(uint64(0), count)
	s.Require().Equal(uint64(0), countWithMentions)

	chats := s.m.Chats()
	for _, c := range chats {
		if c.ID == chat.ID {
			s.Require().Equal(uint(1), c.UnviewedMessagesCount)
			s.Require().Equal(uint(1), c.UnviewedMentionsCount)
			s.Require().Equal(false, c.Highlight)
		}
	}
}

func (s *MessengerSuite) TestMarkAllRead() {
	chat := CreatePublicChat("test-chat", s.m.transport)
	chat.UnviewedMessagesCount = 2
	chat.Highlight = true
	err := s.m.SaveChat(chat)
	s.Require().NoError(err)
	inputMessage1 := buildTestMessage(*chat)
	inputMessage1.ID = "1"
	inputMessage1.Seen = false
	inputMessage2 := buildTestMessage(*chat)
	inputMessage2.ID = "2"
	inputMessage2.Seen = false

	err = s.m.SaveMessages([]*common.Message{inputMessage1, inputMessage2})
	s.Require().NoError(err)

	err = s.m.MarkAllRead(chat.ID)
	s.Require().NoError(err)

	chats := s.m.Chats()
	s.Require().Len(chats, deprecation.AddChatsCount(1))
	for idx := range chats {
		if chats[idx].ID == chat.ID {
			s.Require().Equal(uint(0), chats[idx].UnviewedMessagesCount)
			s.Require().Equal(false, chats[idx].Highlight)
		}
	}
}

func (s *MessengerSuite) TestSendPublic() {
	chat := CreatePublicChat("test-chat", s.m.transport)
	chat.LastClockValue = uint64(100000000000000)
	err := s.m.SaveChat(chat)
	s.NoError(err)
	inputMessage := buildTestMessage(*chat)
	response, err := s.m.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	s.Require().Equal(1, len(response.Messages()), "it returns the message")
	outputMessage := response.Messages()[0]

	s.Require().Equal(uint64(100000000000001), outputMessage.Clock, "it correctly sets the clock")
	s.Require().Equal(uint64(100000000000001), chat.LastClockValue, "it correctly sets the last-clock-value")
	s.Require().NotEqual(uint64(0), chat.Timestamp, "it sets the timestamp")
	s.Require().Equal("0x"+hex.EncodeToString(crypto.FromECDSAPub(&s.privateKey.PublicKey)), outputMessage.From, "it sets the From field")
	s.Require().True(outputMessage.Seen, "it marks the message as seen")
	s.Require().Equal(outputMessage.OutgoingStatus, common.OutgoingStatusSending, "it marks the message as sending")
	s.Require().NotEmpty(outputMessage.ID, "it sets the ID field")
	s.Require().Equal(protobuf.MessageType_PUBLIC_GROUP, outputMessage.MessageType)

	savedMessages, _, err := s.m.MessageByChatID(chat.ID, "", 10)
	s.Require().NoError(err)
	s.Require().Equal(1, len(savedMessages), "it saves the message")
}

func (s *MessengerSuite) TestSendProfile() {
	// Early exit to skip testing deprecated code
	if deprecation.ChatProfileDeprecated {
		return
	}

	chat := CreateProfileChat("0x"+hex.EncodeToString(crypto.FromECDSAPub(&s.privateKey.PublicKey)), s.m.transport)
	chat.LastClockValue = uint64(100000000000000)
	err := s.m.SaveChat(chat)
	s.NoError(err)
	inputMessage := buildTestMessage(*chat)
	response, err := s.m.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	s.Require().Equal(1, len(response.Messages()), "it returns the message")
	outputMessage := response.Messages()[0]

	s.Require().Equal(uint64(100000000000001), outputMessage.Clock, "it correctly sets the clock")
	s.Require().Equal(uint64(100000000000001), chat.LastClockValue, "it correctly sets the last-clock-value")
	s.Require().NotEqual(uint64(0), chat.Timestamp, "it sets the timestamp")
	s.Require().Equal("0x"+hex.EncodeToString(crypto.FromECDSAPub(&s.privateKey.PublicKey)), outputMessage.From, "it sets the From field")
	s.Require().Equal(chat.Profile, outputMessage.From, "From equal to chat Profile")
	s.Require().True(outputMessage.Seen, "it marks the message as seen")
	s.Require().Equal(outputMessage.OutgoingStatus, common.OutgoingStatusSending, "it marks the message as sending")
	s.Require().NotEmpty(outputMessage.ID, "it sets the ID field")
	s.Require().Equal(protobuf.MessageType_PUBLIC_GROUP, outputMessage.MessageType)

	savedMessages, _, err := s.m.MessageByChatID(chat.ID, "", 10)
	s.Require().NoError(err)
	s.Require().Equal(1, len(savedMessages), "it saves the message")
}

func (s *MessengerSuite) TestSendPrivateOneToOne() {
	recipientKey, err := crypto.GenerateKey()
	s.NoError(err)
	pkString := hex.EncodeToString(crypto.FromECDSAPub(&recipientKey.PublicKey))
	chat := CreateOneToOneChat(pkString, &recipientKey.PublicKey, s.m.transport)

	inputMessage := common.NewMessage()
	inputMessage.ChatId = chat.ID
	chat.LastClockValue = uint64(100000000000000)
	err = s.m.SaveChat(chat)
	s.NoError(err)
	response, err := s.m.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Equal(1, len(response.Messages()), "it returns the message")
	outputMessage := response.Messages()[0]

	s.Require().Equal(uint64(100000000000001), outputMessage.Clock, "it correctly sets the clock")
	s.Require().Equal(uint64(100000000000001), chat.LastClockValue, "it correctly sets the last-clock-value")

	s.Require().NotEqual(uint64(0), chat.Timestamp, "it sets the timestamp")
	s.Require().Equal("0x"+hex.EncodeToString(crypto.FromECDSAPub(&s.privateKey.PublicKey)), outputMessage.From, "it sets the From field")
	s.Require().True(outputMessage.Seen, "it marks the message as seen")
	s.Require().Equal(outputMessage.OutgoingStatus, common.OutgoingStatusSending, "it marks the message as sending")
	s.Require().NotEmpty(outputMessage.ID, "it sets the ID field")
	s.Require().Equal(protobuf.MessageType_ONE_TO_ONE, outputMessage.MessageType)
}

func (s *MessengerSuite) TestSendPrivateGroup() {
	response, err := s.m.CreateGroupChatWithMembers(context.Background(), "test", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats(), 1)

	chat := response.Chats()[0]
	key, err := crypto.GenerateKey()
	s.NoError(err)

	s.Require().NoError(makeMutualContact(s.m, &key.PublicKey))

	members := []string{"0x" + hex.EncodeToString(crypto.FromECDSAPub(&key.PublicKey))}
	_, err = s.m.AddMembersToGroupChat(context.Background(), chat.ID, members)
	s.NoError(err)

	inputMessage := common.NewMessage()
	inputMessage.ChatId = chat.ID
	chat.LastClockValue = uint64(100000000000000)
	err = s.m.SaveChat(chat)
	s.NoError(err)
	response, err = s.m.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)
	s.Require().Equal(1, len(response.Messages()), "it returns the message")
	outputMessage := response.Messages()[0]

	s.Require().Equal(uint64(100000000000001), outputMessage.Clock, "it correctly sets the clock")
	s.Require().Equal(uint64(100000000000001), chat.LastClockValue, "it correctly sets the last-clock-value")

	s.Require().NotEqual(uint64(0), chat.Timestamp, "it sets the timestamp")
	s.Require().Equal("0x"+hex.EncodeToString(crypto.FromECDSAPub(&s.privateKey.PublicKey)), outputMessage.From, "it sets the From field")
	s.Require().True(outputMessage.Seen, "it marks the message as seen")
	s.Require().Equal(outputMessage.OutgoingStatus, common.OutgoingStatusSending, "it marks the message as sending")
	s.Require().NotEmpty(outputMessage.ID, "it sets the ID field")
	s.Require().Equal(protobuf.MessageType_PRIVATE_GROUP, outputMessage.MessageType)
}

func (s *MessengerSuite) TestSendPrivateEmptyGroup() {
	response, err := s.m.CreateGroupChatWithMembers(context.Background(), "test", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats(), 1)

	chat := response.Chats()[0]

	inputMessage := common.NewMessage()
	inputMessage.ChatId = chat.ID
	chat.LastClockValue = uint64(100000000000000)
	err = s.m.SaveChat(chat)
	s.NoError(err)
	response, err = s.m.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Equal(1, len(response.Messages()), "it returns the message")
	outputMessage := response.Messages()[0]

	s.Require().Equal(uint64(100000000000001), outputMessage.Clock, "it correctly sets the clock")
	s.Require().Equal(uint64(100000000000001), chat.LastClockValue, "it correctly sets the last-clock-value")

	s.Require().NotEqual(uint64(0), chat.Timestamp, "it sets the timestamp")
	s.Require().Equal("0x"+hex.EncodeToString(crypto.FromECDSAPub(&s.privateKey.PublicKey)), outputMessage.From, "it sets the From field")
	s.Require().True(outputMessage.Seen, "it marks the message as seen")
	s.Require().Equal(outputMessage.OutgoingStatus, common.OutgoingStatusSent, "it marks the message as sent")
	s.Require().NotEmpty(outputMessage.ID, "it sets the ID field")
	s.Require().Equal(protobuf.MessageType_PRIVATE_GROUP, outputMessage.MessageType)
}

// Make sure public messages sent by us are not
func (s *MessengerSuite) TestRetrieveOwnPublic() {
	chat := CreatePublicChat("status", s.m.transport)
	err := s.m.SaveChat(chat)
	s.NoError(err)
	// Right-to-left text
	text := "پيل اندر خانه يي تاريک بود عرضه را آورده بودندش هنود  i\nاز براي ديدنش مردم بسي اندر آن ظلمت همي شد هر کسي"

	inputMessage := buildTestMessage(*chat)
	inputMessage.ChatId = chat.ID
	inputMessage.Text = text

	response, err := s.m.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	s.Require().Len(response.Messages(), 1)

	textMessage := response.Messages()[0]

	s.Equal(textMessage.Text, text)
	s.NotNil(textMessage.ParsedText)
	s.True(textMessage.RTL)
	s.Equal(1, textMessage.LineCount)

	s.Require().Len(response.Chats(), 1)
	actualChat := response.Chats()[0]
	// It does not set the unviewed messages count
	s.Require().Equal(uint(0), actualChat.UnviewedMessagesCount)
	// It updates the last message clock value
	s.Require().Equal(textMessage.Clock, actualChat.LastClockValue)
	// It sets the last message
	s.Require().NotNil(actualChat.LastMessage)
}

// Retrieve their public message
func (s *MessengerSuite) TestRetrieveTheirPublic() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck
	theirChat := CreatePublicChat("status", s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	chat := CreatePublicChat("status", s.m.transport)
	err = s.m.SaveChat(chat)
	s.Require().NoError(err)

	_, err = s.m.Join(chat)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(*chat)

	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	sentMessage := sendResponse.Messages()[0]

	// Wait for the message to reach its destination
	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)

	s.Require().NoError(err)

	s.Require().Len(response.Messages(), 1)
	s.Require().Len(response.Chats(), 1)
	actualChat := response.Chats()[0]
	// It sets the unviewed messages count
	s.Require().Equal(uint(1), actualChat.UnviewedMessagesCount)
	// It updates the last message clock value
	s.Require().Equal(sentMessage.Clock, actualChat.LastClockValue)
	// It sets the last message
	s.Require().NotNil(actualChat.LastMessage)
}

// Drop audio message in public group
func (s *MessengerSuite) TestDropAudioMessageInPublicGroup() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck
	theirChat := CreatePublicChat("status", s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	chat := CreatePublicChat("status", s.m.transport)
	err = s.m.SaveChat(chat)
	s.Require().NoError(err)

	_, err = s.m.Join(chat)
	s.Require().NoError(err)

	inputMessage := buildAudioMessage(s, *chat)

	_, err = theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	time.Sleep(100 * time.Millisecond)
	response, err := s.m.RetrieveAll()
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 0)
}

func (s *MessengerSuite) TestDeletedAtClockValue() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck
	theirChat := CreatePublicChat("status", s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	chat := CreatePublicChat("status", s.m.transport)
	err = s.m.SaveChat(chat)
	s.Require().NoError(err)

	_, err = s.m.Join(chat)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(*chat)

	sentResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	chat.DeletedAtClockValue = sentResponse.Messages()[0].Clock
	err = s.m.SaveChat(chat)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	time.Sleep(100 * time.Millisecond)
	response, err := s.m.RetrieveAll()
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 0)
}

func (s *MessengerSuite) TestRetrieveBlockedContact() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck

	theirChat := CreatePublicChat("status", s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)
	_, err = theirMessenger.Join(theirChat)
	s.Require().NoError(err)

	chat := CreatePublicChat("status", s.m.transport)
	err = s.m.SaveChat(chat)
	s.Require().NoError(err)
	_, err = s.m.Join(chat)
	s.Require().NoError(err)

	publicKeyHex := "0x" + hex.EncodeToString(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	blockedContact := Contact{
		ID:          publicKeyHex,
		EnsName:     "contact-name",
		LastUpdated: 20,
		Blocked:     false,
	}

	requireMessageArrival := func(receiver *Messenger, require bool) {
		// Wait for the message to reach its destination
		time.Sleep(100 * time.Millisecond)
		response, err := receiver.RetrieveAll()
		s.Require().NoError(err)
		if require {
			containTextMsg := false
			for _, msg := range response.Messages() {
				if msg.ContentType == protobuf.ChatMessage_TEXT_PLAIN && msg.Text == "text-input-message" {
					containTextMsg = true
					break
				}
			}
			s.Require().True(containTextMsg)
		} else {
			s.Require().Len(response.Messages(), 0)
		}
	}

	// Block contact
	_, err = s.m.BlockContact(blockedContact.ID, false)
	s.Require().NoError(err)

	// Blocked contact sends message, we should not receive it
	theirMessage := buildTestMessage(*theirChat)
	_, err = theirMessenger.SendChatMessage(context.Background(), theirMessage)
	s.NoError(err)
	requireMessageArrival(s.m, false)

	// We send a message, blocked contact should still receive it
	ourMessage := buildTestMessage(*chat)
	_, err = s.m.SendChatMessage(context.Background(), ourMessage)
	s.NoError(err)
	requireMessageArrival(theirMessenger, true)

	// Unblock contact
	response, err := s.m.UnblockContact(blockedContact.ID)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Equal(false, response.Contacts[0].Blocked)
	s.Require().Equal(true, response.Contacts[0].Removed)

	// Unblocked contact sends message, we should receive it
	_, err = theirMessenger.SendChatMessage(context.Background(), theirMessage)
	s.Require().NoError(err)
	requireMessageArrival(s.m, true)

	// We send a message, unblocked contact should receive it
	_, err = s.m.SendChatMessage(context.Background(), ourMessage)
	s.NoError(err)
	requireMessageArrival(theirMessenger, true)
}

// Resend their public message, receive only once
func (s *MessengerSuite) TestResendPublicMessage() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck
	theirChat := CreatePublicChat("status", s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	chat := CreatePublicChat("status", s.m.transport)
	err = s.m.SaveChat(chat)
	s.Require().NoError(err)

	_, err = s.m.Join(chat)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(*chat)

	sendResponse1, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	sentMessage := sendResponse1.Messages()[0]

	err = theirMessenger.ReSendChatMessage(context.Background(), sentMessage.ID)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().Len(response.Messages(), 1)
	s.Require().Len(response.Chats(), 1)
	actualChat := response.Chats()[0]
	// It sets the unviewed messages count
	s.Require().Equal(uint(1), actualChat.UnviewedMessagesCount)
	// It updates the last message clock value
	s.Require().Equal(sentMessage.Clock, actualChat.LastClockValue)
	// It sets the last message
	s.Require().NotNil(actualChat.LastMessage)

	// We send the messag again
	err = theirMessenger.ReSendChatMessage(context.Background(), sentMessage.ID)
	s.Require().NoError(err)

	// It should not be retrieved anymore
	time.Sleep(100 * time.Millisecond)
	response, err = s.m.RetrieveAll()
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 0)
}

// Test receiving a message on an existing private chat
func (s *MessengerSuite) TestRetrieveTheirPrivateChatExisting() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck
	theirChat := CreateOneToOneChat("XXX", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("our-chat", &theirMessenger.identity.PublicKey, s.m.transport)
	ourChat.UnviewedMessagesCount = 1
	// Make chat inactive
	ourChat.Active = false
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(*theirChat)

	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	sentMessage := sendResponse.Messages()[0]

	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().Len(response.Chats(), 1)
	actualChat := response.Chats()[0]
	// It updates the unviewed messages count
	s.Require().Equal(uint(2), actualChat.UnviewedMessagesCount)
	// It updates the last message clock value
	s.Require().Equal(sentMessage.Clock, actualChat.LastClockValue)
	// It sets the last message
	s.Require().NotNil(actualChat.LastMessage)
	s.Require().False(actualChat.Active)
}

// Test receiving a message on an non-existing private chat
func (s *MessengerSuite) TestRetrieveTheirPrivateChatNonExisting() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck
	chat := CreateOneToOneChat("XXX", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(chat)
	s.NoError(err)

	inputMessage := buildTestMessage(*chat)

	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	sentMessage := sendResponse.Messages()[0]

	// Wait for the message to reach its destination
	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)

	s.Require().NoError(err)

	s.Require().Len(response.Chats(), 1)
	actualChat := response.Chats()[0]
	// It updates the unviewed messages count
	s.Require().Equal(uint(1), actualChat.UnviewedMessagesCount)
	// It updates the last message clock value
	s.Require().Equal(sentMessage.Clock, actualChat.LastClockValue)
	// It sets the last message
	s.Require().NotNil(actualChat.LastMessage)
	// It does not set the chat as active
	s.Require().False(actualChat.Active)
}

// Test receiving a message on an non-existing public chat
func (s *MessengerSuite) TestRetrieveTheirPublicChatNonExisting() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck
	chat := CreatePublicChat("test-chat", s.m.transport)
	err = theirMessenger.SaveChat(chat)
	s.NoError(err)

	inputMessage := buildTestMessage(*chat)

	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	// Wait for the message to reach its destination
	time.Sleep(100 * time.Millisecond)
	response, err := s.m.RetrieveAll()
	s.NoError(err)

	s.Require().Equal(len(response.Messages()), 0)
	s.Require().Equal(len(response.Chats()), 0)
}

// Test receiving a message on an existing private group chat
func (s *MessengerSuite) TestRetrieveTheirPrivateGroupChat() {
	var response *MessengerResponse
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck
	response, err = s.m.CreateGroupChatWithMembers(context.Background(), "id", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats(), 1)

	ourChat := response.Chats()[0]

	err = s.m.SaveChat(ourChat)
	s.NoError(err)

	s.Require().NoError(makeMutualContact(s.m, &theirMessenger.identity.PublicKey))

	members := []string{"0x" + hex.EncodeToString(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))}
	_, err = s.m.AddMembersToGroupChat(context.Background(), ourChat.ID, members)
	s.NoError(err)

	// Retrieve their messages so that the chat is created
	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"chat invitation not received",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.ActivityCenterNotifications(), 1)
	s.Require().False(response.Chats()[0].Active)

	_, err = theirMessenger.ConfirmJoiningGroup(context.Background(), ourChat.ID)
	s.NoError(err)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"no joining group event received",
	)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(*ourChat)

	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	sentMessage := sendResponse.Messages()[0]

	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().Len(response.Chats(), 1)
	actualChat := response.Chats()[0]
	// It updates the unviewed messages count
	s.Require().Equal(uint(1), actualChat.UnviewedMessagesCount)
	// It updates the last message clock value
	s.Require().Equal(sentMessage.Clock, actualChat.LastClockValue)
	// It sets the last message
	s.Require().NotNil(actualChat.LastMessage)
}

// Test receiving a message on an existing private group chat
func (s *MessengerSuite) TestChangeNameGroupChat() {
	var response *MessengerResponse
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck
	response, err = s.m.CreateGroupChatWithMembers(context.Background(), "old-name", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats(), 1)

	ourChat := response.Chats()[0]

	err = s.m.SaveChat(ourChat)
	s.NoError(err)

	s.Require().NoError(makeMutualContact(s.m, &theirMessenger.identity.PublicKey))

	members := []string{"0x" + hex.EncodeToString(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))}
	_, err = s.m.AddMembersToGroupChat(context.Background(), ourChat.ID, members)
	s.NoError(err)

	// Retrieve their messages so that the chat is created
	_, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"chat invitation not received",
	)
	s.Require().NoError(err)

	_, err = theirMessenger.ConfirmJoiningGroup(context.Background(), ourChat.ID)
	s.NoError(err)

	// Wait for join group event
	_, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"no joining group event received",
	)
	s.Require().NoError(err)

	_, err = s.m.ChangeGroupChatName(context.Background(), ourChat.ID, newEnsName)
	s.NoError(err)

	// Retrieve their messages so that the chat is created
	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"chat invitation not received",
	)
	s.Require().NoError(err)

	s.Require().Len(response.Chats(), 1)
	actualChat := response.Chats()[0]
	s.Require().Equal(newEnsName, actualChat.Name)
}

// Test being re-invited to a group chat
func (s *MessengerSuite) TestReInvitedToGroupChat() {
	var response *MessengerResponse
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck
	response, err = s.m.CreateGroupChatWithMembers(context.Background(), "old-name", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats(), 1)

	ourChat := response.Chats()[0]

	err = s.m.SaveChat(ourChat)
	s.NoError(err)

	s.Require().NoError(makeMutualContact(s.m, &theirMessenger.identity.PublicKey))

	members := []string{"0x" + hex.EncodeToString(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))}
	_, err = s.m.AddMembersToGroupChat(context.Background(), ourChat.ID, members)
	s.NoError(err)

	// Retrieve their messages so that the chat is created
	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"chat invitation not received",
	)
	s.Require().NoError(err)
	s.Require().Len(response.ActivityCenterNotifications(), 1)
	s.Require().False(response.Chats()[0].Active)

	_, err = theirMessenger.ConfirmJoiningGroup(context.Background(), ourChat.ID)
	s.NoError(err)

	// Wait for join group event
	_, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"no joining group event received",
	)
	s.Require().NoError(err)

	response, err = theirMessenger.LeaveGroupChat(context.Background(), ourChat.ID, true)
	s.NoError(err)

	s.Require().Len(response.Chats(), 1)
	s.Require().False(response.Chats()[0].Active)

	// Retrieve messages so user is removed
	_, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 && len(r.Chats()[0].Members) == 1 },
		"leave group chat not received",
	)

	s.Require().NoError(err)

	// And we get re-invited
	_, err = s.m.AddMembersToGroupChat(context.Background(), ourChat.ID, members)
	s.NoError(err)

	// Retrieve their messages so that the chat is created
	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"chat invitation not received",
	)

	s.Require().NoError(err)

	s.Require().Len(response.Chats(), 1)
	s.Require().False(response.Chats()[0].Active)
}

func (s *MessengerSuite) TestChatPersistencePublic() {
	chat := &Chat{
		ID:                    "chat-name",
		Name:                  "chat-name",
		Color:                 "#fffff",
		Active:                true,
		ChatType:              ChatTypePublic,
		Timestamp:             10,
		LastClockValue:        20,
		DeletedAtClockValue:   30,
		UnviewedMessagesCount: 40,
		LastMessage:           common.NewMessage(),
		Highlight:             false,
	}

	s.Require().NoError(s.m.SaveChat(chat))
	savedChats := s.m.Chats()
	s.Require().Equal(deprecation.AddChatsCount(1), len(savedChats))
}

func (s *MessengerSuite) TestDeleteChat() {
	chatID := "chatid"
	chat := &Chat{
		ID:                    chatID,
		Name:                  "chat-name",
		Color:                 "#fffff",
		Active:                true,
		ChatType:              ChatTypePublic,
		Timestamp:             10,
		LastClockValue:        20,
		DeletedAtClockValue:   30,
		UnviewedMessagesCount: 40,
		LastMessage:           common.NewMessage(),
		Highlight:             false,
	}

	s.Require().NoError(s.m.SaveChat(chat))
	savedChats := s.m.Chats()
	s.Require().Equal(deprecation.AddChatsCount(1), len(savedChats))

	s.Require().NoError(s.m.DeleteChat(chatID))
	savedChats = s.m.Chats()
	s.Require().Equal(deprecation.AddChatsCount(0), len(savedChats))
}

func (s *MessengerSuite) TestChatPersistenceUpdate() {
	chat := &Chat{
		ID:                    "chat-name",
		Name:                  "chat-name",
		Color:                 "#fffff",
		Active:                true,
		ChatType:              ChatTypePublic,
		Timestamp:             10,
		LastClockValue:        20,
		DeletedAtClockValue:   30,
		UnviewedMessagesCount: 40,
		LastMessage:           common.NewMessage(),
		Highlight:             false,
	}

	s.Require().NoError(s.m.SaveChat(chat))
	savedChats := s.m.Chats()
	s.Require().Equal(deprecation.AddChatsCount(1), len(savedChats))

	var actualChat *Chat
	for idx := range savedChats {
		if savedChats[idx].ID == chat.ID {
			actualChat = chat
		}
	}

	s.Require().NotNil(actualChat)
	s.Require().Equal(chat, actualChat)

	chat.Name = "updated-name-1"
	s.Require().NoError(s.m.SaveChat(chat))

	var actualUpdatedChat *Chat
	updatedChats := s.m.Chats()

	for idx := range updatedChats {
		if updatedChats[idx].ID == chat.ID {
			actualUpdatedChat = chat
		}
	}

	s.Require().Equal(chat, actualUpdatedChat)
}

func (s *MessengerSuite) TestChatPersistenceOneToOne() {
	chat := &Chat{
		ID:                    testPK,
		Name:                  testPK,
		Color:                 "#fffff",
		Active:                true,
		ChatType:              ChatTypeOneToOne,
		Timestamp:             10,
		LastClockValue:        20,
		DeletedAtClockValue:   30,
		UnviewedMessagesCount: 40,
		LastMessage:           common.NewMessage(),
		Highlight:             false,
	}

	publicKeyBytes, err := hex.DecodeString(testPK[2:])
	s.Require().NoError(err)

	pk, err := crypto.UnmarshalPubkey(publicKeyBytes)
	s.Require().NoError(err)

	s.Require().NoError(s.m.SaveChat(chat))
	savedChats := s.m.Chats()
	s.Require().Equal(deprecation.AddChatsCount(1), len(savedChats))

	var actualChat *Chat
	for idx := range savedChats {
		if chat.ID == savedChats[idx].ID {
			actualChat = savedChats[idx]
		}

	}

	actualPk, err := actualChat.PublicKey()
	s.Require().NoError(err)

	s.Require().Equal(pk, actualPk)

	s.Require().Equal(chat, actualChat)
	s.Require().NotEmpty(actualChat.Identicon)
	s.Require().NotEmpty(actualChat.Alias)
}

func (s *MessengerSuite) TestChatPersistencePrivateGroupChat() {

	member1Key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	member1ID := types.EncodeHex(crypto.FromECDSAPub(&member1Key.PublicKey))

	member2Key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	member2ID := types.EncodeHex(crypto.FromECDSAPub(&member2Key.PublicKey))

	member3Key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	member3ID := types.EncodeHex(crypto.FromECDSAPub(&member3Key.PublicKey))

	chat := &Chat{
		ID:        "chat-id",
		Name:      "chat-id",
		Color:     "#fffff",
		Active:    true,
		ChatType:  ChatTypePrivateGroupChat,
		Timestamp: 10,
		Members: []ChatMember{
			{
				ID:    member1ID,
				Admin: false,
			},
			{
				ID:    member2ID,
				Admin: true,
			},
			{
				ID:    member3ID,
				Admin: true,
			},
		},
		MembershipUpdates: []v1protocol.MembershipUpdateEvent{
			{
				Type:       protobuf.MembershipUpdateEvent_CHAT_CREATED,
				Name:       "name-1",
				ClockValue: 1,
				Signature:  []byte("signature-1"),
				From:       "from-1",
				Members:    []string{"member-1", "member-2"},
			},
			{
				Type:       protobuf.MembershipUpdateEvent_MEMBERS_ADDED,
				Name:       "name-2",
				ClockValue: 2,
				Signature:  []byte("signature-2"),
				From:       "from-2",
				Members:    []string{"member-2", "member-3"},
			},
		},
		LastClockValue:        20,
		DeletedAtClockValue:   30,
		UnviewedMessagesCount: 40,
		LastMessage:           common.NewMessage(),
		Highlight:             false,
	}
	s.Require().NoError(s.m.SaveChat(chat))
	savedChats := s.m.Chats()
	s.Require().Equal(deprecation.AddChatsCount(1), len(savedChats))

	var actualChat *Chat
	for idx := range savedChats {
		if savedChats[idx].ID == chat.ID {
			actualChat = savedChats[idx]
		}

	}

	s.Require().Equal(chat, actualChat)
}

func (s *MessengerSuite) TestBlockContact() {
	contact := Contact{
		ID:                       testPK,
		EnsName:                  "contact-name",
		LastUpdated:              20,
		ContactRequestLocalState: ContactRequestStateSent,
	}

	key2, err := crypto.GenerateKey()
	s.Require().NoError(err)

	contact2 := Contact{
		ID:                       common.PubkeyToHex(&key2.PublicKey),
		EnsName:                  "contact-name",
		LastUpdated:              20,
		ContactRequestLocalState: ContactRequestStateSent,
	}

	chat1 := &Chat{
		ID:                    contact.ID,
		Name:                  "chat-name",
		Color:                 "#fffff",
		Active:                true,
		ChatType:              ChatTypeOneToOne,
		Timestamp:             1,
		LastClockValue:        20,
		DeletedAtClockValue:   30,
		UnviewedMessagesCount: 40,
		Highlight:             false,
	}

	chat2 := &Chat{
		ID:                    "chat-2",
		Name:                  "chat-name",
		Color:                 "#fffff",
		Active:                true,
		ChatType:              ChatTypePublic,
		Timestamp:             2,
		LastClockValue:        20,
		DeletedAtClockValue:   30,
		UnviewedMessagesCount: 40,
		Highlight:             false,
	}

	chat3 := &Chat{
		ID:                    "chat-3",
		Name:                  "chat-name",
		Color:                 "#fffff",
		Active:                true,
		ChatType:              ChatTypePublic,
		Timestamp:             3,
		LastClockValue:        20,
		DeletedAtClockValue:   30,
		UnviewedMessagesCount: 40,
		Highlight:             false,
	}

	s.Require().NoError(s.m.SaveChat(chat1))
	s.Require().NoError(s.m.SaveChat(chat2))
	s.Require().NoError(s.m.SaveChat(chat3))

	_, err = s.m.AddContact(context.Background(), &requests.AddContact{ID: contact.ID})
	s.Require().NoError(err)

	messages := []*common.Message{
		{
			ID:          "test-1",
			LocalChatID: chat2.ID,
			ChatMessage: &protobuf.ChatMessage{
				ContentType: 1,
				Text:        "test-1",
				Clock:       1,
			},
			From: contact.ID,
		},
		{
			ID:          "test-2",
			LocalChatID: chat2.ID,
			ChatMessage: &protobuf.ChatMessage{
				ContentType: 2,
				Text:        "test-2",
				Clock:       2,
			},
			From: contact.ID,
		},
		{
			ID:          "test-3",
			LocalChatID: chat2.ID,
			ChatMessage: &protobuf.ChatMessage{
				ContentType: 3,
				Text:        "test-3",
				Clock:       3,
			},
			Seen: false,
			From: contact2.ID,
		},
		{
			ID:          "test-4",
			LocalChatID: chat2.ID,
			ChatMessage: &protobuf.ChatMessage{
				ContentType: 4,
				Text:        "test-4",
				Clock:       4,
			},
			Seen: false,
			From: contact2.ID,
		},
		{
			ID:          "test-5",
			LocalChatID: chat2.ID,
			ChatMessage: &protobuf.ChatMessage{
				ContentType: 5,
				Text:        "test-5",
				Clock:       5,
			},
			Seen: true,
			From: contact2.ID,
		},
		{
			ID:          "test-6",
			LocalChatID: chat3.ID,
			ChatMessage: &protobuf.ChatMessage{
				ContentType: 6,
				Text:        "test-6",
				Clock:       6,
			},
			Seen: false,
			From: contact.ID,
		},
		{
			ID:          "test-7",
			LocalChatID: chat3.ID,
			ChatMessage: &protobuf.ChatMessage{
				ContentType: 7,
				Text:        "test-7",
				Clock:       7,
			},
			Seen: false,
			From: contact2.ID,
		},
	}

	err = s.m.SaveMessages(messages)
	s.Require().NoError(err)

	response, err := s.m.BlockContact(contact.ID, false)
	s.Require().NoError(err)

	blockedContacts := s.m.BlockedContacts()
	s.Require().True(blockedContacts[0].Removed)

	chats := response.Chats()

	var actualChat2, actualChat3 *Chat
	for idx := range chats {
		if chats[idx].ID == chat2.ID {
			actualChat2 = chats[idx]
		} else if chats[idx].ID == chat3.ID {
			actualChat3 = chats[idx]
		}
	}
	// The new unviewed count is updated
	s.Require().Equal(uint(1), actualChat3.UnviewedMessagesCount)
	s.Require().Equal(uint(2), actualChat2.UnviewedMessagesCount)

	// The new message content is updated
	s.Require().NotNil(actualChat3.LastMessage)

	s.Require().Equal("test-7", actualChat3.LastMessage.ID)

	s.Require().NotNil(actualChat2.LastMessage)
	s.Require().Equal("test-5", actualChat2.LastMessage.ID)

	// The contact is updated
	savedContacts := s.m.Contacts()
	s.Require().Equal(1, len(savedContacts))

	// The chat is deleted
	actualChats := s.m.Chats()
	s.Require().Equal(deprecation.AddChatsCount(3), len(actualChats))

	// The messages have been deleted
	chat2Messages, _, err := s.m.MessageByChatID(chat2.ID, "", 20)
	s.Require().NoError(err)
	s.Require().Equal(3, len(chat2Messages))

	chat3Messages, _, err := s.m.MessageByChatID(chat3.ID, "", 20)
	s.Require().NoError(err)
	s.Require().Equal(1, len(chat3Messages))

}

func (s *MessengerSuite) TestContactPersistence() {
	_, err := s.m.AddContact(context.Background(), &requests.AddContact{ID: testPK})
	s.Require().NoError(err)
	savedContacts := s.m.Contacts()

	s.Require().Equal(1, len(savedContacts))

	s.Require().True(savedContacts[0].added())
}

func (s *MessengerSuite) TestSharedSecretHandler() {
	err := s.m.handleSharedSecrets(nil)
	s.NoError(err)
}

func (s *MessengerSuite) TestCreateGroupChatWithMembers() {
	members := []string{testPK}

	pubKey, err := common.HexToPubkey(testPK)
	s.Require().NoError(err)
	s.Require().NoError(makeMutualContact(s.m, pubKey))

	response, err := s.m.CreateGroupChatWithMembers(context.Background(), "test", members)
	s.NoError(err)
	s.Require().Len(response.Chats(), 1)

	chat := response.Chats()[0]

	s.Require().Equal("test", chat.Name)
	publicKeyHex := "0x" + hex.EncodeToString(crypto.FromECDSAPub(&s.m.identity.PublicKey))
	s.Require().Contains(chat.ID, publicKeyHex)
	s.EqualValues([]string{publicKeyHex}, []string{chat.Members[0].ID})
	s.Equal(members[0], chat.Members[1].ID)
}

func (s *MessengerSuite) TestAddMembersToChat() {
	response, err := s.m.CreateGroupChatWithMembers(context.Background(), "test", []string{})
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)

	chat := response.Chats()[0]

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	members := []string{"0x" + hex.EncodeToString(crypto.FromECDSAPub(&key.PublicKey))}

	s.Require().NoError(makeMutualContact(s.m, &key.PublicKey))

	response, err = s.m.AddMembersToGroupChat(context.Background(), chat.ID, members)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	chat = response.Chats()[0]

	publicKeyHex := "0x" + hex.EncodeToString(crypto.FromECDSAPub(&s.m.identity.PublicKey))
	keyHex := "0x" + hex.EncodeToString(crypto.FromECDSAPub(&key.PublicKey))
	s.EqualValues([]string{publicKeyHex, keyHex}, []string{chat.Members[0].ID, chat.Members[1].ID})
}

func (s *MessengerSuite) TestDeclineRequestAddressForTransaction() {
	value := testValue
	contract := testContract
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck
	theirPkString := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	myPkString := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))

	chat := CreateOneToOneChat(theirPkString, &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(chat)
	s.Require().NoError(err)

	myAddress := crypto.PubkeyToAddress(s.m.identity.PublicKey)

	response, err := s.m.RequestAddressForTransaction(context.Background(), theirPkString, myAddress.Hex(), value, contract)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	senderMessage := response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, senderMessage.ContentType)
	initialCommandID := senderMessage.ID

	s.Require().Equal("Request address for transaction", senderMessage.Text)
	s.Require().NotNil(senderMessage.CommandParameters)
	s.Require().Equal(value, senderMessage.CommandParameters.Value)
	s.Require().Equal(contract, senderMessage.CommandParameters.Contract)
	s.Require().Equal(initialCommandID, senderMessage.CommandParameters.ID)
	s.Require().Equal(common.CommandStateRequestAddressForTransaction, senderMessage.CommandParameters.CommandState)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	receiverMessage := response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, receiverMessage.ContentType)
	s.Require().Equal("Request address for transaction", receiverMessage.Text)
	s.Require().NotNil(receiverMessage.CommandParameters)
	s.Require().Equal(value, receiverMessage.CommandParameters.Value)
	s.Require().Equal(contract, receiverMessage.CommandParameters.Contract)
	s.Require().Equal(initialCommandID, receiverMessage.CommandParameters.ID)
	s.Require().Equal(theirPkString, receiverMessage.ChatId)
	s.Require().Equal(common.CommandStateRequestAddressForTransaction, receiverMessage.CommandParameters.CommandState)

	// We decline the request
	response, err = theirMessenger.DeclineRequestAddressForTransaction(context.Background(), receiverMessage.ID)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	senderMessage = response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, senderMessage.ContentType)
	s.Require().Equal("Request address for transaction declined", senderMessage.Text)
	s.Require().NotNil(senderMessage.CommandParameters)
	s.Require().Equal(value, senderMessage.CommandParameters.Value)
	s.Require().Equal(contract, senderMessage.CommandParameters.Contract)
	s.Require().Equal(common.CommandStateRequestAddressForTransactionDeclined, senderMessage.CommandParameters.CommandState)
	s.Require().Equal(initialCommandID, senderMessage.CommandParameters.ID)
	s.Require().Equal(receiverMessage.ID, senderMessage.Replace)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	receiverMessage = response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, receiverMessage.ContentType)
	s.Require().Equal("Request address for transaction declined", receiverMessage.Text)
	s.Require().NotNil(receiverMessage.CommandParameters)
	s.Require().Equal(value, receiverMessage.CommandParameters.Value)
	s.Require().Equal(contract, receiverMessage.CommandParameters.Contract)
	s.Require().Equal(common.CommandStateRequestAddressForTransactionDeclined, receiverMessage.CommandParameters.CommandState)
	s.Require().Equal(initialCommandID, receiverMessage.CommandParameters.ID)
	s.Require().Equal(myPkString, receiverMessage.ChatId)
	s.Require().Equal(initialCommandID, receiverMessage.Replace)
}

func (s *MessengerSuite) TestSendEthTransaction() {
	value := testValue
	contract := testContract

	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck
	theirPkString := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))

	receiverAddress := crypto.PubkeyToAddress(theirMessenger.identity.PublicKey)
	receiverAddressString := strings.ToLower(receiverAddress.Hex())

	chat := CreateOneToOneChat(theirPkString, &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(chat)
	s.Require().NoError(err)

	transactionHash := testTransactionHash
	signature, err := buildSignature(s.m.identity, &s.m.identity.PublicKey, transactionHash)
	s.Require().NoError(err)

	response, err := s.m.SendTransaction(context.Background(), theirPkString, value, contract, transactionHash, signature)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	senderMessage := response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, senderMessage.ContentType)
	s.Require().Equal("Transaction sent", senderMessage.Text)
	s.Require().NotNil(senderMessage.CommandParameters)
	s.Require().Equal(transactionHash, senderMessage.CommandParameters.TransactionHash)
	s.Require().Equal(contract, senderMessage.CommandParameters.Contract)
	s.Require().Equal(value, senderMessage.CommandParameters.Value)
	s.Require().Equal(signature, senderMessage.CommandParameters.Signature)
	s.Require().Equal(common.CommandStateTransactionSent, senderMessage.CommandParameters.CommandState)
	s.Require().NotEmpty(senderMessage.ID)
	s.Require().Equal("", senderMessage.Replace)

	var transactions []*TransactionToValidate
	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		var err error

		_, err = theirMessenger.RetrieveAll()
		if err != nil {
			return err
		}
		transactions, err = theirMessenger.persistence.TransactionsToValidate()
		if err == nil && len(transactions) == 0 {
			err = errors.New("no transactions")
		}
		return err
	})
	s.Require().NoError(err)

	actualTransaction := transactions[0]

	s.Require().Equal(&s.m.identity.PublicKey, actualTransaction.From)
	s.Require().Equal(transactionHash, actualTransaction.TransactionHash)
	s.Require().True(actualTransaction.Validate)

	senderAddress := crypto.PubkeyToAddress(s.m.identity.PublicKey)

	client := MockEthClient{}
	valueBig, ok := big.NewInt(0).SetString(value, 10)
	s.Require().True(ok)
	client.messages = make(map[string]MockTransaction)
	client.messages[transactionHash] = MockTransaction{
		Status: coretypes.TransactionStatusSuccess,
		Message: coretypes.NewMessage(
			senderAddress,
			&receiverAddress,
			1,
			valueBig,
			0,
			nil,
			nil,
			false,
		),
	}
	theirMessenger.verifyTransactionClient = client
	response, err = theirMessenger.ValidateTransactions(context.Background(), []types.Address{receiverAddress})
	s.Require().NoError(err)

	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	receiverMessage := response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, receiverMessage.ContentType)

	s.Require().Equal("Transaction received", receiverMessage.Text)
	s.Require().NotNil(receiverMessage.CommandParameters)
	s.Require().Equal(value, receiverMessage.CommandParameters.Value)
	s.Require().Equal(strings.ToLower(receiverAddress.Hex()), receiverMessage.CommandParameters.Address)
	s.Require().Equal(transactionHash, receiverMessage.CommandParameters.TransactionHash)
	s.Require().Equal(receiverAddressString, receiverMessage.CommandParameters.Address)
	s.Require().Equal("", receiverMessage.CommandParameters.ID)
	s.Require().Equal(common.CommandStateTransactionSent, receiverMessage.CommandParameters.CommandState)
	s.Require().Equal(senderMessage.ID, receiverMessage.ID)
	s.Require().Equal("", receiverMessage.Replace)
}

func (s *MessengerSuite) TestSendTokenTransaction() {
	value := testValue
	contract := testContract

	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck
	theirPkString := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))

	receiverAddress := crypto.PubkeyToAddress(theirMessenger.identity.PublicKey)
	receiverAddressString := strings.ToLower(receiverAddress.Hex())

	chat := CreateOneToOneChat(theirPkString, &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(chat)
	s.Require().NoError(err)

	transactionHash := testTransactionHash
	signature, err := buildSignature(s.m.identity, &s.m.identity.PublicKey, transactionHash)
	s.Require().NoError(err)

	response, err := s.m.SendTransaction(context.Background(), theirPkString, value, contract, transactionHash, signature)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	senderMessage := response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, senderMessage.ContentType)
	s.Require().Equal("Transaction sent", senderMessage.Text)
	s.Require().NotNil(senderMessage.CommandParameters)
	s.Require().Equal(transactionHash, senderMessage.CommandParameters.TransactionHash)
	s.Require().Equal(value, senderMessage.CommandParameters.Value)
	s.Require().Equal(contract, senderMessage.CommandParameters.Contract)
	s.Require().Equal(signature, senderMessage.CommandParameters.Signature)
	s.Require().Equal(common.CommandStateTransactionSent, senderMessage.CommandParameters.CommandState)
	s.Require().NotEmpty(senderMessage.ID)

	var transactions []*TransactionToValidate
	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		var err error

		_, err = theirMessenger.RetrieveAll()
		if err != nil {
			return err
		}
		transactions, err = theirMessenger.persistence.TransactionsToValidate()
		if err == nil && len(transactions) == 0 {
			err = errors.New("no transactions")
		}
		return err
	})
	s.Require().NoError(err)

	actualTransaction := transactions[0]

	s.Require().Equal(&s.m.identity.PublicKey, actualTransaction.From)
	s.Require().Equal(transactionHash, actualTransaction.TransactionHash)
	s.Require().True(actualTransaction.Validate)

	senderAddress := crypto.PubkeyToAddress(s.m.identity.PublicKey)

	contractAddress := types.HexToAddress(contract)
	client := MockEthClient{}
	valueBig, ok := big.NewInt(0).SetString(value, 10)
	s.Require().True(ok)
	client.messages = make(map[string]MockTransaction)
	client.messages[transactionHash] = MockTransaction{
		Status: coretypes.TransactionStatusSuccess,
		Message: coretypes.NewMessage(
			senderAddress,
			&contractAddress,
			1,
			nil,
			0,
			nil,
			buildData(transferFunction, receiverAddress, valueBig),
			false,
		),
	}
	theirMessenger.verifyTransactionClient = client
	response, err = theirMessenger.ValidateTransactions(context.Background(), []types.Address{receiverAddress})
	s.Require().NoError(err)

	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	receiverMessage := response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, receiverMessage.ContentType)

	s.Require().Equal("Transaction received", receiverMessage.Text)
	s.Require().NotNil(receiverMessage.CommandParameters)
	s.Require().Equal(value, receiverMessage.CommandParameters.Value)
	s.Require().Equal(contract, receiverMessage.CommandParameters.Contract)
	s.Require().Equal(transactionHash, receiverMessage.CommandParameters.TransactionHash)
	s.Require().Equal(receiverAddressString, receiverMessage.CommandParameters.Address)
	s.Require().Equal("", receiverMessage.CommandParameters.ID)
	s.Require().Equal(common.CommandStateTransactionSent, receiverMessage.CommandParameters.CommandState)
	s.Require().Equal(senderMessage.ID, receiverMessage.ID)
	s.Require().Equal(senderMessage.Replace, senderMessage.Replace)
}

func (s *MessengerSuite) TestAcceptRequestAddressForTransaction() {
	value := testValue
	contract := testContract
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck
	theirPkString := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	myPkString := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))

	myAddress := crypto.PubkeyToAddress(s.m.identity.PublicKey)

	chat := CreateOneToOneChat(theirPkString, &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(chat)
	s.Require().NoError(err)

	response, err := s.m.RequestAddressForTransaction(context.Background(), theirPkString, myAddress.Hex(), value, contract)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	senderMessage := response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, senderMessage.ContentType)
	initialCommandID := senderMessage.ID

	s.Require().Equal("Request address for transaction", senderMessage.Text)
	s.Require().NotNil(senderMessage.CommandParameters)
	s.Require().Equal(value, senderMessage.CommandParameters.Value)
	s.Require().Equal(contract, senderMessage.CommandParameters.Contract)
	s.Require().Equal(initialCommandID, senderMessage.CommandParameters.ID)
	s.Require().Equal(common.CommandStateRequestAddressForTransaction, senderMessage.CommandParameters.CommandState)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	receiverMessage := response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, receiverMessage.ContentType)
	s.Require().Equal("Request address for transaction", receiverMessage.Text)
	s.Require().NotNil(receiverMessage.CommandParameters)
	s.Require().Equal(value, receiverMessage.CommandParameters.Value)
	s.Require().Equal(contract, receiverMessage.CommandParameters.Contract)
	s.Require().Equal(initialCommandID, receiverMessage.CommandParameters.ID)
	s.Require().Equal(common.CommandStateRequestAddressForTransaction, receiverMessage.CommandParameters.CommandState)
	s.Require().Equal(theirPkString, receiverMessage.ChatId)

	// We accept the request
	response, err = theirMessenger.AcceptRequestAddressForTransaction(context.Background(), receiverMessage.ID, "some-address")
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	senderMessage = response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, senderMessage.ContentType)
	s.Require().Equal("Request address for transaction accepted", senderMessage.Text)
	s.Require().NotNil(senderMessage.CommandParameters)
	s.Require().Equal(value, senderMessage.CommandParameters.Value)
	s.Require().Equal(contract, senderMessage.CommandParameters.Contract)
	s.Require().Equal(common.CommandStateRequestAddressForTransactionAccepted, senderMessage.CommandParameters.CommandState)
	s.Require().Equal(initialCommandID, senderMessage.CommandParameters.ID)
	s.Require().Equal("some-address", senderMessage.CommandParameters.Address)
	s.Require().Equal(receiverMessage.ID, senderMessage.Replace)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	receiverMessage = response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, receiverMessage.ContentType)
	s.Require().Equal("Request address for transaction accepted", receiverMessage.Text)
	s.Require().NotNil(receiverMessage.CommandParameters)
	s.Require().Equal(value, receiverMessage.CommandParameters.Value)
	s.Require().Equal(contract, receiverMessage.CommandParameters.Contract)
	s.Require().Equal(common.CommandStateRequestAddressForTransactionAccepted, receiverMessage.CommandParameters.CommandState)
	s.Require().Equal(initialCommandID, receiverMessage.CommandParameters.ID)
	s.Require().Equal("some-address", receiverMessage.CommandParameters.Address)
	s.Require().Equal(initialCommandID, receiverMessage.Replace)
	s.Require().Equal(myPkString, receiverMessage.ChatId)
}

func (s *MessengerSuite) TestDeclineRequestTransaction() {
	value := testValue
	contract := testContract
	receiverAddress := crypto.PubkeyToAddress(s.m.identity.PublicKey)
	receiverAddressString := strings.ToLower(receiverAddress.Hex())
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck
	theirPkString := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))

	chat := CreateOneToOneChat(theirPkString, &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(chat)
	s.Require().NoError(err)

	response, err := s.m.RequestTransaction(context.Background(), theirPkString, value, contract, receiverAddressString)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	senderMessage := response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, senderMessage.ContentType)
	initialCommandID := senderMessage.ID

	s.Require().Equal("Request transaction", senderMessage.Text)
	s.Require().NotNil(senderMessage.CommandParameters)
	s.Require().Equal(value, senderMessage.CommandParameters.Value)
	s.Require().Equal(contract, senderMessage.CommandParameters.Contract)
	s.Require().Equal(receiverAddressString, senderMessage.CommandParameters.Address)
	s.Require().Equal(initialCommandID, senderMessage.CommandParameters.ID)
	s.Require().Equal(common.CommandStateRequestTransaction, senderMessage.CommandParameters.CommandState)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	receiverMessage := response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, receiverMessage.ContentType)
	s.Require().Equal("Request transaction", receiverMessage.Text)
	s.Require().NotNil(receiverMessage.CommandParameters)
	s.Require().Equal(value, receiverMessage.CommandParameters.Value)
	s.Require().Equal(contract, receiverMessage.CommandParameters.Contract)
	s.Require().Equal(receiverAddressString, receiverMessage.CommandParameters.Address)
	s.Require().Equal(initialCommandID, receiverMessage.CommandParameters.ID)
	s.Require().Equal(common.CommandStateRequestTransaction, receiverMessage.CommandParameters.CommandState)

	response, err = theirMessenger.DeclineRequestTransaction(context.Background(), initialCommandID)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	senderMessage = response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, senderMessage.ContentType)

	s.Require().Equal("Transaction request declined", senderMessage.Text)
	s.Require().Equal(initialCommandID, senderMessage.CommandParameters.ID)
	s.Require().Equal(receiverMessage.ID, senderMessage.Replace)
	s.Require().Equal(common.CommandStateRequestTransactionDeclined, senderMessage.CommandParameters.CommandState)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	receiverMessage = response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, receiverMessage.ContentType)

	s.Require().Equal("Transaction request declined", receiverMessage.Text)
	s.Require().Equal(initialCommandID, receiverMessage.CommandParameters.ID)
	s.Require().Equal(initialCommandID, receiverMessage.Replace)
	s.Require().Equal(common.CommandStateRequestTransactionDeclined, receiverMessage.CommandParameters.CommandState)
}

func (s *MessengerSuite) TestRequestTransaction() {
	value := testValue
	contract := testContract
	receiverAddress := crypto.PubkeyToAddress(s.m.identity.PublicKey)
	receiverAddressString := strings.ToLower(receiverAddress.Hex())
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck
	theirPkString := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))

	chat := CreateOneToOneChat(theirPkString, &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(chat)
	s.Require().NoError(err)

	response, err := s.m.RequestTransaction(context.Background(), theirPkString, value, contract, receiverAddressString)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	senderMessage := response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, senderMessage.ContentType)
	initialCommandID := senderMessage.ID

	s.Require().Equal("Request transaction", senderMessage.Text)
	s.Require().NotNil(senderMessage.CommandParameters)
	s.Require().Equal(value, senderMessage.CommandParameters.Value)
	s.Require().Equal(contract, senderMessage.CommandParameters.Contract)
	s.Require().Equal(receiverAddressString, senderMessage.CommandParameters.Address)
	s.Require().Equal(initialCommandID, senderMessage.CommandParameters.ID)
	s.Require().Equal(common.CommandStateRequestTransaction, senderMessage.CommandParameters.CommandState)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	receiverMessage := response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, receiverMessage.ContentType)
	s.Require().Equal("Request transaction", receiverMessage.Text)
	s.Require().NotNil(receiverMessage.CommandParameters)
	s.Require().Equal(value, receiverMessage.CommandParameters.Value)
	s.Require().Equal(contract, receiverMessage.CommandParameters.Contract)
	s.Require().Equal(receiverAddressString, receiverMessage.CommandParameters.Address)
	s.Require().Equal(initialCommandID, receiverMessage.CommandParameters.ID)
	s.Require().Equal(common.CommandStateRequestTransaction, receiverMessage.CommandParameters.CommandState)

	transactionHash := "0x412a851ac2ae51cad34a56c8a9cfee55d577ac5e1ac71cf488a2f2093a373799"
	signature, err := buildSignature(theirMessenger.identity, &theirMessenger.identity.PublicKey, transactionHash)
	s.Require().NoError(err)
	response, err = theirMessenger.AcceptRequestTransaction(context.Background(), transactionHash, initialCommandID, signature)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	senderMessage = response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, senderMessage.ContentType)

	s.Require().Equal("Transaction sent", senderMessage.Text)
	s.Require().NotNil(senderMessage.CommandParameters)
	s.Require().Equal(value, senderMessage.CommandParameters.Value)
	s.Require().Equal(contract, senderMessage.CommandParameters.Contract)
	s.Require().Equal(transactionHash, senderMessage.CommandParameters.TransactionHash)
	s.Require().Equal(receiverAddressString, senderMessage.CommandParameters.Address)
	s.Require().Equal(initialCommandID, senderMessage.CommandParameters.ID)
	s.Require().Equal(signature, senderMessage.CommandParameters.Signature)
	s.Require().NotEmpty(senderMessage.ID)
	s.Require().Equal(receiverMessage.ID, senderMessage.Replace)
	s.Require().Equal(common.CommandStateTransactionSent, senderMessage.CommandParameters.CommandState)

	var transactions []*TransactionToValidate
	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		var err error

		_, err = s.m.RetrieveAll()
		if err != nil {
			return err
		}
		transactions, err = s.m.persistence.TransactionsToValidate()
		if err == nil && len(transactions) == 0 {
			err = errors.New("no transactions")
		}
		return err
	})
	s.Require().NoError(err)

	actualTransaction := transactions[0]

	s.Require().Equal(&theirMessenger.identity.PublicKey, actualTransaction.From)
	s.Require().Equal(transactionHash, actualTransaction.TransactionHash)
	s.Require().True(actualTransaction.Validate)
	s.Require().Equal(initialCommandID, actualTransaction.CommandID)

	senderAddress := crypto.PubkeyToAddress(theirMessenger.identity.PublicKey)

	contractAddress := types.HexToAddress(contract)
	client := MockEthClient{}
	valueBig, ok := big.NewInt(0).SetString(value, 10)
	s.Require().True(ok)
	client.messages = make(map[string]MockTransaction)
	client.messages[transactionHash] = MockTransaction{
		Status: coretypes.TransactionStatusSuccess,
		Message: coretypes.NewMessage(
			senderAddress,
			&contractAddress,
			1,
			nil,
			0,
			nil,
			buildData(transferFunction, receiverAddress, valueBig),
			false,
		),
	}
	s.m.verifyTransactionClient = client
	response, err = s.m.ValidateTransactions(context.Background(), []types.Address{receiverAddress})
	s.Require().NoError(err)

	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	receiverMessage = response.Messages()[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, receiverMessage.ContentType)

	s.Require().Equal("Transaction received", receiverMessage.Text)
	s.Require().NotNil(receiverMessage.CommandParameters)
	s.Require().Equal(value, receiverMessage.CommandParameters.Value)
	s.Require().Equal(contract, receiverMessage.CommandParameters.Contract)
	s.Require().Equal(transactionHash, receiverMessage.CommandParameters.TransactionHash)
	s.Require().Equal(receiverAddressString, receiverMessage.CommandParameters.Address)
	s.Require().Equal(initialCommandID, receiverMessage.CommandParameters.ID)
	s.Require().Equal(signature, receiverMessage.CommandParameters.Signature)
	s.Require().Equal(common.CommandStateTransactionSent, receiverMessage.CommandParameters.CommandState)
	s.Require().Equal(senderMessage.ID, receiverMessage.ID)
	s.Require().Equal(senderMessage.Replace, senderMessage.Replace)
}

type MockTransaction struct {
	Status  coretypes.TransactionStatus
	Message coretypes.Message
}

type MockEthClient struct {
	messages map[string]MockTransaction
}

func (m MockEthClient) TransactionByHash(ctx context.Context, hash types.Hash) (coretypes.Message, coretypes.TransactionStatus, error) {
	mockTransaction, ok := m.messages[hash.Hex()]
	if !ok {
		return coretypes.Message{}, coretypes.TransactionStatusFailed, nil
	}
	return mockTransaction.Message, mockTransaction.Status, nil
}

func (s *MessengerSuite) TestMessageJSON() {
	message := &common.Message{
		ID:          "test-1",
		LocalChatID: "local-chat-id",
		Alias:       "alias",
		ChatMessage: &protobuf.ChatMessage{
			ChatId:      "remote-chat-id",
			ContentType: 0,
			Text:        "test-1",
			Clock:       1,
		},
		From: testPK,
	}

	_, err := json.Marshal(message)
	s.Require().NoError(err)
}

func (s *MessengerSuite) TestSentEventTracking() {

	//when message sent, its sent field should be "false" until we got confirmation
	chat := CreatePublicChat("test-chat", s.m.transport)
	err := s.m.SaveChat(chat)
	s.NoError(err)
	inputMessage := buildTestMessage(*chat)

	_, err = s.m.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	rawMessage, err := s.m.persistence.RawMessageByID(inputMessage.ID)
	s.NoError(err)
	s.False(rawMessage.Sent)

	//when message sent, its sent field should be true after we got confirmation
	err = s.m.processSentMessages([]string{inputMessage.ID})
	s.NoError(err)

	rawMessage, err = s.m.persistence.RawMessageByID(inputMessage.ID)
	s.NoError(err)
	s.True(rawMessage.Sent)
}

func (s *MessengerSuite) TestLastSentField() {
	//send message
	chat := CreatePublicChat("test-chat", s.m.transport)
	err := s.m.SaveChat(chat)
	s.NoError(err)
	inputMessage := buildTestMessage(*chat)

	_, err = s.m.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	rawMessage, err := s.m.persistence.RawMessageByID(inputMessage.ID)
	s.NoError(err)
	s.Equal(1, rawMessage.SendCount)

	//make sure LastSent is set
	s.NotEqual(uint64(0), rawMessage.LastSent, "rawMessage.LastSent should be non-zero after sending")
}

func (s *MessengerSuite) TestShouldResendEmoji() {
	// shouldn't try to resend non-emoji messages.
	ok, err := shouldResendMessage(&common.RawMessage{
		MessageType: protobuf.ApplicationMetadataMessage_CONTACT_UPDATE,
		Sent:        false,
		SendCount:   2,
	}, s.m.getTimesource())
	s.Error(err)
	s.False(ok)

	// shouldn't try to resend already sent message
	ok, err = shouldResendMessage(&common.RawMessage{
		MessageType: protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
		Sent:        true,
		SendCount:   1,
	}, s.m.getTimesource())
	s.Error(err)
	s.False(ok)

	// messages that already sent to many times shouldn't be resend
	ok, err = shouldResendMessage(&common.RawMessage{
		MessageType: protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
		Sent:        false,
		SendCount:   messageResendMaxCount + 1,
	}, s.m.getTimesource())
	s.NoError(err)
	s.False(ok)

	// message sent one time CAN'T be resend in 15 seconds (only after 30)
	ok, err = shouldResendMessage(&common.RawMessage{
		MessageType: protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
		Sent:        false,
		SendCount:   1,
		LastSent:    s.m.getTimesource().GetCurrentTime() - 15*uint64(time.Second.Milliseconds()),
	}, s.m.getTimesource())
	s.NoError(err)
	s.False(ok)

	// message sent one time CAN be resend in 35 seconds
	ok, err = shouldResendMessage(&common.RawMessage{
		MessageType: protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
		Sent:        false,
		SendCount:   1,
		LastSent:    s.m.getTimesource().GetCurrentTime() - 35*uint64(time.Second.Milliseconds()),
	}, s.m.getTimesource())
	s.NoError(err)
	s.True(ok)

	// message sent three times CAN'T be resend in 100 seconds (only after 120)
	ok, err = shouldResendMessage(&common.RawMessage{
		MessageType: protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
		Sent:        false,
		SendCount:   3,
		LastSent:    s.m.getTimesource().GetCurrentTime() - 100*uint64(time.Second.Milliseconds()),
	}, s.m.getTimesource())
	s.NoError(err)
	s.False(ok)

	// message sent tow times CAN be resend in 65 seconds
	ok, err = shouldResendMessage(&common.RawMessage{
		MessageType: protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
		Sent:        false,
		SendCount:   3,
		LastSent:    s.m.getTimesource().GetCurrentTime() - 125*uint64(time.Second.Milliseconds()),
	}, s.m.getTimesource())
	s.NoError(err)
	s.True(ok)
}

func (s *MessengerSuite) TestSendMessageWithPreviews() {
	httpServer, err := server.NewMediaServer(s.m.database, nil, nil)
	s.Require().NoError(err)
	err = httpServer.SetPort(9876)
	s.NoError(err)
	s.m.httpServer = httpServer

	chat := CreatePublicChat("test-chat", s.m.transport)
	err = s.m.SaveChat(chat)
	s.NoError(err)
	inputMsg := buildTestMessage(*chat)

	preview := common.LinkPreview{
		Type:        protobuf.UnfurledLink_LINK,
		URL:         "https://github.com",
		Title:       "Build software better, together",
		Description: "GitHub is where people build software.",
		Thumbnail: common.LinkPreviewThumbnail{
			DataURI: "data:image/png;base64,iVBORw0KGgoAAAANSUg=",
			Width:   100,
			Height:  200,
		},
	}
	inputMsg.LinkPreviews = []common.LinkPreview{preview}

	sentContactPreview := common.StatusLinkPreview{
		URL: "https://status.app/u/TestUrl",
		Contact: &common.StatusContactLinkPreview{
			PublicKey:   "TestPublicKey",
			DisplayName: "TestDisplayName",
			Description: "Test description",
			Icon: common.LinkPreviewThumbnail{
				Width:   100,
				Height:  200,
				DataURI: "data:image/png;base64,iVBORw0KGgoAAAANSUg=",
			},
		},
	}
	inputMsg.StatusLinkPreviews = []common.StatusLinkPreview{sentContactPreview}

	_, err = s.m.SendChatMessage(context.Background(), inputMsg)
	s.NoError(err)

	savedMsgs, _, err := s.m.MessageByChatID(chat.ID, "", 10)
	s.Require().NoError(err)
	s.Require().Len(savedMsgs, 1)
	savedMsg := savedMsgs[0]

	// Test unfurled links have been saved.
	s.Require().Len(savedMsg.UnfurledLinks, 1)
	savedLinkProto := savedMsg.UnfurledLinks[0]
	s.Require().Equal(preview.Type, savedLinkProto.Type)
	s.Require().Equal(preview.URL, savedLinkProto.Url)
	s.Require().Equal(preview.Title, savedLinkProto.Title)
	s.Require().Equal(preview.Description, savedLinkProto.Description)

	// Test the saved link thumbnail can be encoded as a data URI.
	expectedDataURI, err := images.GetPayloadDataURI(savedLinkProto.ThumbnailPayload)
	s.Require().NoError(err)
	s.Require().Equal(preview.Thumbnail.DataURI, expectedDataURI)

	s.Require().Equal(
		httpServer.MakeLinkPreviewThumbnailURL(inputMsg.ID, preview.URL),
		savedMsg.LinkPreviews[0].Thumbnail.URL,
	)

	// Check saved message protobuf fields
	s.Require().NotNil(savedMsg.UnfurledStatusLinks)
	s.Require().Len(savedMsg.UnfurledStatusLinks.UnfurledStatusLinks, 1)
	savedStatusLinkProto := savedMsg.UnfurledStatusLinks.UnfurledStatusLinks[0]
	s.Require().Equal(sentContactPreview.URL, savedStatusLinkProto.Url)
	s.Require().NotNil(savedStatusLinkProto.GetContact())
	s.Require().Nil(savedStatusLinkProto.GetCommunity())
	s.Require().Nil(savedStatusLinkProto.GetChannel())

	savedContactProto := savedStatusLinkProto.GetContact()
	s.Require().Equal(sentContactPreview.Contact.PublicKey, savedContactProto.PublicKey)
	s.Require().Equal(sentContactPreview.Contact.DisplayName, savedContactProto.DisplayName)
	s.Require().Equal(sentContactPreview.Contact.Description, savedContactProto.Description)
	s.Require().NotNil(savedContactProto.Icon)
	s.Require().Equal(sentContactPreview.Contact.Icon.Width, int(savedContactProto.Icon.Width))
	s.Require().Equal(sentContactPreview.Contact.Icon.Height, int(savedContactProto.Icon.Height))

	iconDataURI, err := images.GetPayloadDataURI(savedContactProto.Icon.Payload)
	s.Require().NoError(err)
	s.Require().Equal(sentContactPreview.Contact.Icon.DataURI, iconDataURI)

	// Check message `StatusLinkPreviews` properties
	s.Require().Len(savedMsg.StatusLinkPreviews, 1)
	savedStatusLinkPreview := savedMsg.StatusLinkPreviews[0]
	s.Require().Equal(sentContactPreview.URL, savedStatusLinkPreview.URL)
	s.Require().NotNil(savedStatusLinkPreview.Contact)

	savedContact := savedStatusLinkPreview.Contact
	s.Require().Equal(sentContactPreview.Contact.PublicKey, savedContact.PublicKey)
	s.Require().Equal(sentContactPreview.Contact.DisplayName, savedContact.DisplayName)
	s.Require().Equal(sentContactPreview.Contact.Description, savedContact.Description)
	s.Require().NotNil(savedContact.Icon)
	s.Require().Equal(sentContactPreview.Contact.Icon.Width, savedContact.Icon.Width)
	s.Require().Equal(sentContactPreview.Contact.Icon.Height, savedContact.Icon.Height)
	expectedIconURL := httpServer.MakeStatusLinkPreviewThumbnailURL(inputMsg.ID, sentContactPreview.URL, "contact-icon")
	s.Require().Equal(expectedIconURL, savedContact.Icon.URL)
}

func (s *MessengerSuite) TestMessageSent() {
	//send message
	chat := CreatePublicChat("test-chat", s.m.transport)
	err := s.m.SaveChat(chat)
	s.NoError(err)
	inputMessage := buildTestMessage(*chat)

	_, err = s.m.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	rawMessage, err := s.m.persistence.RawMessageByID(inputMessage.ID)
	s.NoError(err)
	s.Equal(1, rawMessage.SendCount)
	s.False(rawMessage.Sent)

	//imitate chat message sent
	err = s.m.processSentMessages([]string{inputMessage.ID})
	s.NoError(err)

	rawMessage, err = s.m.persistence.RawMessageByID(inputMessage.ID)
	s.NoError(err)
	s.Equal(1, rawMessage.SendCount)
	s.True(rawMessage.Sent)
}

func (s *MessengerSuite) TestProcessSentMessages() {
	ids := []string{"a"}
	err := s.m.processSentMessages(ids)
	s.Require().NoError(err)
}

func (s *MessengerSuite) TestResendExpiredEmojis() {
	//send message
	chat := CreatePublicChat("test-chat", s.m.transport)
	err := s.m.SaveChat(chat)
	s.NoError(err)
	inputMessage := buildTestMessage(*chat)

	_, err = s.m.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	//create emoji
	_, err = s.m.SendEmojiReaction(context.Background(), chat.ID, inputMessage.ID, protobuf.EmojiReaction_SAD)
	s.Require().NoError(err)

	ids, err := s.m.persistence.RawMessagesIDsByType(protobuf.ApplicationMetadataMessage_EMOJI_REACTION)
	s.Require().NoError(err)
	emojiID := ids[0]

	//check that emoji was sent one time
	rawMessage, err := s.m.persistence.RawMessageByID(emojiID)
	s.NoError(err)
	s.False(rawMessage.Sent)
	s.Equal(1, rawMessage.SendCount)

	//imitate that more than 30 seconds passed since message was sent
	rawMessage.LastSent = rawMessage.LastSent - 35*uint64(time.Second.Milliseconds())
	err = s.m.persistence.SaveRawMessage(rawMessage)
	s.NoError(err)
	time.Sleep(2 * time.Second)

	//make sure it was resent and SendCount incremented
	rawMessage, err = s.m.persistence.RawMessageByID(emojiID)
	s.NoError(err)
	s.True(rawMessage.SendCount >= 2)
}

func buildImageWithAlbumIDMessage(chat Chat, albumID string) (*common.Message, error) {
	file, err := os.Open("../_assets/tests/test.jpg")
	if err != err {
		return nil, err
	}
	defer file.Close()

	payload, err := ioutil.ReadAll(file)
	if err != err {
		return nil, err
	}

	clock, timestamp := chat.NextClockAndTimestamp(&testTimeSource{})
	message := common.NewMessage()
	message.ChatId = chat.ID
	message.Clock = clock
	message.Timestamp = timestamp
	message.WhisperTimestamp = clock
	message.LocalChatID = chat.ID
	message.MessageType = protobuf.MessageType_ONE_TO_ONE
	message.ContentType = protobuf.ChatMessage_IMAGE

	image := protobuf.ImageMessage{
		Payload: payload,
		Type:    protobuf.ImageType_JPEG,
		Width:   1200,
		Height:  1000,
		AlbumId: albumID,
	}
	message.Payload = &protobuf.ChatMessage_Image{Image: &image}

	return message, nil
}

func buildImageWithoutAlbumIDMessage(chat Chat) (*common.Message, error) {
	return buildImageWithAlbumIDMessage(chat, "")
}

type testTimeSource struct{}

func (t *testTimeSource) GetCurrentTime() uint64 {
	return uint64(time.Now().Unix())
}

func (s *MessengerSuite) TestSendMessageMention() {
	// Initialize Alice and Bob's messengers
	alice, bob := s.m, s.newMessenger()
	_, err := bob.Start()
	s.Require().NoError(err)
	defer bob.Shutdown() // nolint: errcheck

	// Set display names for Bob and Alice
	s.Require().NoError(bob.settings.SaveSettingField(settings.DisplayName, "bobby"))
	s.Require().NoError(alice.settings.SaveSettingField(settings.DisplayName, "Alice"))
	s.Require().NoError(alice.settings.SaveSettingField(settings.NotificationsEnabled, true))

	// Create one-to-one chats
	chat, chat2 := CreateOneToOneChat(common.PubkeyToHex(&alice.identity.PublicKey), &alice.identity.PublicKey, bob.transport),
		CreateOneToOneChat(common.PubkeyToHex(&bob.identity.PublicKey), &bob.identity.PublicKey, alice.transport)
	s.Require().NoError(bob.SaveChat(chat))
	s.Require().NoError(alice.SaveChat(chat2))

	// Prepare the message and Send the message from Bob to Alice
	inputMessage := common.NewMessage()
	inputMessage.ChatId = chat.ID
	inputMessage.Text = fmt.Sprintf("@%s talk to @%s", alice.myHexIdentity(), bob.myHexIdentity())
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	_, err = bob.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	// Wait for Alice to receive the message and make sure it's properly formatted
	response, err := WaitOnMessengerResponse(alice, func(r *MessengerResponse) bool { return len(r.Notifications()) >= 1 }, "no messages")
	s.Require().NoError(err)
	s.Require().Equal("Alice talk to bobby", response.Notifications()[0].Message)
}
