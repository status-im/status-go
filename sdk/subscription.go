package sdk

import (
	"fmt"
	"log"
	"time"
)

type Subscription struct {
	unsubscribe chan bool
	channel     *Channel
}

func (s *Subscription) Subscribe(channel *Channel, fn MsgHandler) {
	s.channel = channel
	s.unsubscribe = make(chan bool)
	for {
		select {
		case <-s.unsubscribe:
			return
		default:
			cmd := fmt.Sprintf(getFilterMessagesFormat, channel.filterID)
			response := channel.conn.statusNode.RPCClient().CallRaw(cmd)
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
	s.channel.removeSubscription(s)
}
