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

func TestMessengerBackupSuite(t *testing.T) {
	suite.Run(t, new(MessengerBackupSuite))
}

type MessengerBackupSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerBackupSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.m = s.newMessenger()
	s.privateKey = s.m.identity
	_, err := s.m.Start()
	s.Require().NoError(err)
}

func (s *MessengerBackupSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *MessengerBackupSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerBackupSuite) TestBackupContacts() {
	bob1 := s.m
	// Create bob2
	bob2, err := newMessengerWithKey(s.shh, bob1.identity, s.logger, nil)
	s.Require().NoError(err)
	_, err = bob2.Start()
	s.Require().NoError(err)

	// Create 2 contacts

	contact1Key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	contactID1 := types.EncodeHex(crypto.FromECDSAPub(&contact1Key.PublicKey))

	_, err = bob1.AddContact(context.Background(), contactID1)
	s.Require().NoError(err)

	contact2Key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	contactID2 := types.EncodeHex(crypto.FromECDSAPub(&contact2Key.PublicKey))

	_, err = bob1.AddContact(context.Background(), contactID2)
	s.Require().NoError(err)

	s.Require().Len(bob1.Contacts(), 2)

	actualContacts := bob1.Contacts()
	if actualContacts[0].ID == contactID1 {
		s.Require().Equal(actualContacts[0].ID, contactID1)
		s.Require().Equal(actualContacts[1].ID, contactID2)
	} else {
		s.Require().Equal(actualContacts[0].ID, contactID2)
		s.Require().Equal(actualContacts[1].ID, contactID1)
	}

	// Backup

	clock, err := bob1.BackupData(context.Background())
	s.Require().NoError(err)

	// Safety check
	s.Require().Len(bob2.Contacts(), 0)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		bob2,
		func(r *MessengerResponse) bool {
			_, err := s.m.RetrieveAll()
			if err != nil {
				s.logger.Info("Failed")
				return false
			}

			if len(bob2.Contacts()) < 2 {
				return false
			}

			return true

		},
		"contacts not backed up",
	)
	s.Require().NoError(err)
	s.Require().Len(bob2.AddedContacts(), 2)

	actualContacts = bob2.AddedContacts()
	if actualContacts[0].ID == contactID1 {
		s.Require().Equal(actualContacts[0].ID, contactID1)
		s.Require().Equal(actualContacts[1].ID, contactID2)
	} else {
		s.Require().Equal(actualContacts[0].ID, contactID2)
		s.Require().Equal(actualContacts[1].ID, contactID1)
	}
	lastBackup, err := bob1.lastBackup()
	s.Require().NoError(err)
	s.Require().NotEmpty(lastBackup)
	s.Require().Equal(clock, lastBackup)
}

func (s *MessengerBackupSuite) TestBackupContactsGreaterThanBatch() {
	bob1 := s.m
	// Create bob2
	bob2, err := newMessengerWithKey(s.shh, bob1.identity, s.logger, nil)
	s.Require().NoError(err)
	_, err = bob2.Start()
	s.Require().NoError(err)

	// Create 2 contacts

	for i := 0; i < BackupContactsPerBatch*2; i++ {

		contactKey, err := crypto.GenerateKey()
		s.Require().NoError(err)
		contactID := types.EncodeHex(crypto.FromECDSAPub(&contactKey.PublicKey))

		_, err = bob1.AddContact(context.Background(), contactID)
		s.Require().NoError(err)

	}
	// Backup

	_, err = bob1.BackupData(context.Background())
	s.Require().NoError(err)

	// Safety check
	s.Require().Len(bob2.Contacts(), 0)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		bob2,
		func(r *MessengerResponse) bool {
			_, err := s.m.RetrieveAll()
			if err != nil {
				s.logger.Info("Failed")
				return false
			}

			if len(bob2.Contacts()) < BackupContactsPerBatch*2 {
				return false
			}

			return true

		},
		"contacts not backed up",
	)
	s.Require().NoError(err)
	s.Require().Len(bob2.AddedContacts(), BackupContactsPerBatch*2)
}

func (s *MessengerBackupSuite) TestBackupRemovedContact() {
	bob1 := s.m
	// Create bob2
	bob2, err := newMessengerWithKey(s.shh, bob1.identity, s.logger, nil)
	s.Require().NoError(err)
	_, err = bob2.Start()
	s.Require().NoError(err)

	// Create 2 contacts on bob 1

	contact1Key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	contactID1 := types.EncodeHex(crypto.FromECDSAPub(&contact1Key.PublicKey))

	_, err = bob1.AddContact(context.Background(), contactID1)
	s.Require().NoError(err)

	contact2Key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	contactID2 := types.EncodeHex(crypto.FromECDSAPub(&contact2Key.PublicKey))

	_, err = bob1.AddContact(context.Background(), contactID2)
	s.Require().NoError(err)

	s.Require().Len(bob1.Contacts(), 2)

	actualContacts := bob1.Contacts()
	if actualContacts[0].ID == contactID1 {
		s.Require().Equal(actualContacts[0].ID, contactID1)
		s.Require().Equal(actualContacts[1].ID, contactID2)
	} else {
		s.Require().Equal(actualContacts[0].ID, contactID2)
		s.Require().Equal(actualContacts[1].ID, contactID1)
	}

	// Bob 2 add one of the same contacts

	_, err = bob2.AddContact(context.Background(), contactID2)
	s.Require().NoError(err)

	// Bob 1 now removes one of the contact that was also on bob 2

	_, err = bob1.RemoveContact(context.Background(), contactID2)
	s.Require().NoError(err)

	// Backup

	clock, err := bob1.BackupData(context.Background())
	s.Require().NoError(err)

	// Safety check
	s.Require().Len(bob2.Contacts(), 1)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		bob2,
		func(r *MessengerResponse) bool {
			_, err := s.m.RetrieveAll()
			if err != nil {
				s.logger.Info("Failed")
				return false
			}

			if len(bob2.Contacts()) != 1 && bob2.Contacts()[0].ID != contactID1 {
				return false
			}

			return true

		},
		"contacts not backed up",
	)
	// Bob 2 should remove the contact
	s.Require().NoError(err)
	s.Require().Len(bob2.AddedContacts(), 1)
	s.Require().Equal(contactID1, bob2.AddedContacts()[0].ID)

	lastBackup, err := bob1.lastBackup()
	s.Require().NoError(err)
	s.Require().NotEmpty(lastBackup)
	s.Require().Equal(clock, lastBackup)
}
