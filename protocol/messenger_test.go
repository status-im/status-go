package protocol

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/mutecomm/go-sqlcipher" // require go-sqlcipher that overrides default implementation
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	enstypes "github.com/status-im/status-go/eth-node/types/ens"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/tt"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/whisper/v6"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

func TestMessengerSuite(t *testing.T) {
	suite.Run(t, new(MessengerSuite))
}

func TestMessengerWithDataSyncEnabledSuite(t *testing.T) {
	suite.Run(t, &MessengerSuite{enableDataSync: true})
}

func TestPostProcessorSuite(t *testing.T) {
	suite.Run(t, new(PostProcessorSuite))
}

type MessengerSuite struct {
	suite.Suite

	enableDataSync bool

	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single Whisper service should be shared.
	shh      types.Whisper
	tmpFiles []*os.File // files to clean up
	logger   *zap.Logger
}

type testNode struct {
	shh types.Whisper
}

func (n *testNode) NewENSVerifier(_ *zap.Logger) enstypes.ENSVerifier {
	panic("not implemented")
}

func (n *testNode) GetWhisper(_ interface{}) (types.Whisper, error) {
	return n.shh, nil
}

func (s *MessengerSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := whisper.DefaultConfig
	config.MinimumAcceptedPOW = 0
	shh := whisper.New(&config)
	s.shh = gethbridge.NewGethWhisperWrapper(shh)
	s.Require().NoError(shh.Start(nil))

	s.m = s.newMessenger(s.shh)
	s.privateKey = s.m.identity
}

func (s *MessengerSuite) newMessengerWithKey(shh types.Whisper, privateKey *ecdsa.PrivateKey) *Messenger {
	tmpFile, err := ioutil.TempFile("", "")
	s.Require().NoError(err)

	options := []Option{
		WithCustomLogger(s.logger),
		WithMessagesPersistenceEnabled(),
		WithDatabaseConfig(tmpFile.Name(), "some-key"),
	}
	if s.enableDataSync {
		options = append(options, WithDatasync())
	}
	m, err := NewMessenger(
		privateKey,
		&testNode{shh: shh},
		uuid.New().String(),
		options...,
	)
	s.Require().NoError(err)

	err = m.Init()
	s.Require().NoError(err)

	s.tmpFiles = append(s.tmpFiles, tmpFile)

	return m
}

func (s *MessengerSuite) newMessenger(shh types.Whisper) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return s.newMessengerWithKey(s.shh, privateKey)
}

func (s *MessengerSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
	for _, f := range s.tmpFiles {
		_ = os.Remove(f.Name())
	}
	_ = s.logger.Sync()
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
				contact := Contact{
					ID:         types.EncodeHex(crypto.FromECDSAPub(&key.PublicKey)),
					Name:       "Some Contact",
					SystemTags: []string{contactAdded},
				}
				err = s.m.SaveContact(&contact)
				s.Require().NoError(err)
			},
			AddedFilters: 1,
		},
		{
			Name: "added and blocked contact",
			Prep: func() {
				key, err := crypto.GenerateKey()
				s.Require().NoError(err)
				contact := Contact{
					ID:         types.EncodeHex(crypto.FromECDSAPub(&key.PublicKey)),
					Name:       "Some Contact",
					SystemTags: []string{contactAdded, contactBlocked},
				}
				err = s.m.SaveContact(&contact)
				s.Require().NoError(err)
			},
			AddedFilters: 0,
		},
		{
			Name: "added by them contact",
			Prep: func() {
				key, err := crypto.GenerateKey()
				s.Require().NoError(err)
				contact := Contact{
					ID:         types.EncodeHex(crypto.FromECDSAPub(&key.PublicKey)),
					Name:       "Some Contact",
					SystemTags: []string{contactRequestReceived},
				}
				err = s.m.SaveContact(&contact)
				s.Require().NoError(err)
			},
			AddedFilters: 0,
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
			s.Equal(expectedFilters, len(filters))
		})
	}
}

func buildTestMessage(chat Chat) *Message {

	message := &Message{}
	message.Text = "text-input-message"
	message.ChatId = chat.ID
	message.Clock = 2
	message.WhisperTimestamp = 10
	message.LocalChatID = chat.ID
	message.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	switch chat.ChatType {
	case ChatTypePublic:
		message.MessageType = protobuf.ChatMessage_PUBLIC_GROUP
	case ChatTypeOneToOne:
		message.MessageType = protobuf.ChatMessage_ONE_TO_ONE
	case ChatTypePrivateGroupChat:
		message.MessageType = protobuf.ChatMessage_PRIVATE_GROUP
	}

	return message
}

func (s *MessengerSuite) TestMarkMessagesSeen() {
	chat := CreatePublicChat("test-chat")
	chat.UnviewedMessagesCount = 2
	err := s.m.SaveChat(&chat)
	s.Require().NoError(err)
	inputMessage1 := buildTestMessage(chat)
	inputMessage1.ID = "1"
	inputMessage1.Seen = false
	inputMessage2 := buildTestMessage(chat)
	inputMessage2.ID = "2"
	inputMessage2.Seen = false

	err = s.m.SaveMessages([]*Message{inputMessage1, inputMessage2})
	s.Require().NoError(err)

	err = s.m.MarkMessagesSeen(chat.ID, []string{inputMessage1.ID})
	s.Require().NoError(err)

	chats := s.m.Chats()
	s.Require().Len(chats, 1)
	s.Require().Equal(uint(1), chats[0].UnviewedMessagesCount)
}

func (s *MessengerSuite) TestSendPublic() {
	chat := CreatePublicChat("test-chat")
	chat.LastClockValue = uint64(100000000000000)
	err := s.m.SaveChat(&chat)
	s.NoError(err)
	inputMessage := buildTestMessage(chat)
	response, err := s.m.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	s.Require().Equal(1, len(response.Messages), "it returns the message")
	outputMessage := response.Messages[0]

	s.Require().Equal(uint64(100000000000001), outputMessage.Clock, "it correctly sets the clock")
	s.Require().Equal(uint64(100000000000001), chat.LastClockValue, "it correctly sets the last-clock-value")
	s.Require().NotEqual(uint64(0), chat.Timestamp, "it sets the timestamp")
	s.Require().Equal("0x"+hex.EncodeToString(crypto.FromECDSAPub(&s.privateKey.PublicKey)), outputMessage.From, "it sets the From field")
	s.Require().True(outputMessage.Seen, "it marks the message as seen")
	s.Require().Equal(outputMessage.OutgoingStatus, OutgoingStatusSending, "it marks the message as sending")
	s.Require().NotEmpty(outputMessage.ID, "it sets the ID field")
	s.Require().Equal(protobuf.ChatMessage_PUBLIC_GROUP, outputMessage.MessageType)

	savedMessages, _, err := s.m.MessageByChatID(chat.ID, "", 10)
	s.Require().NoError(err)
	s.Require().Equal(1, len(savedMessages), "it saves the message")
}

