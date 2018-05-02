package sdk

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// Channel : ...
type Channel struct {
	conn          *SDK
	name          string
	filterID      string
	channelKey    string
	topicID       string
	visibility    string
	subscriptions []*Subscription
}

// Subscribe to the current channel by polling the network for new messages
// and executing provided handler
func (c *Channel) Subscribe(fn MsgHandler) (*Subscription, error) {
	log.Println("Subscribed to channel '", c.name, "'")
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

// NewContactKeyRequest first message that is sent to a future contact. At that
// point the only topic we know that the contact is filtering is the
// discovery-topic with his public key so that is what NewContactKey will
// be sent to.
// It contains the sym-key and topic that will be used for future communications
// as well as the actual message that we want to send.
// The sym-key and topic are generated randomly because we don’t want to have
// any correlation between a topic and its participants to avoid leaking
// metadata.
// When one of the contacts recovers his account, a NewContactKey message is
// sent as well to change the symmetric key and topic.
func (c *Channel) NewContactKeyRequest(username string) error {
	format := `["%s",["%s","%s","%s","%s"]]]`
	contactRequest := fmt.Sprintf(format, ContactRequestType, username, "", "", "")

	format = `["%s",["%s","%s",%s]`
	msg := fmt.Sprintf(format, NewContactKeyType, c.conn.address, c.topicID, contactRequest)

	return c.SendPostRawMsg(msg)
}

// ContactRequest wrapped in a NewContactKey message when initiating a contact request.
func (c *Channel) ContactRequest(username, image string) error {
	format := `["%s",["%s","%s","%s","%s"]]]`
	msg := fmt.Sprintf(format, ContactRequestType, username, image, c.conn.address, "")

	return c.SendPostRawMsg(msg)
}

// ConfirmedContactRequest this is the message that will be sent when the
// contact accepts the contact request. It will be sent on the topic that
// was provided in the NewContactKey message and use the sym-key.
// Both users will therefore have the same filter.
func (c *Channel) ConfirmedContactRequest(username, image string) error {
	format := `["%s",["%s","%s","%s","%s"]]`
	msg := fmt.Sprintf(format, ConfirmedContactRequestType, username, image, c.conn.address, "")

	return c.SendPostRawMsg(msg)
}

// Publish a message with the given body on the current channel
func (c *Channel) Publish(body string) error {
	visibility := "~:public-group-user-message"
	if c.visibility != "" {
		visibility = c.visibility
	}

	now := time.Now().Unix()
	format := `["%s",["%s","text/plain","%s",%d,%d]]`

	msg := fmt.Sprintf(format, StandardMessageType, body, visibility, now*100, now*100)
	println("[ SENDING ] : " + msg)

	return c.SendPostRawMsg(msg)
}

// SeenRequest sent when a user sees a message (opens the chat and loads the
// message). Can acknowledge multiple messages at the same time
func (c *Channel) SeenRequest(ids []string) error {
	format := `["%s",["%s","%s"]]`
	body, err := json.Marshal(ids)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf(format, SeenType, body)

	return c.SendPostRawMsg(msg)
}

// ContactUpdateRequest sent when the user changes his name or profile-image.
func (c *Channel) ContactUpdateRequest(username, image string) error {
	format := `["%s",["%s","%s"]]`
	msg := fmt.Sprintf(format, ContactUpdateType, username, image)

	return c.SendPostRawMsg(msg)
}

// SendPostRawMsg sends a shh_post message with the given body.
func (c *Channel) SendPostRawMsg(body string) error {
	param := shhPostParam{
		Signature: c.conn.address,
		SymKeyID:  c.channelKey,
		Payload:   rawrChatMessage(body),
		Topic:     c.topicID,
		TTL:       10,
		PowTarget: c.conn.minimumPoW,
		PowTime:   1,
	}

	_, err := shhPostRequest(c.conn, []*shhPostParam{&param})
	if err != nil {
		log.Println(err.Error())
	}

	return err
}

// PNBroadcastAvailabilityRequest makes a request used by push notification
// servers to broadcast its availability, this request is exposing current
// push notification server Public Key.
func (c *Channel) PNBroadcastAvailabilityRequest() error {
	format := `["%s",["%s"]]`
	msg := fmt.Sprintf(format, PNBroadcastAvailabilityType, c.conn.pubkey)

	return c.SendPostRawMsg(msg)
}

// PNRegistrationRequest request sent by clients wanting to be registered on
// a specific push notification server.
// The client has to provide a channel(topic + symkey) so the future
// communications happen through this channel.
// Additionally a device token will identify the device on the push notification
// provider.
func (c *Channel) PNRegistrationRequest(symkey, topic, deviceToken string, slotAvailabilityRatio float32) error {
	format := `["%s",["%s","%s","%s"]]]`
	msg := fmt.Sprintf(format, PNRegistrationType, symkey, topic, deviceToken, slotAvailabilityRatio)

	return c.SendPostRawMsg(msg)
}

// PNRegistrationConfirmationRequest request sent by the push notification
// server to let a client know what's the pubkey associated with its registered
// token.
func (c *Channel) PNRegistrationConfirmationRequest(pubkey string) error {
	format := `["%s",["%s"]]]`
	msg := fmt.Sprintf(format, PNRegistrationConfirmationType, pubkey)

	return c.SendPostRawMsg(msg)
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

func (c *Channel) pollMessages() (msg *Msg) {
	res, err := shhGetFilterMessagesRequest(c.conn, []string{c.filterID})
	if err != nil {
		log.Fatalf("Error when sending request to server: %s", err)
		return
	}

	switch vv := res.Result.(type) {
	case []interface{}:
		for _, u := range vv {
			msg, err = messageFromEnvelope(u)
			if err == nil && supportedMessage(msg.Type) {
				msg.ChannelName = c.name
				return
			}
			return nil
		}
	default:
		log.Println(res.Result, "is of a type I don't know how to handle")
	}
	return
}
