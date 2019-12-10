package protocol

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/protobuf"
)

type MessageValidatorSuite struct {
	suite.Suite
}

func TestMessageValidatorSuite(t *testing.T) {
	suite.Run(t, new(MessageValidatorSuite))
}

func (s *MessageValidatorSuite) TestValidateRequestAddressForTransaction() {
	testCases := []struct {
		Name    string
		Valid   bool
		Message protobuf.RequestAddressForTransaction
	}{
		{
			Name:  "valid message",
			Valid: true,
			Message: protobuf.RequestAddressForTransaction{
				Clock:    30,
				Value:    "0.34",
				Contract: "some contract",
			},
		},
		{
			Name:  "missing clock value",
			Valid: false,
			Message: protobuf.RequestAddressForTransaction{
				Value:    "0.34",
				Contract: "some contract",
			},
		},
		{
			Name:  "missing value",
			Valid: false,
			Message: protobuf.RequestAddressForTransaction{
				Clock:    30,
				Contract: "some contract",
			},
		},
		{
			Name:  "non number value",
			Valid: false,
			Message: protobuf.RequestAddressForTransaction{
				Clock:    30,
				Value:    "most definitely not a number",
				Contract: "some contract",
			},
		},
	}
	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			err := ValidateReceivedRequestAddressForTransaction(&tc.Message)
			if tc.Valid {
				s.Nil(err)
			} else {
				s.NotNil(err)
			}
		})
	}

}

func (s *MessageValidatorSuite) TestValidatePlainTextMessage() {
	testCases := []struct {
		Name    string
		Valid   bool
		Message protobuf.ChatMessage
	}{
		{
			Name:  "A valid message",
			Valid: true,
			Message: protobuf.ChatMessage{
				ChatId:      "a",
				Clock:       1,
				Timestamp:   2,
				Text:        "some-text",
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.ChatMessage_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			},
		},
		{
			Name:  "Missing chatId",
			Valid: false,
			Message: protobuf.ChatMessage{
				Clock:       1,
				Timestamp:   2,
				Text:        "some-text",
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.ChatMessage_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			},
		},
		{
			Name:  "Missing clock",
			Valid: false,
			Message: protobuf.ChatMessage{
				ChatId:      "a",
				Timestamp:   2,
				Text:        "some-text",
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.ChatMessage_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			},
		},
		{
			Name:  "Missing timestamp",
			Valid: false,
			Message: protobuf.ChatMessage{
				ChatId:      "a",
				Clock:       2,
				Text:        "some-text",
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.ChatMessage_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			},
		},
		{
			Name:  "Missing text",
			Valid: false,
			Message: protobuf.ChatMessage{
				ChatId:      "a",
				Clock:       2,
				Timestamp:   3,
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.ChatMessage_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			},
		},
		{
			Name:  "Blank text",
			Valid: false,
			Message: protobuf.ChatMessage{
				ChatId:      "a",
				Text:        "  \n \t \n  ",
				Clock:       2,
				Timestamp:   3,
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.ChatMessage_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			},
		},
		{
			Name:  "Unknown MessageType",
			Valid: false,
			Message: protobuf.ChatMessage{
				ChatId:      "a",
				Text:        "valid",
				Clock:       2,
				Timestamp:   3,
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.ChatMessage_UNKNOWN_MESSAGE_TYPE,
				ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			},
		},
		{
			Name:  "Unknown ContentType",
			Valid: false,
			Message: protobuf.ChatMessage{
				ChatId:      "a",
				Text:        "valid",
				Clock:       2,
				Timestamp:   3,
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.ChatMessage_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_UNKNOWN_CONTENT_TYPE,
			},
		},
		{
			Name:  "System message MessageType",
			Valid: false,
			Message: protobuf.ChatMessage{
				ChatId:      "a",
				Text:        "valid",
				Clock:       2,
				Timestamp:   3,
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.ChatMessage_SYSTEM_MESSAGE_PRIVATE_GROUP,
				ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			},
		},
		{
			Name:  "Request address for transaction message type",
			Valid: false,
			Message: protobuf.ChatMessage{
				ChatId:      "a",
				Text:        "valid",
				Clock:       2,
				Timestamp:   3,
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.ChatMessage_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_TRANSACTION_COMMAND,
			},
		},
		{
			Name:  "Valid  emoji only emssage",
			Valid: true,
			Message: protobuf.ChatMessage{
				ChatId:      "a",
				Text:        ":+1:",
				Clock:       2,
				Timestamp:   3,
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.ChatMessage_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_EMOJI,
			},
		},
		// TODO: FIX ME
		/*		{
					Name:  "Invalid  emoji only emssage",
					Valid: false,
					Message: protobuf.ChatMessage{
						ChatId:      "a",
						Text:        ":+1: not valid",
						Clock:       2,
						Timestamp:   3,
						ResponseTo:  "",
						EnsName:     "",
						MessageType: protobuf.ChatMessage_ONE_TO_ONE,
						ContentType: protobuf.ChatMessage_EMOJI,
					},
				}
				,*/
		{
			Name:  "Valid sticker message",
			Valid: true,
			Message: protobuf.ChatMessage{
				ChatId:     "a",
				Text:       "valid",
				Clock:      2,
				Timestamp:  3,
				ResponseTo: "",
				EnsName:    "",
				Payload: &protobuf.ChatMessage_Sticker{
					Sticker: &protobuf.StickerMessage{
						Pack: 1,
						Hash: "some-hash",
					},
				},
				MessageType: protobuf.ChatMessage_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_STICKER,
			},
		},
		{
			Name:  "Invalid sticker message without Hash",
			Valid: false,
			Message: protobuf.ChatMessage{
				ChatId:     "a",
				Text:       "valid",
				Clock:      2,
				Timestamp:  3,
				ResponseTo: "",
				EnsName:    "",
				Payload: &protobuf.ChatMessage_Sticker{
					Sticker: &protobuf.StickerMessage{
						Pack: 1,
					},
				},
				MessageType: protobuf.ChatMessage_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_STICKER,
			},
		},
		{
			Name:  "Invalid sticker message without any content",
			Valid: false,
			Message: protobuf.ChatMessage{
				ChatId:      "a",
				Text:        "valid",
				Clock:       2,
				Timestamp:   3,
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.ChatMessage_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_STICKER,
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			err := ValidateReceivedChatMessage(&tc.Message)
			if tc.Valid {
				s.Nil(err)
			} else {
				s.NotNil(err)
			}
		})
	}
}
