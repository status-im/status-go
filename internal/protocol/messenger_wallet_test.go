package protocol

import (
	"testing"

	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"

	"github.com/stretchr/testify/suite"
)

func TestWalletSuite(t *testing.T) {
	suite.Run(t, new(WalletSuite))
}

type WalletSuite struct {
	MessengerBaseTestSuite
}

func (s *WalletSuite) TestRemainingCapacity() {
	profileKeypair, _, _, err := accounts.GetProfileKeypairForTest(true, true, true)
	s.Require().NoError(err)
	seedImportedKeypair, _, _, err := accounts.GetSeedImportedKeypair1ForTest()
	s.Require().NoError(err)
	woAccounts := accounts.GetWatchOnlyAccountsForTest()

	// Empty DB
	capacity, err := s.m.RemainingAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(MaxNumberOfAccounts, capacity)

	capacity, err = s.m.RemainingKeypairCapacity()
	s.Require().NoError(err)
	s.Require().Equal(MaxNumberOfKeypairs, capacity)

	capacity, err = s.m.RemainingWatchOnlyAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(MaxNumberOfWatchOnlyAccounts, capacity)

	// profile keypair with chat account, default wallet account and 2 more derived accounts added
	err = s.m.settings.SaveOrUpdateKeypair(profileKeypair)
	s.Require().NoError(err)

	capacity, err = s.m.RemainingAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(MaxNumberOfAccounts-3, capacity)

	capacity, err = s.m.RemainingKeypairCapacity()
	s.Require().NoError(err)
	s.Require().Equal(MaxNumberOfKeypairs-1, capacity)

	capacity, err = s.m.RemainingWatchOnlyAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(MaxNumberOfWatchOnlyAccounts, capacity)

	// seed keypair with 2 derived accounts added
	err = s.m.settings.SaveOrUpdateKeypair(seedImportedKeypair)
	s.Require().NoError(err)

	capacity, err = s.m.RemainingAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(MaxNumberOfAccounts-(3+2), capacity)

	capacity, err = s.m.RemainingKeypairCapacity()
	s.Require().NoError(err)
	s.Require().Equal(MaxNumberOfKeypairs-(1+1), capacity)

	capacity, err = s.m.RemainingWatchOnlyAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(MaxNumberOfWatchOnlyAccounts, capacity)

	// 1 Watch only accounts added
	err = s.m.settings.SaveOrUpdateAccounts(woAccounts[:1], false)
	s.Require().NoError(err)

	capacity, err = s.m.RemainingAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(MaxNumberOfAccounts-(3+2+1), capacity)

	capacity, err = s.m.RemainingKeypairCapacity()
	s.Require().NoError(err)
	s.Require().Equal(MaxNumberOfKeypairs-(1+1), capacity)

	capacity, err = s.m.RemainingWatchOnlyAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(MaxNumberOfWatchOnlyAccounts-1, capacity)

	// try to add 3 more keypairs
	seedImportedKeypair2, _, _, err := accounts.GetSeedImportedKeypair2ForTest()
	s.Require().NoError(err)
	seedImportedKeypair2.KeyUID = "0000000000000000000000000000000000000000000000000000000000000091"
	seedImportedKeypair2.Accounts[0].Address = types.Address{0x91}
	seedImportedKeypair2.Accounts[0].KeyUID = seedImportedKeypair2.KeyUID
	seedImportedKeypair2.Accounts[1].Address = types.Address{0x92}
	seedImportedKeypair2.Accounts[1].KeyUID = seedImportedKeypair2.KeyUID

	err = s.m.settings.SaveOrUpdateKeypair(seedImportedKeypair2)
	s.Require().NoError(err)

	seedImportedKeypair3, _, _, err := accounts.GetSeedImportedKeypair2ForTest()
	s.Require().NoError(err)
	seedImportedKeypair3.KeyUID = "0000000000000000000000000000000000000000000000000000000000000093"
	seedImportedKeypair3.Accounts[0].Address = types.Address{0x93}
	seedImportedKeypair3.Accounts[0].KeyUID = seedImportedKeypair3.KeyUID
	seedImportedKeypair3.Accounts[1].Address = types.Address{0x94}
	seedImportedKeypair3.Accounts[1].KeyUID = seedImportedKeypair3.KeyUID

	err = s.m.settings.SaveOrUpdateKeypair(seedImportedKeypair3)
	s.Require().NoError(err)

	seedImportedKeypair4, _, _, err := accounts.GetSeedImportedKeypair2ForTest()
	s.Require().NoError(err)
	seedImportedKeypair4.KeyUID = "0000000000000000000000000000000000000000000000000000000000000095"
	seedImportedKeypair4.Accounts[0].Address = types.Address{0x95}
	seedImportedKeypair4.Accounts[0].KeyUID = seedImportedKeypair4.KeyUID
	seedImportedKeypair4.Accounts[1].Address = types.Address{0x96}
	seedImportedKeypair4.Accounts[1].KeyUID = seedImportedKeypair4.KeyUID

	err = s.m.settings.SaveOrUpdateKeypair(seedImportedKeypair4)
	s.Require().NoError(err)

	// check the capacity after adding 3 more keypairs
	capacity, err = s.m.RemainingAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(MaxNumberOfAccounts-(3+2+1+3*2), capacity)

	capacity, err = s.m.RemainingKeypairCapacity()
	s.Require().Error(err)
	s.Require().Equal("no more keypairs can be added", err.Error())
	s.Require().Equal(0, capacity)

	capacity, err = s.m.RemainingWatchOnlyAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(MaxNumberOfWatchOnlyAccounts-1, capacity)

	// add 2 more watch only accounts
	err = s.m.settings.SaveOrUpdateAccounts(woAccounts[1:2], false)
	s.Require().NoError(err)
	err = s.m.settings.SaveOrUpdateAccounts(woAccounts[2:3], false)
	s.Require().NoError(err)

	// check the capacity after adding 8 more watch only accounts
	capacity, err = s.m.RemainingAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(MaxNumberOfAccounts-(3+2+3+3*2), capacity)

	capacity, err = s.m.RemainingKeypairCapacity()
	s.Require().Error(err)
	s.Require().Equal("no more keypairs can be added", err.Error())
	s.Require().Equal(0, capacity)

	capacity, err = s.m.RemainingWatchOnlyAccountCapacity()
	s.Require().Error(err)
	s.Require().Equal("no more watch-only accounts can be added", err.Error())
	s.Require().Equal(0, capacity)

	// add 6 accounts more
	seedImportedKeypair4.Accounts[0].Address = types.Address{0x81}
	err = s.m.settings.SaveOrUpdateAccounts(seedImportedKeypair4.Accounts[:1], false)
	s.Require().NoError(err)

	seedImportedKeypair4.Accounts[0].Address = types.Address{0x82}
	err = s.m.settings.SaveOrUpdateAccounts(seedImportedKeypair4.Accounts[:1], false)
	s.Require().NoError(err)

	seedImportedKeypair4.Accounts[0].Address = types.Address{0x83}
	err = s.m.settings.SaveOrUpdateAccounts(seedImportedKeypair4.Accounts[:1], false)
	s.Require().NoError(err)

	seedImportedKeypair4.Accounts[0].Address = types.Address{0x84}
	err = s.m.settings.SaveOrUpdateAccounts(seedImportedKeypair4.Accounts[:1], false)
	s.Require().NoError(err)

	seedImportedKeypair4.Accounts[0].Address = types.Address{0x85}
	err = s.m.settings.SaveOrUpdateAccounts(seedImportedKeypair4.Accounts[:1], false)
	s.Require().NoError(err)

	seedImportedKeypair4.Accounts[0].Address = types.Address{0x86}
	err = s.m.settings.SaveOrUpdateAccounts(seedImportedKeypair4.Accounts[:1], false)
	s.Require().NoError(err)

	// check the capacity after adding 8 more watch only accounts
	capacity, err = s.m.RemainingAccountCapacity()
	s.Require().Error(err)
	s.Require().Equal("no more accounts can be added", err.Error())
	s.Require().Equal(0, capacity)

	capacity, err = s.m.RemainingKeypairCapacity()
	s.Require().Error(err)
	s.Require().Equal("no more keypairs can be added", err.Error())
	s.Require().Equal(0, capacity)

	capacity, err = s.m.RemainingWatchOnlyAccountCapacity()
	s.Require().Error(err)
	s.Require().Equal("no more watch-only accounts can be added", err.Error())
	s.Require().Equal(0, capacity)
}

