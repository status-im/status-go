package protocol

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	_ "github.com/mutecomm/go-sqlcipher" // require go-sqlcipher that overrides default implementation
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	enstypes "github.com/status-im/status-go/eth-node/types/ens"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/tt"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	whisper "github.com/status-im/whisper/whisperv6"
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

func (s *MessengerSuite) newMessenger(shh types.Whisper) *Messenger {
	tmpFile, err := ioutil.TempFile("", "")
	s.Require().NoError(err)

	privateKey, err := crypto.GenerateKey()
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
		"installation-1",
		options...,
	)
	s.Require().NoError(err)

	err = m.Init()
	s.Require().NoError(err)

	s.tmpFiles = append(s.tmpFiles, tmpFile)

	return m
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
				err := s.m.SaveChat(publicChat)
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
					ID:        types.EncodeHex(crypto.FromECDSAPub(&key.PublicKey)),
					ChatType:  ChatTypeOneToOne,
					PublicKey: &key.PublicKey,
					Active:    true,
				}
				err = s.m.SaveChat(privateChat)
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
				err = s.m.SaveChat(groupChat)
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
				err := s.m.SaveChat(publicChat)
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
				err = s.m.SaveContact(contact)
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
				err = s.m.SaveContact(contact)
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
				err = s.m.SaveContact(contact)
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

func (s *MessengerSuite) TestSendPublic() {
	chat := CreatePublicChat("test-chat")
	err := s.m.SaveChat(chat)
	s.NoError(err)
	_, err = s.m.Send(context.Background(), chat.ID, []byte("test"))
	s.NoError(err)
}

func (s *MessengerSuite) TestSendPrivate() {
	recipientKey, err := crypto.GenerateKey()
	s.NoError(err)
	chat := CreateOneToOneChat("XXX", &recipientKey.PublicKey)
	err = s.m.SaveChat(chat)
	s.NoError(err)
	_, err = s.m.Send(context.Background(), chat.ID, []byte("test"))
	s.NoError(err)
}

func (s *MessengerSuite) TestRetrieveOwnPublic() {
	chat := CreatePublicChat("status")
	err := s.m.SaveChat(chat)
	s.NoError(err)

	_, err = s.m.Send(context.Background(), chat.ID, []byte("test"))
	s.NoError(err)

	// Give Whisper some time to propagate message to filters.
	time.Sleep(time.Millisecond * 500)

	// Retrieve chat
	messages, err := s.m.RetrieveAll(context.Background(), RetrieveLatest)
	s.NoError(err)
	s.Len(messages, 1)

	// Retrieve again to test skipping already existing err.
	messages, err = s.m.RetrieveAll(context.Background(), RetrieveLastDay)
	s.NoError(err)
	s.Require().Len(messages, 1)

	// Verify message fields.
	message := messages[0]
	s.NotEmpty(message.ID)
	s.Equal(&s.privateKey.PublicKey, message.SigPubKey) // this is OUR message
}

func (s *MessengerSuite) TestRetrieveOwnPublicRaw() {
	chat := CreatePublicChat("status")
	err := s.m.SaveChat(chat)
	s.NoError(err)
	text := "پيل اندر خانه يي تاريک بود عرضه را آورده بودندش هنود  i\nاز براي ديدنش مردم بسي اندر آن ظلمت همي شد هر کسي"

	_, err = s.m.Send(context.Background(), chat.ID, []byte(text))
	s.NoError(err)

	// Give Whisper some time to propagate message to filters.
	time.Sleep(time.Millisecond * 500)

	// Retrieve chat
	messages, err := s.m.RetrieveRawAll()
	s.NoError(err)
	s.Len(messages, 1)

	for _, v := range messages {
		s.Len(v, 1)
		textMessage, ok := v[0].ParsedMessage.(v1protocol.Message)
		s.Require().True(ok)
		s.Equal(textMessage.Content.Text, text)
		s.NotNil(textMessage.Content.ParsedText)
		s.True(textMessage.Content.RTL)
		s.Equal(1, textMessage.Content.LineCount)
	}
}

func (s *MessengerSuite) TestRetrieveOwnPrivate() {
	recipientKey, err := crypto.GenerateKey()
	s.NoError(err)
	chat := CreateOneToOneChat("XXX", &recipientKey.PublicKey)
	err = s.m.SaveChat(chat)
	s.NoError(err)

	messageID, err := s.m.Send(context.Background(), chat.ID, []byte("test"))
	s.NoError(err)

	// No need to sleep because the message is returned from own messages in the processor.

	// Retrieve chat
	messages, err := s.m.RetrieveAll(context.Background(), RetrieveLatest)
	s.NoError(err)
	s.Len(messages, 1)

	// Retrieve again to test skipping already existing err.
	messages, err = s.m.RetrieveAll(context.Background(), RetrieveLastDay)
	s.NoError(err)
	s.Len(messages, 1)

	// Verify message fields.
	message := messages[0]
	s.Equal(messageID[0], message.ID)
	s.Equal(&s.privateKey.PublicKey, message.SigPubKey) // this is OUR message
}