func (s *MessengerSuite) TestSendPrivateOneToOne() {
	recipientKey, err := crypto.GenerateKey()
	s.NoError(err)
	pkString := hex.EncodeToString(crypto.FromECDSAPub(&recipientKey.PublicKey))
	chat := CreateOneToOneChat(pkString, &recipientKey.PublicKey)

	inputMessage := &Message{}
	inputMessage.ChatId = chat.ID
	chat.LastClockValue = uint64(100000000000000)
	err = s.m.SaveChat(&chat)
	s.NoError(err)
	response, err := s.m.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Equal(1, len(response.Messages), "it returns the message")
	outputMessage := response.Messages[0]

	s.Require().Equal(uint64(100000000000001), outputMessage.Clock, "it correctly sets the clock")
	s.Require().Equal(uint64(100000000000001), chat.LastClockValue, "it correctly sets the last-clock-value")

	s.Require().NotEqual(uint64(0), chat.Timestamp, "it sets the timestamp")
	s.Require().Equal("0x"+hex.EncodeToString(crypto.FromECDSAPub(&s.privateKey.PublicKey)), outputMessage.From, "it sets the From field")
	s.Require().True(outputMessage.Seen, "it marks the message as seen")
	s.Require().Equal(outputMessage.OutgoingStatus, OutgoingStatusSending, "it marks the message as sending")
	s.Require().NotEmpty(outputMessage.ID, "it sets the ID field")
	s.Require().Equal(protobuf.ChatMessage_ONE_TO_ONE, outputMessage.MessageType)
}

func (s *MessengerSuite) TestSendPrivateGroup() {
	response, err := s.m.CreateGroupChatWithMembers(context.Background(), "test", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats, 1)

	chat := response.Chats[0]
	key, err := crypto.GenerateKey()
	s.NoError(err)
	members := []string{"0x" + hex.EncodeToString(crypto.FromECDSAPub(&key.PublicKey))}
	_, err = s.m.AddMembersToGroupChat(context.Background(), chat.ID, members)
	s.NoError(err)

	inputMessage := &Message{}
	inputMessage.ChatId = chat.ID
	chat.LastClockValue = uint64(100000000000000)
	err = s.m.SaveChat(chat)
	s.NoError(err)
	response, err = s.m.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Equal(1, len(response.Messages), "it returns the message")
	outputMessage := response.Messages[0]

	s.Require().Equal(uint64(100000000000001), outputMessage.Clock, "it correctly sets the clock")
	s.Require().Equal(uint64(100000000000001), chat.LastClockValue, "it correctly sets the last-clock-value")

	s.Require().NotEqual(uint64(0), chat.Timestamp, "it sets the timestamp")
	s.Require().Equal("0x"+hex.EncodeToString(crypto.FromECDSAPub(&s.privateKey.PublicKey)), outputMessage.From, "it sets the From field")
	s.Require().True(outputMessage.Seen, "it marks the message as seen")
	s.Require().Equal(outputMessage.OutgoingStatus, OutgoingStatusSending, "it marks the message as sending")
	s.Require().NotEmpty(outputMessage.ID, "it sets the ID field")
	s.Require().Equal(protobuf.ChatMessage_PRIVATE_GROUP, outputMessage.MessageType)
}

func (s *MessengerSuite) TestSendPrivateEmptyGroup() {
	response, err := s.m.CreateGroupChatWithMembers(context.Background(), "test", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats, 1)

	chat := response.Chats[0]

	inputMessage := &Message{}
	inputMessage.ChatId = chat.ID
	chat.LastClockValue = uint64(100000000000000)
	err = s.m.SaveChat(chat)
	s.NoError(err)
	response, err = s.m.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Equal(1, len(response.Messages), "it returns the message")
	outputMessage := response.Messages[0]

	s.Require().Equal(uint64(100000000000001), outputMessage.Clock, "it correctly sets the clock")
	s.Require().Equal(uint64(100000000000001), chat.LastClockValue, "it correctly sets the last-clock-value")

	s.Require().NotEqual(uint64(0), chat.Timestamp, "it sets the timestamp")
	s.Require().Equal("0x"+hex.EncodeToString(crypto.FromECDSAPub(&s.privateKey.PublicKey)), outputMessage.From, "it sets the From field")
	s.Require().True(outputMessage.Seen, "it marks the message as seen")
	s.Require().Equal(outputMessage.OutgoingStatus, OutgoingStatusSending, "it marks the message as sending")
	s.Require().NotEmpty(outputMessage.ID, "it sets the ID field")
	s.Require().Equal(protobuf.ChatMessage_PRIVATE_GROUP, outputMessage.MessageType)
}

// Make sure public messages sent by us are not
func (s *MessengerSuite) TestRetrieveOwnPublic() {
	chat := CreatePublicChat("status")
	err := s.m.SaveChat(&chat)
	s.NoError(err)
	// Right-to-left text
	text := "پيل اندر خانه يي تاريک بود عرضه را آورده بودندش هنود  i\nاز براي ديدنش مردم بسي اندر آن ظلمت همي شد هر کسي"

	inputMessage := buildTestMessage(chat)
	inputMessage.ChatId = chat.ID
	inputMessage.Text = text

	response, err := s.m.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	s.Require().Len(response.Messages, 1)

	textMessage := response.Messages[0]

	s.Equal(textMessage.Text, text)
	s.NotNil(textMessage.ParsedText)
	s.True(textMessage.RTL)
	s.Equal(1, textMessage.LineCount)

	s.Require().Len(response.Chats, 1)
	actualChat := response.Chats[0]
	// It does not set the unviewed messages count
	s.Require().Equal(uint(0), actualChat.UnviewedMessagesCount)
	// It updates the last message clock value
	s.Require().Equal(textMessage.Clock, actualChat.LastClockValue)
	// It sets the last message
	s.Require().NotNil(actualChat.LastMessage)
}

// Retrieve their public message
func (s *MessengerSuite) TestRetrieveTheirPublic() {
	theirMessenger := s.newMessenger(s.shh)
	theirChat := CreatePublicChat("status")
	err := theirMessenger.SaveChat(&theirChat)
	s.Require().NoError(err)

	chat := CreatePublicChat("status")
	err = s.m.SaveChat(&chat)
	s.Require().NoError(err)

	err = s.m.Join(chat)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(chat)

	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	sentMessage := sendResponse.Messages[0]

	// Wait for the message to reach its destination
	var response *MessengerResponse
	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err = s.m.RetrieveAll()
		if err == nil && len(response.Messages) == 0 {
			err = errors.New("no messages")
		}
		return err
	})
	s.Require().NoError(err)

	s.Require().Len(response.Messages, 1)
	s.Require().Len(response.Chats, 1)
	actualChat := response.Chats[0]
	// It sets the unviewed messages count
	s.Require().Equal(uint(1), actualChat.UnviewedMessagesCount)
	// It updates the last message clock value
	s.Require().Equal(sentMessage.Clock, actualChat.LastClockValue)
	// It sets the last message
	s.Require().NotNil(actualChat.LastMessage)
}

