package protocol

import (
	"strings"
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
		Name             string
		WhisperTimestamp uint64
		Valid            bool
		Message          *protobuf.RequestAddressForTransaction
	}{
		{
			Name:             "valid message",
			WhisperTimestamp: 30,
			Valid:            true,
			Message: &protobuf.RequestAddressForTransaction{
				Clock:    30,
				Value:    "0.34",
				Contract: "some contract",
			},
		},
		{
			Name:             "missing clock value",
			WhisperTimestamp: 30,
			Valid:            false,
			Message: &protobuf.RequestAddressForTransaction{
				Value:    "0.34",
				Contract: "some contract",
			},
		},
		{
			Name:             "missing value",
			WhisperTimestamp: 30,
			Valid:            false,
			Message: &protobuf.RequestAddressForTransaction{
				Clock:    30,
				Contract: "some contract",
			},
		},
		{
			Name:             "non number value",
			WhisperTimestamp: 30,
			Valid:            false,
			Message: &protobuf.RequestAddressForTransaction{
				Clock:    30,
				Value:    "most definitely not a number",
				Contract: "some contract",
			},
		},
		{
			Name:             "Clock value too high",
			WhisperTimestamp: 30,
			Valid:            false,
			Message: &protobuf.RequestAddressForTransaction{
				Clock:    151000,
				Value:    "0.34",
				Contract: "some contract",
			},
		},
	}
	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			err := ValidateReceivedRequestAddressForTransaction(tc.Message, tc.WhisperTimestamp)
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
		Name             string
		WhisperTimestamp uint64
		Valid            bool
		Message          *protobuf.ChatMessage
	}{
		{
			Name:             "A valid message",
			WhisperTimestamp: 2,
			Valid:            true,
			Message: &protobuf.ChatMessage{
				ChatId:      "a",
				Clock:       1,
				Timestamp:   2,
				Text:        strings.Repeat("É", maxChatMessageTextLength),
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			},
		},
		{
			Name:             "Missing chatId",
			WhisperTimestamp: 2,
			Valid:            false,
			Message: &protobuf.ChatMessage{
				Clock:       1,
				Timestamp:   2,
				Text:        "some-text",
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			},
		},
		{
			Name:             "Missing clock",
			WhisperTimestamp: 2,
			Valid:            false,
			Message: &protobuf.ChatMessage{
				ChatId:      "a",
				Timestamp:   2,
				Text:        "some-text",
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			},
		},
		{
			Name:             "Clock value too high",
			WhisperTimestamp: 2,
			Valid:            false,
			Message: &protobuf.ChatMessage{
				ChatId:      "a",
				Clock:       133000,
				Timestamp:   1,
				Text:        "some-text",
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			},
		},
		{
			Name:             "Missing timestamp",
			WhisperTimestamp: 2,
			Valid:            false,
			Message: &protobuf.ChatMessage{
				ChatId:      "a",
				Clock:       2,
				Text:        "some-text",
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			},
		},
		{
			Name:             "Missing text",
			WhisperTimestamp: 2,
			Valid:            false,
			Message: &protobuf.ChatMessage{
				ChatId:      "a",
				Clock:       2,
				Timestamp:   3,
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			},
		},
		{
			Name:             "Blank text",
			WhisperTimestamp: 2,
			Valid:            false,
			Message: &protobuf.ChatMessage{
				ChatId:      "a",
				Text:        "  \n \t \n  ",
				Clock:       2,
				Timestamp:   3,
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			},
		},
		{
			Name:             "Too long text",
			WhisperTimestamp: 2,
			Valid:            false,
			Message: &protobuf.ChatMessage{
				ChatId:      "a",
				Clock:       1,
				Timestamp:   2,
				Text:        strings.Repeat("É", maxChatMessageTextLength+1),
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			},
		},
		{
			Name:             "Unknown MessageType",
			WhisperTimestamp: 2,
			Valid:            false,
			Message: &protobuf.ChatMessage{
				ChatId:      "a",
				Text:        "valid",
				Clock:       2,
				Timestamp:   3,
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.MessageType_UNKNOWN_MESSAGE_TYPE,
				ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			},
		},
		{
			Name:             "Unknown ContentType",
			WhisperTimestamp: 2,
			Valid:            false,
			Message: &protobuf.ChatMessage{
				ChatId:      "a",
				Text:        "valid",
				Clock:       2,
				Timestamp:   3,
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_UNKNOWN_CONTENT_TYPE,
			},
		},
		{
			Name:             "System message MessageType",
			WhisperTimestamp: 2,
			Valid:            false,
			Message: &protobuf.ChatMessage{
				ChatId:      "a",
				Text:        "valid",
				Clock:       2,
				Timestamp:   3,
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.MessageType_SYSTEM_MESSAGE_PRIVATE_GROUP,
				ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			},
		},
		{
			Name:             "Request address for transaction message type",
			WhisperTimestamp: 2,
			Valid:            false,
			Message: &protobuf.ChatMessage{
				ChatId:      "a",
				Text:        "valid",
				Clock:       2,
				Timestamp:   3,
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_TRANSACTION_COMMAND,
			},
		},
		{
			Name:             "Valid  emoji only emssage",
			WhisperTimestamp: 2,
			Valid:            true,
			Message: &protobuf.ChatMessage{
				ChatId:      "a",
				Text:        ":+1:",
				Clock:       2,
				Timestamp:   3,
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_EMOJI,
			},
		},
		// TODO: FIX ME
		/*		{
					Name:  "Invalid  emoji only emssage",
					Valid: false,
					Message: protobuf.ChatMessage{
						ChatID:      "a",
						Text:        ":+1: not valid",
						Clock:       2,
						Timestamp:   3,
						ResponseTo:  "",
						EnsName:     "",
						MessageType: protobuf.MessageType_ONE_TO_ONE,
						ContentType: protobuf.ChatMessage_EMOJI,
					},
				}
				,*/
		{
			Name:             "Valid sticker message",
			WhisperTimestamp: 2,
			Valid:            true,
			Message: &protobuf.ChatMessage{
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
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_STICKER,
			},
		},
		{
			Name:             "Invalid sticker message without Hash",
			WhisperTimestamp: 2,
			Valid:            false,
			Message: &protobuf.ChatMessage{
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
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_STICKER,
			},
		},
		{
			Name:             "Invalid sticker message without any content",
			WhisperTimestamp: 2,
			Valid:            false,
			Message: &protobuf.ChatMessage{
				ChatId:      "a",
				Text:        "valid",
				Clock:       2,
				Timestamp:   3,
				ResponseTo:  "",
				EnsName:     "",
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_STICKER,
			},
		},
		{
			Name:             "Valid image message",
			WhisperTimestamp: 2,
			Valid:            true,
			Message: &protobuf.ChatMessage{
				ChatId:     "a",
				Text:       "valid",
				Clock:      2,
				Timestamp:  3,
				ResponseTo: "",
				EnsName:    "",
				Payload: &protobuf.ChatMessage_Image{
					Image: &protobuf.ImageMessage{
						Type:    1,
						Payload: []byte("some-payload"),
					},
				},
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_IMAGE,
			},
		},
		{
			Name:             "Invalid image message, type unknown",
			WhisperTimestamp: 2,
			Valid:            false,
			Message: &protobuf.ChatMessage{
				ChatId:     "a",
				Text:       "valid",
				Clock:      2,
				Timestamp:  3,
				ResponseTo: "",
				EnsName:    "",
				Payload: &protobuf.ChatMessage_Image{
					Image: &protobuf.ImageMessage{
						Type:    protobuf.ImageType_UNKNOWN_IMAGE_TYPE,
						Payload: []byte("some-payload"),
					},
				},
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_STICKER,
			},
		},
		{
			Name:             "Invalid image message, missing payload",
			WhisperTimestamp: 2,
			Valid:            false,
			Message: &protobuf.ChatMessage{
				ChatId:     "a",
				Text:       "valid",
				Clock:      2,
				Timestamp:  3,
				ResponseTo: "",
				EnsName:    "",
				Payload: &protobuf.ChatMessage_Image{
					Image: &protobuf.ImageMessage{
						Type: 1,
					},
				},
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_IMAGE,
			},
		},
		{
			Name:             "Valid audio message",
			WhisperTimestamp: 2,
			Valid:            true,
			Message: &protobuf.ChatMessage{
				ChatId:     "a",
				Text:       "valid",
				Clock:      2,
				Timestamp:  3,
				ResponseTo: "",
				EnsName:    "",
				Payload: &protobuf.ChatMessage_Audio{
					Audio: &protobuf.AudioMessage{
						Type:    1,
						Payload: []byte("some-payload"),
					},
				},
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_AUDIO,
			},
		},
		{
			Name:             "Invalid audio message, type unknown",
			WhisperTimestamp: 2,
			Valid:            false,
			Message: &protobuf.ChatMessage{
				ChatId:     "a",
				Text:       "valid",
				Clock:      2,
				Timestamp:  3,
				ResponseTo: "",
				EnsName:    "",
				Payload: &protobuf.ChatMessage_Audio{
					Audio: &protobuf.AudioMessage{
						Type:    protobuf.AudioMessage_UNKNOWN_AUDIO_TYPE,
						Payload: []byte("some-payload"),
					},
				},
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_STICKER,
			},
		},
		{
			Name:             "Invalid audio message, missing payload",
			WhisperTimestamp: 2,
			Valid:            false,
			Message: &protobuf.ChatMessage{
				ChatId:     "a",
				Text:       "valid",
				Clock:      2,
				Timestamp:  3,
				ResponseTo: "",
				EnsName:    "",
				Payload: &protobuf.ChatMessage_Audio{
					Audio: &protobuf.AudioMessage{
						Type: 1,
					},
				},
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				ContentType: protobuf.ChatMessage_AUDIO,
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			err := ValidateReceivedChatMessage(tc.Message, tc.WhisperTimestamp)
			if tc.Valid {
				s.Nil(err)
			} else {
				s.NotNil(err)
			}
		})
	}
}

