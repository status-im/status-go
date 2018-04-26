package main

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/messenger"
)

const (
	// defaultWorkTime is a work time reported in messages sent to MailServer nodes.
	defaultWorkTime = 5
)

var (
	// ErrInvalidSymKeyID is returned when it fails to get a symmetric key.
	ErrInvalidSymKeyID = errors.New("invalid symKeyID value")
)

type BroadcastMsg struct {
	PubKey string `json:"pubkey"`
}

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
	symKey         []byte
	discoveryTopic string
	pollInterval   time.Duration
	addressKey     string
	password       string
	client         *messenger.Messenger
}

// NotificationRequestMsg : a registered contact requests sending a push
// notification to one of its contacts
type NotificationRequestMsg struct {
	// TODO (adriacidre) : Check @PombeirP what fields are needed here
}

// NewMessenger Creates a new Messenger
func NewMessenger(sn *node.StatusNode, discoveryTopic string, pollInterval time.Duration) *Messenger {
	password := "password"
	client := messenger.New(sn)
	addr := client.Signup(password)
	log.Println("ADDRESS :", addr)
	addressKey, err := client.Login(addr, password)
	log.Println("ADDRESS KEY :", addr)
	if err != nil {
		log.Println("An error logging in")
	}

	return &Messenger{
		password:       password,
		addressKey:     addressKey,
		pollInterval:   pollInterval,
		discoveryTopic: discoveryTopic,
		client:         client,
	}
}

// BroadcastAvailability : Broadcasts its availability to serve as
// notification server
func (m *Messenger) BroadcastAvailability() error {
	_, _, channelKey := m.client.JoinPublicChannel(m.discoveryTopic)

	go func() {
		for range time.Tick(m.pollInterval) {
			log.Println("Broadcasting availability on", m.discoveryTopic)
			m.client.NewContactKeyRequest(m.addressKey, m.discoveryTopic, channelKey, "me")
		}
	}()

	return nil
}

// ManageRegistrations clients will be sending registration requests to the
// messenger topic, this method retrieves those messages and stores its
// information to allow push notifications
func (m *Messenger) ManageRegistrations() {
	log.Println("Subscribed to discovery topic :", m.discoveryTopic)
	filterID, _, _ := m.client.JoinPublicChannel(m.discoveryTopic)
	m.client.Subscribe(filterID, func(msg *messenger.Msg) {
		m.processRegistration(msg.Text)
	})
}

// processRegistration : processes an input string to get the underlying
// RegistrationRequestMsg and stores the result
func (m *Messenger) processRegistration(r string) error {
	log.Println("Processing registration")
	var req RegistrationRequestMsg

	if err := json.Unmarshal([]byte(r), req); err != nil {
		return err
	}
	go m.subscribeSecureChannel(req)

	return nil
}

func (m *Messenger) subscribeSecureChannel(registration RegistrationRequestMsg) {
	log.Println("Subscribed to secure channel", registration.SecureChannel)
	// TODO (adriacidre) : this is likely to not work as its not a public channel
	filterID, _, _ := m.client.JoinPublicChannel(m.discoveryTopic)
	m.client.Subscribe(filterID, func(msg *messenger.Msg) {
		m.notify(registration.DeviceRegistrationToken, msg.Text)
	})
}

func (m *Messenger) notify(deviceToken, data string) error {
	log.Println("Processing notification")
	// TODO (adriacidre) we should link here development from @PombeirP
	var req NotificationRequestMsg
	if err := json.Unmarshal([]byte(data), req); err != nil {
		return err
	}

	return nil
}
