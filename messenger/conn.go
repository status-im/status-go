package messenger

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/node"
)

// TODO(adriacidre) Probably better move this to structs
var (
	generateSymKeyFromPasswordFormat = `{"jsonrpc":"2.0","id":2950,"method":"shh_generateSymKeyFromPassword","params":["%s"]}`
	newMessageFilterFormat           = `{"jsonrpc":"2.0","id":2,"method":"shh_newMessageFilter","params":[{"allowP2P":true,"topics":["%s"],"type":"sym","symKeyID":"%s"}]}`
	getFilterMessagesFormat          = `{"jsonrpc":"2.0","id":2968,"method":"shh_getFilterMessages","params":["%s"]}`
	web3ShaFormat                    = `{"jsonrpc":"2.0","method":"web3_sha3","params":["%s"],"id":%d}`
	standardMessageFormat            = `{"jsonrpc":"2.0","id":633,"method":"shh_post","params":[{"sig":"%s","symKeyID":"%s","payload":"%s","topic":"%s","ttl":10,"powTarget":%g,"powTime":1}]}`

	messagePayloadMsg          = `["~#c4",["%s","text/plain","~:public-group-user-message",%d,%d]]`
	newContactKeyMsg           = `["~#c1",["%s","%s",%s]`
	contactRequestMsg          = `["~#c2",["%s","%s","%s","%s”]]]`
	confirmedContactRequestMsg = `["~#c3",["%s","%s","%s","%s"]]`
	contactUpdateMsg           = `["~#c6",["%s","%s"]]`
	seenMsg                    = `["~#c5",["%s","%s"]]`
)

// Messenger : interface to manage status communications throygh whisper
type Messenger struct {
	sn *node.StatusNode
}

// New : Messenger constructor
func New(sn *node.StatusNode) *Messenger {
	return &Messenger{sn: sn}
}

// Signup creates a new account with the given password
func (c *Messenger) Signup(password string) string {
	am := account.NewManager(c.sn)
	address, _, _, err := am.CreateAccount(password)
	if err != nil {
		log.Fatalf("could not create an account. ERR: %v", err)
	}

	return address
}

// Login with the provided credentials
func (c *Messenger) Login(addr, password string) (string, error) {
	am := account.NewManager(c.sn)
	w, err := c.sn.WhisperService()
	if err != nil {
		return "", err
	}

	_, accountKey, err := am.AddressToDecryptedAccount(addr, password)
	if err != nil {
		return "", err
	}

	log.Println("ADDING PRIVATE KEY :", accountKey.PrivateKey)
	keyID, err := w.AddKeyPair(accountKey.PrivateKey)
	if err != nil {
		return "", err
	}
	log.Println("Logging in as", keyID)

	if err := am.SelectAccount(addr, password); err != nil {
		return "", err
	}

	return keyID, nil
}

// JoinPublicChannel joins a public channel
func (c *Messenger) JoinPublicChannel(channelName string) (string, string, string) {
	cmd := fmt.Sprintf(generateSymKeyFromPasswordFormat, channelName)
	f, err := unmarshalJSON(c.sn.RPCClient().CallRaw(cmd))
	if err != nil {
		// TODO (adriacidre) return this error
		return "", "", ""
	}

	key := f.(map[string]interface{})["result"].(string)
	id := int(f.(map[string]interface{})["id"].(float64))

	src := []byte(channelName)
	p := "0x" + hex.EncodeToString(src)

	cmd = fmt.Sprintf(web3ShaFormat, p, id)
	f1, err := unmarshalJSON(c.sn.RPCClient().CallRaw(cmd))
	if err != nil {
		// TODO (adriacidre) return this error
		return "", "", ""
	}
	topic := f1.(map[string]interface{})["result"].(string)
	topic = topic[0:10]

	cmd = fmt.Sprintf(newMessageFilterFormat, topic, key)
	res := c.sn.RPCClient().CallRaw(cmd)
	f3, err := unmarshalJSON(res)
	filterID := f3.(map[string]interface{})["result"].(string)
	if err != nil {
		// TODO (adriacidre) return this error
		return "", "", ""
	}

	return filterID, topic, key
}

// Publish : The typical message exchanged between 2 users.
func (c *Messenger) Publish(addressKeyID, chName, chKey, chTopic, body, username string) error {
	cfg := c.sn.Config()

	message := NewMsg(username, body, chName)
	cmd := fmt.Sprintf(standardMessageFormat,
		addressKeyID,
		chKey,
		message.ToPayload(),
		chTopic,
		cfg.WhisperConfig.MinimumPoW)

	c.sn.RPCClient().CallRaw(cmd)

	return nil
}

// Subscribe : subscribes to the given whisper filters and executes the
// given logic for each supported matching message
func (c *Messenger) Subscribe(filters []string, fn MsgHandler) *Subscription {
	// TODO (adriacidre) store and allow subsrcription management
	s := NewSubscription(c.sn, filters, fn)
	go s.Subscribe()

	return s
}

func unmarshalJSON(j string) (interface{}, error) {
	var v interface{}
	return v, json.Unmarshal([]byte(j), &v)
}

// NewContactKeyRequest : First message that is sent to a future contact. At that
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
func (c *Messenger) NewContactKeyRequest(addressKeyID, chTopic, chKey, username string) {
	contactRequest := fmt.Sprintf(contactRequestMsg, username, "", "", "")
	msg := fmt.Sprintf(newContactKeyMsg, addressKeyID, chTopic, contactRequest)

	c.callStandardMsg(msg, addressKeyID, chKey, chTopic)
}

// ContactRequest : Wrapped in a NewContactKey message when initiating a contact request.
func (c *Messenger) ContactRequest(addressKeyID, chKey, chTopic, username, image string) {
	msg := fmt.Sprintf(contactRequestMsg, username, image, addressKeyID, "")
	c.callStandardMsg(msg, addressKeyID, chKey, chTopic)
}

// ConfirmedContactRequest : This is the message that will be sent when the
// contact accepts the contact request. It will be sent on the topic that
// was provided in the NewContactKey message and use the sym-key.
// Both users will therefore have the same filter.
func (c *Messenger) ConfirmedContactRequest(addressKeyID, chKey, chTopic, username, image string) {
	msg := fmt.Sprintf(confirmedContactRequestMsg, username, image, addressKeyID, "")
	c.callStandardMsg(msg, addressKeyID, chKey, chTopic)
}

// ContactUpdateRequest : Sent when the user changes his name or profile-image.
func (c *Messenger) ContactUpdateRequest(addressKeyID, chKey, chTopic, username, image string) {
	msg := fmt.Sprintf(contactUpdateMsg, username, image)
	c.callStandardMsg(msg, addressKeyID, chKey, chTopic)
}

// SeenRequest : Sent when a user sees a message (opens the chat and loads the
// message). Can acknowledge multiple messages at the same time
func (c *Messenger) SeenRequest(addressKeyID, chKey, chTopic string, ids []string) error {
	body, err := json.Marshal(ids)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf(seenMsg, body)
	c.callStandardMsg(msg, addressKeyID, chKey, chTopic)

	return nil
}

func (c *Messenger) callStandardMsg(input, addressKeyID, chKey, chTopic string) {
	cfg := c.sn.Config()
	msg := rawrChatMessage(input)

	cmd := fmt.Sprintf(standardMessageFormat,
		addressKeyID,
		chKey,
		msg,
		chTopic,
		cfg.WhisperConfig.MinimumPoW)

	c.sn.RPCClient().CallRaw(cmd)
}
