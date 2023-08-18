package protocol

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/common"
)

type ChatTestSuite struct {
	suite.Suite
}

func TestChatSuite(t *testing.T) {
	suite.Run(t, new(ChatTestSuite))
}

func (s *ChatTestSuite) TestValidateChat() {
	testCases := []struct {
		Name  string
		Valid bool
		Chat  Chat
	}{
		{
			Name:  "valid one to one chat",
			Valid: true,
			Chat: Chat{
				ID:       "0x0424a68f89ba5fcd5e0640c1e1f591d561fa4125ca4e2a43592bc4123eca10ce064e522c254bb83079ba404327f6eafc01ec90a1444331fe769d3f3a7f90b0dde1",
				Name:     "",
				ChatType: ChatTypeOneToOne,
			},
		},
		{
			Name:  "valid public chat",
			Valid: true,
			Chat: Chat{
				ID:       "status",
				Name:     "status",
				ChatType: ChatTypePublic,
			},
		},
		{
			Name:  "empty chatID",
			Valid: false,
			Chat: Chat{
				ID:       "",
				Name:     "status",
				ChatType: ChatTypePublic,
			},
		},
		{
			Name:  "invalid one to one chat, wrong public key",
			Valid: false,
			Chat: Chat{
				ID:       "0xnotvalid",
				Name:     "",
				ChatType: ChatTypeOneToOne,
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			err := tc.Chat.Validate()
			if tc.Valid {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}

}

func (s *ChatTestSuite) TestUpdateFromMessage() {

	// Base case, clock is higher
	message := common.NewMessage()
	chat := &Chat{}

	message.Clock = 1
	s.Require().NoError(chat.UpdateFromMessage(message, &testTimeSource{}))
	s.Require().NotNil(chat.LastMessage)
	s.Require().Equal(uint64(1), chat.LastClockValue)

	// Clock is lower and lastMessage is not nil
	message = common.NewMessage()
	lastMessage := message
	chat = &Chat{LastClockValue: 2, LastMessage: lastMessage}

	message.Clock = 1
	s.Require().NoError(chat.UpdateFromMessage(message, &testTimeSource{}))
	s.Require().Equal(lastMessage, chat.LastMessage)
	s.Require().Equal(uint64(2), chat.LastClockValue)

	// Clock is lower and lastMessage is nil
	message = common.NewMessage()
	chat = &Chat{LastClockValue: 2}

	message.Clock = 1
	s.Require().NoError(chat.UpdateFromMessage(message, &testTimeSource{}))
	s.Require().NotNil(chat.LastMessage)
	s.Require().Equal(uint64(2), chat.LastClockValue)

	// Clock is higher but lastMessage has lower clock message then the receiving one
	message = common.NewMessage()
	chat = &Chat{LastClockValue: 2}

	message.Clock = 1
	s.Require().NoError(chat.UpdateFromMessage(message, &testTimeSource{}))
	s.Require().NotNil(chat.LastMessage)
	s.Require().Equal(uint64(2), chat.LastClockValue)

	chat.LastClockValue = 4
	message = common.NewMessage()
	message.Clock = 3
	s.Require().NoError(chat.UpdateFromMessage(message, &testTimeSource{}))
	s.Require().Equal(chat.LastMessage, message)
	s.Require().Equal(uint64(4), chat.LastClockValue)

}

func (s *ChatTestSuite) TestSerializeJSON() {

	message := common.NewMessage()
	chat := &Chat{}

	message.From = "0x04deaafa03e3a646e54a36ec3f6968c1d3686847d88420f00c0ab6ee517ee1893398fca28aacd2af74f2654738c21d10bad3d88dc64201ebe0de5cf1e313970d3d"
	message.Clock = 1
	message.Text = "`some markdown text` https://status.im"
	s.Require().NoError(message.PrepareContent(""))
	message.ParsedTextAst = nil
	chat.LastMessage = message

	encodedJSON, err := json.Marshal(chat)
	s.Require().NoError(err)

	decodedChat := &Chat{}

	s.Require().NoError(json.Unmarshal(encodedJSON, decodedChat))
	s.Require().Equal(chat, decodedChat)
}

func (s *ChatTestSuite) TestUpdateFirstMessageTimestamp() {
	chat := &Chat{
		FirstMessageTimestamp: 0,
	}

	setAndCheck := func(newValue uint32, success bool, expectedValue uint32) {
		s.Equal(success, chat.UpdateFirstMessageTimestamp(newValue))
		s.Equal(expectedValue, chat.FirstMessageTimestamp)
	}

	setAndCheck(FirstMessageTimestampUndefined, false, FirstMessageTimestampUndefined)
	setAndCheck(FirstMessageTimestampNoMessage, true, FirstMessageTimestampNoMessage)
	setAndCheck(FirstMessageTimestampUndefined, false, FirstMessageTimestampNoMessage)
	setAndCheck(FirstMessageTimestampNoMessage, false, FirstMessageTimestampNoMessage)
	setAndCheck(200, true, 200)
	setAndCheck(FirstMessageTimestampUndefined, false, 200)
	setAndCheck(FirstMessageTimestampNoMessage, false, 200)
	setAndCheck(100, true, 100)
}
