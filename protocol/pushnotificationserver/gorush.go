package pushnotificationserver

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"go.uber.org/zap"

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
	Topic    string             `json:"topic"`
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
				Topic:    registration.ApnTopic,
				Data: &GoRushRequestData{
					EncryptedMessage: hex.EncodeToString(request.Message),
					ChatID:           request.ChatId,
					PublicKey:        hex.EncodeToString(request.PublicKey),
				},
			})
	}
	return goRushRequests
}

func sendGoRushNotification(request *GoRushRequest, url string, logger *zap.Logger) error {
	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}

	response, err := http.Post(url+"/api/push", "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, _ := ioutil.ReadAll(response.Body)

	logger.Info("Sent gorush request", zap.String("response", string(body)))

	return nil
}
