package pushnotificationserver

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

func TestPushNotificationRegistrationToGoRushRequest(t *testing.T) {
	message1 := []byte("message-1")
	message2 := []byte("message-2")
	message3 := []byte("message-3")
	hexMessage1 := types.EncodeHex(message1)
	hexMessage2 := types.EncodeHex(message2)
	hexMessage3 := types.EncodeHex(message3)
	chatID := []byte("chat-id")
	publicKey1 := []byte("public-key-1")
	publicKey2 := []byte("public-key-2")
	installationID1 := "installation-id-1"
	installationID2 := "installation-id-2"
	installationID3 := "installation-id-3"
	var platform1 uint = 1
	var platform2 uint = 2
	var platform3 uint = 2
	token1 := "token-1"
	token2 := "token-2"
	token3 := "token-3"

	requestAndRegistrations := []*RequestAndRegistration{
		{
			Request: &protobuf.PushNotification{
				ChatId:         chatID,
				Type:           protobuf.PushNotification_MESSAGE,
				PublicKey:      publicKey1,
				InstallationId: installationID1,
				Message:        message1,
			},
			Registration: &protobuf.PushNotificationRegistration{
				DeviceToken: token1,
				TokenType:   protobuf.PushNotificationRegistration_APN_TOKEN,
			},
		},
		{
			Request: &protobuf.PushNotification{
				ChatId:         chatID,
				Type:           protobuf.PushNotification_MESSAGE,
				PublicKey:      publicKey1,
				InstallationId: installationID2,
				Message:        message2,
			},
			Registration: &protobuf.PushNotificationRegistration{
				DeviceToken: token2,
				TokenType:   protobuf.PushNotificationRegistration_FIREBASE_TOKEN,
			},
		},
		{
			Request: &protobuf.PushNotification{
				ChatId:         chatID,
				Type:           protobuf.PushNotification_MENTION,
				PublicKey:      publicKey2,
				InstallationId: installationID3,
				Message:        message3,
			},
			Registration: &protobuf.PushNotificationRegistration{
				DeviceToken: token3,
				TokenType:   protobuf.PushNotificationRegistration_FIREBASE_TOKEN,
			},
		},
	}

	expectedRequests := &GoRushRequest{
		Notifications: []*GoRushRequestNotification{
			{
				Tokens:   []string{token1},
				Platform: platform1,
				Message:  defaultNewMessageNotificationText,
				Data: &GoRushRequestData{
					EncryptedMessage: hexMessage1,
					ChatID:           types.EncodeHex(chatID),
					PublicKey:        types.EncodeHex(publicKey1),
				},
			},
			{
				Tokens:   []string{token2},
				Platform: platform2,
				Message:  defaultNewMessageNotificationText,
				Data: &GoRushRequestData{
					EncryptedMessage: hexMessage2,
					ChatID:           types.EncodeHex(chatID),
					PublicKey:        types.EncodeHex(publicKey1),
				},
			},
			{
				Tokens:   []string{token3},
				Platform: platform3,
				Message:  defaultMentionNotificationText,
				Data: &GoRushRequestData{
					EncryptedMessage: hexMessage3,
					ChatID:           types.EncodeHex(chatID),
					PublicKey:        types.EncodeHex(publicKey2),
				},
			},
		},
	}
	actualRequests := PushNotificationRegistrationToGoRushRequest(requestAndRegistrations)
	require.Equal(t, expectedRequests, actualRequests)
}
