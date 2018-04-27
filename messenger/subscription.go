package messenger

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/status-im/status-go/geth/node"
)

// MsgHandler is a callback function that processes messages delivered to
// asynchronous subscribers.
type MsgHandler func(msg *Msg)

// Subscription : allows you manage subscriptions to a specific topic
type Subscription struct {
	filters     []string
	handler     MsgHandler
	sn          *node.StatusNode
	unsubscribe chan bool
}

// NewSubscription : creates a new subsription object
func NewSubscription(sn *node.StatusNode, filters []string, handler MsgHandler) *Subscription {
	return &Subscription{
		sn:      sn,
		filters: filters,
		handler: handler,
	}
}

// Subscribe : Listens to a specific topic and executes the given function for
// each message
func (s *Subscription) Subscribe() {
	s.unsubscribe = make(chan bool)
	for {
		select {
		case <-s.unsubscribe:
			return
		default:
			switch messages := s.pollMessagesResult().(type) {
			case []interface{}:
				s.handleMessageList(messages)
			default:
				log.Println("unsupported message format")
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// Unsubscribe : stop polling on this Subscription topic
func (s *Subscription) Unsubscribe() {
	s.unsubscribe <- true
}

func (s *Subscription) pollMessagesResult() interface{} {
	filters := strings.Join(s.filters, ",")
	cmd := fmt.Sprintf(getFilterMessagesFormat, filters)

	response := s.sn.RPCClient().CallRaw(cmd)
	f, err := unmarshalJSON(response)
	if err != nil {
		// TODO (adriacidre) return this error
		return nil
	}

	return f.(map[string]interface{})["result"]
}

func (s *Subscription) handleMessageList(messages []interface{}) {
	for _, message := range messages {
		payload := message.(map[string]interface{})["payload"]
		message, err := messageFromPayload(payload.(string))
		if err != nil {
			log.Println(err)
		} else {
			s.handler(message)
		}
	}
}

// messageFromPayload : builds a valid Msg from the given payload
func messageFromPayload(payload string) (*Msg, error) {
	var msg []interface{}

	rawMsg, err := unrawrChatMessage(payload)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(rawMsg, &msg); err != nil {
		return nil, err
	}

	if len(msg) < 1 {
		return nil, errors.New("unknown message format")
	}

	msgType := msg[0].(string)
	if !supportedMessage(msgType) {
		return nil, errors.New("unsupported message type")
	}

	properties := msg[1].([]interface{})

	return &Msg{
		Type:      msg[0].(string),
		From:      "TODO : someone",
		Text:      properties[0].(string),
		Timestamp: int64(properties[3].(float64)),
		Raw:       string(rawMsg),
	}, nil
}
