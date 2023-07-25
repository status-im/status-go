package accounts

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/common"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
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
	require.NoError(t, db.SaveOrUpdateAccounts(accounts, false))
	addresses, err := db.GetAddresses()
	require.NoError(t, err)
	require.Equal(t, []types.Address{{0x01}, {0x02}}, addresses)
}

func TestMoveWalletAccount(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	networks := json.RawMessage("{}")
	setting := settings.Settings{
		Networks: &networks,
	}
	config := params.NodeConfig{}
	err := db.CreateSettings(setting, config)
	require.NoError(t, err)

	accounts := []*Account{
		{Address: types.Address{0x01}, Type: AccountTypeWatch, Position: 0},
		{Address: types.Address{0x02}, Type: AccountTypeWatch, Position: 1},
		{Address: types.Address{0x03}, Type: AccountTypeWatch, Position: 2},
		{Address: types.Address{0x04}, Type: AccountTypeWatch, Position: 3},
		{Address: types.Address{0x05}, Type: AccountTypeWatch, Position: 4},
		{Address: types.Address{0x06}, Type: AccountTypeWatch, Position: 5},
	}
	require.NoError(t, db.SaveOrUpdateAccounts(accounts, false))
	dbAccounts, err := db.GetActiveAccounts()
	require.NoError(t, err)
	require.Len(t, dbAccounts, len(accounts))
	for i := 0; i < len(accounts); i++ {
		require.True(t, SameAccounts(accounts[i], dbAccounts[i]))
	}

	clock := uint64(1000)
	err = db.MoveWalletAccount(-1, 4, clock)
	require.ErrorIs(t, err, ErrMovingAccountToWrongPosition)
	err = db.MoveWalletAccount(4, -1, clock)
	require.ErrorIs(t, err, ErrMovingAccountToWrongPosition)
	err = db.MoveWalletAccount(4, 4, clock)
	require.ErrorIs(t, err, ErrMovingAccountToWrongPosition)

	// Move down account from position 1 to position 4
	err = db.MoveWalletAccount(1, 4, clock)
	require.NoError(t, err)

	// Expected after moving down
	accounts = []*Account{
		{Address: types.Address{0x01}, Type: AccountTypeWatch, Position: 0},
		{Address: types.Address{0x03}, Type: AccountTypeWatch, Position: 1},
		{Address: types.Address{0x04}, Type: AccountTypeWatch, Position: 2},
		{Address: types.Address{0x05}, Type: AccountTypeWatch, Position: 3},
		{Address: types.Address{0x02}, Type: AccountTypeWatch, Position: 4}, // acc with addr 0x02 is at position 4 (moved from position 1)
		{Address: types.Address{0x06}, Type: AccountTypeWatch, Position: 5},
	}

	dbAccounts, err = db.GetActiveAccounts()
	require.NoError(t, err)
	for i := 0; i < len(accounts); i++ {
		require.True(t, SameAccounts(accounts[i], dbAccounts[i]))
	}

	// Check clock
	dbClock, err := db.GetClockOfLastAccountsPositionChange()
	require.NoError(t, err)
	require.Equal(t, clock, dbClock)

	// Move up account from position 5 to position 0
	clock = 2000
	err = db.MoveWalletAccount(5, 0, clock)
	require.NoError(t, err)

	// Expected after moving up
	accounts = []*Account{
		{Address: types.Address{0x06}, Type: AccountTypeWatch, Position: 0}, // acc with addr 0x06 is at position 0 (moved from position 5)
		{Address: types.Address{0x01}, Type: AccountTypeWatch, Position: 1},
		{Address: types.Address{0x03}, Type: AccountTypeWatch, Position: 2},
		{Address: types.Address{0x04}, Type: AccountTypeWatch, Position: 3},
		{Address: types.Address{0x05}, Type: AccountTypeWatch, Position: 4},
		{Address: types.Address{0x02}, Type: AccountTypeWatch, Position: 5},
	}

	dbAccounts, err = db.GetActiveAccounts()
	require.NoError(t, err)
	for i := 0; i < len(accounts); i++ {
		require.True(t, SameAccounts(accounts[i], dbAccounts[i]))
	}

	// Check clock
	dbClock, err = db.GetClockOfLastAccountsPositionChange()
	require.NoError(t, err)
	require.Equal(t, clock, dbClock)
}

