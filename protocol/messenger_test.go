package protocol

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	_ "github.com/mutecomm/go-sqlcipher" // require go-sqlcipher that overrides default implementation
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/account/generator"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	coretypes "github.com/status-im/status-go/eth-node/core/types"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	enstypes "github.com/status-im/status-go/eth-node/types/ens"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/tt"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/waku"
)

const (
	testPK              = "0x0424a68f89ba5fcd5e0640c1e1f591d561fa4125ca4e2a43592bc4123eca10ce064e522c254bb83079ba404327f6eafc01ec90a1444331fe769d3f3a7f90b0dde1"
	testPublicChatID    = "super-chat"
	testContract        = "0x314159265dd8dbb310642f98f50c066173c1259b"
	testValue           = "2000"
	testTransactionHash = "0x412a851ac2ae51cad34a56c8a9cfee55d577ac5e1ac71cf488a2f2093a373799"
	testIdenticon       = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAnElEQVR4nOzXQaqDMBRG4bZkLR10e12H23PgZuJUjJAcE8kdnG/44IXDhZ9iyjm/4vnMDrhmFmEWYRZhFpH6n1jW7fSX/+/b+WbQa5lFmEVUljhqZfSdoNcyizCLeNMvn3JTLeh+g17LLMIsorLElt2VK7v3X0dBr2UWYRaBfxNLfifOZhYRNGvAEp8Q9FpmEWYRZhFmEXsAAAD//5K5JFhu0M0nAAAAAElFTkSuQmCC"
	testAlias           = "Concrete Lavender Xiphias"
	newName             = "new-name"
)

func TestMessengerSuite(t *testing.T) {
	suite.Run(t, new(MessengerSuite))
}

func TestMessengerWithDataSyncEnabledSuite(t *testing.T) {
	suite.Run(t, &MessengerSuite{enableDataSync: true})
}

func TestMessageHandlerSuite(t *testing.T) {
	suite.Run(t, new(MessageHandlerSuite))
}

type MessengerSuite struct {
	suite.Suite

	enableDataSync bool

	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single Whisper service should be shared.
	shh    types.Waku
	logger *zap.Logger
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

func (n *testNode) GetWhisper(_ interface{}) (types.Whisper, error) {
	return nil, nil
}

func (n *testNode) PeersCount() int {
	return 1
}

func (s *MessengerSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start(nil))

	s.m = s.newMessenger(s.shh)
	s.privateKey = s.m.identity
	_, err := s.m.Start()
	s.Require().NoError(err)
}