func (s *MessengerSuite) TestDeletedAtClockValue() {
	theirMessenger := s.newMessenger(s.shh)
	theirChat := CreatePublicChat("status")
	err := theirMessenger.SaveChat(&theirChat)
	s.Require().NoError(err)

	chat := CreatePublicChat("status")
	err = s.m.SaveChat(&chat)
	s.Require().NoError(err)

	err = s.m.Join(chat)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(chat)

	sentResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	chat.DeletedAtClockValue = sentResponse.Messages[0].Clock
	err = s.m.SaveChat(&chat)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	time.Sleep(100 * time.Millisecond)
	response, err := s.m.RetrieveAll()
	s.Require().NoError(err)
	s.Require().Len(response.Messages, 0)
}

func (s *MessengerSuite) TestRetrieveBlockedContact() {
	theirMessenger := s.newMessenger(s.shh)
	theirChat := CreatePublicChat("status")
	err := theirMessenger.SaveChat(&theirChat)
	s.Require().NoError(err)

	chat := CreatePublicChat("status")
	err = s.m.SaveChat(&chat)
	s.Require().NoError(err)

	err = s.m.Join(chat)
	s.Require().NoError(err)

	publicKeyHex := "0x" + hex.EncodeToString(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	blockedContact := Contact{
		ID:            publicKeyHex,
		Address:       "contact-address",
		Name:          "contact-name",
		Photo:         "contact-photo",
		LastUpdated:   20,
		SystemTags:    []string{contactBlocked},
		TributeToTalk: "talk",
	}

	s.Require().NoError(s.m.SaveContact(&blockedContact))

	inputMessage := buildTestMessage(chat)

	_, err = theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	// Wait for the message to reach its destination
	time.Sleep(100 * time.Millisecond)
	response, err := s.m.RetrieveAll()
	s.Require().NoError(err)
	s.Require().Len(response.Messages, 0)
}

// Resend their public message, receive only once
func (s *MessengerSuite) TestResendPublicMessage() {
	theirMessenger := s.newMessenger(s.shh)
	theirChat := CreatePublicChat("status")
	err := theirMessenger.SaveChat(&theirChat)
	s.Require().NoError(err)

	chat := CreatePublicChat("status")
	err = s.m.SaveChat(&chat)
	s.Require().NoError(err)

	err = s.m.Join(chat)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(chat)

	sendResponse1, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	sentMessage := sendResponse1.Messages[0]

	sendResponse2, err := theirMessenger.ReSendChatMessage(context.Background(), sentMessage.ID)
	s.Require().NoError(err)

	s.Require().Equal(sendResponse1.Messages[0].ID, sendResponse2.Messages[0].ID)

	// Wait for the message to reach its destination
	var response *MessengerResponse
	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err = s.m.RetrieveAll()
		if err == nil && len(response.Messages) == 0 {
			err = errors.New("no messages")
		}
		return err
	})
	s.Require().NoError(err)

	s.Require().Len(response.Messages, 1)
	s.Require().Len(response.Chats, 1)
	actualChat := response.Chats[0]
	// It sets the unviewed messages count
	s.Require().Equal(uint(1), actualChat.UnviewedMessagesCount)
	// It updates the last message clock value
	s.Require().Equal(sentMessage.Clock, actualChat.LastClockValue)
	// It sets the last message
	s.Require().NotNil(actualChat.LastMessage)

	// We send the messag again
	_, err = theirMessenger.ReSendChatMessage(context.Background(), sentMessage.ID)
	s.Require().NoError(err)

	// It should not be retrieved anymore
	time.Sleep(100 * time.Millisecond)
	response, err = s.m.RetrieveAll()
	s.Require().NoError(err)
	s.Require().Len(response.Messages, 0)
}

// Test receiving a message on an existing private chat
func (s *MessengerSuite) TestRetrieveTheirPrivateChatExisting() {
	theirMessenger := s.newMessenger(s.shh)
	theirChat := CreateOneToOneChat("XXX", &s.privateKey.PublicKey)
	err := theirMessenger.SaveChat(&theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("our-chat", &theirMessenger.identity.PublicKey)
	ourChat.UnviewedMessagesCount = 1
	// Make chat inactive
	ourChat.Active = false
	err = s.m.SaveChat(&ourChat)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(theirChat)

	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages, 1)

	sentMessage := sendResponse.Messages[0]

	var response *MessengerResponse
	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err = s.m.RetrieveAll()
		if err == nil && len(response.Messages) == 0 {
			err = errors.New("no messages")
		}
		return err
	})
	s.Require().NoError(err)

	s.Require().Equal(len(response.Chats), 1)
	actualChat := response.Chats[0]
	// It updates the unviewed messages count
	s.Require().Equal(uint(2), actualChat.UnviewedMessagesCount)
	// It updates the last message clock value
	s.Require().Equal(sentMessage.Clock, actualChat.LastClockValue)
	// It sets the last message
	s.Require().NotNil(actualChat.LastMessage)
	s.Require().True(actualChat.Active)
}

// Test receiving a message on an non-existing private chat
func (s *MessengerSuite) TestRetrieveTheirPrivateChatNonExisting() {
	theirMessenger := s.newMessenger(s.shh)
	chat := CreateOneToOneChat("XXX", &s.privateKey.PublicKey)
	err := theirMessenger.SaveChat(&chat)
	s.NoError(err)

	inputMessage := buildTestMessage(chat)

	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages, 1)

	sentMessage := sendResponse.Messages[0]

	// Wait for the message to reach its destination
	var response *MessengerResponse
	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err = s.m.RetrieveAll()
		if err == nil && len(response.Messages) == 0 {
			err = errors.New("no messages")
		}
		return err
	})
	s.Require().NoError(err)

	s.Require().Len(response.Chats, 1)
	actualChat := response.Chats[0]
	// It updates the unviewed messages count
	s.Require().Equal(uint(1), actualChat.UnviewedMessagesCount)
	// It updates the last message clock value
	s.Require().Equal(sentMessage.Clock, actualChat.LastClockValue)
	// It sets the last message
	s.Require().NotNil(actualChat.LastMessage)
	// It sets the chat as active
	s.Require().True(actualChat.Active)
}

