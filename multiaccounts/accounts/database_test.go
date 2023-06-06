package accounts

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/common"
)

func setupTestDB(t *testing.T) (*Database, func()) {
	db, stop, err := appdatabase.SetupTestSQLDB("settings-tests-")
	if err != nil {
		require.NoError(t, stop())
	}
	require.NoError(t, err)

	d, err := NewDB(db)
	if err != nil {
		require.NoError(t, stop())
	}
	require.NoError(t, err)

	return d, func() {
		require.NoError(t, stop())
	}
}

func TestGetAddresses(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	accounts := []*Account{
		{Address: types.Address{0x01}, Chat: true, Wallet: true},
		{Address: types.Address{0x02}},
	}
	require.NoError(t, db.SaveOrUpdateAccounts(accounts))
	addresses, err := db.GetAddresses()
	require.NoError(t, err)
	require.Equal(t, []types.Address{{0x01}, {0x02}}, addresses)
}

func TestGetWalletAddress(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	address := types.Address{0x01}
	_, err := db.GetWalletAddress()
	require.Equal(t, err, sql.ErrNoRows)
	require.NoError(t, db.SaveOrUpdateAccounts([]*Account{{Address: address, Wallet: true}}))
	wallet, err := db.GetWalletAddress()
	require.NoError(t, err)
	require.Equal(t, address, wallet)
}

func TestGetChatAddress(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	address := types.Address{0x01}
	_, err := db.GetChatAddress()
	require.Equal(t, err, sql.ErrNoRows)
	require.NoError(t, db.SaveOrUpdateAccounts([]*Account{{Address: address, Chat: true}}))
	chat, err := db.GetChatAddress()
	require.NoError(t, err)
	require.Equal(t, address, chat)
}

func TestAddressExists(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	accounts := []*Account{
		{Address: types.Address{0x01}, Chat: true, Wallet: true},
	}
	require.NoError(t, db.SaveOrUpdateAccounts(accounts))

	exists, err := db.AddressExists(accounts[0].Address)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestAddressDoesntExist(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	exists, err := db.AddressExists(types.Address{1, 1, 1})
	require.NoError(t, err)
	require.False(t, exists)
}

func TestWatchOnlyAccounts(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	// check the db
	dbAccounts, err := db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, 0, len(dbAccounts))

	woAccounts := GetWatchOnlyAccountsForTest()

	// try to save keypair with watch only accounts
	kp := &Keypair{}
	kp.Accounts = append(kp.Accounts, woAccounts...)
	err = db.SaveOrUpdateKeypair(kp)
	require.Error(t, err)

	// check the db after that trying to save keypair with watch only accounts
	dbAccounts, err = db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, 0, len(dbAccounts))

	// save watch only accounts
	err = db.SaveOrUpdateAccounts(woAccounts)
	require.NoError(t, err)
	_, err = db.GetKeypairByKeyUID(woAccounts[0].KeyUID)
	require.Error(t, err)
	require.True(t, err == ErrDbKeypairNotFound)
	dbAccounts, err = db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, len(woAccounts), len(dbAccounts))
	require.Equal(t, woAccounts[0].Address, dbAccounts[0].Address)

	// try to save the same watch only account again
	err = db.SaveOrUpdateAccounts(woAccounts[:1])
	require.NoError(t, err)
	dbAccounts, err = db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, len(woAccounts), len(dbAccounts))
	dbAcc, err := db.GetAccountByAddress(woAccounts[:1][0].Address)
	require.NoError(t, err)
	require.Equal(t, woAccounts[:1][0].Address, dbAcc.Address)

	// try to save new watch only account
	wo4 := &Account{
		Address: types.Address{0x14},
		Type:    AccountTypeWatch,
		Name:    "WatchOnlyAcc4",
		ColorID: common.CustomizationColorPrimary,
		Emoji:   "emoji-1",
	}
	err = db.SaveOrUpdateAccounts([]*Account{wo4})
	require.NoError(t, err)
	dbAccounts, err = db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, len(woAccounts)+1, len(dbAccounts))
	dbAcc, err = db.GetAccountByAddress(wo4.Address)
	require.NoError(t, err)
	require.Equal(t, wo4.Address, dbAcc.Address)

	// updated watch onl to save the same account after it's saved
	wo4.Name = wo4.Name + "updated"
	wo4.ColorID = common.CustomizationColorCamel
	wo4.Emoji = wo4.Emoji + "updated"
	err = db.SaveOrUpdateAccounts([]*Account{wo4})
	require.NoError(t, err)
	dbAccounts, err = db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, len(woAccounts)+1, len(dbAccounts))
	dbAcc, err = db.GetAccountByAddress(wo4.Address)
	require.NoError(t, err)
	require.Equal(t, wo4.Address, dbAcc.Address)

	// try to delete keypair for watch only account
	err = db.DeleteKeypair(wo4.KeyUID)
	require.Error(t, err)
	require.True(t, err == ErrDbKeypairNotFound)

	// try to delete watch only account
	err = db.DeleteAccount(wo4.Address)
	require.NoError(t, err)
	dbAccounts, err = db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, len(woAccounts), len(dbAccounts))
	_, err = db.GetAccountByAddress(wo4.Address)
	require.Error(t, err)
	require.True(t, err == ErrDbAccountNotFound)
}

