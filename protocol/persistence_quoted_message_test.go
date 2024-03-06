package protocol

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/server"
)

func TestMessengerPrepareMessage(t *testing.T) {
	suite.Run(t, new(TestMessengerPrepareMessageSuite))
}

type TestMessengerPrepareMessageSuite struct {
	MessengerBaseTestSuite
	chatID string
	p      *sqlitePersistence
}

func (s *TestMessengerPrepareMessageSuite) SetupTest() {
	s.MessengerBaseTestSuite.SetupTest()
	s.chatID, s.p = s.setUpTestDatabase()
}

func (s *TestMessengerPrepareMessageSuite) setUpTestDatabase() (string, *sqlitePersistence) {
	chat := CreatePublicChat("test-chat", s.m.transport)
	err := s.m.SaveChat(chat)
	s.NoError(err)

	db, err := openTestDB()
	s.NoError(err)

	p := newSQLitePersistence(db)
	return chat.ID, p
}

func (s *TestMessengerPrepareMessageSuite) generateTextMessage(ID string, From string, Clock uint64, responseTo string) *common.Message {
	return &common.Message{
		ID:          ID,
		From:        From,
		LocalChatID: s.chatID,
		ChatMessage: &protobuf.ChatMessage{
			Text:       RandomLettersString(5),
			Clock:      Clock,
			ResponseTo: responseTo,
		},
	}
}

func (s *TestMessengerPrepareMessageSuite) testMessageContainsImage(testAlbum bool) {
	message1 := s.generateTextMessage("id-1", "1", 1, "")
	message1.ContentType = protobuf.ChatMessage_IMAGE
	message1.Payload = &protobuf.ChatMessage_Image{
		Image: &protobuf.ImageMessage{
			Format:  1,
			Payload: RandomBytes(10),
		},
	}

	message2 := s.generateTextMessage("id-2", "2", 2, message1.ID)
	messages := []*common.Message{message1, message2}

	var message3 *common.Message

	if testAlbum {
		albumID := RandomLettersString(5)
		message1.GetImage().AlbumId = albumID

		message3 = s.generateTextMessage("id-3", "1", 0, "")
		message3.ContentType = protobuf.ChatMessage_IMAGE
		message3.Payload = &protobuf.ChatMessage_Image{
			Image: &protobuf.ImageMessage{
				Format:  1,
				Payload: RandomBytes(10),
				AlbumId: albumID,
			},
		}

		messages = append(messages, message3)
	}

	err := s.m.SaveMessages(messages)
	s.Require().NoError(err)

	err = s.p.SaveMessages(messages)
	s.Require().NoError(err)

	mediaServer, err := server.NewMediaServer(s.m.database, nil, nil, nil)
	s.Require().NoError(err)
	s.Require().NoError(mediaServer.Start())

	retrievedMessages, _, err := s.p.MessageByChatID(s.chatID, "", 10)
	s.Require().NoError(err)
	if testAlbum {
		s.Require().Len(retrievedMessages, 3)
	} else {
		s.Require().Len(retrievedMessages, 2)
	}
	s.Require().Equal(message2.ID, retrievedMessages[0].ID)
	s.Require().Equal(message1.ID, retrievedMessages[0].ResponseTo)

	err = s.m.prepareMessage(retrievedMessages[0], mediaServer)
	s.Require().NoError(err)

	mediaServerImageLink := func(messageID string) string {
		return fmt.Sprintf(`https://Localhost:%d/messages/images?messageId=%s`,
			mediaServer.GetPort(),
			messageID)
	}

	if testAlbum {
		expectedJSON := fmt.Sprintf(`["%s","%s"]`,
			mediaServerImageLink(message1.ID),
			mediaServerImageLink(message3.ID),
		)
		s.Require().Equal(json.RawMessage(expectedJSON), retrievedMessages[0].QuotedMessage.AlbumImages)
	} else {
		expectedURL := mediaServerImageLink(message1.ID)
		s.Require().Equal(expectedURL, retrievedMessages[0].QuotedMessage.ImageLocalURL)
	}
}

func (s *TestMessengerPrepareMessageSuite) Test_WHEN_MessageContainsImage_THEN_preparedMessageAddsAlbumImageWithImageGeneratedLink() {
	testCases := []struct {
		name  string
		album bool
	}{
		{
			name:  "single image",
			album: false,
		},
		{
			name:  "album",
			album: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.testMessageContainsImage(tc.album)
		})
	}
}

func (s *TestMessengerPrepareMessageSuite) Test_WHEN_NoQuotedMessage_THEN_RetrievedMessageDoesNotContainQuotedMessage() {
	message1 := s.generateTextMessage("id-1", "1", 1, "")
	message2 := s.generateTextMessage("id-2", "2", 2, "")
	messages := []*common.Message{message1, message2}

	err := s.m.SaveMessages([]*common.Message{message1, message2})
	s.Require().NoError(err)

	err = s.p.SaveMessages(messages)
	s.Require().NoError(err)

	mediaServer, err := server.NewMediaServer(s.m.database, nil, nil, nil)
	s.Require().NoError(err)
	s.Require().NoError(mediaServer.Start())

	retrievedMessages, _, err := s.p.MessageByChatID(s.chatID, "", 10)
	s.Require().NoError(err)
	s.Require().Equal(message2.ID, retrievedMessages[0].ID)
	s.Require().Empty(retrievedMessages[0].ResponseTo)

	err = s.m.prepareMessage(retrievedMessages[0], mediaServer)
	s.Require().NoError(err)

	s.Require().Equal((*common.QuotedMessage)(nil), retrievedMessages[0].QuotedMessage)
}

func (s *TestMessengerPrepareMessageSuite) Test_WHEN_QuotedMessageDoesNotContainsImage_THEN_RetrievedMessageContainsNoImages() {
	message1 := s.generateTextMessage("id-1", "1", 1, "")
	message2 := s.generateTextMessage("id-2", "2", 2, message1.ID)
	messages := []*common.Message{message1, message2}

	err := s.m.SaveMessages([]*common.Message{message1, message2})
	s.Require().NoError(err)

	err = s.p.SaveMessages(messages)
	s.Require().NoError(err)

	mediaServer, err := server.NewMediaServer(s.m.database, nil, nil, nil)
	s.Require().NoError(err)
	s.Require().NoError(mediaServer.Start())

	retrievedMessages, _, err := s.p.MessageByChatID(s.chatID, "", 10)
	s.Require().NoError(err)
	s.Require().Equal(message2.ID, retrievedMessages[0].ID)
	s.Require().Equal(message1.ID, retrievedMessages[0].ResponseTo)

	err = s.m.prepareMessage(retrievedMessages[0], mediaServer)
	s.Require().NoError(err)

	s.Require().Equal(json.RawMessage(nil), retrievedMessages[0].QuotedMessage.AlbumImages)
}
