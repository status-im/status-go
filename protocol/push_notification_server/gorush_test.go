package push_notification_server

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/protobuf"
)

func TestPushNotificationRegistrationToGoRushRequest(t *testing.T) {
	message1 := []byte("message-1")
	message2 := []byte("message-2")
	message3 := []byte("message-3")
	hexMessage1 := hex.EncodeToString(message1)
	hexMessage2 := hex.EncodeToString(message2)
	hexMessage3 := hex.EncodeToString(message3)
	chatID := "chat-id"
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
				PublicKey:      publicKey1,
				InstallationId: installationID1,
				Message:        message1,
			},
			Registration: &protobuf.PushNotificationRegistration{
				Token:     token1,
				TokenType: protobuf.PushNotificationRegistration_APN_TOKEN,
			},
		},
		{
			Request: &protobuf.PushNotification{
				ChatId:         chatID,
				PublicKey:      publicKey1,
				InstallationId: installationID2,
				Message:        message2,
			},
			Registration: &protobuf.PushNotificationRegistration{
				Token:     token2,
				TokenType: protobuf.PushNotificationRegistration_FIREBASE_TOKEN,
			},
		},
		{
			Request: &protobuf.PushNotification{
				ChatId:         chatID,
				PublicKey:      publicKey2,
				InstallationId: installationID3,
				Message:        message3,
			},
			Registration: &protobuf.PushNotificationRegistration{
				Token:     token3,
				TokenType: protobuf.PushNotificationRegistration_FIREBASE_TOKEN,
			},
		},
	}

	expectedRequests := &GoRushRequest{
		Notifications: []*GoRushRequestNotification{
			{
				Tokens:   []string{token1},
				Platform: platform1,
				Message:  defaultNotificationMessage,
				Data: &GoRushRequestData{
					EncryptedMessage: hexMessage1,
					ChatID:           chatID,
					PublicKey:        hex.EncodeToString(publicKey1),
				},
			},
			{
				Tokens:   []string{token2},
				Platform: platform2,
				Message:  defaultNotificationMessage,
				Data: &GoRushRequestData{
					EncryptedMessage: hexMessage2,
					ChatID:           chatID,
					PublicKey:        hex.EncodeToString(publicKey1),
				},
			},
			{
				Tokens:   []string{token3},
				Platform: platform3,
				Message:  defaultNotificationMessage,
				Data: &GoRushRequestData{
					EncryptedMessage: hexMessage3,
					ChatID:           chatID,
					PublicKey:        hex.EncodeToString(publicKey2),
				},
			},
		},
	}
	actualRequests := PushNotificationRegistrationToGoRushRequest(requestAndRegistrations)
	require.Equal(t, expectedRequests, actualRequests)
}
