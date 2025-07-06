package protocol

import (
	"crypto/ecdsa"

	"github.com/stretchr/testify/suite"

	wakutypes "github.com/status-im/status-go/waku/types"
)

const DefaultProfileDisplayName = ""

func (s *MessengerBaseTestSuite) setupWaku() {
	shh, err := newTestWakuNode()
	s.Require().NoError(err)
	s.Require().NoError(shh.Start())
	s.shh = shh
}

func (s *MessengerBaseTestSuite) SetupTest() {
	s.setupWaku()

	s.m = s.newMessenger()
	s.privateKey = s.m.identity
}

func (s *MessengerBaseTestSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.m)
	if s.shh != nil {
		s.Require().NoError(s.shh.Stop())
	}
}

func (s *MessengerBaseTestSuite) newMessenger() *Messenger {
	messenger, err := newRunningTestMessenger(s.shh, testMessengerConfig{})
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerBaseTestSuite) anotherMessenger() *Messenger {
	messenger, err := newRunningTestMessenger(s.shh, testMessengerConfig{privateKey: s.privateKey})
	s.Require().NoError(err)
	return messenger
}

type MessengerBaseTestSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single waku service should be shared.
	shh wakutypes.Waku
}
