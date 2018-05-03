package notifier

import (
	"encoding/json"
	"log"
	"time"

	sdk "github.com/status-im/status-go-sdk"
	"github.com/status-im/status-go/geth/node"
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
	client         *sdk.SDK
}

// NotificationRequestMsg : a registered contact requests sending a push
// notification to one of its contacts
type NotificationRequestMsg struct {
	// TODO (adriacidre) : Check @PombeirP what fields are needed here
}

// NewMessenger Creates a new Messenger
func NewMessenger(sn *node.StatusNode, discoveryTopic string, pollInterval time.Duration) *Messenger {
	password := "password"
	client := sdk.New("")
	address, _, _, err := client.SignupAndLogin(password)
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	return &Messenger{
		password:       password,
		addressKey:     address,
		pollInterval:   pollInterval,
		discoveryTopic: discoveryTopic,
		client:         client,
	}
}

// BroadcastAvailability : Broadcasts its availability to serve as
// notification server
func (m *Messenger) BroadcastAvailability() error {
	// TODO (pombeirp): Use a different method so that an asym key is exchanged, not a sym key
	ch, err := m.client.JoinPublicChannel(m.discoveryTopic)
	if err != nil {
		return err
	}

	go func() {
		for range time.Tick(m.pollInterval) {
			// TODO (pombeirp): Listen to channel to determine when is time to exit
			log.Println("Broadcasting availability on", m.discoveryTopic)
			ch.NewContactKeyRequest(m.addressKey)
		}
	}()

	return nil
}

// ManageRegistrations clients will be sending registration requests to the
// messenger topic, this method retrieves those messages and stores its
// information to allow push notifications
func (m *Messenger) ManageRegistrations() {
	log.Println("Subscribed to discovery topic :", m.discoveryTopic)
	ch, err := m.client.JoinPublicChannel(m.discoveryTopic)
	if err != nil {
		log.Println("Can't manage registrations")
		log.Println(err.Error())
		return
	}

	ch.Subscribe(func(msg *sdk.Msg) {
		if err := m.processRegistration(msg.Text); err != nil {
			log.Println("Error processing registration : " + err.Error())
		}
	})
}

// processRegistration : processes an input string to get the underlying
// RegistrationRequestMsg and stores the result
func (m *Messenger) processRegistration(r string) error {
	log.Println("Processing registration")
	var req RegistrationRequestMsg

	if err := json.Unmarshal([]byte(r), &req); err != nil {
		return err
	}
	go m.subscribeSecureChannel(req)

	return nil
}

func (m *Messenger) subscribeSecureChannel(registration RegistrationRequestMsg) {
	log.Println("Subscribed to secure channel", registration.SecureChannel)
	// TODO (adriacidre) : this is likely to not work as its not a public channel
	ch, err := m.client.JoinPublicChannel(m.discoveryTopic)
	if err != nil {
		log.Println("Can't manage registrations")
		log.Println(err.Error())
		return
	}

	ch.Subscribe(func(msg *sdk.Msg) {
		if err := m.notify(registration.DeviceRegistrationToken, msg.Text); err != nil {
			log.Println("Error notifying over a secure channel : " + err.Error())
		}
	})
}

func (m *Messenger) notify(deviceToken, data string) error {
	log.Println("Processing notification")
	// TODO (adriacidre) we should link here development from @PombeirP
	var req NotificationRequestMsg

	return json.Unmarshal([]byte(data), &req)
}
