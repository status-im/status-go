package sdk

import (
	"fmt"
	"log"
)

// Channel : ...
type Channel struct {
	conn          *Conn
	channelName   string
	filterID      string
	channelKey    string
	topic         string
	subscriptions []*Subscription
}

// Publish : ...
func (c *Channel) Publish(body string) error {
	cfg, _ := c.conn.statusNode.Config()
	powTime := cfg.WhisperConfig.MinimumPoW

	message := NewMsg(c.conn.userName, body, c.channelName)
	cmd := fmt.Sprintf(standardMessageFormat, c.conn.address, c.channelKey, message.ToPayload(), c.topic, powTime)

	log.Println("-> Sending:", cmd)
	log.Println("-> SENT:", c.conn.statusNode.RPCClient().CallRaw(cmd))

	return nil
}

// MsgHandler is a callback function that processes messages delivered to
// asynchronous subscribers.
type MsgHandler func(msg *Msg)

// Subscribe : ...
func (c *Channel) Subscribe(fn MsgHandler) (*Subscription, error) {
	log.Println("Subscribed to channel '", c.channelName, "'")
	subscription := &Subscription{}
	go subscription.Subscribe(c, fn)
	c.subscriptions = append(c.subscriptions, subscription)

	return subscription, nil
}

func (c *Channel) removeSubscription(sub *Subscription) {
	var subs []*Subscription
	for _, s := range c.subscriptions {
		if s != sub {
			subs = append(subs, s)
		}
	}
	c.subscriptions = subs
}
