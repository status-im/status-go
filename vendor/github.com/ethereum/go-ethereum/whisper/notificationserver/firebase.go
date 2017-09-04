package notificationserver

import (
	"encoding/json"

	"github.com/NaySoftware/go-fcm"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/static"
)

// FirebaseNotifier provides push notifications via Firebase
type FirebaseNotifier struct {
	template *fcm.FcmClient // client template
}

// NewFirebaseNotifier returns a firebase client
func NewFirebaseNotifier(config *params.FirebaseConfig) (*FirebaseNotifier, error) {
	apiKey, err := config.ReadAuthorizationKeyFile()
	if err != nil {
		return nil, err
	}
	msgTemplate, err := loadMsgTemplate()
	if err != nil {
		return nil, err
	}
	client := fcm.NewFcmClient(apiKey)
	client.Message = *msgTemplate
	return &FirebaseNotifier{
		template: client,
	}, nil
}

// Notify includes a data payload in the notification & notifies a list of devices
func (f *FirebaseNotifier) Notify(devices []string, payload interface{}) error {
	if len(devices) == 0 {
		return ErrNoTarget
	}
	// copy template (msg & apiKey)
	client := *f.template
	// data payload
	data := map[string]string{
		"msg": payload.(string),
	}
	if len(devices) > 1 {
		// multicast message
		client.NewFcmRegIdsMsg(devices, data)
	} else {
		// spec: to send a message to a single device, use the to parameter.
		client.NewFcmMsgTo(devices[0], data)
	}
	_, err := client.Send()
	if err != nil {
		return err
	}
	return nil
}

// loadTemplate loads the template of the firebase notification
func loadMsgTemplate() (*fcm.FcmMsg, error) {
	var msg fcm.FcmMsg
	rawMessage := static.MustAsset("notifications/templates/firebase/notification.json")
	if err := json.Unmarshal(rawMessage, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