// Test retrieve paired message
func (s *MessengerSuite) TestRetrieveOurPairedMessage() {
	pairedMessenger := s.newMessengerWithKey(s.shh, s.privateKey)
	chat := CreateOneToOneChat("XXX", &s.privateKey.PublicKey)
	err := pairedMessenger.SaveChat(&chat)
	s.NoError(err)

	inputMessage := buildTestMessage(chat)

	// Send a message so we now of the installation
	sendResponse, err := pairedMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages, 1)

	sentMessage := sendResponse.Messages[0]

	// Wait for the message to reach its destination

	var response *MessengerResponse
	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err = s.m.RetrieveAll()
		if err == nil && len(response.Messages) == 0 {
			err = errors.New("no messages")
		}
		return err
	})
	s.Require().NoError(err)
	// Check message is received
	s.Require().Len(response.Messages, 1)

	actualChat := response.Chats[0]
	// It does not update the unviewed message count
	s.Require().Equal(uint(0), actualChat.UnviewedMessagesCount)
	// It updates the last message clock value
	s.Require().Equal(sentMessage.Clock, actualChat.LastClockValue)
	// It sets the last message
	s.Require().NotNil(actualChat.LastMessage)

	// Get installations
	installations, err := s.m.Installations()
	s.Require().NoError(err)
	s.Require().Len(installations, 2)

	// Enable installations
	err = s.m.EnableInstallation(installations[0].ID)
	s.Require().NoError(err)
	err = s.m.EnableInstallation(installations[1].ID)
	s.Require().NoError(err)

	// We create new one to one chat
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	chat = CreateOneToOneChat("new-chat", &key.PublicKey)
	err = s.m.SaveChat(&chat)
	s.NoError(err)

	inputMessage = buildTestMessage(chat)
	_, err = s.m.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err = pairedMessenger.RetrieveAll()
		if err == nil && len(response.Messages) == 0 {
			err = errors.New("no messages")
		}
		return err
	})
	s.Require().NoError(err)

	// Check message is received
	s.Require().Len(response.Messages, 1)

	message := response.Messages[0]

	// The chatID is the same chatID as the received one
	s.Require().Equal(message.LocalChatID, chat.ID)

	// Sets the outgoing status
	s.Equal(message.OutgoingStatus, OutgoingStatusSent)
}

// Test receiving a message on an non-existing public chat
func (s *MessengerSuite) TestRetrieveTheirPublicChatNonExisting() {
	theirMessenger := s.newMessenger(s.shh)
	chat := CreatePublicChat("test-chat")
	err := theirMessenger.SaveChat(&chat)
	s.NoError(err)

	inputMessage := buildTestMessage(chat)

	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages, 1)

	// Wait for the message to reach its destination
	time.Sleep(100 * time.Millisecond)
	response, err := s.m.RetrieveAll()
	s.NoError(err)

	s.Require().Equal(len(response.Messages), 0)
	s.Require().Equal(len(response.Chats), 0)
}

// Test receiving a message on an non-existing private public chat
func (s *MessengerSuite) TestRetrieveTheirGroupChatNonExisting() {
	theirMessenger := s.newMessenger(s.shh)
	response, err := s.m.CreateGroupChatWithMembers(context.Background(), "test", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats, 1)

	chat := response.Chats[0]

	err = theirMessenger.SaveChat(chat)
	s.NoError(err)

	inputMessage := buildTestMessage(*chat)

	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages, 1)

	// Retrieve their messages so that the chat is created
	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err = s.m.RetrieveAll()
		if err == nil && len(response.Chats) == 1 {
			err = errors.New("chat membership update not received")
		}
		return err
	})
	s.Require().NoError(err)

	// The message is discarded
	s.Require().Equal(0, len(response.Messages))
	s.Require().Equal(0, len(response.Chats))
}

// Test receiving a message on an existing private group chat
func (s *MessengerSuite) TestRetrieveTheirPrivateGroupChat() {
	var response *MessengerResponse
	theirMessenger := s.newMessenger(s.shh)
	response, err := s.m.CreateGroupChatWithMembers(context.Background(), "id", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats, 1)

	ourChat := response.Chats[0]

	err = s.m.SaveChat(ourChat)
	s.NoError(err)

	members := []string{"0x" + hex.EncodeToString(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))}
	_, err = s.m.AddMembersToGroupChat(context.Background(), ourChat.ID, members)
	s.NoError(err)

	// Retrieve their messages so that the chat is created
	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err = theirMessenger.RetrieveAll()
		if err == nil && len(response.Chats) == 0 {
			err = errors.New("chat invitation not received")
		}
		return err
	})
	s.Require().NoError(err)

	_, err = theirMessenger.ConfirmJoiningGroup(context.Background(), ourChat.ID)
	s.NoError(err)

	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err = s.m.RetrieveAll()
		if err == nil && len(response.Chats) == 0 {
			err = errors.New("no joining group event received")
		}
		return err
	})
	s.Require().NoError(err)

	inputMessage := buildTestMessage(*ourChat)

	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages, 1)

	sentMessage := sendResponse.Messages[0]

	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err = s.m.RetrieveAll()
		if err == nil && len(response.Messages) == 0 {
			err = errors.New("no messages")
		}
		return err
	})
	s.Require().NoError(err)

	s.Require().Len(response.Chats, 1)
	actualChat := response.Chats[0]
	// It updates the unviewed messages count
	s.Require().Equal(uint(1), actualChat.UnviewedMessagesCount)
	// It updates the last message clock value
	s.Require().Equal(sentMessage.Clock, actualChat.LastClockValue)
	// It sets the last message
	s.Require().NotNil(actualChat.LastMessage)
}

// Test receiving a message on an existing private group chat, if messages
// are not wrapped this will fail as they'll likely come out of order
func (s *MessengerSuite) TestRetrieveTheirPrivateGroupWrappedMessageChat() {
	var response *MessengerResponse
	theirMessenger := s.newMessenger(s.shh)
	response, err := s.m.CreateGroupChatWithMembers(context.Background(), "id", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats, 1)

	ourChat := response.Chats[0]

	err = s.m.SaveChat(ourChat)
	s.NoError(err)

	members := []string{"0x" + hex.EncodeToString(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))}
	_, err = s.m.AddMembersToGroupChat(context.Background(), ourChat.ID, members)
	s.NoError(err)

	// Retrieve their messages so that the chat is created
	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err = theirMessenger.RetrieveAll()
		if err == nil && len(response.Chats) == 0 {
			err = errors.New("chat invitation not received")
		}
		return err
	})
	s.Require().NoError(err)

	_, err = theirMessenger.ConfirmJoiningGroup(context.Background(), ourChat.ID)
	s.NoError(err)

	inputMessage := buildTestMessage(*ourChat)

	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages, 1)

	sentMessage := sendResponse.Messages[0]

	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err = s.m.RetrieveAll()
		if err == nil && len(response.Messages) == 0 {
			err = errors.New("no messages")
		}
		return err
	})
	s.Require().NoError(err)

	s.Require().Len(response.Chats, 1)
	actualChat := response.Chats[0]
	// It updates the unviewed messages count
	s.Require().Equal(uint(1), actualChat.UnviewedMessagesCount)
	// It updates the last message clock value
	s.Require().Equal(sentMessage.Clock, actualChat.LastClockValue)
	// It sets the last message
	s.Require().NotNil(actualChat.LastMessage)
}

