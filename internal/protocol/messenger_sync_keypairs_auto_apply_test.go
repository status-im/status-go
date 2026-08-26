package protocol

import (
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/accounts/accountsevent"
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

// Regression: SaveOrUpdateKeypair writes xpub unconditionally, so an empty wire
// xpub must not wipe a previously stored wallet xpub. UpdateKeypairXPub already
// treats empty xpub as "leave unchanged"; the sync path should match.
func (s *MessengerSyncKeypairsAutoApplySuite) TestEmptyWireXPubDoesNotWipeStoredXPub() {
	const storedXPub = "xpub6BugbotRegressionWalletXPubDoNotWipeOnEmptySyncMessage"

	kp, _, _, err := accounts.GetSeedImportedKeypair1ForTest()
	s.Require().NoError(err)
	kp.Clock = 1
	kp.XPub = storedXPub
	kp.ColdWallet = accsmanagementtypes.ColdWalletTypeStatusKeycard
	for _, acc := range kp.Accounts {
		acc.Operable = accsmanagementtypes.AccountFullyOperable
	}
	s.Require().NoError(s.m.settings.SaveOrUpdateKeypair(kp))

	message := s.syncMessageFor(kp, 2, "Renamed On Paired Device", accsmanagementtypes.ColdWalletTypeStatusKeycard)
	message.Xpub = ""

	s.handleSync(message, false)

	dbKp := s.dbKeypair(kp.KeyUID)
	s.Require().Equal("Renamed On Paired Device", dbKp.Name)
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeStatusKeycard, dbKp.ColdWallet)
	s.Require().Equal(storedXPub, dbKp.XPub, "empty SyncKeypair.xpub must leave the stored wallet xpub unchanged")
}

