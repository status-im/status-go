package protocol

import (
	"testing"

	"github.com/status-im/status-go/constants"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"

	"github.com/stretchr/testify/suite"
)

func TestWalletSuite(t *testing.T) {
	suite.Run(t, new(WalletSuite))
}

type WalletSuite struct {
	MessengerBaseTestSuite
}

func (s *WalletSuite) TestRemainingCapacity() {
	profileKeypair := accounts.GetProfileKeypairForTest(true, true, true)
	seedImportedKeypair := accounts.GetSeedImportedKeypair1ForTest()
	woAccounts := accounts.GetWatchOnlyAccountsForTest()

	// Empty DB
	capacity, err := s.m.RemainingAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(constants.MaxNumberOfAccounts, capacity)

	capacity, err = s.m.RemainingKeypairCapacity()
	s.Require().NoError(err)
	s.Require().Equal(constants.MaxNumberOfKeypairs, capacity)

	capacity, err = s.m.RemainingWatchOnlyAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(constants.MaxNumberOfWatchOnlyAccounts, capacity)

	// profile keypair with chat account, default wallet account and 2 more derived accounts added
	err = s.m.SaveOrUpdateKeypair(profileKeypair)
	s.Require().NoError(err)

	capacity, err = s.m.RemainingAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(constants.MaxNumberOfAccounts-3, capacity)

	capacity, err = s.m.RemainingKeypairCapacity()
	s.Require().NoError(err)
	s.Require().Equal(constants.MaxNumberOfKeypairs-1, capacity)

	capacity, err = s.m.RemainingWatchOnlyAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(constants.MaxNumberOfWatchOnlyAccounts, capacity)

	// seed keypair with 2 derived accounts added
	err = s.m.SaveOrUpdateKeypair(seedImportedKeypair)
	s.Require().NoError(err)

	capacity, err = s.m.RemainingAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(constants.MaxNumberOfAccounts-(3+2), capacity)

	capacity, err = s.m.RemainingKeypairCapacity()
	s.Require().NoError(err)
	s.Require().Equal(constants.MaxNumberOfKeypairs-(1+1), capacity)

	capacity, err = s.m.RemainingWatchOnlyAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(constants.MaxNumberOfWatchOnlyAccounts, capacity)

	// 1 Watch only accounts added
	err = s.m.SaveOrUpdateAccount(woAccounts[0])
	s.Require().NoError(err)

	capacity, err = s.m.RemainingAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(constants.MaxNumberOfAccounts-(3+2+1), capacity)

	capacity, err = s.m.RemainingKeypairCapacity()
	s.Require().NoError(err)
	s.Require().Equal(constants.MaxNumberOfKeypairs-(1+1), capacity)

	capacity, err = s.m.RemainingWatchOnlyAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(constants.MaxNumberOfWatchOnlyAccounts-1, capacity)

	// try to add 3 more keypairs
	seedImportedKeypair2 := accounts.GetSeedImportedKeypair2ForTest()
	seedImportedKeypair2.KeyUID = "0000000000000000000000000000000000000000000000000000000000000091"
	seedImportedKeypair2.Accounts[0].Address = types.Address{0x91}
	seedImportedKeypair2.Accounts[0].KeyUID = seedImportedKeypair2.KeyUID
	seedImportedKeypair2.Accounts[1].Address = types.Address{0x92}
	seedImportedKeypair2.Accounts[1].KeyUID = seedImportedKeypair2.KeyUID

	err = s.m.SaveOrUpdateKeypair(seedImportedKeypair2)
	s.Require().NoError(err)

	seedImportedKeypair3 := accounts.GetSeedImportedKeypair2ForTest()
	seedImportedKeypair3.KeyUID = "0000000000000000000000000000000000000000000000000000000000000093"
	seedImportedKeypair3.Accounts[0].Address = types.Address{0x93}
	seedImportedKeypair3.Accounts[0].KeyUID = seedImportedKeypair3.KeyUID
	seedImportedKeypair3.Accounts[1].Address = types.Address{0x94}
	seedImportedKeypair3.Accounts[1].KeyUID = seedImportedKeypair3.KeyUID

	err = s.m.SaveOrUpdateKeypair(seedImportedKeypair3)
	s.Require().NoError(err)

	seedImportedKeypair4 := accounts.GetSeedImportedKeypair2ForTest()
	seedImportedKeypair4.KeyUID = "0000000000000000000000000000000000000000000000000000000000000095"
	seedImportedKeypair4.Accounts[0].Address = types.Address{0x95}
	seedImportedKeypair4.Accounts[0].KeyUID = seedImportedKeypair4.KeyUID
	seedImportedKeypair4.Accounts[1].Address = types.Address{0x96}
	seedImportedKeypair4.Accounts[1].KeyUID = seedImportedKeypair4.KeyUID

	err = s.m.SaveOrUpdateKeypair(seedImportedKeypair4)
	s.Require().NoError(err)

	// check the capacity after adding 3 more keypairs
	capacity, err = s.m.RemainingAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(constants.MaxNumberOfAccounts-(3+2+1+3*2), capacity)

	capacity, err = s.m.RemainingKeypairCapacity()
	s.Require().Error(err)
	s.Require().Equal("no more keypairs can be added", err.Error())
	s.Require().Equal(0, capacity)

	capacity, err = s.m.RemainingWatchOnlyAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(constants.MaxNumberOfWatchOnlyAccounts-1, capacity)

	// add 2 more watch only accounts
	err = s.m.SaveOrUpdateAccount(woAccounts[1])
	s.Require().NoError(err)
	err = s.m.SaveOrUpdateAccount(woAccounts[2])
	s.Require().NoError(err)

	// check the capacity after adding 8 more watch only accounts
	capacity, err = s.m.RemainingAccountCapacity()
	s.Require().NoError(err)
	s.Require().Equal(constants.MaxNumberOfAccounts-(3+2+3+3*2), capacity)

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
	err = s.m.SaveOrUpdateAccount(seedImportedKeypair4.Accounts[0])
	s.Require().NoError(err)

	seedImportedKeypair4.Accounts[0].Address = types.Address{0x82}
	err = s.m.SaveOrUpdateAccount(seedImportedKeypair4.Accounts[0])
	s.Require().NoError(err)

	seedImportedKeypair4.Accounts[0].Address = types.Address{0x83}
	err = s.m.SaveOrUpdateAccount(seedImportedKeypair4.Accounts[0])
	s.Require().NoError(err)

	seedImportedKeypair4.Accounts[0].Address = types.Address{0x84}
	err = s.m.SaveOrUpdateAccount(seedImportedKeypair4.Accounts[0])
	s.Require().NoError(err)

	seedImportedKeypair4.Accounts[0].Address = types.Address{0x85}
	err = s.m.SaveOrUpdateAccount(seedImportedKeypair4.Accounts[0])
	s.Require().NoError(err)

	seedImportedKeypair4.Accounts[0].Address = types.Address{0x86}
	err = s.m.SaveOrUpdateAccount(seedImportedKeypair4.Accounts[0])
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
