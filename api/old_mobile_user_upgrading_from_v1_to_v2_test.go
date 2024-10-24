package api

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"go.uber.org/zap"

	d_common "github.com/status-im/status-go/common"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/common/dbsetup"
	"github.com/status-im/status-go/sqlite"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/common"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/t/utils"
)

const (
	oldMobileUserKeyUID = "0x855ab0a932e5325daab7a550b9fcd78d2a17de5e2b7a52241f82505ea9d87629"
	oldMobileUserPasswd = "0x20756cad9b728c8225fd8cedb6badaf8731e174506950219ea657cd54f35f46c" // #nosec G101

	// what we have in table `accounts` before test run:
	// 1. Address: 0x23A5CEF34B18920785F4B895849936F65CBDEF73
	//    Wallet: 1, Chat: 0, Type: '', Path: m/44'/60'/0'/0/0, Name: 'Main account', Derived_from: '', Pubkey: 0x047B67AD2...
	// 2. Address: 0x4851276E2B7DC3B8BEF1749127031BCB3578492D
	//    Wallet: 0, Chat: 1, Type: '', Path: m/43'/60'/1581'/0'/0, Name: 'Cadetblue Fuzzy Flickertailsquirrel', Derived_from: '', Pubkey: 0x04F96F6F5...
	// 3. Address: 0x4D26E5C2F85BA5D10BDA6B031E1C1579F8ECFA1F
	//    Wallet: 0, Chat: 0, Type: 'generated', Path: m/44'/60'/0'/1, Name: 'generated', Derived_from: '', Pubkey: 0x04488EDA7...
	// 4. Address: 0x516312D69737C5E6EF16F22E0097FF5D9F0C4196
	//    Wallet: 0, Chat: 0, Type: 'key', Path: m/44'/60'/0'/0/0, Name: 'key', Derived_from: '', Pubkey: 0x040D5E4E3...
	// 5. Address: 0x95222290DD7278AA3DDD389CC1E1D165CC4BAFE5
	//    Wallet: 0, Chat: 0, Type: 'watch', Path: '', Name: 'watch-only', Derived_from: '', Pubkey: <null>
	// 6. Address: 0xB7A1233D1309CE665A3A4DB088E4A046EB333545
	//    Wallet: 0, Chat: 0, Type: 'seed', Path: m/44'/60'/0'/0/0, Name: 'seed', Derived_from: '', Pubkey: 0x04FDE3E5...
	// seed phrase for 0xB7A1233D1309CE665A3A4DB088E4A046EB333545: vocal blouse script census island armor seek catch wool narrow peasant attract
	// private key for 0x516312D69737C5E6EF16F22E0097FF5D9F0C4196: c3ad0b50652318f845565c13761e5369ce75dcbc2a94616e15b829d4b07410fe
	// status account seed phrase: coin globe kit hamster notable proof orphan always mistake usual morning usage
	srcFolder = "../static/test-mobile-release-1.20.x-aa6e4b2-account/"
)

type OldMobileUserUpgradingFromV1ToV2Test struct {
	suite.Suite
	tmpdir string
	logger *zap.Logger
}

type PostLoginCheckCallback func(b *GethStatusBackend)

func (s *OldMobileUserUpgradingFromV1ToV2Test) SetupTest() {
	utils.Init()
	s.tmpdir = s.T().TempDir()
	copyDir(srcFolder, s.tmpdir, s.T())

	var err error
	s.logger, err = zap.NewDevelopment()
	s.Require().NoError(err)
}

func TestOldMobileUserUpgradingFromV1ToV2(t *testing.T) {
	suite.Run(t, new(OldMobileUserUpgradingFromV1ToV2Test))
}

func (s *OldMobileUserUpgradingFromV1ToV2Test) loginMobileUser(check PostLoginCheckCallback) {
	b := NewGethStatusBackend(s.logger)
	b.UpdateRootDataDir(s.tmpdir)
	s.Require().NoError(b.OpenAccounts())
	s.Require().NoError(b.Login(oldMobileUserKeyUID, oldMobileUserPasswd))

	check(b)

	s.Require().NoError(b.Logout())
}

func (s *OldMobileUserUpgradingFromV1ToV2Test) TestOptimizeMobileWakuV2SettingsForMobileV1() {
	bkFunc := d_common.IsMobilePlatform
	d_common.IsMobilePlatform = func() bool {
		return true
	}
	defer func() {
		d_common.IsMobilePlatform = bkFunc
	}()

	s.loginMobileUser(func(b *GethStatusBackend) {
		nc, err := b.GetNodeConfig()
		s.Require().NoError(err)
		s.Require().True(nc.WakuV2Config.LightClient)
		s.Require().False(nc.WakuV2Config.EnableStoreConfirmationForMessagesSent)
	})
}