func (s *MessageValidatorSuite) TestValidateEmojiReaction() {
	testCases := []struct {
		Name             string
		Valid            bool
		WhisperTimestamp uint64
		Message          *protobuf.EmojiReaction
	}{
		{
			Name:             "valid emoji reaction",
			Valid:            true,
			WhisperTimestamp: 30,
			Message: &protobuf.EmojiReaction{
				Clock:       30,
				ChatId:      "chat-id",
				MessageId:   "message-id",
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				Type:        protobuf.EmojiReaction_LOVE,
			},
		},
		{
			Name:             "valid emoji retraction",
			Valid:            true,
			WhisperTimestamp: 30,
			Message: &protobuf.EmojiReaction{
				Clock:       30,
				ChatId:      "0.34",
				MessageId:   "message-id",
				Type:        protobuf.EmojiReaction_LOVE,
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				Retracted:   true,
			},
		},
		{
			Name:             "missing chatID",
			Valid:            false,
			WhisperTimestamp: 30,
			Message: &protobuf.EmojiReaction{
				Clock:       30,
				MessageId:   "message-id",
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				Type:        protobuf.EmojiReaction_LOVE,
			},
		},
		{
			Name:             "missing messageID",
			Valid:            false,
			WhisperTimestamp: 30,
			Message: &protobuf.EmojiReaction{
				Clock:       30,
				ChatId:      "chat-id",
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				Type:        protobuf.EmojiReaction_LOVE,
			},
		},
		{
			Name:             "missing type",
			Valid:            false,
			WhisperTimestamp: 30,
			Message: &protobuf.EmojiReaction{
				Clock:       30,
				ChatId:      "chat-id",
				MessageId:   "message-id",
				MessageType: protobuf.MessageType_ONE_TO_ONE,
			},
		},
		{
			Name:             "missing message type",
			Valid:            false,
			WhisperTimestamp: 30,
			Message: &protobuf.EmojiReaction{
				Clock:     30,
				ChatId:    "chat-id",
				MessageId: "message-id",
				Type:      protobuf.EmojiReaction_LOVE,
			},
		},
		{
			Name:             "clock value too high",
			Valid:            false,
			WhisperTimestamp: 30,
			Message: &protobuf.EmojiReaction{
				Clock:       900000,
				ChatId:      "chat-id",
				MessageId:   "message-id",
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				Type:        protobuf.EmojiReaction_LOVE,
			},
		},
	}
	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			err := ValidateReceivedEmojiReaction(tc.Message, tc.WhisperTimestamp)
			if tc.Valid {
				s.Nil(err)
			} else {
				s.NotNil(err)
			}
		})
	}

}
