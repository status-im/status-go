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

// Publish : Publishes a message with the given body on the current channel
func (c *Channel) Publish(body string) error {
	message := NewMsg(c.conn.userName, body, c.channelName)
	cmd := fmt.Sprintf(standardMessageFormat,
		c.conn.address,
		c.channelKey,
		message.ToPayload(),
		c.topic,
		c.conn.minimumPoW,
	)

	c.conn.rpc.Call(cmd)

	return nil
}

// Subscribe : ...
func (c *Channel) Subscribe(fn MsgHandler) (*Subscription, error) {
	log.Println("Subscribed to channel '", c.channelName, "'")
	subscription := &Subscription{}
	go subscription.Subscribe(c, fn)
	c.subscriptions = append(c.subscriptions, subscription)

	return subscription, nil
}

// Close current channel and all its subscriptions
func (c *Channel) Close() {
	for _, sub := range c.subscriptions {
		c.removeSubscription(sub)
	}
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
