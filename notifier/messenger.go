package notifier

import (
	"crypto/rand"
	b64 "encoding/base64"
	"log"
	"time"

	sdk "github.com/status-im/status-go-sdk"
)

// RegistrationRequestMsg : peers wanting to use this notification server will
// send anonymous whisper messages with the device registration token, and
// a secure channel, so communication between PN server and client can happen
// securely
type RegistrationRequestMsg struct {
	DeviceRegistrationToken string `json:"token"`
	SecureChannel           []byte `json:"channel"`
}

// Messenger : whisper interface for the notifier
type Messenger struct {
	discoveryTopic string
	pollInterval   time.Duration
	addressKey     string
	password       string
	notifier       NotificationProvider
	client         *sdk.SDK
	baseAccount    *sdk.Account
	// TODO(adriacidre) this is only in memory key pair db to store all
	// `pubkey/device tokens`, should be moved to something persistant
	tokenDB map[string]string
}

// NotificationProvider represents the Push Notification Provider microservice,
// which is capable of forwarding the actual Push Notifications to devices
type NotificationProvider interface {
	Send(tokens []string, message string) error
}

// NewMessenger Creates a new Messenger
func NewMessenger(rpc sdk.RPCClient, n NotificationProvider, discoveryTopic string, pollInterval time.Duration) (*Messenger, error) {
	password := "password"

	client := sdk.New(rpc)
	ac, err := client.SignupAndLogin(password)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	return &Messenger{
		notifier:       n,
		password:       password,
		baseAccount:    ac,
		pollInterval:   pollInterval,
		discoveryTopic: discoveryTopic,
		client:         client,
		tokenDB:        make(map[string]string, 0),
	}, nil
}

// BroadcastAvailability : Broadcasts its availability to serve as
// notification server
func (m *Messenger) BroadcastAvailability() error {
	// TODO (pombeirp): Use a different method so that an asym key is exchanged, not a sym key
	ch, err := m.baseAccount.JoinPublicChannel(m.discoveryTopic)
	if err != nil {
		return err
	}

	go func() {
		for range time.Tick(m.pollInterval) {
			// TODO (pombeirp): Listen to channel to determine when is time to exit
			log.Println("Broadcasting availability on", m.discoveryTopic)
			_ = ch.PNBroadcastAvailabilityRequest()
		}
	}()

	return nil
}

// ManageRegistrations clients will be sending registration requests to the
// messenger topic, this method retrieves those messages and stores its
// information to allow push notifications
func (m *Messenger) ManageRegistrations() error {
	log.Println("Subscribed to discovery topic :", m.discoveryTopic)
	ch, err := m.baseAccount.JoinPublicChannel(m.discoveryTopic)
	if err != nil {
		log.Println("Can't manage registrations")
		log.Println(err.Error())
		return err
	}

	_, err = ch.Subscribe(m.processRegistration)

	return err
}

// processRegistration : processes an input string to get the underlying
// RegistrationRequestMsg and stores the result
func (m *Messenger) processRegistration(msg *sdk.Msg) {
	req := msg.Properties.(sdk.PNRegistrationMsg)

	// Generate a new asymetric key (AK2)
	key := make([]byte, 64)
	_, err := rand.Read(key)
	if err != nil {
		log.Println("Error generating random key")
	}
	ak2 := b64.StdEncoding.EncodeToString(key)

	// Store locally the AK2 / device token key pair
	m.tokenDB[ak2] = req.DeviceToken

	// Subscribe to proposed topic with given symkey
	log.Println("Subscribed to secure channel", req.Topic)
	ch, err := m.baseAccount.Join("isThisNameNeededAtAll?", req.Topic, req.Symkey)
	if err != nil {
		log.Println(err.Error())
		return
	}

	// Send a registration confirmation with the new public key
	err = ch.PNRegistrationConfirmationRequest(ak2)
	if err == nil {
		_, _ = ch.Subscribe(m.manageNotificationRequests)
	}
}

func (m *Messenger) manageNotificationRequests(msg *sdk.Msg) {
	// Get persisted device token based on the provided AK2
	token, ok := m.tokenDB[msg.Channel.PubKey()]
	if !ok {
		log.Println("Could not retrieve existing token for this channel")
	}

	// Get the text from the input request
	properties := msg.Properties.(sdk.PublishMsg)

	// Send notification to notifier
	if err := m.notifier.Send([]string{token}, properties.Text); err != nil {
		log.Println("Error notifying : " + err.Error())
	}
}
