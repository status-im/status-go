package protocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	mock_protocol_accounts_manager "github.com/status-im/status-go/protocol/mock"
)

func TestMessengerSyncKeycardsStateSuite(t *testing.T) {
	suite.Run(t, new(MessengerSyncKeycardsStateSuite))
}

type MessengerSyncKeycardsStateSuite struct {
	MessengerBaseTestSuite
	main                     *Messenger // main instance of Messenger paired with `other`
	accountsManagerMock      *mock_protocol_accounts_manager.MockAccountsManager
	other                    *Messenger
	accountsManagerOtherMock *mock_protocol_accounts_manager.MockAccountsManager
}

func (s *MessengerSyncKeycardsStateSuite) SetupTest() {
	s.MessengerBaseTestSuite.SetupTest()

	ctrl := gomock.NewController(s.T())

	s.main = s.m
	s.accountsManagerMock = mock_protocol_accounts_manager.NewMockAccountsManager(ctrl)
	s.m.accountsManager = s.accountsManagerMock

	s.other = s.anotherMessenger()
	s.accountsManagerOtherMock = mock_protocol_accounts_manager.NewMockAccountsManager(ctrl)
	s.other.accountsManager = s.accountsManagerOtherMock

	// Pair devices (main and other)
	imOther := &multidevice.InstallationMetadata{
		Name:       "other-device",
		DeviceType: "other-device-type",
	}
	err := s.other.SetInstallationMetadata(s.other.installationID, imOther)
	s.Require().NoError(err)
	response, err := s.other.SendPairInstallation(context.Background(), "", nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		s.main,
		func(r *MessengerResponse) bool { return len(r.Installations()) > 0 },
		"installation not received",
	)
	s.Require().NoError(err)

	_, err = s.main.EnableInstallation(s.other.installationID)
	s.Require().NoError(err)

	// Pre-condition - both sides have to know about keypairs migrated to a keycards
	kp1, _, _, err := accounts.GetProfileKeypairForTest(true, true, true)
	s.Require().NoError(err)
	kp2, _, _, err := accounts.GetSeedImportedKeypair1ForTest()
	s.Require().NoError(err)
	kp3, _, _, err := accounts.GetSeedImportedKeypair2ForTest()
	s.Require().NoError(err)
	kp1.Clock = 1
	kp2.Clock = 1
	kp3.Clock = 1

	err = s.main.settings.SaveOrUpdateKeypair(kp1)
	s.Require().NoError(err)
	err = s.main.settings.SaveOrUpdateKeypair(kp2)
	s.Require().NoError(err)
	err = s.main.settings.SaveOrUpdateKeypair(kp3)
	s.Require().NoError(err)
	dbKeypairs, err := s.main.settings.GetActiveKeypairs()
	s.Require().NoError(err)
	s.Require().Equal(3, len(dbKeypairs))

	kp1.Clock = 0
	kp2.Clock = 0
	kp3.Clock = 0

	err = s.other.settings.SaveOrUpdateKeypair(kp1)
	s.Require().NoError(err)
	err = s.other.settings.SaveOrUpdateKeypair(kp2)
	s.Require().NoError(err)
	err = s.other.settings.SaveOrUpdateKeypair(kp3)
	s.Require().NoError(err)
	dbKeypairs, err = s.other.settings.GetActiveKeypairs()
	s.Require().NoError(err)
	s.Require().Equal(3, len(dbKeypairs))
}

func (s *MessengerSyncKeycardsStateSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.other)
	s.MessengerBaseTestSuite.TearDownTest()
}

func (s *MessengerSyncKeycardsStateSuite) TestSyncKeycardsIfReceiverHasNoKeycards() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	keycard1 := accounts.GetProfileKeycardForTest()

	keycard2 := accounts.GetKeycardForSeedImportedKeypair1ForTest()

	keycard2Copy := accounts.GetKeycardForSeedImportedKeypair1ForTest()
	keycard2Copy.KeycardUID = keycard2Copy.KeycardUID + "C"
	keycard2Copy.KeycardName = keycard2Copy.KeycardName + "Copy"

	keycard3 := accounts.GetKeycardForSeedImportedKeypair2ForTest()

	// Add keycards on sender
	err := senderDb.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)
	err = senderDb.SaveOrUpdateKeycard(*keycard2, 0, false)
	s.Require().NoError(err)
	err = senderDb.SaveOrUpdateKeycard(*keycard2Copy, 0, false)
	s.Require().NoError(err)
	err = senderDb.SaveOrUpdateKeycard(*keycard3, 0, false)
	s.Require().NoError(err)

	// Trigger's a sync between devices
	err = s.main.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	s.Require().NoError(err)

	s.accountsManagerOtherMock.EXPECT().DeleteKeystoreFilesForKeypair(gomock.Any()).Return(nil).AnyTimes()

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			success := len(r.Keypairs) == 3
			for _, kp := range r.Keypairs {
				if kp.KeyUID == keycard1.KeyUID {
					success = success && len(kp.Keycards) == 1
				} else if kp.KeyUID == keycard2.KeyUID {
					success = success && len(kp.Keycards) == 2
				} else if kp.KeyUID == keycard3.KeyUID {
					success = success && len(kp.Keycards) == 1
				}
			}
			return success
		},
		"expected to receive keycards",
	)
	s.Require().NoError(err)

	syncedKeycards, err := dbOnReceiver.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(4, len(syncedKeycards))
	s.Require().True(accounts.Contains(syncedKeycards, keycard1, accounts.SameKeycards))
	s.Require().True(accounts.Contains(syncedKeycards, keycard2, accounts.SameKeycards))
	s.Require().True(accounts.Contains(syncedKeycards, keycard2Copy, accounts.SameKeycards))
	s.Require().True(accounts.Contains(syncedKeycards, keycard3, accounts.SameKeycards))
}