func (s *WalletSuite) saveColdSeedKeypairWithXPub() *accsmanagementtypes.Keypair {
	kp, _, _, err := accounts.GetSeedImportedKeypair1ForTest()
	s.Require().NoError(err)
	kp.Clock = 1
	kp.ColdWallet = accsmanagementtypes.ColdWalletTypeStatusKeycard
	kp.XPub = "xpub6ColdWalletKeypairStoredXPub"
	for _, acc := range kp.Accounts {
		acc.Operable = accsmanagementtypes.AccountFullyOperable
	}
	s.Require().NoError(s.m.settings.SaveOrUpdateKeypair(kp))
	return kp
}

func (s *WalletSuite) TestUpdateKeypairPreservesColdWalletStateOnRename() {
	kp := s.saveColdSeedKeypairWithXPub()

	dbKp, err := s.m.settings.GetKeypairByKeyUID(kp.KeyUID)
	s.Require().NoError(err)
	dbKp.Name = "Renamed Cold Keypair"
	s.Require().NoError(s.m.UpdateKeypair(dbKp))

	dbKp, err = s.m.settings.GetKeypairByKeyUID(kp.KeyUID)
	s.Require().NoError(err)
	s.Require().Equal("Renamed Cold Keypair", dbKp.Name, "the rename itself must apply")
	s.Require().Equal(accsmanagementtypes.ColdWalletTypeStatusKeycard, dbKp.ColdWallet, "renaming a keypair must not change its signing method")
	s.Require().Equal(kp.XPub, dbKp.XPub, "renaming a keypair must not drop the stored xpub, password-less derivation depends on it")
}
