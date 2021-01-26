package protocol

import (
	"context"
	"crypto/ecdsa"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerContactUpdateSuite(t *testing.T) {
	suite.Run(t, new(MessengerContactUpdateSuite))
}

type MessengerContactUpdateSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerContactUpdateSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start(nil))

	s.m = s.newMessenger(s.shh)
	s.privateKey = s.m.identity
	_, err := s.m.Start()
	s.Require().NoError(err)
}

func (s *MessengerContactUpdateSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *MessengerContactUpdateSuite) newMessenger(shh types.Waku) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerContactUpdateSuite) TestReceiveContactUpdate() {
	theirName := "ens-name.stateofus.eth"
	theirPicture := "their-picture"
	contactID := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	theirContactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))

	response, err := theirMessenger.SendContactUpdate(context.Background(), contactID, theirName, theirPicture)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	s.Require().Len(response.Contacts, 1)
	contact := response.Contacts[0]
	// It should not add the contact, as that's left to `SaveContact`
	s.Require().False(contact.IsAdded())

	// add contact
	contact.SystemTags = []string{contactAdded}
	s.Require().NoError(theirMessenger.SaveContact(contact))

	s.Require().Len(response.Chats, 1)
	chat := response.Chats[0]
	s.Require().False(chat.Active, "It does not create an active chat")

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Contacts) > 0 },
		"contact request not received",
	)
	s.Require().NoError(err)

	receivedContact := response.Contacts[0]
	s.Require().Equal(theirName, receivedContact.Name)
	s.Require().False(receivedContact.ENSVerified)
	s.Require().True(receivedContact.HasBeenAdded())
	s.Require().NotEmpty(receivedContact.LastUpdated)

	newPicture := "new-picture"
	err = theirMessenger.SendContactUpdates(context.Background(), newName, newPicture)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && response.Contacts[0].ID == theirContactID
		},
		"contact request not received",
	)

	s.Require().NoError(err)

	receivedContact = response.Contacts[0]
	s.Require().Equal(theirContactID, receivedContact.ID)
	s.Require().Equal(newName, receivedContact.Name)
	s.Require().False(receivedContact.ENSVerified)
	s.Require().True(receivedContact.HasBeenAdded())
	s.Require().NotEmpty(receivedContact.LastUpdated)
	s.Require().NoError(theirMessenger.Shutdown())
}

func (s *MessengerContactUpdateSuite) TestAddContact() {
	contactID := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))

	theirMessenger := s.newMessenger(s.shh)
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	response, err := theirMessenger.AddContact(context.Background(), contactID)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	s.Require().Len(response.Contacts, 1)
	contact := response.Contacts[0]

	// It adds the profile chat and the one to one chat
	s.Require().Len(response.Chats, 2)

	// It should add the contact
	s.Require().True(contact.IsAdded())

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Contacts) > 0 },
		"contact request not received",
	)
	s.Require().NoError(err)

	receivedContact := response.Contacts[0]
	s.Require().True(receivedContact.HasBeenAdded())
	s.Require().NotEmpty(receivedContact.LastUpdated)
}
