package notifier

import (
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
}

// NotificationRequestMsg : a registered contact requests sending a push
// notification to one of its contacts
type NotificationRequestMsg struct {
	// TODO (adriacidre) : Check @PombeirP what fields are needed here
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
	address, _, _, err := client.SignupAndLogin(password)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	return &Messenger{
		notifier:       n,
		password:       password,
		addressKey:     address,
		pollInterval:   pollInterval,
		discoveryTopic: discoveryTopic,
		client:         client,
	}, nil
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
	ch, err := m.client.JoinPublicChannel(m.discoveryTopic)
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
	// TODO (adriacidre) generate a new asymetric key (AK2)
	pubkey := "..."
	// TODO (adriacidre) store locally the AK2 / device token key pair
	// TODO (adriacidre) subscribe to proposed topic with given symkey
	log.Println("Subscribed to secure channel", req.Topic)
	ch, err := m.client.Join("isThisNameNeededAtAll?", req.Topic, req.Symkey)
	if err != nil {
		log.Println(err.Error())
		return
	}
	// TODO (adriacidre) send a registration confirmation with a new public key
	err = ch.PNRegistrationConfirmationRequest(pubkey)
	if err == nil {
		_, _ = ch.Subscribe(m.manageNotificationRequests)
	}
}

func (m *Messenger) manageNotificationRequests(msg *sdk.Msg) {
	// TODO (adriacidre) get device token based on the provided AK2
	tokens := []string{"FAKE_TOKEN"}
	// TODO (adriacidre) get the text from the input request
	text := "Ping!"

	if err := m.notifier.Send(tokens, text); err != nil {
		log.Println("Error notifying over a secure channel : " + err.Error())
	}
}