func TestGetWalletAddress(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	address := types.Address{0x01}
	_, err := db.GetWalletAddress()
	require.Equal(t, err, sql.ErrNoRows)
	require.NoError(t, db.SaveOrUpdateAccounts([]*Account{{Address: address, Wallet: true}}, false))
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
	require.NoError(t, db.SaveOrUpdateAccounts([]*Account{{Address: address, Chat: true}}, false))
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
	require.NoError(t, db.SaveOrUpdateAccounts(accounts, false))

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
	dbAccounts, err := db.GetActiveAccounts()
	require.NoError(t, err)
	require.Equal(t, 0, len(dbAccounts))

	woAccounts := GetWatchOnlyAccountsForTest()

	// try to save keypair with watch only accounts
	kp := &Keypair{}
	kp.Accounts = append(kp.Accounts, woAccounts...)
	err = db.SaveOrUpdateKeypair(kp)
	require.Error(t, err)

	// check the db after that trying to save keypair with watch only accounts
	dbAccounts, err = db.GetActiveAccounts()
	require.NoError(t, err)
	require.Equal(t, 0, len(dbAccounts))

	// save watch only accounts
	err = db.SaveOrUpdateAccounts(woAccounts, false)
	require.NoError(t, err)
	_, err = db.GetKeypairByKeyUID(woAccounts[0].KeyUID)
	require.Error(t, err)
	require.True(t, err == ErrDbKeypairNotFound)
	dbAccounts, err = db.GetActiveAccounts()
	require.NoError(t, err)
	require.Equal(t, len(woAccounts), len(dbAccounts))
	require.Equal(t, woAccounts[0].Address, dbAccounts[0].Address)

	// try to save the same watch only account again
	err = db.SaveOrUpdateAccounts(woAccounts[:1], false)
	require.NoError(t, err)
	dbAccounts, err = db.GetActiveAccounts()
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
	err = db.SaveOrUpdateAccounts([]*Account{wo4}, false)
	require.NoError(t, err)
	dbAccounts, err = db.GetActiveAccounts()
	require.NoError(t, err)
	require.Equal(t, len(woAccounts)+1, len(dbAccounts))
	dbAcc, err = db.GetAccountByAddress(wo4.Address)
	require.NoError(t, err)
	require.Equal(t, wo4.Address, dbAcc.Address)

	// updated watch onl to save the same account after it's saved
	wo4.Name = wo4.Name + "updated"
	wo4.ColorID = common.CustomizationColorCamel
	wo4.Emoji = wo4.Emoji + "updated"
	err = db.SaveOrUpdateAccounts([]*Account{wo4}, false)
	require.NoError(t, err)
	dbAccounts, err = db.GetActiveAccounts()
	require.NoError(t, err)
	require.Equal(t, len(woAccounts)+1, len(dbAccounts))
	dbAcc, err = db.GetAccountByAddress(wo4.Address)
	require.NoError(t, err)
	require.Equal(t, wo4.Address, dbAcc.Address)

	// try to delete keypair for watch only account
	err = db.RemoveKeypair(wo4.KeyUID, 0)
	require.Error(t, err)
	require.True(t, err == ErrDbKeypairNotFound)

	// try to delete watch only account
	err = db.RemoveAccount(wo4.Address, 0)
	require.NoError(t, err)
	dbAccounts, err = db.GetActiveAccounts()
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
	dbAccounts, err := db.GetActiveAccounts()
	require.NoError(t, err)
	require.Equal(t, 0, len(dbAccounts))

	// save keypair
	err = db.SaveOrUpdateKeypair(kp)
	require.NoError(t, err)
	dbKeypairs, err := db.GetActiveKeypairs()
	require.NoError(t, err)
	require.Equal(t, 1, len(dbKeypairs))
	require.True(t, SameKeypairs(kp, dbKeypairs[0]))

	// update keypair name
	kp.Name = kp.Name + "updated"
	kp.Accounts[0].Name = kp.Name
	err = db.UpdateKeypairName(kp.KeyUID, kp.Name, kp.Clock, true)
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
			dbKeypairs, err := db.GetActiveKeypairs()
			require.NoError(t, err)
			require.Equal(t, 0, len(dbKeypairs))
			dbAccounts, err := db.GetActiveAccounts()
			require.NoError(t, err)
			require.Equal(t, 0, len(dbAccounts))

			expectedLastUsedDerivationIndex := uint64(len(kp.Accounts) - 1)
			if kp.Type == KeypairTypeProfile {
				expectedLastUsedDerivationIndex-- // subtract one more in case of profile keypair because of chat account
			}

			// save keypair
			err = db.SaveOrUpdateKeypair(kp)
			require.NoError(t, err)
			dbKeypairs, err = db.GetActiveKeypairs()
			require.NoError(t, err)
			require.Equal(t, 1, len(dbKeypairs))
			dbKp, err := db.GetKeypairByKeyUID(kp.KeyUID)
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts), len(dbKp.Accounts))
			kp.LastUsedDerivationIndex = expectedLastUsedDerivationIndex
			require.Equal(t, kp.KeyUID, dbKp.KeyUID)
			dbAccounts, err = db.GetActiveAccounts()
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts), len(dbAccounts))

			// delete keypair
			err = db.RemoveKeypair(kp.KeyUID, 0)
			require.NoError(t, err)
			_, err = db.GetKeypairByKeyUID(kp.KeyUID)
			require.Error(t, err)
			require.True(t, err == ErrDbKeypairNotFound)

			// save keypair again to test the flow below
			err = db.SaveOrUpdateKeypair(kp)
			require.NoError(t, err)
			dbKeypairs, err = db.GetActiveKeypairs()
			require.NoError(t, err)
			require.Equal(t, 1, len(dbKeypairs))

			ind := len(kp.Accounts) - 1
			accToUpdate := kp.Accounts[ind]

			// try to save the same account again
			err = db.SaveOrUpdateAccounts([]*Account{accToUpdate}, false)
			require.NoError(t, err)
			dbKp, err = db.GetKeypairByKeyUID(kp.KeyUID)
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts), len(dbKp.Accounts))
			require.Equal(t, kp.KeyUID, dbKp.KeyUID)
			dbAccounts, err = db.GetActiveAccounts()
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts), len(dbAccounts))

			// update an existing account
			accToUpdate.Name = accToUpdate.Name + "updated"
			accToUpdate.ColorID = common.CustomizationColorBrown
			accToUpdate.Emoji = accToUpdate.Emoji + "updated"

			err = db.SaveOrUpdateAccounts([]*Account{accToUpdate}, false)
			require.NoError(t, err)
			dbKp, err = db.GetKeypairByKeyUID(kp.KeyUID)
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts), len(dbKp.Accounts))
			dbAccounts, err = db.GetActiveAccounts()
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
			dbKeypairs, err = db.GetActiveKeypairs()
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

			err = db.SaveOrUpdateAccounts([]*Account{accToAdd}, false)
			require.NoError(t, err)
			dbKp, err = db.GetKeypairByKeyUID(kp.KeyUID)
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts)+1, len(dbKp.Accounts))
			require.Equal(t, kp.LastUsedDerivationIndex, dbKp.LastUsedDerivationIndex)
			dbAccounts, err = db.GetActiveAccounts()
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

			err = db.SaveOrUpdateAccounts([]*Account{accToAdd}, false)
			require.NoError(t, err)
			dbKp, err = db.GetKeypairByKeyUID(kp.KeyUID)
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts)+2, len(dbKp.Accounts))
			require.Equal(t, expectedLastUsedDerivationIndex, dbKp.LastUsedDerivationIndex)
			dbAccounts, err = db.GetActiveAccounts()
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts)+2, len(dbAccounts))
			dbAcc, err = db.GetAccountByAddress(accToUpdate.Address)
			require.NoError(t, err)
			require.Equal(t, accToAdd.Address, dbAcc.Address)

			// delete account
			err = db.RemoveAccount(accToAdd.Address, 0)
			require.NoError(t, err)
			dbAccounts, err = db.GetActiveAccounts()
			require.NoError(t, err)
			require.Equal(t, len(kp.Accounts)+1, len(dbAccounts))
			_, err = db.GetAccountByAddress(accToAdd.Address)
			require.Error(t, err)
			require.True(t, err == ErrDbAccountNotFound)

			for _, acc := range dbAccounts {
				err = db.RemoveAccount(acc.Address, 0)
				require.NoError(t, err)
			}

			_, err = db.GetKeypairByKeyUID(kp.KeyUID)
			require.Error(t, err)
			require.True(t, err == ErrDbKeypairNotFound)
		})
	}
}
