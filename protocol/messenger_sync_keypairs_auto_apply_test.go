package protocol

import (
	"testing"

	"github.com/stretchr/testify/suite"

	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/protobuf"
)

func TestMessengerSyncKeypairsAutoApplyTest(t *testing.T) {
	suite.Run(t, new(MessengerSyncKeypairsAutoApplySuite))
}

type MessengerSyncKeypairsAutoApplySuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerSyncKeypairsAutoApplySuite) setAutoApply(enabled bool) {
	err := s.m.settings.SaveSettingField(settings.AutoApplyKeypairMigrations, enabled)
	s.Require().NoError(err)
}

func (s *MessengerSyncKeypairsAutoApplySuite) saveSeedKeypair(coldWallet accsmanagementtypes.ColdWalletType) *accsmanagementtypes.Keypair {
	kp, _, _, err := accounts.GetSeedImportedKeypair1ForTest()
	s.Require().NoError(err)
	kp.Clock = 1
	kp.ColdWallet = coldWallet
	for _, acc := range kp.Accounts {
		acc.Operable = accsmanagementtypes.AccountFullyOperable
	}
	s.Require().NoError(s.m.settings.SaveOrUpdateKeypair(kp))
	return kp
}

func (s *MessengerSyncKeypairsAutoApplySuite) saveProfileKeypair(coldWallet accsmanagementtypes.ColdWalletType) *accsmanagementtypes.Keypair {
	kp, _, _, err := accounts.GetProfileKeypairForTest(true, true, false)
	s.Require().NoError(err)
	kp.KeyUID = s.m.account.KeyUID
	kp.Clock = 1
	kp.ColdWallet = coldWallet
	for _, acc := range kp.Accounts {
		acc.KeyUID = s.m.account.KeyUID
		acc.Operable = accsmanagementtypes.AccountFullyOperable
	}
	s.Require().NoError(s.m.settings.SaveOrUpdateKeypair(kp))
	return kp
}

// Builds the SyncKeypair message a paired device would send after applying the given
// mutations to its own copy of the keypair.
func (s *MessengerSyncKeypairsAutoApplySuite) syncMessageFor(kp *accsmanagementtypes.Keypair, clock uint64, name string, coldWallet accsmanagementtypes.ColdWalletType) *protobuf.SyncKeypair {
	remoteKp := *kp
	remoteKp.Clock = clock
	remoteKp.Name = name
	remoteKp.ColdWallet = coldWallet
	message, err := s.m.prepareSyncKeypairMessage(&remoteKp)
	s.Require().NoError(err)
	for _, sAcc := range message.Accounts {
		sAcc.Clock = clock
	}
	return message
}

func (s *MessengerSyncKeypairsAutoApplySuite) handleSync(message *protobuf.SyncKeypair, fromLocalPairing bool) {
	err := s.m.handleSyncKeypairInternal(s.m.buildMessageState(), message, fromLocalPairing)
	s.Require().NoError(err)
}

