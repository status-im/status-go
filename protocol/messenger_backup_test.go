package protocol

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
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

	_, err = bob1.AddContact(context.Background(), &requests.AddContact{ID: contactID1})
	s.Require().NoError(err)

	contact2Key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	contactID2 := types.EncodeHex(crypto.FromECDSAPub(&contact2Key.PublicKey))

	_, err = bob1.AddContact(context.Background(), &requests.AddContact{ID: contactID2})
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

	s.Require().Equal(ContactRequestStateSent, actualContacts[0].ContactRequestLocalState)
	s.Require().Equal(ContactRequestStateSent, actualContacts[1].ContactRequestLocalState)
	s.Require().True(actualContacts[0].added())
	s.Require().True(actualContacts[1].added())

	// Backup

	clock, err := bob1.BackupData(context.Background())
	s.Require().NoError(err)

	// Safety check
	s.Require().Len(bob2.Contacts(), 0)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		bob2,
		func(r *MessengerResponse) bool {
			return r.BackupHandled
		},
		"no messages",
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
	s.Require().Equal(ContactRequestStateSent, actualContacts[0].ContactRequestLocalState)
	s.Require().Equal(ContactRequestStateSent, actualContacts[1].ContactRequestLocalState)
	lastBackup, err := bob1.lastBackup()
	s.Require().NoError(err)
	s.Require().NotEmpty(lastBackup)
	s.Require().Equal(clock, lastBackup)
}

