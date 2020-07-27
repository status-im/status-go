package pushnotificationserver

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/status-im/status-go/protocol/protobuf"
)

const defaultNotificationMessage = "You have a new message"

type GoRushRequestData struct {
	EncryptedMessage string `json:"encryptedMessage"`
	ChatID           string `json:"chatId"`
	PublicKey        string `json:"publicKey"`
}

type GoRushRequestNotification struct {
	Tokens   []string           `json:"tokens"`
	Platform uint               `json:"platform"`
	Message  string             `json:"message"`
	Data     *GoRushRequestData `json:"data"`
}

type GoRushRequest struct {
	Notifications []*GoRushRequestNotification `json:"notifications"`
}

type RequestAndRegistration struct {
	Request      *protobuf.PushNotification
	Registration *protobuf.PushNotificationRegistration
}

func tokenTypeToGoRushPlatform(tokenType protobuf.PushNotificationRegistration_TokenType) uint {
	switch tokenType {
	case protobuf.PushNotificationRegistration_APN_TOKEN:
		return 1
	case protobuf.PushNotificationRegistration_FIREBASE_TOKEN:
		return 2
	}
	return 0
}

func PushNotificationRegistrationToGoRushRequest(requestAndRegistrations []*RequestAndRegistration) *GoRushRequest {
	goRushRequests := &GoRushRequest{}
	for _, requestAndRegistration := range requestAndRegistrations {
		request := requestAndRegistration.Request
		registration := requestAndRegistration.Registration
		goRushRequests.Notifications = append(goRushRequests.Notifications,
			&GoRushRequestNotification{
				Tokens:   []string{registration.DeviceToken},
				Platform: tokenTypeToGoRushPlatform(registration.TokenType),
				Message:  defaultNotificationMessage,
				Data: &GoRushRequestData{
					EncryptedMessage: hex.EncodeToString(request.Message),
					ChatID:           request.ChatId,
					PublicKey:        hex.EncodeToString(request.PublicKey),
				},
			})
	}
	return goRushRequests
}

func sendGoRushNotification(request *GoRushRequest, url string) error {
	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}

	_, err = http.Post(url+"/api/push", "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	return nil
}