func (s *MessengerSuite) TestChatPersistencePublic() {
	chat := Chat{
		ID:                    "chat-name",
		Name:                  "chat-name",
		Color:                 "#fffff",
		Active:                true,
		ChatType:              ChatTypePublic,
		Timestamp:             10,
		LastClockValue:        20,
		DeletedAtClockValue:   30,
		UnviewedMessagesCount: 40,
		LastMessage:           []byte("test"),
	}

	s.Require().NoError(s.m.SaveChat(&chat))
	savedChats := s.m.Chats()
	s.Require().Equal(1, len(savedChats))

	actualChat := savedChats[0]
	expectedChat := &chat

	s.Require().Equal(actualChat, expectedChat)
}

func (s *MessengerSuite) TestDeleteChat() {
	chatID := "chatid"
	chat := Chat{
		ID:                    chatID,
		Name:                  "chat-name",
		Color:                 "#fffff",
		Active:                true,
		ChatType:              ChatTypePublic,
		Timestamp:             10,
		LastClockValue:        20,
		DeletedAtClockValue:   30,
		UnviewedMessagesCount: 40,
		LastMessage:           []byte("test"),
	}

	s.Require().NoError(s.m.SaveChat(&chat))
	savedChats := s.m.Chats()
	s.Require().Equal(1, len(savedChats))

	s.Require().NoError(s.m.DeleteChat(chatID))
	savedChats = s.m.Chats()
	s.Require().Equal(0, len(savedChats))
}

func (s *MessengerSuite) TestChatPersistenceUpdate() {
	chat := Chat{
		ID:                    "chat-name",
		Name:                  "chat-name",
		Color:                 "#fffff",
		Active:                true,
		ChatType:              ChatTypePublic,
		Timestamp:             10,
		LastClockValue:        20,
		DeletedAtClockValue:   30,
		UnviewedMessagesCount: 40,
		LastMessage:           []byte("test"),
	}

	s.Require().NoError(s.m.SaveChat(&chat))
	savedChats := s.m.Chats()
	s.Require().Equal(1, len(savedChats))

	actualChat := savedChats[0]
	expectedChat := &chat

	s.Require().Equal(expectedChat, actualChat)

	chat.Name = "updated-name"
	s.Require().NoError(s.m.SaveChat(&chat))
	updatedChats := s.m.Chats()
	s.Require().Equal(1, len(updatedChats))

	actualUpdatedChat := updatedChats[0]
	expectedUpdatedChat := &chat

	s.Require().Equal(expectedUpdatedChat, actualUpdatedChat)
}

func (s *MessengerSuite) TestChatPersistenceOneToOne() {
	pkStr := "0x0424a68f89ba5fcd5e0640c1e1f591d561fa4125ca4e2a43592bc4123eca10ce064e522c254bb83079ba404327f6eafc01ec90a1444331fe769d3f3a7f90b0dde1"
	chat := Chat{
		ID:                    pkStr,
		Name:                  pkStr,
		Color:                 "#fffff",
		Active:                true,
		ChatType:              ChatTypeOneToOne,
		Timestamp:             10,
		LastClockValue:        20,
		DeletedAtClockValue:   30,
		UnviewedMessagesCount: 40,
		LastMessage:           []byte("test"),
	}
	publicKeyBytes, err := hex.DecodeString(pkStr[2:])
	s.Require().NoError(err)

	pk, err := crypto.UnmarshalPubkey(publicKeyBytes)
	s.Require().NoError(err)

	s.Require().NoError(s.m.SaveChat(&chat))
	savedChats := s.m.Chats()
	s.Require().Equal(1, len(savedChats))

	actualChat := savedChats[0]
	expectedChat := &chat

	actualPk, err := actualChat.PublicKey()
	s.Require().NoError(err)

	s.Require().Equal(pk, actualPk)

	s.Require().Equal(expectedChat, actualChat)
}