func newMessengerWithKey(shh types.Waku, privateKey *ecdsa.PrivateKey, logger *zap.Logger, extraOptions []Option) (*Messenger, error) {
	tmpfile, err := ioutil.TempFile("", "accounts-tests-")
	if err != nil {
		return nil, err
	}
	madb, err := multiaccounts.InitializeDB(tmpfile.Name())
	if err != nil {
		return nil, err
	}

	acc := generator.NewAccount(privateKey, nil)
	iai := acc.ToIdentifiedAccountInfo("")

	options := []Option{
		WithCustomLogger(logger),
		WithDatabaseConfig(":memory:", "some-key"),
		WithMultiAccounts(madb),
		WithAccount(iai.ToMultiAccount()),
		WithDatasync(),
	}

	options = append(options, extraOptions...)

	m, err := NewMessenger(
		privateKey,
		&testNode{shh: shh},
		uuid.New().String(),
		options...,
	)
	if err != nil {
		return nil, err
	}

	err = m.Init()
	if err != nil {
		return nil, err
	}

	_, err = m.Start()
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (s *MessengerSuite) newMessenger(shh types.Waku) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	messenger, err := newMessengerWithKey(shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
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

func buildTestMessage(chat Chat) *common.Message {
	clock, timestamp := chat.NextClockAndTimestamp(&testTimeSource{})
	message := &common.Message{}
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

func (s *MessengerSuite) TestMarkMessagesSeen() {
	chat := CreatePublicChat("test-chat", s.m.transport)
	chat.UnviewedMessagesCount = 2
	err := s.m.SaveChat(&chat)
	s.Require().NoError(err)
	inputMessage1 := buildTestMessage(chat)
	inputMessage1.ID = "1"
	inputMessage1.Seen = false
	inputMessage2 := buildTestMessage(chat)
	inputMessage2.ID = "2"
	inputMessage2.Seen = false

	err = s.m.SaveMessages([]*common.Message{inputMessage1, inputMessage2})
	s.Require().NoError(err)

	count, err := s.m.MarkMessagesSeen(chat.ID, []string{inputMessage1.ID})
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), count)

	// Make sure that if it's not seen, it does not return a count of 1
	count, err = s.m.MarkMessagesSeen(chat.ID, []string{inputMessage1.ID})
	s.Require().NoError(err)
	s.Require().Equal(uint64(0), count)

	chats := s.m.Chats()
	s.Require().Len(chats, 1)
	s.Require().Equal(uint(1), chats[0].UnviewedMessagesCount)
}

func (s *MessengerSuite) TestMarkAllRead() {
	chat := CreatePublicChat("test-chat", s.m.transport)
	chat.UnviewedMessagesCount = 2
	err := s.m.SaveChat(&chat)
	s.Require().NoError(err)
	inputMessage1 := buildTestMessage(chat)
	inputMessage1.ID = "1"
	inputMessage1.Seen = false
	inputMessage2 := buildTestMessage(chat)
	inputMessage2.ID = "2"
	inputMessage2.Seen = false

	err = s.m.SaveMessages([]*common.Message{inputMessage1, inputMessage2})
	s.Require().NoError(err)

	err = s.m.MarkAllRead(chat.ID)
	s.Require().NoError(err)

	chats := s.m.Chats()
	s.Require().Len(chats, 1)
	s.Require().Equal(uint(0), chats[0].UnviewedMessagesCount)
}

func (s *MessengerSuite) TestSendPublic() {
	chat := CreatePublicChat("test-chat", s.m.transport)
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
	s.Require().Equal(outputMessage.OutgoingStatus, common.OutgoingStatusSending, "it marks the message as sending")
	s.Require().NotEmpty(outputMessage.ID, "it sets the ID field")
	s.Require().Equal(protobuf.MessageType_PUBLIC_GROUP, outputMessage.MessageType)

	savedMessages, _, err := s.m.MessageByChatID(chat.ID, "", 10)
	s.Require().NoError(err)
	s.Require().Equal(1, len(savedMessages), "it saves the message")
}

func (s *MessengerSuite) TestSendProfile() {
	chat := CreateProfileChat("test-chat-profile", "0x"+hex.EncodeToString(crypto.FromECDSAPub(&s.privateKey.PublicKey)), s.m.transport)
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

	inputMessage := &common.Message{}
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
	s.Require().Equal(outputMessage.OutgoingStatus, common.OutgoingStatusSending, "it marks the message as sending")
	s.Require().NotEmpty(outputMessage.ID, "it sets the ID field")
	s.Require().Equal(protobuf.MessageType_ONE_TO_ONE, outputMessage.MessageType)
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

	inputMessage := &common.Message{}
	inputMessage.ChatId = chat.ID
	chat.LastClockValue = uint64(100000000000000)
	err = s.m.SaveChat(chat)
	s.NoError(err)
	response, err = s.m.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)
	s.Require().Equal(1, len(response.Messages), "it returns the message")
	outputMessage := response.Messages[0]

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
	s.Require().Len(response.Chats, 1)

	chat := response.Chats[0]

	inputMessage := &common.Message{}
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
	s.Require().Equal(outputMessage.OutgoingStatus, common.OutgoingStatusSending, "it marks the message as sending")
	s.Require().NotEmpty(outputMessage.ID, "it sets the ID field")
	s.Require().Equal(protobuf.MessageType_PRIVATE_GROUP, outputMessage.MessageType)
}

// Make sure public messages sent by us are not
func (s *MessengerSuite) TestRetrieveOwnPublic() {
	chat := CreatePublicChat("status", s.m.transport)
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
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	theirChat := CreatePublicChat("status", s.m.transport)
	err = theirMessenger.SaveChat(&theirChat)
	s.Require().NoError(err)

	chat := CreatePublicChat("status", s.m.transport)
	err = s.m.SaveChat(&chat)
	s.Require().NoError(err)

	err = s.m.Join(chat)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(chat)

	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	sentMessage := sendResponse.Messages[0]

	// Wait for the message to reach its destination
	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages) > 0 },
		"no messages",
	)

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
	s.Require().NoError(theirMessenger.Shutdown())
}

func (s *MessengerSuite) TestDeletedAtClockValue() {
	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	theirChat := CreatePublicChat("status", s.m.transport)
	err = theirMessenger.SaveChat(&theirChat)
	s.Require().NoError(err)

	chat := CreatePublicChat("status", s.m.transport)
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
	s.Require().NoError(theirMessenger.Shutdown())
}

func (s *MessengerSuite) TestRetrieveBlockedContact() {
	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	theirChat := CreatePublicChat("status", s.m.transport)
	err = theirMessenger.SaveChat(&theirChat)
	s.Require().NoError(err)

	chat := CreatePublicChat("status", s.m.transport)
	err = s.m.SaveChat(&chat)
	s.Require().NoError(err)

	err = s.m.Join(chat)
	s.Require().NoError(err)

	publicKeyHex := "0x" + hex.EncodeToString(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	blockedContact := Contact{
		ID:            publicKeyHex,
		Name:          "contact-name",
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
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	theirChat := CreatePublicChat("status", s.m.transport)
	err = theirMessenger.SaveChat(&theirChat)
	s.Require().NoError(err)

	chat := CreatePublicChat("status", s.m.transport)
	err = s.m.SaveChat(&chat)
	s.Require().NoError(err)

	err = s.m.Join(chat)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(chat)

	sendResponse1, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	sentMessage := sendResponse1.Messages[0]

	err = theirMessenger.ReSendChatMessage(context.Background(), sentMessage.ID)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages) > 0 },
		"no messages",
	)
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
	err = theirMessenger.ReSendChatMessage(context.Background(), sentMessage.ID)
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
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	theirChat := CreateOneToOneChat("XXX", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(&theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("our-chat", &theirMessenger.identity.PublicKey, s.m.transport)
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

	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages) > 0 },
		"no messages",
	)
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
	s.Require().NoError(theirMessenger.Shutdown())
}

