package sdk

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto/sha3"
)

const (
	// NewContactKeyType message type for newContactKeyFormat
	NewContactKeyType = "~#c1"
	// ContactRequestType message type for contactRequestFormat
	ContactRequestType = "~#c2"
	// ConfirmedContactRequestType message type for confirmedContactRequestFormat
	ConfirmedContactRequestType = "~#c3"
	// StandardMessageType message type for StandardMessageFormat
	StandardMessageType = "~#c4"
	// SeenType message type for SeentType
	SeenType = "~#c5"
	// ContactUpdateType message type for contactUpdateMsg
	ContactUpdateType = "~#c6"
	// PNBroadcastAvailabilityType message type for push notification broadcast
	// availability
	PNBroadcastAvailabilityType = "~#c90"
	// PNRegistrationType message type for sending a registration request to
	// a push notification server
	PNRegistrationType = "~#c91"
	// PNRegistrationConfirmationType message type to allow a push notification
	// server confirm a registration
	PNRegistrationConfirmationType = "~#c92"
)

// supportedMessage check if the message type is supported
func supportedMessage(msgType string) bool {
	_, ok := map[string]bool{
		NewContactKeyType:              true,
		ContactRequestType:             true,
		ConfirmedContactRequestType:    true,
		StandardMessageType:            true,
		SeenType:                       true,
		ContactUpdateType:              true,
		PNBroadcastAvailabilityType:    true,
		PNRegistrationType:             true,
		PNRegistrationConfirmationType: true,
	}[msgType]

	return ok
}

// Msg is a structure used by Subscribers and Publish().
type Msg struct {
	From        string   `json:"from"`
	ChannelName string   `json:"channel"`
	Channel     *Channel `json:"-"`
	Raw         string   `json:"-"`
	Type        string   `json:"-"`
	Properties  interface{}
}

// ID gets the message id
func (m *Msg) ID() string {
	return fmt.Sprintf("%X", sha3.Sum256([]byte(m.Raw)))
}

func rawrChatMessage(raw string) string {
	bytes := []byte(raw)

	return fmt.Sprintf("0x%s", hex.EncodeToString(bytes))
}

func unrawrChatMessage(message string) ([]byte, error) {
	return hex.DecodeString(message[2:])
}

func messageFromEnvelope(u interface{}) (msg *Msg, err error) {
	payload := u.(map[string]interface{})["payload"]
	return messageFromPayload(payload.(string))
}

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

	message := Msg{
		From: "TODO : someone",
		Type: msgType,
		Raw:  string(rawMsg),
	}

	properties := msg[1].([]interface{})
	switch msgType {
	case NewContactKeyType:
		message.Properties = newContactKeyMsgFromProperties(properties)
	case ContactRequestType:
		message.Properties = contactMsgFromProperties(properties)
	case ConfirmedContactRequestType:
		message.Properties = confirmedContactMsgFromProperties(properties)
	case StandardMessageType:
		message.Properties = publishMsgFromProperties(properties)
	case SeenType:
		message.Properties = seenMsgFromProperties(properties)
	case ContactUpdateType:
		message.Properties = contactUpdateMsgFromProperties(properties)
	case PNBroadcastAvailabilityType:
		message.Properties = pnBroadcastAvailabilityMsgFromProperties(properties)
	case PNRegistrationType:
		message.Properties = pnRegistrationMsgFromProperties(properties)
	case PNRegistrationConfirmationType:
		// message.Properties = newPublishMessageFromProperties(properties)
	default:
		return nil, errors.New("unsupported message type")
	}

	return &message, nil
}

// PublishMsg representation of a StandardMessageType
type PublishMsg struct {
	Text       string
	MimeType   string
	Visibility string
	ClockValue float64
	Timestamp  float64
}

func publishMsgFromProperties(properties []interface{}) *PublishMsg {
	return &PublishMsg{
		Text:       properties[0].(string),
		MimeType:   properties[1].(string),
		Visibility: properties[2].(string),
		ClockValue: properties[3].(float64),
		Timestamp:  properties[4].(float64),
	}
}

// ContactMsg parsed struct for ContactRequestType
type ContactMsg struct {
	Name     string
	Image    string
	Address  string
	FCMToken string // This will be deprecated
}

func contactMsgFromProperties(properties []interface{}) *ContactMsg {
	crProperties := properties[2].([]interface{})

	return &ContactMsg{
		Name:     crProperties[0].(string),
		Image:    crProperties[1].(string),
		Address:  crProperties[2].(string),
		FCMToken: crProperties[3].(string),
	}
}

// NewContactKeyMsg parsed struct for NewContactKeyType
type NewContactKeyMsg struct {
	Address string
	Topic   string
	Contact *ContactMsg
}

func newContactKeyMsgFromProperties(properties []interface{}) *NewContactKeyMsg {
	crProperties := properties[2].([]interface{})

	return &NewContactKeyMsg{
		Address: properties[0].(string),
		Topic:   properties[1].(string),
		Contact: contactMsgFromProperties(crProperties),
	}
}

// ConfirmedContactMsg parsed struct for ConfirmedContactRequestType
type ConfirmedContactMsg struct {
	Name     string
	Image    string
	Address  string
	FCMToken string // This will be deprecated
}

func confirmedContactMsgFromProperties(properties []interface{}) *ConfirmedContactMsg {
	return &ConfirmedContactMsg{
		Name:     properties[0].(string),
		Image:    properties[1].(string),
		Address:  properties[2].(string),
		FCMToken: properties[3].(string),
	}
}

// SeenMsg parsed struct for SeenType
type SeenMsg struct {
	ID1 string
	ID2 string
}

func seenMsgFromProperties(properties []interface{}) *SeenMsg {
	return &SeenMsg{
		ID1: properties[0].(string),
		ID2: properties[1].(string),
	}
}

// ContactUpdateMsg parsed struct for ContactUpdateType
type ContactUpdateMsg struct {
	Name  string
	Image string
}

func contactUpdateMsgFromProperties(properties []interface{}) *ContactUpdateMsg {
	return &ContactUpdateMsg{
		Name:  properties[0].(string),
		Image: properties[1].(string),
	}
}

// PNBroadcastAvailabilityMsg parsed struct for PNBroadcastAvailabilityType
type PNBroadcastAvailabilityMsg struct {
	Pubkey string
}

func pnBroadcastAvailabilityMsgFromProperties(properties []interface{}) *PNBroadcastAvailabilityMsg {
	return &PNBroadcastAvailabilityMsg{
		Pubkey: properties[0].(string),
	}
}

// PNRegistrationMsg parsed struct for PNRegistrationType
type PNRegistrationMsg struct {
	Symkey           string
	Topic            string
	DeviceToken      string
	SlotAvailability float32
}

func pnRegistrationMsgFromProperties(properties []interface{}) *PNRegistrationMsg {
	return &PNRegistrationMsg{
		Symkey:           properties[0].(string),
		Topic:            properties[1].(string),
		DeviceToken:      properties[2].(string),
		SlotAvailability: properties[3].(float32),
	}
}

// PNRegistrationConfirmationMsg parsed struct for PNRegistrationConfirmationType
type PNRegistrationConfirmationMsg struct {
	Pubkey string
}

func pnRegistrationConfirmationMsgFromProperties(properties []interface{}) *PNRegistrationConfirmationMsg {
	return &PNRegistrationConfirmationMsg{
		Pubkey: properties[0].(string),
	}
}