func (s *MessengerBackupSuite) TestBackupProfile() {
	const bob1DisplayName = "bobby"

	// Create bob1
	bob1 := s.m
	err := bob1.SetDisplayName(bob1DisplayName)
	s.Require().NoError(err)
	bob1KeyUID := bob1.account.KeyUID
	imagesExpected := fmt.Sprintf(`[{"keyUid":"%s","type":"large","uri":"data:image/png;base64,iVBORw0KGgoAAAANSUg=","width":240,"height":300,"fileSize":1024,"resizeTarget":240,"clock":0},{"keyUid":"%s","type":"thumbnail","uri":"data:image/jpeg;base64,/9j/2wCEAFA3PEY8MlA=","width":80,"height":80,"fileSize":256,"resizeTarget":80,"clock":0}]`,
		bob1KeyUID, bob1KeyUID)

	iis := images.SampleIdentityImages()
	s.Require().NoError(bob1.multiAccounts.StoreIdentityImages(bob1KeyUID, iis, false))

	// Create bob2
	bob2, err := newMessengerWithKey(s.shh, bob1.identity, s.logger, nil)
	s.Require().NoError(err)
	_, err = bob2.Start()
	s.Require().NoError(err)

	// Check bob1
	storedBob1DisplayName, err := bob1.settings.DisplayName()
	s.Require().NoError(err)
	s.Require().Equal(bob1DisplayName, storedBob1DisplayName)

	bob1Images, err := bob1.multiAccounts.GetIdentityImages(bob1KeyUID)
	s.Require().NoError(err)
	jBob1Images, err := json.Marshal(bob1Images)
	s.Require().NoError(err)
	s.Require().Equal(imagesExpected, string(jBob1Images))

	// Check bob2
	storedBob2DisplayName, err := bob2.settings.DisplayName()
	s.Require().NoError(err)
	s.Require().Equal("", storedBob2DisplayName)

	var expectedEmpty []*images.IdentityImage
	bob2Images, err := bob2.multiAccounts.GetIdentityImages(bob1KeyUID)
	s.Require().NoError(err)
	s.Require().Equal(expectedEmpty, bob2Images)

	// Backup
	clock, err := bob1.BackupData(context.Background())
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		bob2,
		func(r *MessengerResponse) bool {
			return r.BackupHandled
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Check bob2
	storedBob2DisplayName, err = bob2.settings.DisplayName()
	s.Require().NoError(err)
	s.Require().Equal(bob1DisplayName, storedBob2DisplayName)

	bob2Images, err = bob2.multiAccounts.GetIdentityImages(bob1KeyUID)
	s.Require().NoError(err)
	s.Require().Equal(2, len(bob2Images))
	s.Require().Equal(bob2Images[0].Payload, bob1Images[0].Payload)
	s.Require().Equal(bob2Images[1].Payload, bob1Images[1].Payload)

	lastBackup, err := bob1.lastBackup()
	s.Require().NoError(err)
	s.Require().NotEmpty(lastBackup)
	s.Require().Equal(clock, lastBackup)
}

func (s *MessengerBackupSuite) TestBackupSettings() {
	const (
		bob1DisplayName               = "bobby"
		bob1Currency                  = "eur"
		bob1MessagesFromContactsOnly  = true
		bob1ProfilePicturesShowTo     = settings.ProfilePicturesShowToEveryone
		bob1ProfilePicturesVisibility = settings.ProfilePicturesVisibilityEveryone
		bob1Bio                       = "bio"
	)

	// Create bob1 and set fields which are supposed to be backed up to/fetched from waku
	bob1 := s.m
	err := bob1.settings.SaveSettingField(settings.DisplayName, bob1DisplayName)
	s.Require().NoError(err)
	err = bob1.settings.SaveSettingField(settings.Currency, bob1Currency)
	s.Require().NoError(err)
	err = bob1.settings.SaveSettingField(settings.MessagesFromContactsOnly, bob1MessagesFromContactsOnly)
	s.Require().NoError(err)
	err = bob1.settings.SaveSettingField(settings.ProfilePicturesShowTo, bob1ProfilePicturesShowTo)
	s.Require().NoError(err)
	err = bob1.settings.SaveSettingField(settings.ProfilePicturesVisibility, bob1ProfilePicturesVisibility)
	s.Require().NoError(err)
	err = bob1.settings.SaveSettingField(settings.Bio, bob1Bio)
	s.Require().NoError(err)

	// Create bob2
	bob2, err := newMessengerWithKey(s.shh, bob1.identity, s.logger, nil)
	s.Require().NoError(err)
	_, err = bob2.Start()
	s.Require().NoError(err)

	// Check bob1
	storedBob1DisplayName, err := bob1.settings.DisplayName()
	s.Require().NoError(err)
	s.Require().Equal(bob1DisplayName, storedBob1DisplayName)
	storedBob1Currency, err := bob1.settings.GetCurrency()
	s.Require().NoError(err)
	s.Require().Equal(bob1Currency, storedBob1Currency)
	storedBob1MessagesFromContactsOnly, err := bob1.settings.GetMessagesFromContactsOnly()
	s.Require().NoError(err)
	s.Require().Equal(bob1MessagesFromContactsOnly, storedBob1MessagesFromContactsOnly)
	storedBob1ProfilePicturesShowTo, err := bob1.settings.GetProfilePicturesShowTo()
	s.Require().NoError(err)
	s.Require().Equal(int64(bob1ProfilePicturesShowTo), storedBob1ProfilePicturesShowTo)
	storedBob1ProfilePicturesVisibility, err := bob1.settings.GetProfilePicturesVisibility()
	s.Require().NoError(err)
	s.Require().Equal(int(bob1ProfilePicturesVisibility), storedBob1ProfilePicturesVisibility)
	storedBob1Bio, err := bob1.settings.Bio()
	s.Require().NoError(err)
	s.Require().Equal(bob1Bio, storedBob1Bio)

	// Check bob2
	storedBob2DisplayName, err := bob2.settings.DisplayName()
	s.Require().NoError(err)
	s.Require().NotEqual(storedBob1DisplayName, storedBob2DisplayName)
	storedBob2Currency, err := bob2.settings.GetCurrency()
	s.Require().NoError(err)
	s.Require().NotEqual(storedBob1Currency, storedBob2Currency)
	storedBob2MessagesFromContactsOnly, err := bob2.settings.GetMessagesFromContactsOnly()
	s.Require().NoError(err)
	s.Require().NotEqual(storedBob1MessagesFromContactsOnly, storedBob2MessagesFromContactsOnly)
	storedBob2ProfilePicturesShowTo, err := bob2.settings.GetProfilePicturesShowTo()
	s.Require().NoError(err)
	s.Require().NotEqual(storedBob1ProfilePicturesShowTo, storedBob2ProfilePicturesShowTo)
	storedBob2ProfilePicturesVisibility, err := bob2.settings.GetProfilePicturesVisibility()
	s.Require().NoError(err)
	s.Require().NotEqual(storedBob1ProfilePicturesVisibility, storedBob2ProfilePicturesVisibility)
	storedBob2Bio, err := bob2.settings.Bio()
	s.Require().NoError(err)
	s.Require().NotEqual(storedBob1Bio, storedBob2Bio)

	// Backup
	clock, err := bob1.BackupData(context.Background())
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		bob2,
		func(r *MessengerResponse) bool {
			return r.BackupHandled
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Check bob2
	storedBob2DisplayName, err = bob2.settings.DisplayName()
	s.Require().NoError(err)
	s.Require().Equal(storedBob1DisplayName, storedBob2DisplayName)
	storedBob2Currency, err = bob2.settings.GetCurrency()
	s.Require().NoError(err)
	s.Require().Equal(storedBob1Currency, storedBob2Currency)
	storedBob2MessagesFromContactsOnly, err = bob2.settings.GetMessagesFromContactsOnly()
	s.Require().NoError(err)
	s.Require().Equal(storedBob1MessagesFromContactsOnly, storedBob2MessagesFromContactsOnly)
	storedBob2ProfilePicturesShowTo, err = bob2.settings.GetProfilePicturesShowTo()
	s.Require().NoError(err)
	s.Require().Equal(storedBob1ProfilePicturesShowTo, storedBob2ProfilePicturesShowTo)
	storedBob2ProfilePicturesVisibility, err = bob2.settings.GetProfilePicturesVisibility()
	s.Require().NoError(err)
	s.Require().Equal(storedBob1ProfilePicturesVisibility, storedBob2ProfilePicturesVisibility)
	storedBob2Bio, err = bob2.settings.Bio()
	s.Require().NoError(err)
	s.Require().Equal(storedBob1Bio, storedBob2Bio)

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

	// Create contacts

	for i := 0; i < BackupContactsPerBatch*2; i++ {

		contactKey, err := crypto.GenerateKey()
		s.Require().NoError(err)
		contactID := types.EncodeHex(crypto.FromECDSAPub(&contactKey.PublicKey))

		_, err = bob1.AddContact(context.Background(), &requests.AddContact{ID: contactID})
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
			return r.BackupHandled
		},
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().Less(BackupContactsPerBatch*2, len(bob2.Contacts()))
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

	_, err = bob1.AddContact(context.Background(), &requests.AddContact{ID: contactID1})
	s.Require().NoError(err)

	contact2Key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	contactID2 := types.EncodeHex(crypto.FromECDSAPub(&contact2Key.PublicKey))

	_, err = bob1.AddContact(context.Background(), &requests.AddContact{ID: contactID2})
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

	_, err = bob2.AddContact(context.Background(), &requests.AddContact{ID: contactID2})
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
			return r.BackupHandled
		},
		"no messages",
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

func (s *MessengerBackupSuite) TestBackupLocalNickname() {
	bob1 := s.m
	// Create bob2
	bob2, err := newMessengerWithKey(s.shh, bob1.identity, s.logger, nil)
	nickname := "don van vliet"
	s.Require().NoError(err)
	_, err = bob2.Start()
	s.Require().NoError(err)

	// Set contact nickname

	contact1Key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	contactID1 := types.EncodeHex(crypto.FromECDSAPub(&contact1Key.PublicKey))

	_, err = bob1.SetContactLocalNickname(&requests.SetContactLocalNickname{ID: types.Hex2Bytes(contactID1), Nickname: nickname})
	s.Require().NoError(err)

	// Backup

	clock, err := bob1.BackupData(context.Background())
	s.Require().NoError(err)

	// Safety check
	s.Require().Len(bob2.Contacts(), 0)

	var actualContact *Contact
	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		bob2,
		func(r *MessengerResponse) bool {
			return r.BackupHandled
		},
		"no messages",
	)
	s.Require().NoError(err)

	for _, c := range bob2.Contacts() {
		if c.ID == contactID1 {
			actualContact = c
			break
		}
	}

	s.Require().Equal(actualContact.LocalNickname, nickname)
	lastBackup, err := bob1.lastBackup()
	s.Require().NoError(err)
	s.Require().NotEmpty(lastBackup)
	s.Require().Equal(clock, lastBackup)
}

func (s *MessengerBackupSuite) TestBackupBlockedContacts() {
	bob1 := s.m
	// Create bob2
	bob2, err := newMessengerWithKey(s.shh, bob1.identity, s.logger, nil)
	s.Require().NoError(err)
	_, err = bob2.Start()
	s.Require().NoError(err)

	// Create contact
	contact1Key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	contactID1 := types.EncodeHex(crypto.FromECDSAPub(&contact1Key.PublicKey))

	_, err = bob1.AddContact(context.Background(), &requests.AddContact{ID: contactID1})
	s.Require().NoError(err)

	// Backup
	_, err = bob1.BackupData(context.Background())
	s.Require().NoError(err)

	// Safety check
	s.Require().Len(bob2.Contacts(), 0)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		bob2,
		func(r *MessengerResponse) bool {
			return r.BackupHandled
		},
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().Len(bob2.AddedContacts(), 1)

	actualContacts := bob2.AddedContacts()
	s.Require().Equal(actualContacts[0].ID, contactID1)

	_, err = bob1.BlockContact(contactID1)
	s.Require().NoError(err)

	// Backup
	_, err = bob1.BackupData(context.Background())
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(
		bob2,
		func(r *MessengerResponse) bool {
			return r.BackupHandled
		},
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(bob2.BlockedContacts(), 1)
}

func (s *MessengerBackupSuite) TestBackupCommunities() {
	bob1 := s.m
	// Create bob2
	bob2, err := newMessengerWithKey(s.shh, bob1.identity, s.logger, nil)
	s.Require().NoError(err)
	_, err = bob2.Start()
	s.Require().NoError(err)

	// Create a communitie

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create a community chat
	response, err := bob1.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	// Backup
	clock, err := bob1.BackupData(context.Background())
	s.Require().NoError(err)

	// Safety check
	communities, err := bob2.Communities()
	s.Require().NoError(err)
	s.Require().Len(communities, 1)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		bob2,
		func(r *MessengerResponse) bool {
			return r.BackupHandled
		},
		"no messages",
	)

	s.Require().NoError(err)

	communities, err = bob2.JoinedCommunities()
	s.Require().NoError(err)
	s.Require().Len(communities, 1)

	lastBackup, err := bob1.lastBackup()
	s.Require().NoError(err)
	s.Require().NotEmpty(lastBackup)
	s.Require().Equal(clock, lastBackup)
}

func (s *MessengerBackupSuite) TestBackupKeycards() {
	// Create bob1
	bob1 := s.m
	allKeycardsToSync := getKeycardsForTest()
	for _, kp := range allKeycardsToSync {
		addedKc, addedAccs, err := bob1.settings.AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(*kp)
		s.Require().NoError(err)
		s.Require().Equal(true, addedKc)
		s.Require().Equal(false, addedAccs)
	}

	// Create bob2
	bob2, err := newMessengerWithKey(s.shh, bob1.identity, s.logger, nil)
	s.Require().NoError(err)
	_, err = bob2.Start()
	s.Require().NoError(err)

	// Backup
	_, err = bob1.BackupData(context.Background())
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		bob2,
		func(r *MessengerResponse) bool {
			return r.BackupHandled
		},
		"no messages",
	)
	s.Require().NoError(err)

	syncedKeycards, err := bob2.settings.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(len(allKeycardsToSync), len(syncedKeycards))
	s.Require().True(haveSameElements(syncedKeycards, allKeycardsToSync, sameKeycards))
}