func (s *OldMobileUserUpgradingFromV1ToV2Test) TestLoginAndMigrationsStillWorkWithExistingMobileUser() {
	checkAfterLogin := func(b *GethStatusBackend) {
		db, err := accounts.NewDB(b.appDB)
		s.Require().NoError(err)
		accs, err := db.GetAllAccounts()
		s.Require().NoError(err)
		s.Require().True(len(accs) == 6)
		kps, err := db.GetAllKeypairs()
		s.Require().NoError(err)
		s.Require().True(len(kps) == 3)

		// Create a map to categorize keypairs by their type
		keypairMap := make(map[accounts.KeypairType][]*accounts.Keypair)
		for _, kp := range kps {
			keypairMap[kp.Type] = append(keypairMap[kp.Type], kp)
		}

		// Check profile keypair
		profileKps, ok := keypairMap[accounts.KeypairTypeProfile]
		s.Require().True(ok, "Profile keypair not found")
		s.Require().True(len(profileKps) == 1, "Unexpected number of profile keypairs")
		s.Require().True(len(profileKps[0].Accounts) == 3)
		for _, a := range profileKps[0].Accounts {
			s.Require().Equal(a.KeyUID, oldMobileUserKeyUID)
		}

		generator := b.AccountManager().AccountsGenerator()
		// Check seed keypair
		seedKps, ok := keypairMap[accounts.KeypairTypeSeed]
		s.Require().True(ok, "Seed keypair not found")
		s.Require().True(len(seedKps) == 1, "Unexpected number of seed keypairs")
		s.Require().True(len(seedKps[0].Accounts) == 1)
		info, err := generator.LoadAccount(seedKps[0].Accounts[0].Address.Hex(), oldMobileUserPasswd)
		s.Require().NoError(err)
		s.Require().Equal(seedKps[0].KeyUID, info.KeyUID)
		s.Require().Equal(seedKps[0].Accounts[0].KeyUID, info.KeyUID)
		mnemonicNoExtraSpaces := strings.Join(strings.Fields("vocal blouse script census island armor seek catch wool narrow peasant attract"), " ")
		importedSeedAccountInfo, err := generator.ImportMnemonic(mnemonicNoExtraSpaces, "")
		s.Require().NoError(err)
		derivedAddresses, err := generator.DeriveAddresses(importedSeedAccountInfo.ID, paths)
		s.Require().NoError(err)
		s.Require().Equal(derivedAddresses[pathDefaultWallet].PublicKey, "0x04fde3e58a7379161da2adf033fbee076e2ba11fca8b07c4d06610b399911a60017e4c108eae243487d19e273f99c2d6af13ff5e330783f4389212092b01cc616c")
		//following line shows: we're unable to calculate the right KeyUID with the wrong public key from existing records for the imported seed account
		s.Require().False(importedSeedAccountInfo.KeyUID == seedKps[0].KeyUID)

		// Check key keypair
		keyKps, ok := keypairMap[accounts.KeypairTypeKey]
		s.Require().True(ok, "Key keypair not found")
		s.Require().True(len(keyKps) == 1, "Unexpected number of key keypairs")
		s.Require().True(len(keyKps[0].Accounts) == 1)
		info, err = generator.LoadAccount(keyKps[0].Accounts[0].Address.Hex(), oldMobileUserPasswd)
		s.Require().NoError(err)

		// The user should manually accept terms, so we make sure we don't set it
		// automatically by mistake.
		s.Require().False(info.ToMultiAccount().HasAcceptedTerms)

		s.Require().Equal(keyKps[0].KeyUID, info.KeyUID)
		s.Require().Equal(keyKps[0].Accounts[0].KeyUID, info.KeyUID)
		info, err = generator.ImportPrivateKey("c3ad0b50652318f845565c13761e5369ce75dcbc2a94616e15b829d4b07410fe")
		s.Require().NoError(err)
		s.Require().Equal(info.KeyUID, keyKps[0].KeyUID)
	}

	s.loginMobileUser(checkAfterLogin)
	s.loginMobileUser(checkAfterLogin) // Login twice to catch weird errors that only appear after logout
}