func (s *MessengerSuite) TestRetrieveTheirPrivate() {
	theirMessenger := s.newMessenger(s.shh)
	chat := CreateOneToOneChat("XXX", &s.privateKey.PublicKey)
	err := theirMessenger.SaveChat(chat)
	s.NoError(err)

	messageID, err := theirMessenger.Send(context.Background(), chat.ID, []byte("test"))
	s.NoError(err)

	var messages []*v1protocol.Message

	err = tt.RetryWithBackOff(func() error {
		var err error
		messages, err = s.m.RetrieveAll(context.Background(), RetrieveLatest)
		if err == nil && len(messages) == 0 {
			err = errors.New("no messages")
		}
		return err
	})
	s.NoError(err)

	// Validate received message.
	s.Require().Len(messages, 1)
	message := messages[0]
	s.Equal(messageID[0], message.ID)
	s.Equal(&theirMessenger.identity.PublicKey, message.SigPubKey)
}

func (s *MessengerSuite) TestChatPersistencePublic() {
	chat := Chat{
		ID:                     "chat-name",
		Name:                   "chat-name",
		Color:                  "#fffff",
		Active:                 true,
		ChatType:               ChatTypePublic,
		Timestamp:              10,
		LastClockValue:         20,
		DeletedAtClockValue:    30,
		UnviewedMessagesCount:  40,
		LastMessageContentType: "something",
		LastMessageContent:     "something-else",
	}

	s.Require().NoError(s.m.SaveChat(chat))
	savedChats, err := s.m.Chats()
	s.Require().NoError(err)
	s.Require().Equal(1, len(savedChats))

	actualChat := savedChats[0]
	expectedChat := &chat

	s.Require().Equal(actualChat, expectedChat)
}

func (s *MessengerSuite) TestDeleteChat() {
	chatID := "chatid"
	chat := Chat{
		ID:                     chatID,
		Name:                   "chat-name",
		Color:                  "#fffff",
		Active:                 true,
		ChatType:               ChatTypePublic,
		Timestamp:              10,
		LastClockValue:         20,
		DeletedAtClockValue:    30,
		UnviewedMessagesCount:  40,
		LastMessageContentType: "something",
		LastMessageContent:     "something-else",
	}

	s.Require().NoError(s.m.SaveChat(chat))
	savedChats, err := s.m.Chats()
	s.Require().NoError(err)
	s.Require().Equal(1, len(savedChats))

	s.Require().NoError(s.m.DeleteChat(chatID))
	savedChats, err = s.m.Chats()
	s.Require().NoError(err)
	s.Require().Equal(0, len(savedChats))
}

func (s *MessengerSuite) TestChatPersistenceUpdate() {
	chat := Chat{
		ID:                     "chat-name",
		Name:                   "chat-name",
		Color:                  "#fffff",
		Active:                 true,
		ChatType:               ChatTypePublic,
		Timestamp:              10,
		LastClockValue:         20,
		DeletedAtClockValue:    30,
		UnviewedMessagesCount:  40,
		LastMessageContentType: "something",
		LastMessageContent:     "something-else",
	}

	s.Require().NoError(s.m.SaveChat(chat))
	savedChats, err := s.m.Chats()
	s.Require().NoError(err)
	s.Require().Equal(1, len(savedChats))

	actualChat := savedChats[0]
	expectedChat := &chat

	s.Require().Equal(expectedChat, actualChat)

	chat.Name = "updated-name"
	s.Require().NoError(s.m.SaveChat(chat))
	updatedChats, err := s.m.Chats()
	s.Require().NoError(err)
	s.Require().Equal(1, len(updatedChats))

	actualUpdatedChat := updatedChats[0]
	expectedUpdatedChat := &chat

	s.Require().Equal(expectedUpdatedChat, actualUpdatedChat)
}