func (s *MessengerSyncKeypairsAutoApplySuite) dbKeypair(keyUID string) *accsmanagementtypes.Keypair {
	dbKp, err := s.m.settings.GetKeypairByKeyUID(keyUID)
	s.Require().NoError(err)
	return dbKp
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestAutoApplyDefaultsToEnabled() {
	autoApply, err := s.m.settings.AutoApplyKeypairMigrations()
	s.Require().NoError(err)
	s.Require().True(autoApply)
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestMigrationAppliedWhenAutoApplyEnabled() {
	kp := s.saveSeedKeypair(accsmanagementtypes.ColdWalletTypeNone)

	s.handleSync(s.syncMessageFor(kp, 2, kp.Name, accsmanagementtypes.ColdWalletTypeStatusKeycard), false)

	dbKp := s.dbKeypair(kp.KeyUID)
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeStatusKeycard, dbKp.ColdWallet)
	for _, acc := range dbKp.Accounts {
		s.Require().Equal(accsmanagementtypes.AccountFullyOperable, acc.Operable)
	}
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestRevertAppliedWhenAutoApplyEnabled() {
	kp := s.saveSeedKeypair(accsmanagementtypes.ColdWalletTypeStatusKeycard)

	s.handleSync(s.syncMessageFor(kp, 2, kp.Name, accsmanagementtypes.ColdWalletTypeNone), false)

	dbKp := s.dbKeypair(kp.KeyUID)
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeNone, dbKp.ColdWallet)
	for _, acc := range dbKp.Accounts {
		s.Require().Equal(accsmanagementtypes.AccountNonOperable, acc.Operable)
	}
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestMigrationIgnoredWhenAutoApplyDisabled() {
	s.setAutoApply(false)
	kp := s.saveSeedKeypair(accsmanagementtypes.ColdWalletTypeNone)

	s.handleSync(s.syncMessageFor(kp, 2, "Renamed On Paired Device", accsmanagementtypes.ColdWalletTypeStatusKeycard), false)

	dbKp := s.dbKeypair(kp.KeyUID)
	// metadata applied, signing method untouched
	s.Require().Equal("Renamed On Paired Device", dbKp.Name)
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeNone, dbKp.ColdWallet)
	for _, acc := range dbKp.Accounts {
		s.Require().Equal(accsmanagementtypes.AccountFullyOperable, acc.Operable)
	}
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestRevertIgnoredWhenAutoApplyDisabled() {
	s.setAutoApply(false)
	kp := s.saveSeedKeypair(accsmanagementtypes.ColdWalletTypeStatusKeycard)

	s.handleSync(s.syncMessageFor(kp, 2, "Renamed On Paired Device", accsmanagementtypes.ColdWalletTypeNone), false)

	dbKp := s.dbKeypair(kp.KeyUID)
	s.Require().Equal("Renamed On Paired Device", dbKp.Name)
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeStatusKeycard, dbKp.ColdWallet)
	for _, acc := range dbKp.Accounts {
		s.Require().Equal(accsmanagementtypes.AccountFullyOperable, acc.Operable)
	}
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestUnknownKeypairAdoptsWireStateWhenAutoApplyDisabled() {
	s.setAutoApply(false)
	kp, _, _, err := accounts.GetSeedImportedKeypair1ForTest()
	s.Require().NoError(err)
	// not saved locally — this device has never seen the keypair

	s.handleSync(s.syncMessageFor(kp, 2, kp.Name, accsmanagementtypes.ColdWalletTypeStatusKeycard), false)

	dbKp := s.dbKeypair(kp.KeyUID)
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeStatusKeycard, dbKp.ColdWallet)
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestLocalPairingBypassesAutoApply() {
	s.setAutoApply(false)
	kp := s.saveSeedKeypair(accsmanagementtypes.ColdWalletTypeNone)

	s.handleSync(s.syncMessageFor(kp, 2, kp.Name, accsmanagementtypes.ColdWalletTypeStatusKeycard), true)

	dbKp := s.dbKeypair(kp.KeyUID)
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeStatusKeycard, dbKp.ColdWallet)
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestProfileMigrationNeededSetWhenAutoApplyEnabled() {
	kp := s.saveProfileKeypair(accsmanagementtypes.ColdWalletTypeNone)

	s.handleSync(s.syncMessageFor(kp, 2, "Renamed On Paired Device", accsmanagementtypes.ColdWalletTypeStatusKeycard), false)

	migrationNeeded, err := s.m.settings.ProfileMigrationNeeded()
	s.Require().NoError(err)
	s.Require().True(migrationNeeded)

	dbKp := s.dbKeypair(kp.KeyUID)
	// metadata applies even while the signing method is preserved until the user completes the migration flow
	s.Require().Equal("Renamed On Paired Device", dbKp.Name)
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeNone, dbKp.ColdWallet)
	for _, acc := range dbKp.Accounts {
		s.Require().Equal(accsmanagementtypes.AccountFullyOperable, acc.Operable)
	}
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestProfileRevertDivergenceAppliesMetadataWhenAutoApplyEnabled() {
	kp := s.saveProfileKeypair(accsmanagementtypes.ColdWalletTypeStatusKeycard)

	s.handleSync(s.syncMessageFor(kp, 2, "Renamed On Paired Device", accsmanagementtypes.ColdWalletTypeNone), false)

	migrationNeeded, err := s.m.settings.ProfileMigrationNeeded()
	s.Require().NoError(err)
	s.Require().True(migrationNeeded)

	dbKp := s.dbKeypair(kp.KeyUID)
	s.Require().Equal("Renamed On Paired Device", dbKp.Name)
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeStatusKeycard, dbKp.ColdWallet)
	for _, acc := range dbKp.Accounts {
		s.Require().Equal(accsmanagementtypes.AccountFullyOperable, acc.Operable)
	}
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestPartiallyOperableProfileAccountPreservedWhenAutoApplyDisabled() {
	s.setAutoApply(false)

	kp, _, _, err := accounts.GetProfileKeypairForTest(true, true, false)
	s.Require().NoError(err)
	kp.KeyUID = s.m.account.KeyUID
	kp.Clock = 1
	var partiallyOperableAddress types.Address
	for _, acc := range kp.Accounts {
		acc.KeyUID = s.m.account.KeyUID
		if acc.Chat {
			acc.Operable = accsmanagementtypes.AccountFullyOperable
			continue
		}
		acc.Operable = accsmanagementtypes.AccountPartiallyOperable
		partiallyOperableAddress = acc.Address
	}
	s.Require().NotEqual(types.Address{}, partiallyOperableAddress)
	s.Require().NoError(s.m.settings.SaveOrUpdateKeypair(kp))

	message := s.syncMessageFor(kp, 2, "Renamed On Paired Device", accsmanagementtypes.ColdWalletTypeStatusKeycard)
	// the sending device reports the account as fully operable, so a preserved local "partially" below proves the wire value was not copied
	for _, sAcc := range message.Accounts {
		if types.BytesToAddress(sAcc.Address) == partiallyOperableAddress {
			sAcc.Operable = string(accsmanagementtypes.AccountFullyOperable)
		}
	}

	s.handleSync(message, false)

	migrationNeeded, err := s.m.settings.ProfileMigrationNeeded()
	s.Require().NoError(err)
	s.Require().False(migrationNeeded)

	dbKp := s.dbKeypair(kp.KeyUID)
	s.Require().Equal("Renamed On Paired Device", dbKp.Name)
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeNone, dbKp.ColdWallet)
	partiallyOperableFound := false
	for _, acc := range dbKp.Accounts {
		if acc.Address == partiallyOperableAddress {
			s.Require().Equal(accsmanagementtypes.AccountPartiallyOperable, acc.Operable)
			partiallyOperableFound = true
		}
	}
	s.Require().True(partiallyOperableFound)
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestProfileMigrationNeededNotSetWhenAutoApplyDisabled() {
	s.setAutoApply(false)
	kp := s.saveProfileKeypair(accsmanagementtypes.ColdWalletTypeNone)

	s.handleSync(s.syncMessageFor(kp, 2, "Renamed On Paired Device", accsmanagementtypes.ColdWalletTypeStatusKeycard), false)

	migrationNeeded, err := s.m.settings.ProfileMigrationNeeded()
	s.Require().NoError(err)
	s.Require().False(migrationNeeded)

	dbKp := s.dbKeypair(kp.KeyUID)
	// metadata applied even for the profile keypair, signing method untouched
	s.Require().Equal("Renamed On Paired Device", dbKp.Name)
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeNone, dbKp.ColdWallet)
	for _, acc := range dbKp.Accounts {
		s.Require().Equal(accsmanagementtypes.AccountFullyOperable, acc.Operable)
	}
}