// TestAddWalletAccount we should be able to add a wallet account after upgrading from mobile v1
func (s *OldMobileUserUpgradingFromV1ToV2Test) TestAddWalletAccountAfterUpgradingFromMobileV1() {
	b := NewGethStatusBackend(s.logger)
	b.UpdateRootDataDir(s.tmpdir)
	s.Require().NoError(b.OpenAccounts())
	s.Require().NoError(b.Login(oldMobileUserKeyUID, oldMobileUserPasswd))
	db, _ := accounts.NewDB(b.appDB)
	walletRootAddress, err := db.GetWalletRootAddress()
	s.Require().NoError(err)
	masterAddress, err := db.GetMasterAddress()
	s.Require().NoError(err)

	kps, _ := db.GetAllKeypairs()
	// Create a map to categorize keypairs by their type
	keypairMap := make(map[accounts.KeypairType][]*accounts.Keypair)
	for _, kp := range kps {
		keypairMap[kp.Type] = append(keypairMap[kp.Type], kp)
	}
	profileKps := keypairMap[accounts.KeypairTypeProfile]
	profileKp := profileKps[0]
	s.Require().True(profileKp.DerivedFrom == walletRootAddress.Hex())
	s.Require().False(masterAddress.Hex() == walletRootAddress.Hex())
	s.T().Logf("masterAddress: %s, walletRootAddress: %s", masterAddress.Hex(), walletRootAddress.Hex())

	// simulate mobile frontend adding a wallet account
	suggestedPath, err := db.ResolveSuggestedPathForKeypair(oldMobileUserKeyUID)
	s.Require().NoError(err)
	generator := b.AccountManager().AccountsGenerator()
	accountInfo, err := generator.LoadAccount(profileKp.DerivedFrom, oldMobileUserPasswd)
	s.Require().NoError(err)
	infoMap, err := generator.DeriveAddresses(accountInfo.ID, []string{suggestedPath})
	s.Require().NoError(err)
	s.Require().Len(infoMap, 1)
	deriveAccountInfo := infoMap[suggestedPath]
	expectedDerivedAddress := "0xf44F8Ebc5b088e0eA8a0f7309A4a0c525AD783DA"
	s.Require().Equal(expectedDerivedAddress, deriveAccountInfo.Address)
	derivedAddress := types.HexToAddress(deriveAccountInfo.Address)
	accountsAPI := b.StatusNode().AccountService().AccountsAPI()
	err = accountsAPI.AddAccount(context.Background(), oldMobileUserPasswd, &accounts.Account{
		Address:   derivedAddress,
		KeyUID:    oldMobileUserKeyUID,
		Wallet:    false,
		Chat:      false,
		Type:      accounts.AccountTypeGenerated,
		Path:      suggestedPath,
		PublicKey: types.Hex2Bytes(deriveAccountInfo.PublicKey),
		Name:      "GeneratedAccount2",
		Emoji:     "emoji",
		ColorID:   common.CustomizationColorBlue,
	})
	s.Require().NoError(err)
	// need retry since there's a possible of getting "no key for given address or file" error
	err = tt.RetryWithBackOff(func() error {
		return accountsAPI.DeleteAccount(context.Background(), derivedAddress)
	})
	s.Require().NoError(err)
	s.Require().NoError(b.Logout())
}

func (s *OldMobileUserUpgradingFromV1ToV2Test) TestFixMissingKeyUIDForAccounts() {
	db, err := sqlite.OpenDB(sqlite.InMemoryPath, "1234567890", dbsetup.ReducedKDFIterationsNumber)
	s.Require().NoError(err)
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	s.Require().NoError(err)
	s.Require().ErrorContains(appdatabase.FixMissingKeyUIDForAccounts(tx), "no such table: accounts")
	s.Require().NoError(tx.Rollback())

	_, err = db.Exec(`
create table accounts
(
    address      VARCHAR not null primary key,
    wallet       BOOLEAN,
    chat         BOOLEAN,
    type         TEXT,
    storage      TEXT,
    pubkey       BLOB,
    path         TEXT,
    name         TEXT,
    color        TEXT,
    created_at   DATETIME           not null,
    updated_at   DATETIME           not null,
    hidden       BOOL default FALSE not null,
    emoji        TEXT default ""    not null,
    derived_from TEXT default ""    not null,
    clock        INT  default 0     not null
) without rowid;`)
	s.Require().NoError(err)
	tx, err = db.BeginTx(context.Background(), &sql.TxOptions{})
	s.Require().NoError(err)
	s.Require().ErrorContains(appdatabase.FixMissingKeyUIDForAccounts(tx), "no such table: settings")
	s.Require().NoError(tx.Rollback())

	_, err = db.Exec(`
create table settings
(
    address                               VARCHAR                    not null,
    key_uid                               VARCHAR                    not null,
    latest_derived_path                   UNSIGNED INT default 0,
    public_key                            VARCHAR                    not null,
    synthetic_id                          VARCHAR      default 'id'  not null primary key,
    wallet_root_address                   VARCHAR                    not null
) without rowid;`)
	s.Require().NoError(err)
	tx, err = db.BeginTx(context.Background(), &sql.TxOptions{})
	s.Require().NoError(err)
	// no rows in `settings` table, but we expect no error
	s.Require().NoError(appdatabase.FixMissingKeyUIDForAccounts(tx))
	s.Require().NoError(tx.Commit())
}
