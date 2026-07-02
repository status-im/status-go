package pushnotificationserver

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

const defaultNewMessageNotificationText = "New message in Messenger"
const defaultMentionNotificationText = "New mention or reply in Communities"
const defaultRequestToJoinCommunityNotificationText = "Someone requested to join a community you are an admin of"
const defaultContactRequestNotificationText = "Someone sent you a contact request"

const deepLinkChats = "status-app://chats"                      // plain messages -> messenger
const deepLinkActivityCenter = "status-app://ac"                // mentions, community requests, etc. -> activity center
const deepLinkContactRequests = "status-app://contact-requests" // contact requests -> activity center contact requests

type GoRushRequestData struct {
	EncryptedMessage string `json:"encryptedMessage"`
	ChatID           string `json:"chatId"`
	PublicKey        string `json:"publicKey"`
	DeepLink         string `json:"deepLink,omitempty"`
}

type GoRushRequestNotification struct {
	Tokens           []string           `json:"tokens"`
	Platform         uint               `json:"platform"`
	Message          string             `json:"message"`
	Topic            string             `json:"topic"`
	ContentAvailable bool               `json:"content_available,omitempty"`
	Sound            string             `json:"sound,omitempty"`
	Priority         string             `json:"priority,omitempty"`
	PushType         string             `json:"push_type,omitempty"`
	Data             *GoRushRequestData `json:"data"`
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
		var text string
		if request.Type == protobuf.PushNotification_MESSAGE {
			text = defaultNewMessageNotificationText
		} else if request.Type == protobuf.PushNotification_REQUEST_TO_JOIN_COMMUNITY {
			text = defaultRequestToJoinCommunityNotificationText
		} else if request.Type == protobuf.PushNotification_CONTACT_REQUEST {
			text = defaultContactRequestNotificationText
		} else {
			text = defaultMentionNotificationText
		}
		isAPN := registration.TokenType == protobuf.PushNotificationRegistration_APN_TOKEN
		var sound, priority, pushType, deepLink string
		if isAPN {
			sound = "default"
			priority = "high"
			pushType = "alert"
			if request.Type == protobuf.PushNotification_MESSAGE {
				deepLink = deepLinkChats
			} else if request.Type == protobuf.PushNotification_CONTACT_REQUEST {
				deepLink = deepLinkContactRequests
			} else {
				deepLink = deepLinkActivityCenter
			}
		}
		goRushRequests.Notifications = append(goRushRequests.Notifications,
			&GoRushRequestNotification{
				Tokens:           []string{registration.DeviceToken},
				Platform:         tokenTypeToGoRushPlatform(registration.TokenType),
				Message:          text,
				Topic:            registration.ApnTopic,
				ContentAvailable: isAPN,
				Sound:            sound,
				Priority:         priority,
				PushType:         pushType,
				Data: &GoRushRequestData{
					EncryptedMessage: types.EncodeHex(request.Message),
					ChatID:           types.EncodeHex(request.ChatId),
					PublicKey:        types.EncodeHex(request.PublicKey),
					DeepLink:         deepLink,
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