func (s *MessengerSyncKeypairsAutoApplySuite) handleSyncWithState(message *protobuf.SyncKeypair, fromLocalPairing bool) *ReceivedMessageState {
	state := s.m.buildMessageState()
	err := s.m.handleSyncKeypairInternal(state, message, fromLocalPairing)
	s.Require().NoError(err)
	return state
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestNonEmptyWireXPubAdoptedForUnknownKeypair() {
	const wireXPub = "xpub6WireXPubForUnknownKeypairAdoption"

	kp, _, _, err := accounts.GetSeedImportedKeypair1ForTest()
	s.Require().NoError(err)
	kp.XPub = wireXPub
	// not saved locally — this device has never seen the keypair

	s.handleSync(s.syncMessageFor(kp, 2, kp.Name, accsmanagementtypes.ColdWalletTypeStatusKeycard), false)

	dbKp := s.dbKeypair(kp.KeyUID)
	s.Require().Equal(wireXPub, dbKp.XPub, "a newly adopted keypair must store the wire xpub, otherwise no-password derivation is impossible on this device")
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestNonEmptyWireXPubBackfillsMissingStoredXPubOnKeypairUpdate() {
	const wireXPub = "xpub6ValueSentByThePairedDevice"

	// the xpub is derived at a fixed path from the keypair's master key, so two
	// devices never hold two different non-empty values for it; the state that
	// does occur is a device predating the xpub column, holding none
	kp, _, _, err := accounts.GetSeedImportedKeypair1ForTest()
	s.Require().NoError(err)
	kp.Clock = 1
	kp.XPub = ""
	for _, acc := range kp.Accounts {
		acc.Operable = accsmanagementtypes.AccountFullyOperable
	}
	s.Require().NoError(s.m.settings.SaveOrUpdateKeypair(kp))
	s.Require().Empty(s.dbKeypair(kp.KeyUID).XPub, "the keypair must start without an xpub, else the backfill proves nothing")

	message := s.syncMessageFor(kp, 2, kp.Name, accsmanagementtypes.ColdWalletTypeNone)
	message.Xpub = wireXPub

	s.handleSync(message, false)

	dbKp := s.dbKeypair(kp.KeyUID)
	s.Require().Equal(wireXPub, dbKp.XPub, "a keypair stored without an xpub must take the one a paired device sends")
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestLedgerMigrationAdoptedVerbatimWhenAutoApplyEnabled() {
	kp := s.saveSeedKeypair(accsmanagementtypes.ColdWalletTypeNone)

	s.handleSync(s.syncMessageFor(kp, 2, kp.Name, accsmanagementtypes.ColdWalletTypeLedger), false)

	dbKp := s.dbKeypair(kp.KeyUID)
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeLedger, dbKp.ColdWallet, "the wire cold wallet string must be stored verbatim")
	s.Require().True(dbKp.MigratedToColdWallet(), "a ledger keypair must count as migrated to a cold wallet")
	for _, acc := range dbKp.Accounts {
		s.Require().Equal(accsmanagementtypes.AccountFullyOperable, acc.Operable, "accounts must stay operable after a ledger migration because no keystore is needed")
	}
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestTrezorMigrationAdoptedVerbatimWhenAutoApplyEnabled() {
	kp := s.saveSeedKeypair(accsmanagementtypes.ColdWalletTypeNone)

	s.handleSync(s.syncMessageFor(kp, 2, kp.Name, accsmanagementtypes.ColdWalletTypeTrezor), false)

	dbKp := s.dbKeypair(kp.KeyUID)
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeTrezor, dbKp.ColdWallet, "the wire cold wallet string must be stored verbatim")
	s.Require().True(dbKp.MigratedToColdWallet(), "a trezor keypair must count as migrated to a cold wallet")
	for _, acc := range dbKp.Accounts {
		s.Require().Equal(accsmanagementtypes.AccountFullyOperable, acc.Operable, "accounts must stay operable after a trezor migration because no keystore is needed")
	}
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestUnknownColdWalletStringAdoptedVerbatimForUnknownKeypair() {
	kp, _, _, err := accounts.GetSeedImportedKeypair1ForTest()
	s.Require().NoError(err)
	// not saved locally — this device has never seen the keypair

	message := s.syncMessageFor(kp, 2, kp.Name, accsmanagementtypes.ColdWalletTypeNone)
	message.ColdWallet = "future-hw-wallet"

	s.handleSync(message, false)

	dbKp := s.dbKeypair(kp.KeyUID)
	s.Require().Equal(accsmanagementtypes.ColdWalletType("future-hw-wallet"), dbKp.ColdWallet, "an unrecognized cold wallet string from a newer client must be adopted verbatim, not normalized away")
	s.Require().True(dbKp.MigratedToColdWallet(), "any non-empty cold wallet value must count as migrated")
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestEqualClockSyncKeypairLeavesKeypairUntouched() {
	kp, _, _, err := accounts.GetSeedImportedKeypair1ForTest()
	s.Require().NoError(err)
	kp.Clock = 5
	kp.ColdWallet = accsmanagementtypes.ColdWalletTypeStatusKeycard
	for _, acc := range kp.Accounts {
		acc.Operable = accsmanagementtypes.AccountFullyOperable
	}
	s.Require().NoError(s.m.settings.SaveOrUpdateKeypair(kp))

	state := s.handleSyncWithState(s.syncMessageFor(kp, 5, "Stale Name", accsmanagementtypes.ColdWalletTypeNone), false)

	s.Require().Empty(state.Response.Keypairs, "a stale sync must not be echoed to clients")
	dbKp := s.dbKeypair(kp.KeyUID)
	s.Require().Equal("Seed Imported 1", dbKp.Name, "an equal-clock sync must not apply metadata")
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeStatusKeycard, dbKp.ColdWallet, "an equal-clock sync must not undo the cold wallet migration")
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestLowerClockSyncKeypairLeavesKeypairUntouched() {
	kp, _, _, err := accounts.GetSeedImportedKeypair1ForTest()
	s.Require().NoError(err)
	kp.Clock = 5
	kp.ColdWallet = accsmanagementtypes.ColdWalletTypeStatusKeycard
	for _, acc := range kp.Accounts {
		acc.Operable = accsmanagementtypes.AccountFullyOperable
	}
	s.Require().NoError(s.m.settings.SaveOrUpdateKeypair(kp))

	state := s.handleSyncWithState(s.syncMessageFor(kp, 4, "Stale Name", accsmanagementtypes.ColdWalletTypeNone), false)

	s.Require().Empty(state.Response.Keypairs, "a stale sync must not be echoed to clients")
	dbKp := s.dbKeypair(kp.KeyUID)
	s.Require().Equal("Seed Imported 1", dbKp.Name, "a lower-clock sync must not apply metadata")
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeStatusKeycard, dbKp.ColdWallet, "a lower-clock sync must not undo the cold wallet migration")
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestStaleProfileSyncDoesNotSetProfileMigrationNeeded() {
	kp := s.saveProfileKeypair(accsmanagementtypes.ColdWalletTypeNone)

	state := s.handleSyncWithState(s.syncMessageFor(kp, 1, kp.Name, accsmanagementtypes.ColdWalletTypeStatusKeycard), false)

	s.Require().Empty(state.Response.Keypairs, "a stale profile sync must not be echoed to clients")
	migrationNeeded, err := s.m.settings.ProfileMigrationNeeded()
	s.Require().NoError(err)
	s.Require().False(migrationNeeded, "a replayed stale sync must not prompt the user with a profile migration banner")
	dbKp := s.dbKeypair(kp.KeyUID)
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeNone, dbKp.ColdWallet)
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestRemovedKeypairSyncRemovesKnownKeypair() {
	kp := s.saveSeedKeypair(accsmanagementtypes.ColdWalletTypeNone)

	message := s.syncMessageFor(kp, 2, kp.Name, accsmanagementtypes.ColdWalletTypeNone)
	message.Removed = true
	for _, sAcc := range message.Accounts {
		sAcc.Removed = true
	}

	state := s.handleSyncWithState(message, false)

	s.Require().Len(state.Response.Keypairs, 1, "clients need the removal echoed in the response")
	s.Require().True(state.Response.Keypairs[0].Removed, "the echoed keypair must carry the removal flag")
	s.Require().Empty(state.Response.ActivityCenterNotifications(), "removing a keypair must not raise a new-keypair notification")

	_, err := s.m.settings.GetKeypairByKeyUID(kp.KeyUID)
	s.Require().ErrorIs(err, accsmanagementtypes.ErrDbKeypairNotFound, "a keypair removed on a paired device must no longer be active on this device")
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestRevertSyncMarksLocallyUnknownAccountNonOperable() {
	kp := s.saveSeedKeypair(accsmanagementtypes.ColdWalletTypeStatusKeycard)

	extraAddress := types.Address{0xde, 0xad, 0xbe, 0xef}
	message := s.syncMessageFor(kp, 2, kp.Name, accsmanagementtypes.ColdWalletTypeNone)
	message.Accounts = append(message.Accounts, &protobuf.SyncAccount{
		Clock:    2,
		Address:  extraAddress.Bytes(),
		KeyUid:   kp.KeyUID,
		Path:     "m/44'/60'/1'/0/0",
		Name:     "Never Seen Locally",
		Operable: string(accsmanagementtypes.AccountFullyOperable),
	})

	s.handleSync(message, false)

	dbKp := s.dbKeypair(kp.KeyUID)
	extraFound := false
	for _, acc := range dbKp.Accounts {
		if acc.Address == extraAddress {
			s.Require().Equal(accsmanagementtypes.AccountNonOperable, acc.Operable, "an account this device never saw, arriving in a keycard-to-app revert, has no key material here and must be non operable")
			extraFound = true
		}
	}
	s.Require().True(extraFound)
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestNewKeypairFromPairedDeviceAddsACNotification() {
	kp, _, _, err := accounts.GetSeedImportedKeypair1ForTest()
	s.Require().NoError(err)
	// not saved locally — this device has never seen the keypair

	state := s.handleSyncWithState(s.syncMessageFor(kp, 2, kp.Name, accsmanagementtypes.ColdWalletTypeNone), false)

	notifications := state.Response.ActivityCenterNotifications()
	s.Require().Len(notifications, 1, "the user must be told another device added keys to the profile")
	s.Require().Equal(ActivityCenterNotificationTypeNewKeypairAddedToPairedDevice, notifications[0].Type)
	s.Require().Equal(kp.KeyUID, notifications[0].Message.ID)
	s.Require().Equal(kp.Name, notifications[0].Message.Text)
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestKnownKeypairUpdateDoesNotAddACNotification() {
	kp := s.saveSeedKeypair(accsmanagementtypes.ColdWalletTypeNone)

	state := s.handleSyncWithState(s.syncMessageFor(kp, 2, "Renamed On Paired Device", accsmanagementtypes.ColdWalletTypeNone), false)

	s.Require().Empty(state.Response.ActivityCenterNotifications(), "routine keypair updates must not spam the activity centre")
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestSyncedFromPreservedOnKeypairUpdate() {
	kp, _, _, err := accounts.GetSeedImportedKeypair1ForTest()
	s.Require().NoError(err)
	kp.Clock = 1
	kp.SyncedFrom = "origin-device"
	for _, acc := range kp.Accounts {
		acc.Operable = accsmanagementtypes.AccountFullyOperable
	}
	s.Require().NoError(s.m.settings.SaveOrUpdateKeypair(kp))

	message := s.syncMessageFor(kp, 2, kp.Name, accsmanagementtypes.ColdWalletTypeNone)
	message.SyncedFrom = "other-device"

	s.handleSync(message, false)

	dbKp := s.dbKeypair(kp.KeyUID)
	s.Require().Equal("origin-device", dbKp.SyncedFrom, "keypair provenance must not be rewritten by routine syncs")
}

func (s *MessengerSyncKeypairsAutoApplySuite) TestAccountsPublisherEmitsAddedEventForSyncedNewKeypair() {
	publisher := pubsub.NewPublisher()
	m2, err := newRunningTestMessenger(s.T(), s.messagingEnv, testMessengerConfig{
		extraOptions: []Option{WithAccountsPublisher(publisher)},
	})
	s.Require().NoError(err)

	addedCh, unsubAdded := pubsub.Subscribe[accountsevent.AccountsAddedEvent](publisher, 8)
	defer unsubAdded()
	removedCh, unsubRemoved := pubsub.Subscribe[accountsevent.AccountsRemovedEvent](publisher, 8)
	defer unsubRemoved()

	kp, _, _, err := accounts.GetSeedImportedKeypair1ForTest()
	s.Require().NoError(err)
	remoteKp := *kp
	remoteKp.Clock = 2
	message, err := m2.prepareSyncKeypairMessage(&remoteKp)
	s.Require().NoError(err)
	for _, sAcc := range message.Accounts {
		sAcc.Clock = 2
	}

	s.Require().NoError(m2.handleSyncKeypairInternal(m2.buildMessageState(), message, false))

	select {
	case ev := <-addedCh:
		s.Require().Len(ev.Accounts, len(kp.Accounts), "every synced-in wallet account must be announced so balances get fetched")
		announced := make(map[gethcommon.Address]bool)
		for _, addr := range ev.Accounts {
			announced[addr] = true
		}
		for _, acc := range kp.Accounts {
			s.Require().True(announced[gethcommon.Address(acc.Address)], "account %s arrived via sync but was not announced", acc.Address.String())
		}
	default:
		s.Require().FailNow("expected an AccountsAddedEvent because a keypair with wallet accounts arrived via sync")
	}

	// publishing happens before handleSyncKeypairInternal returns, so a second
	// event would already be waiting here; a duplicate means duplicate balance work
	select {
	case <-addedCh:
		s.Require().FailNow("expected exactly one AccountsAddedEvent, a second announces the same accounts twice")
	default:
	}

	select {
	case <-removedCh:
		s.Require().FailNow("no AccountsRemovedEvent expected when a new keypair arrives")
	default:
	}
}