// Test receiving a message on an non-existing private chat
func (s *MessengerSuite) TestRetrieveTheirPrivateChatNonExisting() {
	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	chat := CreateOneToOneChat("XXX", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(&chat)
	s.NoError(err)

	inputMessage := buildTestMessage(chat)

	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages, 1)

	sentMessage := sendResponse.Messages[0]

	// Wait for the message to reach its destination
	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages) > 0 },
		"no messages",
	)

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

// Test receiving a message on an non-existing public chat
func (s *MessengerSuite) TestRetrieveTheirPublicChatNonExisting() {
	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	chat := CreatePublicChat("test-chat", s.m.transport)
	err = theirMessenger.SaveChat(&chat)
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
	s.Require().NoError(theirMessenger.Shutdown())
}

// Test receiving a message on an existing private group chat
func (s *MessengerSuite) TestRetrieveTheirPrivateGroupChat() {
	var response *MessengerResponse
	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	response, err = s.m.CreateGroupChatWithMembers(context.Background(), "id", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats, 1)

	ourChat := response.Chats[0]

	err = s.m.SaveChat(ourChat)
	s.NoError(err)

	members := []string{"0x" + hex.EncodeToString(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))}
	_, err = s.m.AddMembersToGroupChat(context.Background(), ourChat.ID, members)
	s.NoError(err)

	// Retrieve their messages so that the chat is created
	_, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.Chats) > 0 },
		"chat invitation not received",
	)
	s.Require().NoError(err)

	_, err = theirMessenger.ConfirmJoiningGroup(context.Background(), ourChat.ID)
	s.NoError(err)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Chats) > 0 },
		"no joining group event received",
	)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(*ourChat)

	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages, 1)

	sentMessage := sendResponse.Messages[0]

	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages) > 0 },
		"no messages",
	)
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

// Test receiving a message on an existing private group chat
func (s *MessengerSuite) TestChangeNameGroupChat() {
	var response *MessengerResponse
	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	response, err = s.m.CreateGroupChatWithMembers(context.Background(), "old-name", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats, 1)

	ourChat := response.Chats[0]

	err = s.m.SaveChat(ourChat)
	s.NoError(err)

	members := []string{"0x" + hex.EncodeToString(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))}
	_, err = s.m.AddMembersToGroupChat(context.Background(), ourChat.ID, members)
	s.NoError(err)

	// Retrieve their messages so that the chat is created
	_, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.Chats) > 0 },
		"chat invitation not received",
	)
	s.Require().NoError(err)

	_, err = theirMessenger.ConfirmJoiningGroup(context.Background(), ourChat.ID)
	s.NoError(err)

	// Wait for join group event
	_, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Chats) > 0 },
		"no joining group event received",
	)
	s.Require().NoError(err)

	_, err = s.m.ChangeGroupChatName(context.Background(), ourChat.ID, newName)
	s.NoError(err)

	// Retrieve their messages so that the chat is created
	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.Chats) > 0 },
		"chat invitation not received",
	)
	s.Require().NoError(err)

	s.Require().Len(response.Chats, 1)
	actualChat := response.Chats[0]
	s.Require().Equal(newName, actualChat.Name)
	s.Require().NoError(theirMessenger.Shutdown())
}

// Test being re-invited to a group chat
func (s *MessengerSuite) TestReInvitedToGroupChat() {
	var response *MessengerResponse
	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	response, err = s.m.CreateGroupChatWithMembers(context.Background(), "old-name", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats, 1)

	ourChat := response.Chats[0]

	err = s.m.SaveChat(ourChat)
	s.NoError(err)

	members := []string{"0x" + hex.EncodeToString(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))}
	_, err = s.m.AddMembersToGroupChat(context.Background(), ourChat.ID, members)
	s.NoError(err)

	// Retrieve their messages so that the chat is created
	_, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.Chats) > 0 },
		"chat invitation not received",
	)
	s.Require().NoError(err)

	_, err = theirMessenger.ConfirmJoiningGroup(context.Background(), ourChat.ID)
	s.NoError(err)

	// Wait for join group event
	_, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Chats) > 0 },
		"no joining group event received",
	)
	s.Require().NoError(err)

	response, err = theirMessenger.LeaveGroupChat(context.Background(), ourChat.ID, true)
	s.NoError(err)

	s.Require().Len(response.Chats, 1)
	s.Require().False(response.Chats[0].Active)

	// Retrieve messages so user is removed
	_, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Chats) > 0 && len(r.Chats[0].Members) == 1 },
		"leave group chat not received",
	)

	s.Require().NoError(err)

	// And we get re-invited
	_, err = s.m.AddMembersToGroupChat(context.Background(), ourChat.ID, members)
	s.NoError(err)

	// Retrieve their messages so that the chat is created
	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.Chats) > 0 },
		"chat invitation not received",
	)

	s.Require().NoError(err)

	s.Require().Len(response.Chats, 1)
	s.Require().True(response.Chats[0].Active)
	s.Require().NoError(theirMessenger.Shutdown())
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
		LastMessage:           &common.Message{},
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
		LastMessage:           &common.Message{},
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
		LastMessage:           &common.Message{},
	}

	s.Require().NoError(s.m.SaveChat(&chat))
	savedChats := s.m.Chats()
	s.Require().Equal(1, len(savedChats))

	actualChat := savedChats[0]
	expectedChat := &chat

	s.Require().Equal(expectedChat, actualChat)

	chat.Name = "updated-name-1"
	s.Require().NoError(s.m.SaveChat(&chat))
	updatedChats := s.m.Chats()
	s.Require().Equal(1, len(updatedChats))

	actualUpdatedChat := updatedChats[0]
	expectedUpdatedChat := &chat

	s.Require().Equal(expectedUpdatedChat, actualUpdatedChat)
}