func (s *MessengerSuite) TestChatPersistenceOneToOne() {
	pkStr := "0x0424a68f89ba5fcd5e0640c1e1f591d561fa4125ca4e2a43592bc4123eca10ce064e522c254bb83079ba404327f6eafc01ec90a1444331fe769d3f3a7f90b0dde1"
	chat := Chat{
		ID:                     pkStr,
		Name:                   pkStr,
		Color:                  "#fffff",
		Active:                 true,
		ChatType:               ChatTypeOneToOne,
		Timestamp:              10,
		LastClockValue:         20,
		DeletedAtClockValue:    30,
		UnviewedMessagesCount:  40,
		LastMessageContentType: "something",
		LastMessageContent:     "something-else",
	}
	publicKeyBytes, err := hex.DecodeString(pkStr[2:])
	s.Require().NoError(err)

	pk, err := crypto.UnmarshalPubkey(publicKeyBytes)
	s.Require().NoError(err)

	s.Require().NoError(s.m.SaveChat(chat))
	savedChats, err := s.m.Chats()
	s.Require().NoError(err)
	s.Require().Equal(1, len(savedChats))

	actualChat := savedChats[0]
	expectedChat := &chat
	expectedChat.PublicKey = pk

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
			{
				ID:     "1",
				Admin:  false,
				Joined: true,
			},
			{
				ID:     "2",
				Admin:  true,
				Joined: false,
			},
			{
				ID:     "3",
				Admin:  true,
				Joined: true,
			},
		},
		MembershipUpdates: []ChatMembershipUpdate{
			{
				ID:         "1",
				Type:       "type-1",
				Name:       "name-1",
				ClockValue: 1,
				Signature:  "signature-1",
				From:       "from-1",
				Member:     "member-1",
				Members:    []string{"member-1", "member-2"},
			},
			{
				ID:         "2",
				Type:       "type-2",
				Name:       "name-2",
				ClockValue: 2,
				Signature:  "signature-2",
				From:       "from-2",
				Member:     "member-2",
				Members:    []string{"member-2", "member-3"},
			},
		},
		LastClockValue:         20,
		DeletedAtClockValue:    30,
		UnviewedMessagesCount:  40,
		LastMessageContentType: "something",
		LastMessageContent:     "something-else",
	}
	s.Require().NoError(s.m.SaveChat(chat))
	savedChats, err := s.m.Chats()
	s.Require().NoError(err)
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

	s.Require().NoError(s.m.SaveChat(chat1))
	s.Require().NoError(s.m.SaveChat(chat2))
	s.Require().NoError(s.m.SaveChat(chat3))

	s.Require().NoError(s.m.SaveContact(contact))

	contact.Name = "blocked"

	messages := []*Message{
		{
			ID:          "test-1",
			ChatID:      chat2.ID,
			ContentType: "content-type-1",
			Content:     "test-1",
			ClockValue:  1,
			From:        contact.ID,
		},
		{
			ID:          "test-2",
			ChatID:      chat2.ID,
			ContentType: "content-type-2",
			Content:     "test-2",
			ClockValue:  2,
			From:        contact.ID,
		},
		{
			ID:          "test-3",
			ChatID:      chat2.ID,
			ContentType: "content-type-3",
			Content:     "test-3",
			ClockValue:  3,
			Seen:        false,
			From:        "test",
		},
		{
			ID: "test-4",

			ChatID:      chat2.ID,
			ContentType: "content-type-4",
			Content:     "test-4",
			ClockValue:  4,
			Seen:        false,
			From:        "test",
		},
		{
			ID:          "test-5",
			ChatID:      chat2.ID,
			ContentType: "content-type-5",
			Content:     "test-5",
			ClockValue:  5,
			Seen:        true,
			From:        "test",
		},
		{
			ID:          "test-6",
			ChatID:      chat3.ID,
			ContentType: "content-type-6",
			Content:     "test-6",
			ClockValue:  6,
			Seen:        false,
			From:        contact.ID,
		},
		{
			ID:          "test-7",
			ChatID:      chat3.ID,
			ContentType: "content-type-7",
			Content:     "test-7",
			ClockValue:  7,
			Seen:        false,
			From:        "test",
		},
	}

	err := s.m.SaveMessages(messages)
	s.Require().NoError(err)

	response, err := s.m.BlockContact(contact)
	s.Require().NoError(err)

	// The new unviewed count is updated
	s.Require().Equal(uint(1), response[0].UnviewedMessagesCount)
	s.Require().Equal(uint(2), response[1].UnviewedMessagesCount)

	// The new message content is updated
	s.Require().Equal("test-7", response[0].LastMessageContent)
	s.Require().Equal("test-5", response[1].LastMessageContent)

	// The new message content-type is updated
	s.Require().Equal("content-type-7", response[0].LastMessageContentType)
	s.Require().Equal("content-type-5", response[1].LastMessageContentType)

	// The contact is updated
	savedContacts, err := s.m.Contacts()
	s.Require().NoError(err)
	s.Require().Equal(1, len(savedContacts))
	s.Require().Equal("blocked", savedContacts[0].Name)

	// The chat is deleted
	actualChats, err := s.m.Chats()
	s.Require().NoError(err)
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

	s.Require().NoError(s.m.SaveContact(contact))
	savedContacts, err := s.m.Contacts()
	s.Require().NoError(err)
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
		{
			Name:            "pedro.stateofus.eth",
			PublicKeyString: pk2,
		},
		// Not existing name
		{
			Name:            "definitelynotpedro.stateofus.eth",
			PublicKeyString: pk3,
		},
		// Malformed pk
		{
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
	savedContacts, err := s.m.Contacts()
	s.Require().NoError(err)

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

	s.Require().NoError(s.m.SaveContact(contact))
	savedContacts, err := s.m.Contacts()
	s.Require().NoError(err)
	s.Require().Equal(1, len(savedContacts))

	actualContact := savedContacts[0]
	expectedContact := &contact

	expectedContact.Alias = "Concrete Lavender Xiphias"
	expectedContact.Identicon = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAnElEQVR4nOzXQaqDMBRG4bZkLR10e12H23PgZuJUjJAcE8kdnG/44IXDhZ9iyjm/4vnMDrhmFmEWYRZhFpH6n1jW7fSX/+/b+WbQa5lFmEVUljhqZfSdoNcyizCLeNMvn3JTLeh+g17LLMIsorLElt2VK7v3X0dBr2UWYRaBfxNLfifOZhYRNGvAEp8Q9FpmEWYRZhFmEXsAAAD//5K5JFhu0M0nAAAAAElFTkSuQmCC"

	s.Require().Equal(expectedContact, actualContact)

	contact.Name = "updated-name"
	s.Require().NoError(s.m.SaveContact(contact))
	updatedContact, err := s.m.Contacts()
	s.Require().NoError(err)
	s.Require().Equal(1, len(updatedContact))

	actualUpdatedContact := updatedContact[0]
	expectedUpdatedContact := &contact

	s.Require().Equal(expectedUpdatedContact, actualUpdatedContact)
}

