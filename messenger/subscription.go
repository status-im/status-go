package messenger

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/status-im/status-go/geth/node"
)

// MsgHandler is a callback function that processes messages delivered to
// asynchronous subscribers.
type MsgHandler func(msg *Msg)

var (
	newContactKeyPrefix           = "~#c1"
	contactRequestPrefix          = "~#c2"
	confirmedContactRequestPrefix = "~#c3"
	messagePrefix                 = "~#c4"
	seenPrefix                    = "~#c5"
	contactUpdatePrefix           = "~#c6"
)

type Subscription struct {
	sn          *node.StatusNode
	unsubscribe chan bool
}

func (s *Subscription) Subscribe(filterID string, fn MsgHandler) {
	s.unsubscribe = make(chan bool)
	for {
		select {
		case <-s.unsubscribe:
			return
		default:
			cmd := fmt.Sprintf(getFilterMessagesFormat, filterID)
			response := s.sn.RPCClient().CallRaw(cmd)
			f := unmarshalJSON(response)
			v := f.(map[string]interface{})["result"]
			switch vv := v.(type) {
			case []interface{}:
				for _, u := range vv {
					payload := u.(map[string]interface{})["payload"]
					message, err := MessageFromPayload(payload.(string))
					if err != nil {
						log.Println(err)
					} else {
						fn(message)
					}
				}
			default:
				log.Println(v, "is of a type I don't know how to handle")
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (s *Subscription) Unsubscribe() {
	s.unsubscribe <- true
}

// MessageFromPayload : TODO ...
func MessageFromPayload(payload string) (*Msg, error) {
	message, err := unrawrChatMessage(payload)
	if err != nil {
		return nil, err
	}
	var x []interface{}
	json.Unmarshal([]byte(message), &x)
	if len(x) < 1 {
		return nil, errors.New("unsupported message type")
	}
	// TODO (adriacidre) add support for other message types
	if x[0].(string) != messagePrefix {
		return nil, errors.New("unsupported message type")
	}
	properties := x[1].([]interface{})

	return &Msg{
		From:      "TODO : someone",
		Text:      properties[0].(string),
		Timestamp: int64(properties[3].(float64)),
		Raw:       string(message),
	}, nil
}