func (s *MessengerSuite) TestChatPersistencePrivateGroupChat() {
	chat := Chat{
		ID:        "chat-id",
		Name:      "chat-id",
		Color:     "#fffff",
		Active:    true,
		ChatType:  ChatTypePrivateGroupChat,
		Timestamp: 10,
		Members: []ChatMember{
			ChatMember{
				ID:     "1",
				Admin:  false,
				Joined: true,
			},
			ChatMember{
				ID:     "2",
				Admin:  true,
				Joined: false,
			},
			ChatMember{
				ID:     "3",
				Admin:  true,
				Joined: true,
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
		LastMessage:           []byte("test"),
	}
	s.Require().NoError(s.m.SaveChat(&chat))
	savedChats := s.m.Chats()
	s.Require().Equal(1, len(savedChats))

	actualChat := savedChats[0]
	expectedChat := &chat

	s.Require().Equal(expectedChat, actualChat)
}

func (s *MessengerSuite) TestBlockContact() {
	pk := "0x0424a68f89ba5fcd5e0640c1e1f591d561fa4125ca4e2a43592bc4123eca10ce064e522c254bb83079ba404327f6eafc01ec90a1444331fe769d3f3a7f90b0dde1"

	contact := Contact{
		ID:          pk,
		Address:     "contact-address",
		Name:        "contact-name",
		Photo:       "contact-photo",
		LastUpdated: 20,
		SystemTags:  []string{"1", "2"},
		DeviceInfo: []ContactDeviceInfo{
			ContactDeviceInfo{
				InstallationID: "1",
				Timestamp:      2,
				FCMToken:       "token",
			},
			ContactDeviceInfo{
				InstallationID: "2",
				Timestamp:      3,
				FCMToken:       "token-2",
			},
		},
		TributeToTalk: "talk",
	}

	chat1 := Chat{
		ID:                    contact.ID,
		Name:                  "chat-name",
		Color:                 "#fffff",
		Active:                true,
		ChatType:              ChatTypeOneToOne,
		Timestamp:             1,
		LastClockValue:        20,
		DeletedAtClockValue:   30,
		UnviewedMessagesCount: 40,
	}

	chat2 := Chat{
		ID:                    "chat-2",
		Name:                  "chat-name",
		Color:                 "#fffff",
		Active:                true,
		ChatType:              ChatTypePublic,
		Timestamp:             2,
		LastClockValue:        20,
		DeletedAtClockValue:   30,
		UnviewedMessagesCount: 40,
	}

	chat3 := Chat{
		ID:                    "chat-3",
		Name:                  "chat-name",
		Color:                 "#fffff",
		Active:                true,
		ChatType:              ChatTypePublic,
		Timestamp:             3,
		LastClockValue:        20,
		DeletedAtClockValue:   30,
		UnviewedMessagesCount: 40,
	}

	s.Require().NoError(s.m.SaveChat(&chat1))
	s.Require().NoError(s.m.SaveChat(&chat2))
	s.Require().NoError(s.m.SaveChat(&chat3))

	s.Require().NoError(s.m.SaveContact(&contact))

	contact.Name = "blocked"

	messages := []*Message{
		&Message{
			ID:          "test-1",
			LocalChatID: chat2.ID,
			ChatMessage: protobuf.ChatMessage{
				ContentType: 1,
				Text:        "test-1",
				Clock:       1,
			},
			From: contact.ID,
		},
		&Message{
			ID:          "test-2",
			LocalChatID: chat2.ID,
			ChatMessage: protobuf.ChatMessage{
				ContentType: 2,
				Text:        "test-2",
				Clock:       2,
			},
			From: contact.ID,
		},
		&Message{
			ID:          "test-3",
			LocalChatID: chat2.ID,
			ChatMessage: protobuf.ChatMessage{
				ContentType: 3,
				Text:        "test-3",
				Clock:       3,
			},
			Seen: false,
			From: "test",
		},
		&Message{
			ID:          "test-4",
			LocalChatID: chat2.ID,
			ChatMessage: protobuf.ChatMessage{
				ContentType: 4,
				Text:        "test-4",
				Clock:       4,
			},
			Seen: false,
			From: "test",
		},
		&Message{
			ID:          "test-5",
			LocalChatID: chat2.ID,
			ChatMessage: protobuf.ChatMessage{
				ContentType: 5,
				Text:        "test-5",
				Clock:       5,
			},
			Seen: true,
			From: "test",
		},
		&Message{
			ID:          "test-6",
			LocalChatID: chat3.ID,
			ChatMessage: protobuf.ChatMessage{
				ContentType: 6,
				Text:        "test-6",
				Clock:       6,
			},
			Seen: false,
			From: contact.ID,
		},
		&Message{
			ID:          "test-7",
			LocalChatID: chat3.ID,
			ChatMessage: protobuf.ChatMessage{
				ContentType: 7,
				Text:        "test-7",
				Clock:       7,
			},
			Seen: false,
			From: "test",
		},
	}

	err := s.m.SaveMessages(messages)
	s.Require().NoError(err)

	response, err := s.m.BlockContact(&contact)
	s.Require().NoError(err)

	// The new unviewed count is updated
	s.Require().Equal(uint(1), response[0].UnviewedMessagesCount)
	s.Require().Equal(uint(2), response[1].UnviewedMessagesCount)

	// The new message content is updated
	decodedMessage := &Message{}
	s.Require().NotNil(response[0].LastMessage)

	s.Require().NoError(json.Unmarshal(response[0].LastMessage, decodedMessage))
	s.Require().Equal("test-7", decodedMessage.ID)

	decodedMessage = &Message{}
	s.Require().NotNil(response[1].LastMessage)

	s.Require().NoError(json.Unmarshal(response[1].LastMessage, decodedMessage))
	s.Require().Equal("test-5", decodedMessage.ID)

	// The contact is updated
	savedContacts := s.m.Contacts()
	s.Require().Equal(1, len(savedContacts))
	s.Require().Equal("blocked", savedContacts[0].Name)

	// The chat is deleted
	actualChats := s.m.Chats()
	s.Require().Equal(2, len(actualChats))

	// The messages have been deleted
	chat2Messages, _, err := s.m.MessageByChatID(chat2.ID, "", 20)
	s.Require().NoError(err)
	s.Require().Equal(3, len(chat2Messages))

	chat3Messages, _, err := s.m.MessageByChatID(chat3.ID, "", 20)
	s.Require().NoError(err)
	s.Require().Equal(1, len(chat3Messages))

}

func (s *MessengerSuite) TestContactPersistence() {
	contact := Contact{
		ID: "0x0424a68f89ba5fcd5e0640c1e1f591d561fa4125ca4e2a43592bc4123eca10ce064e522c254bb83079ba404327f6eafc01ec90a1444331fe769d3f3a7f90b0dde1",

		Address:     "contact-address",
		Name:        "contact-name",
		Photo:       "contact-photo",
		LastUpdated: 20,
		SystemTags:  []string{"1", "2"},
		DeviceInfo: []ContactDeviceInfo{
			ContactDeviceInfo{
				InstallationID: "1",
				Timestamp:      2,
				FCMToken:       "token",
			},
			ContactDeviceInfo{
				InstallationID: "2",
				Timestamp:      3,
				FCMToken:       "token-2",
			},
		},
		TributeToTalk: "talk",
	}

	s.Require().NoError(s.m.SaveContact(&contact))
	savedContacts := s.m.Contacts()
	s.Require().Equal(1, len(savedContacts))

	actualContact := savedContacts[0]
	expectedContact := &contact
	expectedContact.Alias = "Concrete Lavender Xiphias"
	expectedContact.Identicon = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAnElEQVR4nOzXQaqDMBRG4bZkLR10e12H23PgZuJUjJAcE8kdnG/44IXDhZ9iyjm/4vnMDrhmFmEWYRZhFpH6n1jW7fSX/+/b+WbQa5lFmEVUljhqZfSdoNcyizCLeNMvn3JTLeh+g17LLMIsorLElt2VK7v3X0dBr2UWYRaBfxNLfifOZhYRNGvAEp8Q9FpmEWYRZhFmEXsAAAD//5K5JFhu0M0nAAAAAElFTkSuQmCC"
	s.Require().Equal(expectedContact, actualContact)
}

func (s *MessengerSuite) TestVerifyENSNames() {
	rpcEndpoint := os.Getenv("RPC_ENDPOINT")
	if rpcEndpoint == "" {
		s.T().Skip()
	}
	contractAddress := "0x314159265dd8dbb310642f98f50c066173c1259b"
	pk1 := "04325367620ae20dd878dbb39f69f02c567d789dd21af8a88623dc5b529827c2812571c380a2cd8236a2851b8843d6486481166c39debf60a5d30b9099c66213e4"
	pk2 := "044580b6aef9ddebd88c373b43c91237dcc95a8307bc5837d11d3ad2fa5d1dc696b598e7ccc498b414ba80b86b129c48e1eb9464cc9ea26224321539f2f54024cc"
	pk3 := "044fee950d9748606da2f77d3c51bf16134a59bde4903aa68076a45d9eefbb54182a24f0c74b381bad0525a90e78770d11559aa02f77343d172f386e3b521c277a"
	pk4 := "not a valid pk"

	ensDetails := []enstypes.ENSDetails{
		{
			Name:            "pedro.stateofus.eth",
			PublicKeyString: pk1,
		},
		// Not matching pk -> name
		enstypes.ENSDetails{
			Name:            "pedro.stateofus.eth",
			PublicKeyString: pk2,
		},
		// Not existing name
		enstypes.ENSDetails{
			Name:            "definitelynotpedro.stateofus.eth",
			PublicKeyString: pk3,
		},
		// Malformed pk
		enstypes.ENSDetails{
			Name:            "pedro.stateofus.eth",
			PublicKeyString: pk4,
		},
	}

	response, err := s.m.VerifyENSNames(rpcEndpoint, contractAddress, ensDetails)
	s.Require().NoError(err)
	s.Require().Equal(4, len(response))

	s.Require().Nil(response[pk1].Error)
	s.Require().Nil(response[pk2].Error)
	s.Require().NotNil(response[pk3].Error)
	s.Require().NotNil(response[pk4].Error)

	s.Require().True(response[pk1].Verified)
	s.Require().False(response[pk2].Verified)
	s.Require().False(response[pk3].Verified)
	s.Require().False(response[pk4].Verified)

	// The contacts are updated
	savedContacts := s.m.Contacts()

	s.Require().Equal(2, len(savedContacts))

	var verifiedContact *Contact
	var notVerifiedContact *Contact

	if savedContacts[0].ID == pk1 {
		verifiedContact = savedContacts[0]
		notVerifiedContact = savedContacts[1]
	} else {
		notVerifiedContact = savedContacts[0]
		verifiedContact = savedContacts[1]
	}

	s.Require().Equal("pedro.stateofus.eth", verifiedContact.Name)
	s.Require().NotEqual(0, verifiedContact.ENSVerifiedAt)
	s.Require().True(verifiedContact.ENSVerified)

	s.Require().Equal("pedro.stateofus.eth", notVerifiedContact.Name)
	s.Require().NotEqual(0, notVerifiedContact.ENSVerifiedAt)
	s.Require().True(notVerifiedContact.ENSVerified)
}

func (s *MessengerSuite) TestContactPersistenceUpdate() {
	contactID := "0x0424a68f89ba5fcd5e0640c1e1f591d561fa4125ca4e2a43592bc4123eca10ce064e522c254bb83079ba404327f6eafc01ec90a1444331fe769d3f3a7f90b0dde1"

	contact := Contact{
		ID:          contactID,
		Address:     "contact-address",
		Name:        "contact-name",
		Photo:       "contact-photo",
		LastUpdated: 20,
		SystemTags:  []string{"1", "2"},
		DeviceInfo: []ContactDeviceInfo{
			ContactDeviceInfo{
				InstallationID: "1",
				Timestamp:      2,
				FCMToken:       "token",
			},
			ContactDeviceInfo{
				InstallationID: "2",
				Timestamp:      3,
				FCMToken:       "token-2",
			},
		},
		TributeToTalk: "talk",
	}

	s.Require().NoError(s.m.SaveContact(&contact))
	savedContacts := s.m.Contacts()
	s.Require().Equal(1, len(savedContacts))

	actualContact := savedContacts[0]
	expectedContact := &contact

	expectedContact.Alias = "Concrete Lavender Xiphias"
	expectedContact.Identicon = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAnElEQVR4nOzXQaqDMBRG4bZkLR10e12H23PgZuJUjJAcE8kdnG/44IXDhZ9iyjm/4vnMDrhmFmEWYRZhFpH6n1jW7fSX/+/b+WbQa5lFmEVUljhqZfSdoNcyizCLeNMvn3JTLeh+g17LLMIsorLElt2VK7v3X0dBr2UWYRaBfxNLfifOZhYRNGvAEp8Q9FpmEWYRZhFmEXsAAAD//5K5JFhu0M0nAAAAAElFTkSuQmCC"

	s.Require().Equal(expectedContact, actualContact)

	contact.Name = "updated-name"
	s.Require().NoError(s.m.SaveContact(&contact))
	updatedContact := s.m.Contacts()
	s.Require().Equal(1, len(updatedContact))

	actualUpdatedContact := updatedContact[0]
	expectedUpdatedContact := &contact

	s.Require().Equal(expectedUpdatedContact, actualUpdatedContact)
}

func (s *MessengerSuite) TestSharedSecretHandler() {
	_, err := s.m.handleSharedSecrets(nil)
	s.NoError(err)
}

func (s *MessengerSuite) TestCreateGroupChatWithMembers() {
	members := []string{"0x0424a68f89ba5fcd5e0640c1e1f591d561fa4125ca4e2a43592bc4123eca10ce064e522c254bb83079ba404327f6eafc01ec90a1444331fe769d3f3a7f90b0dde1"}
	response, err := s.m.CreateGroupChatWithMembers(context.Background(), "test", members)
	s.NoError(err)
	s.Require().Len(response.Chats, 1)

	chat := response.Chats[0]

	s.Require().Equal("test", chat.Name)
	publicKeyHex := "0x" + hex.EncodeToString(crypto.FromECDSAPub(&s.m.identity.PublicKey))
	s.Require().Contains(chat.ID, publicKeyHex)
	s.EqualValues([]string{publicKeyHex}, []string{chat.Members[0].ID})
	s.Equal(members[0], chat.Members[1].ID)
}

func (s *MessengerSuite) TestAddMembersToChat() {
	response, err := s.m.CreateGroupChatWithMembers(context.Background(), "test", []string{})
	s.Require().NoError(err)
	s.Require().Len(response.Chats, 1)

	chat := response.Chats[0]

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	members := []string{"0x" + hex.EncodeToString(crypto.FromECDSAPub(&key.PublicKey))}

	response, err = s.m.AddMembersToGroupChat(context.Background(), chat.ID, members)
	s.Require().NoError(err)
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	chat = response.Chats[0]

	publicKeyHex := "0x" + hex.EncodeToString(crypto.FromECDSAPub(&s.m.identity.PublicKey))
	keyHex := "0x" + hex.EncodeToString(crypto.FromECDSAPub(&key.PublicKey))
	s.EqualValues([]string{publicKeyHex, keyHex}, []string{chat.Members[0].ID, chat.Members[1].ID})
}

type mockSendMessagesRequest struct {
	types.Whisper
	req types.MessagesRequest
}

func (m *mockSendMessagesRequest) SendMessagesRequest(peerID []byte, request types.MessagesRequest) error {
	m.req = request
	return nil
}

func (s *MessengerSuite) TestMessageJSON() {
	message := &Message{
		ID:          "test-1",
		LocalChatID: "local-chat-id",
		Alias:       "alias",
		ChatMessage: protobuf.ChatMessage{
			ChatId:      "remote-chat-id",
			ContentType: 0,
			Text:        "test-1",
			Clock:       1,
		},
		From: "from-field",
	}

	expectedJSON := `{"id":"test-1","whisperTimestamp":0,"from":"from-field","alias":"alias","identicon":"","retryCount":0,"seen":false,"quotedMessage":null,"rtl":false,"parsedText":null,"lineCount":0,"text":"test-1","chatId":"remote-chat-id","localChatId":"local-chat-id","clock":1,"responseTo":"","ensName":"","sticker":null,"timestamp":0,"contentType":0,"messageType":0}`

	messageJSON, err := json.Marshal(message)
	s.Require().NoError(err)
	s.Require().Equal(expectedJSON, string(messageJSON))

	decodedMessage := &Message{}
	err = json.Unmarshal([]byte(expectedJSON), decodedMessage)
	s.Require().NoError(err)
	s.Require().Equal(message, decodedMessage)
}

// TODO: For some reason is not mocking the method anymore, help?
func (s *MessengerSuite) testRequestHistoricMessagesRequest() {
	shh := &mockSendMessagesRequest{
		Whisper: s.shh,
	}
	m := s.newMessenger(shh)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	cursor, err := m.RequestHistoricMessages(ctx, nil, 10, 20, []byte{0x01})
	s.EqualError(err, ctx.Err().Error())
	s.Empty(cursor)
	// verify request is correct
	s.NotEmpty(shh.req.ID)
	s.EqualValues(10, shh.req.From)
	s.EqualValues(20, shh.req.To)
	s.EqualValues(100, shh.req.Limit)
	s.Equal([]byte{0x01}, shh.req.Cursor)
	s.NotEmpty(shh.req.Bloom)
}

type PostProcessorSuite struct {
	suite.Suite

	postProcessor *postProcessor
	logger        *zap.Logger
}

func (s *PostProcessorSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	db, err := sqlite.OpenInMemory()
	s.Require().NoError(err)

	s.postProcessor = &postProcessor{
		myPublicKey: &privateKey.PublicKey,
		persistence: &sqlitePersistence{db: db},
		logger:      s.logger,
		config: postProcessorConfig{
			MatchChat: true,
			Persist:   true,
			Parse:     true,
		},
	}
}

func (s *PostProcessorSuite) TearDownTest() {
	_ = s.logger.Sync()
}

func (s *PostProcessorSuite) TestRun() {
	key1, err := crypto.GenerateKey()
	s.Require().NoError(err)
	key2, err := crypto.GenerateKey()
	s.Require().NoError(err)

	testCases := []struct {
		Name           string
		Error          bool
		Chat           Chat // Chat to create
		Message        Message
		SigPubKey      *ecdsa.PublicKey
		ExpectedChatID string
	}{
		{
			Name: "Public chat",
			Chat: CreatePublicChat("test-chat"),
			Message: Message{
				ChatMessage: protobuf.ChatMessage{
					ChatId:      "test-chat",
					MessageType: protobuf.ChatMessage_PUBLIC_GROUP,
					Text:        "test-text"},
			},
			SigPubKey:      &key1.PublicKey,
			ExpectedChatID: "test-chat",
		},
		{
			Name: "Private message from myself with existing chat",
			Chat: CreateOneToOneChat("test-private-chat", &key1.PublicKey),
			Message: Message{
				ChatMessage: protobuf.ChatMessage{
					ChatId:      "test-chat",
					MessageType: protobuf.ChatMessage_ONE_TO_ONE,
					Text:        "test-text"},
			},
			SigPubKey:      &key1.PublicKey,
			ExpectedChatID: oneToOneChatID(&key1.PublicKey),
		},
		{
			Name: "Private message from other with existing chat",
			Chat: CreateOneToOneChat("test-private-chat", &key2.PublicKey),
			Message: Message{
				ChatMessage: protobuf.ChatMessage{
					ChatId:      "test-chat",
					MessageType: protobuf.ChatMessage_ONE_TO_ONE,
					Text:        "test-text"},
			},

			SigPubKey:      &key2.PublicKey,
			ExpectedChatID: oneToOneChatID(&key2.PublicKey),
		},
		{
			Name: "Private message from myself without chat",
			Message: Message{
				ChatMessage: protobuf.ChatMessage{
					ChatId:      "test-chat",
					MessageType: protobuf.ChatMessage_ONE_TO_ONE,
					Text:        "test-text"},
			},

			SigPubKey:      &key1.PublicKey,
			ExpectedChatID: oneToOneChatID(&key1.PublicKey),
		},
		{
			Name: "Private message from other without chat",
			Message: Message{
				ChatMessage: protobuf.ChatMessage{
					ChatId:      "test-chat",
					MessageType: protobuf.ChatMessage_ONE_TO_ONE,
					Text:        "test-text"},
			},

			SigPubKey:      &key2.PublicKey,
			ExpectedChatID: oneToOneChatID(&key2.PublicKey),
		},
		{
			Name:      "Private message without public key",
			SigPubKey: nil,
			Error:     true,
		},
		{
			Name: "Private group message",
			Message: Message{
				ChatMessage: protobuf.ChatMessage{
					ChatId:      "non-existing-chat",
					MessageType: protobuf.ChatMessage_PRIVATE_GROUP,
					Text:        "test-text"},
			},
			Error:     true,
			SigPubKey: &key2.PublicKey,
		},
	}

	for idx, tc := range testCases {
		s.Run(tc.Name, func() {
			chatsMap := make(map[string]*Chat)
			if tc.Chat.ID != "" {
				chatsMap[tc.Chat.ID] = &tc.Chat
				err := s.postProcessor.persistence.SaveChat(tc.Chat)
				s.Require().NoError(err)
				defer func() {
					err := s.postProcessor.persistence.DeleteChat(tc.Chat.ID)
					s.Require().NoError(err)
				}()
			}

			message := tc.Message
			message.SigPubKey = tc.SigPubKey
			// ChatID is not set at the beginning.
			s.Empty(message.LocalChatID)

			message.ID = strconv.Itoa(idx) // manually set the ID because messages does not go through messageProcessor
			chat, err := s.postProcessor.matchMessage(&message, chatsMap)
			if tc.Error {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				if tc.ExpectedChatID != "" {

					s.Require().NotNil(chat)
					s.Require().Equal(tc.ExpectedChatID, chat.ID)
				}
			}
		})
	}
}