func (s *MessengerSuite) TestSharedSecretHandler() {
	_, err := s.m.handleSharedSecrets(nil)
	s.NoError(err)
}

func (s *MessengerSuite) TestCreateGroupChat() {
	chat, err := s.m.CreateGroupChat("test")
	s.Require().NoError(err)
	s.Require().Equal("test", chat.Name)
	publicKeyHex := "0x" + hex.EncodeToString(crypto.FromECDSAPub(&s.m.identity.PublicKey))
	s.Require().Contains(chat.ID, publicKeyHex)
	s.EqualValues([]string{publicKeyHex}, []string{chat.Members[0].ID})
}

func (s *MessengerSuite) TestAddMembersToChat() {
	chat, err := s.m.CreateGroupChat("test")
	s.Require().NoError(err)
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	err = s.m.AddMembersToChat(context.Background(), chat, []*ecdsa.PublicKey{&key.PublicKey})
	s.Require().NoError(err)
	publicKeyHex := "0x" + hex.EncodeToString(crypto.FromECDSAPub(&s.m.identity.PublicKey))
	keyHex := "0x" + hex.EncodeToString(crypto.FromECDSAPub(&key.PublicKey))
	s.EqualValues([]string{publicKeyHex, keyHex}, []string{chat.Members[0].ID, chat.Members[1].ID})
}

// TestGroupChatAutocreate verifies that after receiving a membership update message
// for non-existing group chat, a new one is created.
func (s *MessengerSuite) TestGroupChatAutocreate() {
	theirMessenger := s.newMessenger(s.shh)
	chat, err := theirMessenger.CreateGroupChat("test-group")
	s.Require().NoError(err)
	err = theirMessenger.SaveChat(*chat)
	s.NoError(err)
	err = theirMessenger.AddMembersToChat(
		context.Background(),
		chat,
		[]*ecdsa.PublicKey{&s.privateKey.PublicKey},
	)
	s.NoError(err)
	s.Equal(2, len(chat.Members))

	var chats []*Chat

	err = tt.RetryWithBackOff(func() error {
		_, err := s.m.RetrieveAll(context.Background(), RetrieveLatest)
		if err != nil {
			return err
		}
		chats, err = s.m.Chats()
		if err != nil {
			return err
		}
		if len(chats) == 0 {
			return errors.New("expected a group chat to be created")
		}
		return nil
	})
	s.NoError(err)
	s.Equal(chat.ID, chats[0].ID)
	s.Equal("test-group", chats[0].Name)
	s.Equal(2, len(chats[0].Members))

	// Send confirmation.
	err = s.m.ConfirmJoiningGroup(context.Background(), chats[0])
	s.Require().NoError(err)
}