func (s *MessengerSyncKeycardsStateSuite) TestSyncKeycardsIfKeycardsWereDeletedOnSenderSide() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	// Add keycards on sender
	keycard1 := accounts.GetProfileKeycardForTest()

	keycard2 := accounts.GetKeycardForSeedImportedKeypair1ForTest()

	keycard2Copy := accounts.GetKeycardForSeedImportedKeypair1ForTest()
	keycard2Copy.KeycardUID = keycard2Copy.KeycardUID + "C"
	keycard2Copy.KeycardName = keycard2Copy.KeycardName + "Copy"

	keycard3 := accounts.GetKeycardForSeedImportedKeypair2ForTest()

	// Add keycards on sender
	err := senderDb.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)
	err = senderDb.SaveOrUpdateKeycard(*keycard2, 0, false)
	s.Require().NoError(err)

	// Add keycards on receiver
	err = dbOnReceiver.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)
	err = dbOnReceiver.SaveOrUpdateKeycard(*keycard2, 0, false)
	s.Require().NoError(err)
	err = dbOnReceiver.SaveOrUpdateKeycard(*keycard2Copy, 0, false)
	s.Require().NoError(err)
	err = dbOnReceiver.SaveOrUpdateKeycard(*keycard3, 0, false)
	s.Require().NoError(err)

	// Trigger's a sync between devices
	err = s.main.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			success := len(r.Keypairs) == 3
			for _, kp := range r.Keypairs {
				if kp.KeyUID == keycard1.KeyUID {
					success = success && len(kp.Keycards) == 1
				} else if kp.KeyUID == keycard2.KeyUID {
					success = success && len(kp.Keycards) == 1
				}
			}
			return success
		},
		"expected to receive keycards",
	)
	s.Require().NoError(err)

	syncedKeycards, err := dbOnReceiver.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(2, len(syncedKeycards))
	s.Require().True(accounts.Contains(syncedKeycards, keycard1, accounts.SameKeycards))
	s.Require().True(accounts.Contains(syncedKeycards, keycard2, accounts.SameKeycards))
}

func (s *MessengerSyncKeycardsStateSuite) TestSyncKeycardsIfReceiverAndSenderHasNoKeycardsInCommon() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	// Add keycards on sender
	keycard1 := accounts.GetProfileKeycardForTest()

	keycard2 := accounts.GetKeycardForSeedImportedKeypair1ForTest()

	keycard2Copy := accounts.GetKeycardForSeedImportedKeypair1ForTest()
	keycard2Copy.KeycardUID = keycard2Copy.KeycardUID + "C"
	keycard2Copy.KeycardName = keycard2Copy.KeycardName + "Copy"

	keycard3 := accounts.GetKeycardForSeedImportedKeypair2ForTest()

	// Add keycards on sender
	err := senderDb.SaveOrUpdateKeycard(*keycard2, 0, false)
	s.Require().NoError(err)
	err = senderDb.SaveOrUpdateKeycard(*keycard2Copy, 0, false)
	s.Require().NoError(err)

	// Add keycards on receiver
	err = dbOnReceiver.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)
	err = dbOnReceiver.SaveOrUpdateKeycard(*keycard3, 0, false)
	s.Require().NoError(err)

	// Trigger's a sync between devices
	err = s.main.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	s.Require().NoError(err)

	s.accountsManagerOtherMock.EXPECT().DeleteKeystoreFilesForKeypair(gomock.Any()).Return(nil).AnyTimes()

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			success := len(r.Keypairs) == 3
			for _, kp := range r.Keypairs {
				if kp.KeyUID == keycard2.KeyUID {
					success = success && len(kp.Keycards) == 2
				}
			}
			return success
		},
		"expected to receive keycards",
	)
	s.Require().NoError(err)

	syncedKeycards, err := dbOnReceiver.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(2, len(syncedKeycards))
	s.Require().True(accounts.Contains(syncedKeycards, keycard2, accounts.SameKeycards))
	s.Require().True(accounts.Contains(syncedKeycards, keycard2Copy, accounts.SameKeycards))
}
