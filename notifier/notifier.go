package notifier

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	// IOS identifier for an iOS notification.
	IOS = 1
	// Android identifier for an android notification.
	Android             = 2
	failedPushErrorType = "failed-push"
	pushEndpoint        = "/api/push"
)

// Notifier handles android and ios push notifications.
type Notifier struct {
	client *http.Client
	url    string
}

// New notifier connected to the specified server.
func New(url string) *Notifier {
	client := &http.Client{}

	return &Notifier{url: url, client: client}
}

// Notification details for gorush.
type Notification struct {
	Tokens   []string `json:"tokens"`
	Platform float32  `json:"platform"`
	Message  string   `json:"message"`
}

type request struct {
	Notifications []*Notification `json:"notifications"`
}

// Response from gorush.
type Response struct {
	Logs []struct {
		Type  string `json:"type"`
		Error string `json:"error"`
	} `json:"logs"`
}

// Send a push notification to given devices.
func (n *Notifier) Send(notifications []*Notification) error {
	url := n.url + pushEndpoint
	r := request{Notifications: notifications}

	body, err := json.Marshal(r)
	if err != nil {
		return err
	}

	res, err := n.doRequest(url, body)
	if err != nil {
		return err
	}

	if len(res.Logs) > 0 {
		if res.Logs[0].Type == failedPushErrorType {
			return errors.New(res.Logs[0].Error)
		}
	}

	return err
}

func (n *Notifier) doRequest(url string, body []byte) (res Response, err error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Println(err.Error())
		}
	}()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	return res, json.Unmarshal(body, &res)
}