func (s *MessengerSuite) TestGroupChatMessages() {
	theirMessenger := s.newMessenger(s.shh)
	chat, err := theirMessenger.CreateGroupChat("test-group")
	s.NoError(err)
	err = theirMessenger.SaveChat(*chat)
	s.NoError(err)
	err = theirMessenger.AddMembersToChat(
		context.Background(),
		chat,
		[]*ecdsa.PublicKey{&s.privateKey.PublicKey},
	)
	s.NoError(err)
	_, err = theirMessenger.Send(context.Background(), chat.ID, []byte("hello!"))
	s.NoError(err)

	var messages []*v1protocol.Message

	err = tt.RetryWithBackOff(func() error {
		var err error
		messages, err = s.m.RetrieveAll(context.Background(), RetrieveLatest)
		if err == nil && len(messages) == 0 {
			err = errors.New("no messages")
		}
		return err
	})
	s.NoError(err)

	// Validate received message.
	s.Require().Len(messages, 1)
	message := messages[0]
	s.Equal(&theirMessenger.identity.PublicKey, message.SigPubKey)
}

type mockSendMessagesRequest struct {
	types.Whisper
	req types.MessagesRequest
}

func (m *mockSendMessagesRequest) SendMessagesRequest(peerID []byte, request types.MessagesRequest) error {
	m.req = request
	return nil
}

func (s *MessengerSuite) TestRequestHistoricMessagesRequest() {
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
		Name            string
		Chat            Chat // Chat to create
		Message         v1protocol.Message
		SigPubKey       *ecdsa.PublicKey
		ExpectedChatIDs []string
	}{
		{
			Name:            "Public chat",
			Chat:            CreatePublicChat("test-chat"),
			Message:         v1protocol.CreatePublicTextMessage([]byte("test"), 0, "test-chat"),
			SigPubKey:       &key1.PublicKey,
			ExpectedChatIDs: []string{"test-chat"},
		},
		{
			Name:            "Private message from myself with existing chat",
			Chat:            CreateOneToOneChat("test-private-chat", &key1.PublicKey),
			Message:         v1protocol.CreatePrivateTextMessage([]byte("test"), 0, oneToOneChatID(&key1.PublicKey)),
			SigPubKey:       &key1.PublicKey,
			ExpectedChatIDs: []string{oneToOneChatID(&key1.PublicKey)},
		},
		{
			Name:            "Private message from other with existing chat",
			Chat:            CreateOneToOneChat("test-private-chat", &key2.PublicKey),
			Message:         v1protocol.CreatePrivateTextMessage([]byte("test"), 0, oneToOneChatID(&key1.PublicKey)),
			SigPubKey:       &key2.PublicKey,
			ExpectedChatIDs: []string{oneToOneChatID(&key2.PublicKey)},
		},
		{
			Name:            "Private message from myself without chat",
			Message:         v1protocol.CreatePrivateTextMessage([]byte("test"), 0, oneToOneChatID(&key1.PublicKey)),
			SigPubKey:       &key1.PublicKey,
			ExpectedChatIDs: []string{oneToOneChatID(&key1.PublicKey)},
		},
		{
			Name:            "Private message from other without chat",
			Message:         v1protocol.CreatePrivateTextMessage([]byte("test"), 0, oneToOneChatID(&key1.PublicKey)),
			SigPubKey:       &key2.PublicKey,
			ExpectedChatIDs: []string{oneToOneChatID(&key2.PublicKey)},
		},
		{
			Name:      "Private message without public key",
			SigPubKey: nil,
		},
		{
			Name:      "Private group message",
			Message:   v1protocol.CreatePrivateGroupTextMessage([]byte("test"), 0, "not-existing-chat-id"),
			SigPubKey: &key2.PublicKey,
		},

		// TODO: add test for group messages
	}

	for idx, tc := range testCases {
		s.Run(tc.Name, func() {
			if tc.Chat.ID != "" {
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
			s.Empty(message.ChatID)

			message.ID = []byte(strconv.Itoa(idx)) // manually set the ID because messages does not go through messageProcessor
			messages, err := s.postProcessor.Run([]*v1protocol.Message{&message})
			s.NoError(err)
			s.Require().Len(messages, len(tc.ExpectedChatIDs))
			if len(tc.ExpectedChatIDs) != 0 {
				s.Equal(tc.ExpectedChatIDs[0], message.ChatID)
				s.EqualValues(&message, messages[0])
			}
		})
	}
}