func (s *MessengerSuite) TestChatPersistenceOneToOne() {
	chat := Chat{
		ID:                    testPK,
		Name:                  testPK,
		Color:                 "#fffff",
		Active:                true,
		ChatType:              ChatTypeOneToOne,
		Timestamp:             10,
		LastClockValue:        20,
		DeletedAtClockValue:   30,
		UnviewedMessagesCount: 40,
		LastMessage:           &common.Message{},
	}
	contact := Contact{
		ID: testPK,
	}

	publicKeyBytes, err := hex.DecodeString(testPK[2:])
	s.Require().NoError(err)

	pk, err := crypto.UnmarshalPubkey(publicKeyBytes)
	s.Require().NoError(err)

	s.Require().NoError(s.m.SaveChat(&chat))
	s.Require().NoError(s.m.SaveContact(&contact))
	savedChats := s.m.Chats()
	s.Require().Equal(1, len(savedChats))

	actualChat := savedChats[0]
	expectedChat := &chat

	actualPk, err := actualChat.PublicKey()
	s.Require().NoError(err)

	s.Require().Equal(pk, actualPk)

	s.Require().Equal(expectedChat, actualChat)
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

	chat := Chat{
		ID:        "chat-id",
		Name:      "chat-id",
		Color:     "#fffff",
		Active:    true,
		ChatType:  ChatTypePrivateGroupChat,
		Timestamp: 10,
		Members: []ChatMember{
			{
				ID:     member1ID,
				Admin:  false,
				Joined: true,
			},
			{
				ID:     member2ID,
				Admin:  true,
				Joined: false,
			},
			{
				ID:     member3ID,
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
		LastMessage:           &common.Message{},
	}
	s.Require().NoError(s.m.SaveChat(&chat))
	savedChats := s.m.Chats()
	s.Require().Equal(1, len(savedChats))

	actualChat := savedChats[0]
	expectedChat := &chat

	s.Require().Equal(expectedChat, actualChat)
}

func (s *MessengerSuite) TestBlockContact() {
	contact := Contact{
		ID:          testPK,
		Name:        "contact-name",
		LastUpdated: 20,
		SystemTags:  []string{contactAdded, contactRequestReceived},
		DeviceInfo: []ContactDeviceInfo{
			{
				InstallationID: "1",
				Timestamp:      2,
				FCMToken:       "token",
			},
			{
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

	messages := []*common.Message{
		{
			ID:          "test-1",
			LocalChatID: chat2.ID,
			ChatMessage: protobuf.ChatMessage{
				ContentType: 1,
				Text:        "test-1",
				Clock:       1,
			},
			From: contact.ID,
		},
		{
			ID:          "test-2",
			LocalChatID: chat2.ID,
			ChatMessage: protobuf.ChatMessage{
				ContentType: 2,
				Text:        "test-2",
				Clock:       2,
			},
			From: contact.ID,
		},
		{
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
		{
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
		{
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
		{
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
		{
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
	s.Require().NotNil(response[0].LastMessage)

	s.Require().Equal("test-7", response[0].LastMessage.ID)

	s.Require().NotNil(response[1].LastMessage)
	s.Require().Equal("test-5", response[1].LastMessage.ID)

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
		ID: testPK,

		Name:        "contact-name",
		LastUpdated: 20,
		SystemTags:  []string{contactAdded, contactRequestReceived},
		DeviceInfo: []ContactDeviceInfo{
			{
				InstallationID: "1",
				Timestamp:      2,
				FCMToken:       "token",
			},
			{
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
	expectedContact.Alias = testAlias
	expectedContact.Identicon = testIdenticon
	s.Require().Equal(expectedContact, actualContact)
}

func (s *MessengerSuite) TestContactPersistenceUpdate() {
	contactID := testPK

	contact := Contact{
		ID:          contactID,
		Name:        "contact-name",
		LastUpdated: 20,
		SystemTags:  []string{contactAdded, contactRequestReceived},
		DeviceInfo: []ContactDeviceInfo{
			{
				InstallationID: "1",
				Timestamp:      2,
				FCMToken:       "token",
			},
			{
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

	expectedContact.Alias = testAlias
	expectedContact.Identicon = testIdenticon

	s.Require().Equal(expectedContact, actualContact)

	contact.Name = "updated-name-2"
	s.Require().NoError(s.m.SaveContact(&contact))
	updatedContact := s.m.Contacts()
	s.Require().Equal(1, len(updatedContact))

	actualUpdatedContact := updatedContact[0]
	expectedUpdatedContact := &contact

	s.Require().Equal(expectedUpdatedContact, actualUpdatedContact)
}

func (s *MessengerSuite) TestSharedSecretHandler() {
	err := s.m.handleSharedSecrets(nil)
	s.NoError(err)
}

func (s *MessengerSuite) TestCreateGroupChatWithMembers() {
	members := []string{testPK}
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

func (s *MessengerSuite) TestDeclineRequestAddressForTransaction() {
	value := testValue
	contract := testContract
	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	theirPkString := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))

	chat := CreateOneToOneChat(theirPkString, &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(&chat)
	s.Require().NoError(err)

	myAddress := crypto.PubkeyToAddress(s.m.identity.PublicKey)

	response, err := s.m.RequestAddressForTransaction(context.Background(), theirPkString, myAddress.Hex(), value, contract)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	senderMessage := response.Messages[0]
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
		func(r *MessengerResponse) bool { return len(r.Messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().NotNil(response)
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	receiverMessage := response.Messages[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, receiverMessage.ContentType)
	s.Require().Equal("Request address for transaction", receiverMessage.Text)
	s.Require().NotNil(receiverMessage.CommandParameters)
	s.Require().Equal(value, receiverMessage.CommandParameters.Value)
	s.Require().Equal(contract, receiverMessage.CommandParameters.Contract)
	s.Require().Equal(initialCommandID, receiverMessage.CommandParameters.ID)
	s.Require().Equal(common.CommandStateRequestAddressForTransaction, receiverMessage.CommandParameters.CommandState)

	// We decline the request
	response, err = theirMessenger.DeclineRequestAddressForTransaction(context.Background(), receiverMessage.ID)
	s.Require().NoError(err)
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	senderMessage = response.Messages[0]
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
		func(r *MessengerResponse) bool { return len(r.Messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	receiverMessage = response.Messages[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, receiverMessage.ContentType)
	s.Require().Equal("Request address for transaction declined", receiverMessage.Text)
	s.Require().NotNil(receiverMessage.CommandParameters)
	s.Require().Equal(value, receiverMessage.CommandParameters.Value)
	s.Require().Equal(contract, receiverMessage.CommandParameters.Contract)
	s.Require().Equal(common.CommandStateRequestAddressForTransactionDeclined, receiverMessage.CommandParameters.CommandState)
	s.Require().Equal(initialCommandID, receiverMessage.CommandParameters.ID)
	s.Require().Equal(initialCommandID, receiverMessage.Replace)
	s.Require().NoError(theirMessenger.Shutdown())
}

func (s *MessengerSuite) TestSendEthTransaction() {
	value := testValue
	contract := testContract

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	theirPkString := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))

	receiverAddress := crypto.PubkeyToAddress(theirMessenger.identity.PublicKey)
	receiverAddressString := strings.ToLower(receiverAddress.Hex())

	chat := CreateOneToOneChat(theirPkString, &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(&chat)
	s.Require().NoError(err)

	transactionHash := testTransactionHash
	signature, err := buildSignature(s.m.identity, &s.m.identity.PublicKey, transactionHash)
	s.Require().NoError(err)

	response, err := s.m.SendTransaction(context.Background(), theirPkString, value, contract, transactionHash, signature)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	senderMessage := response.Messages[0]
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
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	receiverMessage := response.Messages[0]
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
	s.Require().NoError(theirMessenger.Shutdown())
}

func (s *MessengerSuite) TestSendTokenTransaction() {
	value := testValue
	contract := testContract

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	theirPkString := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))

	receiverAddress := crypto.PubkeyToAddress(theirMessenger.identity.PublicKey)
	receiverAddressString := strings.ToLower(receiverAddress.Hex())

	chat := CreateOneToOneChat(theirPkString, &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(&chat)
	s.Require().NoError(err)

	transactionHash := testTransactionHash
	signature, err := buildSignature(s.m.identity, &s.m.identity.PublicKey, transactionHash)
	s.Require().NoError(err)

	response, err := s.m.SendTransaction(context.Background(), theirPkString, value, contract, transactionHash, signature)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	senderMessage := response.Messages[0]
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
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	receiverMessage := response.Messages[0]
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
	s.Require().NoError(theirMessenger.Shutdown())
}

func (s *MessengerSuite) TestAcceptRequestAddressForTransaction() {
	value := testValue
	contract := testContract
	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	theirPkString := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))

	myAddress := crypto.PubkeyToAddress(s.m.identity.PublicKey)

	chat := CreateOneToOneChat(theirPkString, &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(&chat)
	s.Require().NoError(err)

	response, err := s.m.RequestAddressForTransaction(context.Background(), theirPkString, myAddress.Hex(), value, contract)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	senderMessage := response.Messages[0]
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
		func(r *MessengerResponse) bool { return len(r.Messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().NotNil(response)
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	receiverMessage := response.Messages[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, receiverMessage.ContentType)
	s.Require().Equal("Request address for transaction", receiverMessage.Text)
	s.Require().NotNil(receiverMessage.CommandParameters)
	s.Require().Equal(value, receiverMessage.CommandParameters.Value)
	s.Require().Equal(contract, receiverMessage.CommandParameters.Contract)
	s.Require().Equal(initialCommandID, receiverMessage.CommandParameters.ID)
	s.Require().Equal(common.CommandStateRequestAddressForTransaction, receiverMessage.CommandParameters.CommandState)

	// We accept the request
	response, err = theirMessenger.AcceptRequestAddressForTransaction(context.Background(), receiverMessage.ID, "some-address")
	s.Require().NoError(err)
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	senderMessage = response.Messages[0]
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
		func(r *MessengerResponse) bool { return len(r.Messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	receiverMessage = response.Messages[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, receiverMessage.ContentType)
	s.Require().Equal("Request address for transaction accepted", receiverMessage.Text)
	s.Require().NotNil(receiverMessage.CommandParameters)
	s.Require().Equal(value, receiverMessage.CommandParameters.Value)
	s.Require().Equal(contract, receiverMessage.CommandParameters.Contract)
	s.Require().Equal(common.CommandStateRequestAddressForTransactionAccepted, receiverMessage.CommandParameters.CommandState)
	s.Require().Equal(initialCommandID, receiverMessage.CommandParameters.ID)
	s.Require().Equal("some-address", receiverMessage.CommandParameters.Address)
	s.Require().Equal(initialCommandID, receiverMessage.Replace)
	s.Require().NoError(theirMessenger.Shutdown())
}

func (s *MessengerSuite) TestDeclineRequestTransaction() {
	value := testValue
	contract := testContract
	receiverAddress := crypto.PubkeyToAddress(s.m.identity.PublicKey)
	receiverAddressString := strings.ToLower(receiverAddress.Hex())
	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	theirPkString := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))

	chat := CreateOneToOneChat(theirPkString, &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(&chat)
	s.Require().NoError(err)

	response, err := s.m.RequestTransaction(context.Background(), theirPkString, value, contract, receiverAddressString)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	senderMessage := response.Messages[0]
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
		func(r *MessengerResponse) bool { return len(r.Messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().NotNil(response)
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	receiverMessage := response.Messages[0]
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
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	senderMessage = response.Messages[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, senderMessage.ContentType)

	s.Require().Equal("Transaction request declined", senderMessage.Text)
	s.Require().Equal(initialCommandID, senderMessage.CommandParameters.ID)
	s.Require().Equal(receiverMessage.ID, senderMessage.Replace)
	s.Require().Equal(common.CommandStateRequestTransactionDeclined, senderMessage.CommandParameters.CommandState)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().NotNil(response)
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	receiverMessage = response.Messages[0]
	s.Require().Equal(protobuf.ChatMessage_TRANSACTION_COMMAND, receiverMessage.ContentType)

	s.Require().Equal("Transaction request declined", receiverMessage.Text)
	s.Require().Equal(initialCommandID, receiverMessage.CommandParameters.ID)
	s.Require().Equal(initialCommandID, receiverMessage.Replace)
	s.Require().Equal(common.CommandStateRequestTransactionDeclined, receiverMessage.CommandParameters.CommandState)
	s.Require().NoError(theirMessenger.Shutdown())
}

func (s *MessengerSuite) TestRequestTransaction() {
	value := testValue
	contract := testContract
	receiverAddress := crypto.PubkeyToAddress(s.m.identity.PublicKey)
	receiverAddressString := strings.ToLower(receiverAddress.Hex())
	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	theirPkString := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))

	chat := CreateOneToOneChat(theirPkString, &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(&chat)
	s.Require().NoError(err)

	response, err := s.m.RequestTransaction(context.Background(), theirPkString, value, contract, receiverAddressString)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	senderMessage := response.Messages[0]
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
		func(r *MessengerResponse) bool { return len(r.Messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().NotNil(response)
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	receiverMessage := response.Messages[0]
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
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	senderMessage = response.Messages[0]
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
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Messages, 1)

	receiverMessage = response.Messages[0]
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
	s.Require().NoError(theirMessenger.Shutdown())
}

type MockTransaction struct {
	Status  coretypes.TransactionStatus
	Message coretypes.Message
}

type MockEthClient struct {
	messages map[string]MockTransaction
}

type mockSendMessagesRequest struct {
	types.Waku
	req types.MessagesRequest
}

func (m MockEthClient) TransactionByHash(ctx context.Context, hash types.Hash) (coretypes.Message, coretypes.TransactionStatus, error) {
	mockTransaction, ok := m.messages[hash.Hex()]
	if !ok {
		return coretypes.Message{}, coretypes.TransactionStatusFailed, nil
	}
	return mockTransaction.Message, mockTransaction.Status, nil
}

func (m *mockSendMessagesRequest) SendMessagesRequest(peerID []byte, request types.MessagesRequest) error {
	m.req = request
	return nil
}

func (s *MessengerSuite) TestMessageJSON() {
	message := &common.Message{
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

	expectedJSON := `{"id":"test-1","whisperTimestamp":0,"from":"from-field","alias":"alias","identicon":"","seen":false,"quotedMessage":null,"rtl":false,"lineCount":0,"text":"test-1","chatId":"remote-chat-id","localChatId":"local-chat-id","clock":1,"replace":"","responseTo":"","ensName":"","sticker":null,"commandParameters":null,"timestamp":0,"contentType":0,"messageType":0}`

	messageJSON, err := json.Marshal(message)
	s.Require().NoError(err)
	s.Require().Equal(expectedJSON, string(messageJSON))

	decodedMessage := &common.Message{}
	err = json.Unmarshal([]byte(expectedJSON), decodedMessage)
	s.Require().NoError(err)
	s.Require().Equal(message, decodedMessage)
}

func (s *MessengerSuite) TestRequestHistoricMessagesRequest() {
	shh := &mockSendMessagesRequest{
		Waku: s.shh,
	}
	m := s.newMessenger(shh)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	m.mailserver = []byte("mailserver-id")
	cursor, err := m.RequestHistoricMessages(ctx, 10, 20, []byte{0x01})
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

func (s *MessengerSuite) TestSentEventTracking() {

	//when message sent, its sent field should be "false" until we got confirmation
	chat := CreatePublicChat("test-chat", s.m.transport)
	err := s.m.SaveChat(&chat)
	s.NoError(err)
	inputMessage := buildTestMessage(chat)

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
	err := s.m.SaveChat(&chat)
	s.NoError(err)
	inputMessage := buildTestMessage(chat)

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
	ok, err := shouldResendEmojiReaction(&common.RawMessage{
		MessageType: protobuf.ApplicationMetadataMessage_CONTACT_UPDATE,
		Sent:        false,
		SendCount:   2,
	}, s.m.getTimesource())
	s.Error(err)
	s.False(ok)

	// shouldn't try to resend already sent message
	ok, err = shouldResendEmojiReaction(&common.RawMessage{
		MessageType: protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
		Sent:        true,
		SendCount:   1,
	}, s.m.getTimesource())
	s.Error(err)
	s.False(ok)

	// messages that already sent to many times shouldn't be resend
	ok, err = shouldResendEmojiReaction(&common.RawMessage{
		MessageType: protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
		Sent:        false,
		SendCount:   emojiResendMaxCount + 1,
	}, s.m.getTimesource())
	s.NoError(err)
	s.False(ok)

	// message sent one time CAN'T be resend in 15 seconds (only after 30)
	ok, err = shouldResendEmojiReaction(&common.RawMessage{
		MessageType: protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
		Sent:        false,
		SendCount:   1,
		LastSent:    s.m.getTimesource().GetCurrentTime() - 15*uint64(time.Second),
	}, s.m.getTimesource())
	s.NoError(err)
	s.False(ok)

	// message sent one time CAN be resend in 35 seconds
	ok, err = shouldResendEmojiReaction(&common.RawMessage{
		MessageType: protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
		Sent:        false,
		SendCount:   1,
		LastSent:    s.m.getTimesource().GetCurrentTime() - 35*uint64(time.Second),
	}, s.m.getTimesource())
	s.NoError(err)
	s.True(ok)

	// message sent three times CAN'T be resend in 100 seconds (only after 120)
	ok, err = shouldResendEmojiReaction(&common.RawMessage{
		MessageType: protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
		Sent:        false,
		SendCount:   3,
		LastSent:    s.m.getTimesource().GetCurrentTime() - 100*uint64(time.Second),
	}, s.m.getTimesource())
	s.NoError(err)
	s.False(ok)

	// message sent tow times CAN be resend in 65 seconds
	ok, err = shouldResendEmojiReaction(&common.RawMessage{
		MessageType: protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
		Sent:        false,
		SendCount:   3,
		LastSent:    s.m.getTimesource().GetCurrentTime() - 125*uint64(time.Second),
	}, s.m.getTimesource())
	s.NoError(err)
	s.True(ok)
}

func (s *MessengerSuite) TestMessageSent() {
	//send message
	chat := CreatePublicChat("test-chat", s.m.transport)
	err := s.m.SaveChat(&chat)
	s.NoError(err)
	inputMessage := buildTestMessage(chat)

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

func (s *MessengerSuite) TestResendExpiredEmojis() {
	//send message
	chat := CreatePublicChat("test-chat", s.m.transport)
	err := s.m.SaveChat(&chat)
	s.NoError(err)
	inputMessage := buildTestMessage(chat)

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
	rawMessage.LastSent = rawMessage.LastSent - 35*uint64(time.Second)
	err = s.m.persistence.SaveRawMessage(rawMessage)
	s.NoError(err)
	time.Sleep(2 * time.Second)

	//make sure it was resent and SendCount incremented
	rawMessage, err = s.m.persistence.RawMessageByID(emojiID)
	s.NoError(err)
	s.Equal(2, rawMessage.SendCount)
}

type MessageHandlerSuite struct {
	suite.Suite

	messageHandler *MessageHandler
	logger         *zap.Logger
}

func (s *MessageHandlerSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	s.messageHandler = &MessageHandler{
		identity: privateKey,
		logger:   s.logger,
	}
}

func (s *MessageHandlerSuite) TearDownTest() {
	_ = s.logger.Sync()
}

type testTimeSource struct{}

func (t *testTimeSource) GetCurrentTime() uint64 {
	return uint64(time.Now().Unix())
}

func (s *MessageHandlerSuite) TestRun() {
	key1, err := crypto.GenerateKey()
	s.Require().NoError(err)
	key2, err := crypto.GenerateKey()
	s.Require().NoError(err)

	testCases := []struct {
		Name           string
		Error          bool
		Chat           Chat // Chat to create
		Message        common.Message
		SigPubKey      *ecdsa.PublicKey
		ExpectedChatID string
	}{
		{
			Name: "Public chat",
			Chat: CreatePublicChat("test-chat", &testTimeSource{}),
			Message: common.Message{
				ChatMessage: protobuf.ChatMessage{
					ChatId:      "test-chat",
					MessageType: protobuf.MessageType_PUBLIC_GROUP,
					Text:        "test-text"},
			},
			SigPubKey:      &key1.PublicKey,
			ExpectedChatID: "test-chat",
		},
		{
			Name: "Private message from myself with existing chat",
			Chat: CreateOneToOneChat("test-private-chat", &key1.PublicKey, &testTimeSource{}),
			Message: common.Message{
				ChatMessage: protobuf.ChatMessage{
					ChatId:      "test-chat",
					MessageType: protobuf.MessageType_ONE_TO_ONE,
					Text:        "test-text"},
			},
			SigPubKey:      &key1.PublicKey,
			ExpectedChatID: oneToOneChatID(&key1.PublicKey),
		},
		{
			Name: "Private message from other with existing chat",
			Chat: CreateOneToOneChat("test-private-chat", &key2.PublicKey, &testTimeSource{}),
			Message: common.Message{
				ChatMessage: protobuf.ChatMessage{
					ChatId:      "test-chat",
					MessageType: protobuf.MessageType_ONE_TO_ONE,
					Text:        "test-text"},
			},

			SigPubKey:      &key2.PublicKey,
			ExpectedChatID: oneToOneChatID(&key2.PublicKey),
		},
		{
			Name: "Private message from myself without chat",
			Message: common.Message{
				ChatMessage: protobuf.ChatMessage{
					ChatId:      "test-chat",
					MessageType: protobuf.MessageType_ONE_TO_ONE,
					Text:        "test-text"},
			},

			SigPubKey:      &key1.PublicKey,
			ExpectedChatID: oneToOneChatID(&key1.PublicKey),
		},
		{
			Name: "Private message from other without chat",
			Message: common.Message{
				ChatMessage: protobuf.ChatMessage{
					ChatId:      "test-chat",
					MessageType: protobuf.MessageType_ONE_TO_ONE,
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
			Message: common.Message{
				ChatMessage: protobuf.ChatMessage{
					ChatId:      "non-existing-chat",
					MessageType: protobuf.MessageType_PRIVATE_GROUP,
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
			}

			message := tc.Message
			message.SigPubKey = tc.SigPubKey
			// ChatID is not set at the beginning.
			s.Empty(message.LocalChatID)

			message.ID = strconv.Itoa(idx) // manually set the ID because messages does not go through messageProcessor
			chat, err := s.messageHandler.matchChatEntity(&message, chatsMap, &testTimeSource{})
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

func WaitOnMessengerResponse(m *Messenger, condition func(*MessengerResponse) bool, errorMessage string) (*MessengerResponse, error) {
	var response *MessengerResponse
	return response, tt.RetryWithBackOff(func() error {
		var err error
		response, err = m.RetrieveAll()
		if err == nil && !condition(response) {
			err = errors.New(errorMessage)
		}
		return err
	})
}

func (s *MessengerSuite) TestChatIdentity() {
	keyUID := "0xdeadbeef"
	s.m.account = &multiaccounts.Account{KeyUID: keyUID}

	err := s.m.multiAccounts.SaveAccount(multiaccounts.Account{Name: "string", KeyUID: keyUID})
	s.Require().NoError(err)

	iis := images.SampleIdentityImages()
	s.Require().NoError(s.m.multiAccounts.StoreIdentityImages(keyUID, iis))

	ci, err := s.m.createChatIdentity("private-chat")
	s.Require().NoError(err)

	s.Require().Exactly(len(iis), len(ci.Images))

	spew.Dump(ci, len(ci.Images))
}