func TestUpdateKeypairName(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	kp := GetProfileKeypairForTest(true, false, false)

	// check the db
	dbAccounts, err := db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, 0, len(dbAccounts))

	// save keypair
	err = db.SaveOrUpdateKeypair(kp)
	require.NoError(t, err)
	dbKeypairs, err := db.GetKeypairs()
	require.NoError(t, err)
	require.Equal(t, 1, len(dbKeypairs))
	require.True(t, SameKeypairs(kp, dbKeypairs[0]))

	// update keypair name
	kp.Name = kp.Name + "updated"
	err = db.UpdateKeypairName(kp.KeyUID, kp.Name, kp.Clock)
	require.NoError(t, err)

	// check keypair
	dbKp, err := db.GetKeypairByKeyUID(kp.KeyUID)
	require.NoError(t, err)
	require.Equal(t, len(kp.Accounts), len(dbKp.Accounts))
	require.True(t, SameKeypairs(kp, dbKp))
}

func TestKeypairs(t *testing.T) {
	keypairs := []*Keypair{
		GetProfileKeypairForTest(true, true, true),
		GetSeedImportedKeypair1ForTest(),
		GetPrivKeyImportedKeypairForTest(), // in this context (when testing db functions) there is not limitations for private key imported keypair
	}

	for _, kp := range keypairs {
		t.Run("test keypair "+kp.Name, func(t *testing.T) {
			db, stop := setupTestDB(t)
			defer stop()

			// check the db
			dbKeypairs, err := db.GetKeypairs()
			require.NoError(t, err)
			require.Equal(t, 0, len(dbKeypairs))
			dbAccounts, err := db.GetAccounts()
			require.NoError(t, err)
			require.Equal(t, 0, len(dbAccounts))

			expectedLastUsedDerivationIndex := uint64(len(kp.Accounts) - 1)
			if kp.Type == KeypairTypeProfile {
				expectedLastUsedDerivationIndex-- // subtract one more in case of profile keypair because of chat account
			}

			// save keypair
			err = db.SaveOrUpdateKeypair(kp)
			require.NoError(t, err)
			dbKeypairs, err = db.GetKeypairs()
			require.NoError(t, err)
			require.Equal(t, 1, len(dbKeypairs))
			dbKp, err := db.GetKeypairByKeyUID(kp.KeyUID)
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts), len(dbKp.Accounts))
			kp.LastUsedDerivationIndex = expectedLastUsedDerivationIndex
			require.Equal(t, kp.KeyUID, dbKp.KeyUID)
			dbAccounts, err = db.GetAccounts()
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts), len(dbAccounts))

			// delete keypair
			err = db.DeleteKeypair(kp.KeyUID)
			require.NoError(t, err)
			_, err = db.GetKeypairByKeyUID(kp.KeyUID)
			require.Error(t, err)
			require.True(t, err == ErrDbKeypairNotFound)

			// save keypair again to test the flow below
			err = db.SaveOrUpdateKeypair(kp)
			require.NoError(t, err)
			dbKeypairs, err = db.GetKeypairs()
			require.NoError(t, err)
			require.Equal(t, 1, len(dbKeypairs))

			ind := len(kp.Accounts) - 1
			accToUpdate := kp.Accounts[ind]

			// try to save the same account again
			err = db.SaveOrUpdateAccounts([]*Account{accToUpdate})
			require.NoError(t, err)
			dbKp, err = db.GetKeypairByKeyUID(kp.KeyUID)
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts), len(dbKp.Accounts))
			require.Equal(t, kp.KeyUID, dbKp.KeyUID)
			dbAccounts, err = db.GetAccounts()
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts), len(dbAccounts))

			// update an existing account
			accToUpdate.Name = accToUpdate.Name + "updated"
			accToUpdate.ColorID = common.CustomizationColorBrown
			accToUpdate.Emoji = accToUpdate.Emoji + "updated"

			err = db.SaveOrUpdateAccounts([]*Account{accToUpdate})
			require.NoError(t, err)
			dbKp, err = db.GetKeypairByKeyUID(kp.KeyUID)
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts), len(dbKp.Accounts))
			dbAccounts, err = db.GetAccounts()
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts), len(dbAccounts))
			dbAcc, err := db.GetAccountByAddress(accToUpdate.Address)
			require.NoError(t, err)
			require.Equal(t, accToUpdate.Address, dbAcc.Address)

			// update keypair name
			kpToUpdate := kp
			kpToUpdate.Name = kpToUpdate.Name + "updated"
			err = db.SaveOrUpdateKeypair(kp)
			require.NoError(t, err)
			dbKeypairs, err = db.GetKeypairs()
			require.NoError(t, err)
			require.Equal(t, 1, len(dbKeypairs))
			dbKp, err = db.GetKeypairByKeyUID(kp.KeyUID)
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts), len(dbKp.Accounts))
			require.Equal(t, kpToUpdate.KeyUID, dbKp.KeyUID)

			// save new account to an existing keypair which is out of the default Status' derivation root path
			accToAdd := kp.Accounts[ind]
			accToAdd.Address = types.Address{0x08}
			accToAdd.Path = "m/44'/60'/0'/0/10"
			accToAdd.PublicKey = types.Hex2Bytes("0x000000008")
			accToAdd.Name = "Generated Acc 8"

			err = db.SaveOrUpdateAccounts([]*Account{accToAdd})
			require.NoError(t, err)
			dbKp, err = db.GetKeypairByKeyUID(kp.KeyUID)
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts)+1, len(dbKp.Accounts))
			require.Equal(t, kp.LastUsedDerivationIndex, dbKp.LastUsedDerivationIndex)
			dbAccounts, err = db.GetAccounts()
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts)+1, len(dbAccounts))
			dbAcc, err = db.GetAccountByAddress(accToUpdate.Address)
			require.NoError(t, err)
			require.Equal(t, accToAdd.Address, dbAcc.Address)

			// save new account to an existing keypair which follows Status' default derivation root path
			accToAdd = kp.Accounts[ind]
			accToAdd.Address = types.Address{0x09}
			accToAdd.Path = "m/44'/60'/0'/0/3"
			accToAdd.PublicKey = types.Hex2Bytes("0x000000009")
			accToAdd.Name = "Generated Acc 9"

			expectedLastUsedDerivationIndex = 3
			if kp.Type == KeypairTypeSeed {
				accToAdd.Path = "m/44'/60'/0'/0/2"
				expectedLastUsedDerivationIndex = 2
			} else if kp.Type == KeypairTypeKey {
				accToAdd.Path = "m/44'/60'/0'/0/1"
				expectedLastUsedDerivationIndex = 1
			}

			err = db.SaveOrUpdateAccounts([]*Account{accToAdd})
			require.NoError(t, err)
			dbKp, err = db.GetKeypairByKeyUID(kp.KeyUID)
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts)+2, len(dbKp.Accounts))
			require.Equal(t, expectedLastUsedDerivationIndex, dbKp.LastUsedDerivationIndex)
			dbAccounts, err = db.GetAccounts()
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts)+2, len(dbAccounts))
			dbAcc, err = db.GetAccountByAddress(accToUpdate.Address)
			require.NoError(t, err)
			require.Equal(t, accToAdd.Address, dbAcc.Address)

			// delete account
			err = db.DeleteAccount(accToAdd.Address)
			require.NoError(t, err)
			dbAccounts, err = db.GetAccounts()
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts)+1, len(dbAccounts))
			_, err = db.GetAccountByAddress(accToAdd.Address)
			require.Error(t, err)
			require.True(t, err == ErrDbAccountNotFound)

			for _, acc := range dbAccounts {
				err = db.DeleteAccount(acc.Address)
				require.NoError(t, err)
			}

			_, err = db.GetKeypairByKeyUID(kp.KeyUID)
			require.Error(t, err)
			require.True(t, err == ErrDbKeypairNotFound)
		})
	}
}
