package protocol

import (
	"encoding/json"
	"fmt"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/server"
)

func (s *MessengerSuite) setUpTestDatabase() (string, *sqlitePersistence) {
	chat := CreatePublicChat("test-chat", s.m.transport)
	err := s.m.SaveChat(chat)
	s.NoError(err)
	db, err := openTestDB()
	s.NoError(err)
	p := newSQLitePersistence(db)

	return chat.ID, p
}

func (s *MessengerSuite) Test_WHEN_MessageContainsImage_Then_preparedMessageAddsAlbumImageWithImageGeneratedLink() {
	chatID, p := s.setUpTestDatabase()

	message1 := &common.Message{
		ID:          "id-1",
		LocalChatID: chatID,
		ChatMessage: &protobuf.ChatMessage{
			Text:        "content-1",
			Clock:       uint64(1),
			ContentType: protobuf.ChatMessage_IMAGE,
			Payload: &protobuf.ChatMessage_Image{
				Image: &protobuf.ImageMessage{
					Type:    1,
					Payload: []byte("some-payload"),
				},
			},
		},
		From: "1",
	}
	message2 := &common.Message{
		ID:          "id-2",
		LocalChatID: chatID,
		ChatMessage: &protobuf.ChatMessage{
			Text:       "content-2",
			Clock:      uint64(2),
			ResponseTo: "id-1",
		},

		From: "2",
	}

	messages := []*common.Message{message1, message2}

	err := s.m.SaveMessages([]*common.Message{message1, message2})
	s.Require().NoError(err)

	err = p.SaveMessages(messages)
	s.Require().NoError(err)

	mediaServer, err := server.NewMediaServer(s.m.database, nil, nil, nil)
	s.Require().NoError(err)
	s.Require().NoError(mediaServer.Start())

	retrievedMessages, _, err := p.MessageByChatID(chatID, "", 10)
	s.Require().NoError(err)
	s.Require().Equal("id-2", retrievedMessages[0].ID)
	s.Require().Equal("id-1", retrievedMessages[0].ResponseTo)

	err = s.m.prepareMessage(retrievedMessages[0], mediaServer)
	s.Require().NoError(err)

	expectedURL := fmt.Sprintf(`["https://Localhost:%d/messages/images?messageId=id-1"]`, mediaServer.GetPort())

	s.Require().Equal(json.RawMessage(expectedURL), retrievedMessages[0].QuotedMessage.AlbumImages)
}

func (s *MessengerSuite) Test_WHEN_NoQuotedMessage_THEN_RetrievedMessageDoesNotContainQuotedMessage() {
	chatID, p := s.setUpTestDatabase()

	message1 := &common.Message{
		ID:          "id-1",
		LocalChatID: chatID,
		ChatMessage: &protobuf.ChatMessage{
			Text:  "content-1",
			Clock: uint64(1),
		},
		From: "1",
	}

	message2 := &common.Message{
		ID:          "id-2",
		LocalChatID: chatID,
		ChatMessage: &protobuf.ChatMessage{
			Text:  "content-2",
			Clock: uint64(2),
		},

		From: "2",
	}

	messages := []*common.Message{message1, message2}

	err := s.m.SaveMessages([]*common.Message{message1, message2})
	s.Require().NoError(err)

	err = p.SaveMessages(messages)
	s.Require().NoError(err)

	mediaServer, err := server.NewMediaServer(s.m.database, nil, nil, nil)
	s.Require().NoError(err)
	s.Require().NoError(mediaServer.Start())

	retrievedMessages, _, err := p.MessageByChatID(chatID, "", 10)
	s.Require().NoError(err)
	s.Require().Equal("id-2", retrievedMessages[0].ID)
	s.Require().Equal("", retrievedMessages[0].ResponseTo)

	err = s.m.prepareMessage(retrievedMessages[0], mediaServer)
	s.Require().NoError(err)

	s.Require().Equal((*common.QuotedMessage)(nil), retrievedMessages[0].QuotedMessage)
}

func (s *MessengerSuite) Test_WHEN_QuotedMessageDoesNotContainsImage_THEN_RetrievedMessageContainsNoImages() {
	chatID, p := s.setUpTestDatabase()

	message1 := &common.Message{
		ID:          "id-1",
		LocalChatID: chatID,
		ChatMessage: &protobuf.ChatMessage{
			Text:  "content-1",
			Clock: uint64(1),
		},
		From: "1",
	}

	message2 := &common.Message{
		ID:          "id-2",
		LocalChatID: chatID,
		ChatMessage: &protobuf.ChatMessage{
			Text:       "content-2",
			Clock:      uint64(2),
			ResponseTo: "id-1",
		},

		From: "2",
	}

	messages := []*common.Message{message1, message2}

	err := s.m.SaveMessages([]*common.Message{message1, message2})
	s.Require().NoError(err)

	err = p.SaveMessages(messages)
	s.Require().NoError(err)

	mediaServer, err := server.NewMediaServer(s.m.database, nil, nil, nil)
	s.Require().NoError(err)
	s.Require().NoError(mediaServer.Start())

	retrievedMessages, _, err := p.MessageByChatID(chatID, "", 10)
	s.Require().NoError(err)
	s.Require().Equal("id-2", retrievedMessages[0].ID)
	s.Require().Equal("id-1", retrievedMessages[0].ResponseTo)

	err = s.m.prepareMessage(retrievedMessages[0], mediaServer)
	s.Require().NoError(err)

	s.Require().Equal(json.RawMessage(nil), retrievedMessages[0].QuotedMessage.AlbumImages)
}
