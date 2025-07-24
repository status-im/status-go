package protocol

import (
	"crypto/ecdsa"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/messaging"
)

const DefaultProfileDisplayName = ""

func (s *MessengerBaseTestSuite) setupMessaging() {
	var err error
	s.messagingEnv, err = messaging.NewTestMessagingEnvironment()
	s.Require().NoError(err)
}

func (s *MessengerBaseTestSuite) SetupTest() {
	s.setupMessaging()

	s.m = s.newMessenger()
	s.privateKey = s.m.identity
}

func (s *MessengerBaseTestSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.m)
	if s.messagingEnv != nil {
		s.Require().NoError(s.messagingEnv.TearDown())
	}
}

func (s *MessengerBaseTestSuite) newMessenger() *Messenger {
	messenger, err := newRunningTestMessenger(s.messagingEnv, testMessengerConfig{})
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerBaseTestSuite) anotherMessenger() *Messenger {
	messenger, err := newRunningTestMessenger(s.messagingEnv, testMessengerConfig{privateKey: s.privateKey})
	s.Require().NoError(err)

	return messenger
}

type MessengerBaseTestSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger

	messagingEnv *messaging.TestMessagingEnvironment
}
