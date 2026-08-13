package backend

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	keystorepkg "github.com/status-im/status-go/internal/accounts-management/keystore"
	"github.com/status-im/status-go/internal/accounts-management/keystore/envelope"
	types "github.com/status-im/status-go/internal/crypto/types"
	settings "github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/db/sqlite"
	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/internal/signal"
)

// requireDBOpensWith asserts that the sqlcipher database at path opens with the given key.
func requireDBOpensWith(t *testing.T, path, key string, kdfIter int) {
	t.Helper()
	db, err := sqlite.OpenDB(path, key, kdfIter)
	require.NoError(t, err)
	require.NoError(t, db.Close())
}

func requireDBDoesNotOpenWith(t *testing.T, path, key string, kdfIter int) {
	t.Helper()
	db, err := sqlite.OpenDB(path, key, kdfIter)
	if err == nil {
		_ = db.Close()
	}
	require.Error(t, err)
}

// TestChangeDatabasePasswordMigratesToDEK verifies that the first password change of a
// legacy profile performs the one-time migration to the DEK scheme, and that subsequent
// changes take the fast (re-wrap only) path without touching the databases.
func TestChangeDatabasePasswordMigratesToDEK(t *testing.T) {
	testContext := setupTestContext(t, testPassword, true, true, false)
	b := testContext.backend
	keyUID := testContext.profileKeypair.KeyUID
	masterAddress := types.HexToAddress(testContext.profileKeypair.DerivedFrom)

	require.NoError(t, b.StartNode(testContext.config))
	defer func() {
		require.NoError(t, b.StopNode())
	}()

	require.False(t, envelope.Exists(b.rootDataDir, keyUID))

	const password1 = "new-password-1"
	require.NoError(t, b.ChangeDatabasePassword(keyUID, testPassword, password1, false))
	b.UpdateRootDataDir(testContext.config.RootDataDir)

	// The envelope exists, unwraps with the new password only, and prescribes reduced kdf iterations.
	require.True(t, envelope.Exists(b.rootDataDir, keyUID))
	dek, dekIter, err := envelope.Unwrap(b.rootDataDir, keyUID, password1)
	require.NoError(t, err)
	require.Equal(t, sqlite.ReducedKDFIterationsNumber, dekIter)
	_, _, err = envelope.Unwrap(b.rootDataDir, keyUID, testPassword)
	require.ErrorIs(t, err, envelope.ErrInvalidKEK)

	// The kdfIterations column follows.
	columnIter, err := b.multiaccountsDB.GetAccountKDFIterationsNumber(keyUID)
	require.NoError(t, err)
	require.Equal(t, sqlite.ReducedKDFIterationsNumber, columnIter)

	// Both databases are now encrypted with the DEK.
	appDBPath, err := b.getAppDBPath(keyUID)
	require.NoError(t, err)
	walletDBPath, err := b.getWalletDBPath(keyUID)
	require.NoError(t, err)
	requireDBOpensWith(t, appDBPath, dek, sqlite.ReducedKDFIterationsNumber)
	requireDBOpensWith(t, walletDBPath, dek, sqlite.ReducedKDFIterationsNumber)
	requireDBDoesNotOpenWith(t, appDBPath, testPassword, 1)

	// The keystore resolves through the DEK with the new password.
	ok, err := b.AccountsManager().VerifyAccountPassword(masterAddress, password1)
	require.NoError(t, err)
	require.True(t, ok)

	// Second change: fast path — the database files must not be rewritten.
	appStatBefore, err := os.Stat(appDBPath)
	require.NoError(t, err)
	walletStatBefore, err := os.Stat(walletDBPath)
	require.NoError(t, err)

	const password2 = "new-password-2"
	require.NoError(t, b.ChangeDatabasePassword(keyUID, password1, password2, false))

	appStatAfter, err := os.Stat(appDBPath)
	require.NoError(t, err)
	walletStatAfter, err := os.Stat(walletDBPath)
	require.NoError(t, err)
	require.Equal(t, appStatBefore.ModTime(), appStatAfter.ModTime())
	require.Equal(t, walletStatBefore.ModTime(), walletStatAfter.ModTime())

	// Same DEK, new wrapper.
	dekAfter, _, err := envelope.Unwrap(b.rootDataDir, keyUID, password2)
	require.NoError(t, err)
	require.Equal(t, dek, dekAfter)
	_, _, err = envelope.Unwrap(b.rootDataDir, keyUID, password1)
	require.ErrorIs(t, err, envelope.ErrInvalidKEK)

	// Wrong old password is rejected by the fast path.
	require.ErrorIs(t, b.ChangeDatabasePassword(keyUID, "wrong-password", "whatever", false), envelope.ErrInvalidKEK)

	// Keystore operations follow the new password.
	ok, err = b.AccountsManager().VerifyAccountPassword(masterAddress, password2)
	require.NoError(t, err)
	require.True(t, ok)
}

// TestMigrationRejectsWrongOldPassword verifies that a migration attempt with a wrong old
// password fails before anything is persisted: no envelope may be left behind (it would be
// wrapped with an unknown KEK and make the profile unopenable with the real password).
func TestMigrationRejectsWrongOldPassword(t *testing.T) {
	testContext := setupTestContext(t, testPassword, true, true, false)
	b := testContext.backend
	keyUID := testContext.profileKeypair.KeyUID
	masterAddress := types.HexToAddress(testContext.profileKeypair.DerivedFrom)

	require.NoError(t, b.StartNode(testContext.config))
	defer func() {
		require.NoError(t, b.StopNode())
	}()

	require.False(t, envelope.Exists(b.rootDataDir, keyUID))

	require.Error(t, b.ChangeDatabasePassword(keyUID, "wrong-password", "new-password", false))
	b.UpdateRootDataDir(testContext.config.RootDataDir)

	// Nothing was persisted: no envelope, databases and keystore still on the old password.
	require.False(t, envelope.Exists(b.rootDataDir, keyUID))
	kdfIter, err := b.multiaccountsDB.GetAccountKDFIterationsNumber(keyUID)
	require.NoError(t, err)
	appDBPath, err := b.getAppDBPath(keyUID)
	require.NoError(t, err)
	requireDBOpensWith(t, appDBPath, testPassword, kdfIter)
	ok, err := b.AccountsManager().VerifyAccountPassword(masterAddress, testPassword)
	require.NoError(t, err)
	require.True(t, ok)

	// The profile is intact: migration with the correct password still works.
	require.NoError(t, b.ChangeDatabasePassword(keyUID, testPassword, "new-password", false))
	require.True(t, envelope.Exists(b.rootDataDir, keyUID))
	_, _, err = envelope.Unwrap(b.rootDataDir, keyUID, "new-password")
	require.NoError(t, err)
}

// TestChangeDatabasePasswordRekey verifies that a deep rekey rotates the DEK and
// re-encrypts the databases and keystore with it.
func TestChangeDatabasePasswordRekey(t *testing.T) {
	testContext := setupTestContext(t, testPassword, true, true, false)
	b := testContext.backend
	keyUID := testContext.profileKeypair.KeyUID
	masterAddress := types.HexToAddress(testContext.profileKeypair.DerivedFrom)

	require.NoError(t, b.StartNode(testContext.config))
	defer func() {
		require.NoError(t, b.StopNode())
	}()

	const password1 = "new-password-1"
	require.NoError(t, b.ChangeDatabasePassword(keyUID, testPassword, password1, false))
	b.UpdateRootDataDir(testContext.config.RootDataDir)
	dek1, _, err := envelope.Unwrap(b.rootDataDir, keyUID, password1)
	require.NoError(t, err)

	const password2 = "new-password-2"
	require.NoError(t, b.ChangeDatabasePassword(keyUID, password1, password2, true))
	b.UpdateRootDataDir(testContext.config.RootDataDir)

	// Fresh DEK, committed envelope, no pending leftover.
	dek2, _, err := envelope.Unwrap(b.rootDataDir, keyUID, password2)
	require.NoError(t, err)
	require.NotEqual(t, dek1, dek2)
	require.False(t, envelope.PendingExists(b.rootDataDir, keyUID))

	// Databases are on the new DEK; the old one no longer opens them.
	appDBPath, err := b.getAppDBPath(keyUID)
	require.NoError(t, err)
	walletDBPath, err := b.getWalletDBPath(keyUID)
	require.NoError(t, err)
	requireDBOpensWith(t, appDBPath, dek2, sqlite.ReducedKDFIterationsNumber)
	requireDBOpensWith(t, walletDBPath, dek2, sqlite.ReducedKDFIterationsNumber)
	requireDBDoesNotOpenWith(t, appDBPath, dek1, sqlite.ReducedKDFIterationsNumber)

	// Keystore follows.
	ok, err := b.AccountsManager().VerifyAccountPassword(masterAddress, password2)
	require.NoError(t, err)
	require.True(t, ok)
}

// TestCreateAccountUsesDEKFromDayOne verifies that newly created profiles are on the DEK
// scheme immediately, with reduced kdf iterations.
func TestCreateAccountUsesDEKFromDayOne(t *testing.T) {
	testContext := setupTestContext(t, testPassword, false, false, true)

	createAccountRequest := &requests.CreateAccount{
		DisplayName:        "some-display-name",
		CustomizationColor: "#ffffff",
		Password:           testPassword,
		RootDataDir:        testContext.config.RootDataDir,
		LogFilePath:        testContext.config.RootDataDir + "/log",
	}

	c := make(chan interface{}, 10)
	signal.SetMobileSignalHandler(func(data []byte) {
		if strings.Contains(string(data), "node.login") {
			c <- struct{}{}
		}
	})
	t.Cleanup(signal.ResetMobileSignalHandler)

	account, err := testContext.backend.CreateAccountAndLogin(createAccountRequest)
	require.NoError(t, err)
	<-c

	b := testContext.backend
	require.True(t, envelope.Exists(b.rootDataDir, account.KeyUID))
	require.True(t, b.ProfileEncryptionInfo(account.KeyUID))
	require.Equal(t, sqlite.ReducedKDFIterationsNumber, account.KDFIterations)

	columnIter, err := b.multiaccountsDB.GetAccountKDFIterationsNumber(account.KeyUID)
	require.NoError(t, err)
	require.Equal(t, sqlite.ReducedKDFIterationsNumber, columnIter)

	dek, _, err := envelope.Unwrap(b.rootDataDir, account.KeyUID, testPassword)
	require.NoError(t, err)

	appDBPath, err := b.getAppDBPath(account.KeyUID)
	require.NoError(t, err)
	requireDBOpensWith(t, appDBPath, dek, sqlite.ReducedKDFIterationsNumber)

	require.NoError(t, b.Logout())
	require.NoError(t, b.StopNode())
}

// reEncryptDBFileForTest re-encrypts a closed sqlcipher database file in place.
func reEncryptDBFileForTest(t *testing.T, path, oldKey string, oldIter int, newKey string, newIter int) {
	t.Helper()
	tmpPath := path + ".reencrypted"
	require.NoError(t, sqlite.ExportDBWithKDFChange(path, oldKey, oldIter, tmpPath, newKey, newIter, nil, nil))
	require.NoError(t, os.Rename(tmpPath, path))
	_ = os.Remove(path + "-wal")
	_ = os.Remove(path + "-shm")
}

// migratedFixture returns a profile already migrated to the DEK scheme (logged out), with
// the password it is wrapped with and its DEK.
func migratedFixture(t *testing.T) (testContext *setupContext, password, dek string) {
	t.Helper()
	testContext = setupTestContext(t, testPassword, true, true, false)
	b := testContext.backend
	keyUID := testContext.profileKeypair.KeyUID

	password = "migrated-password"
	require.NoError(t, b.ChangeDatabasePassword(keyUID, testPassword, password, false))
	b.UpdateRootDataDir(testContext.config.RootDataDir)
	var err error
	dek, _, err = envelope.Unwrap(b.rootDataDir, keyUID, password)
	require.NoError(t, err)

	require.NoError(t, b.Logout())
	return testContext, password, dek
}

func profilePaths(t *testing.T, b *StatusBackend, keyUID string) (appDBPath, walletDBPath, keystoreDir string) {
	t.Helper()
	var err error
	appDBPath, err = b.getAppDBPath(keyUID)
	require.NoError(t, err)
	walletDBPath, err = b.getWalletDBPath(keyUID)
	require.NoError(t, err)
	_, keystoreDir = DefaultKeystorePath(b.rootDataDir, keyUID)
	return
}

// TestRepairMigrationInterruptedAfterKeystore simulates a crash after the migration
// re-encrypted the keystore (databases still on the raw password). This is also the state
// left behind when keystore re-encryption returns an ambiguous error after installing the
// files. Repair must roll the profile back to a clean legacy state.
func TestRepairMigrationInterruptedAfterKeystore(t *testing.T) {
	testContext := setupTestContext(t, testPassword, true, true, false)
	b := testContext.backend
	keyUID := testContext.profileKeypair.KeyUID
	_, _, keystoreDir := profilePaths(t, b, keyUID)

	dek, err := envelope.Generate()
	require.NoError(t, err)
	require.NoError(t, envelope.Write(b.rootDataDir, keyUID, dek, testPassword, sqlite.ReducedKDFIterationsNumber))
	require.NoError(t, keystorepkg.ReEncryptKeyStoreDirAtPath(keystoreDir, testPassword, dek))

	require.NoError(t, b.Logout())
	require.NoError(t, b.ensureDBsOpened(*testContext.multiAcc, testPassword))
	appSecret, _ := b.dbOpenSecrets()
	require.Equal(t, testPassword, appSecret)

	require.NoError(t, b.repairProfileEncryption(keyUID, testPassword, testContext.multiAcc.KDFIterations))

	require.False(t, envelope.Exists(b.rootDataDir, keyUID))
	require.NoError(t, keystorepkg.VerifyKeyStoreDirAtPath(keystoreDir, testPassword))
	require.NoError(t, b.Logout())
}

// TestRepairMigrationInterruptedBetweenSwaps simulates a crash between the two database
// swaps: app DB on the DEK, wallet DB still on the raw password. Repair must realign the
// wallet DB forward to the DEK.
func TestRepairMigrationInterruptedBetweenSwaps(t *testing.T) {
	testContext := setupTestContext(t, testPassword, true, true, false)
	b := testContext.backend
	keyUID := testContext.profileKeypair.KeyUID
	appDBPath, walletDBPath, keystoreDir := profilePaths(t, b, keyUID)

	require.NoError(t, b.Logout())

	dek, err := envelope.Generate()
	require.NoError(t, err)
	require.NoError(t, envelope.Write(b.rootDataDir, keyUID, dek, testPassword, sqlite.ReducedKDFIterationsNumber))
	require.NoError(t, keystorepkg.ReEncryptKeyStoreDirAtPath(keystoreDir, testPassword, dek))
	reEncryptDBFileForTest(t, appDBPath, testPassword, testContext.multiAcc.KDFIterations, dek, sqlite.ReducedKDFIterationsNumber)

	require.NoError(t, b.ensureDBsOpened(*testContext.multiAcc, testPassword))
	appSecret, walletSecret := b.dbOpenSecrets()
	require.Equal(t, dek, appSecret)
	require.Equal(t, testPassword, walletSecret)

	require.NoError(t, b.repairProfileEncryption(keyUID, testPassword, testContext.multiAcc.KDFIterations))

	// Fully on the DEK now; envelope kept (still wrapped with the login password).
	require.True(t, envelope.Exists(b.rootDataDir, keyUID))
	require.NoError(t, keystorepkg.VerifyKeyStoreDirAtPath(keystoreDir, dek))
	require.NoError(t, b.Logout())
	requireDBOpensWith(t, walletDBPath, dek, sqlite.ReducedKDFIterationsNumber)
}

// TestRepairMigrationInterruptedBeforeCommit simulates a crash after both swaps but before
// the envelope was re-wrapped to the new password: a fully consistent DEK profile still
// wrapped with the old password. Login with the old password must work without repair.
func TestRepairMigrationInterruptedBeforeCommit(t *testing.T) {
	testContext := setupTestContext(t, testPassword, true, true, false)
	b := testContext.backend
	keyUID := testContext.profileKeypair.KeyUID
	appDBPath, walletDBPath, keystoreDir := profilePaths(t, b, keyUID)

	require.NoError(t, b.Logout())

	dek, err := envelope.Generate()
	require.NoError(t, err)
	require.NoError(t, envelope.Write(b.rootDataDir, keyUID, dek, testPassword, sqlite.ReducedKDFIterationsNumber))
	require.NoError(t, keystorepkg.ReEncryptKeyStoreDirAtPath(keystoreDir, testPassword, dek))
	reEncryptDBFileForTest(t, appDBPath, testPassword, testContext.multiAcc.KDFIterations, dek, sqlite.ReducedKDFIterationsNumber)
	reEncryptDBFileForTest(t, walletDBPath, testPassword, testContext.multiAcc.KDFIterations, dek, sqlite.ReducedKDFIterationsNumber)

	require.NoError(t, b.ensureDBsOpened(*testContext.multiAcc, testPassword))
	appSecret, walletSecret := b.dbOpenSecrets()
	require.Equal(t, dek, appSecret)
	require.Equal(t, dek, walletSecret)

	require.NoError(t, b.repairProfileEncryption(keyUID, testPassword, testContext.multiAcc.KDFIterations))

	// Nothing to repair: consistent DEK profile, old password stays valid.
	require.True(t, envelope.Exists(b.rootDataDir, keyUID))
	unwrapped, _, err := envelope.Unwrap(b.rootDataDir, keyUID, testPassword)
	require.NoError(t, err)
	require.Equal(t, dek, unwrapped)
	require.NoError(t, b.Logout())
}

// TestRepairRekeyInterruptedAfterPendingWrite simulates a crash right after the rekey wrote
// the pending envelope (nothing else changed). Repair must drop the pending envelope.
func TestRepairRekeyInterruptedAfterPendingWrite(t *testing.T) {
	testContext, password, dek1 := migratedFixture(t)
	b := testContext.backend
	keyUID := testContext.profileKeypair.KeyUID
	_, _, keystoreDir := profilePaths(t, b, keyUID)

	dek2, err := envelope.Generate()
	require.NoError(t, err)
	require.NoError(t, envelope.WritePending(b.rootDataDir, keyUID, dek2, password, sqlite.ReducedKDFIterationsNumber))

	require.NoError(t, b.ensureDBsOpened(*testContext.multiAcc, password))
	require.NoError(t, b.repairProfileEncryption(keyUID, password, testContext.multiAcc.KDFIterations))

	require.False(t, envelope.PendingExists(b.rootDataDir, keyUID))
	unwrapped, _, err := envelope.Unwrap(b.rootDataDir, keyUID, password)
	require.NoError(t, err)
	require.Equal(t, dek1, unwrapped)
	require.NoError(t, keystorepkg.VerifyKeyStoreDirAtPath(keystoreDir, dek1))
	require.NoError(t, b.Logout())
}

// TestRepairRekeyInterruptedBetweenSwaps simulates a crash after the rekey swapped the app
// DB and re-encrypted the keystore, with the wallet DB still on the old DEK. Repair must
// roll everything forward to the new DEK and commit it.
func TestRepairRekeyInterruptedBetweenSwaps(t *testing.T) {
	testContext, password, dek1 := migratedFixture(t)
	b := testContext.backend
	keyUID := testContext.profileKeypair.KeyUID
	appDBPath, walletDBPath, keystoreDir := profilePaths(t, b, keyUID)

	dek2, err := envelope.Generate()
	require.NoError(t, err)
	require.NoError(t, envelope.WritePending(b.rootDataDir, keyUID, dek2, password, sqlite.ReducedKDFIterationsNumber))
	reEncryptDBFileForTest(t, appDBPath, dek1, sqlite.ReducedKDFIterationsNumber, dek2, sqlite.ReducedKDFIterationsNumber)
	require.NoError(t, keystorepkg.ReEncryptKeyStoreDirAtPath(keystoreDir, dek1, dek2))

	require.NoError(t, b.ensureDBsOpened(*testContext.multiAcc, password))
	appSecret, walletSecret := b.dbOpenSecrets()
	require.Equal(t, dek2, appSecret)
	require.Equal(t, dek1, walletSecret)

	require.NoError(t, b.repairProfileEncryption(keyUID, password, testContext.multiAcc.KDFIterations))

	require.False(t, envelope.PendingExists(b.rootDataDir, keyUID))
	unwrapped, _, err := envelope.Unwrap(b.rootDataDir, keyUID, password)
	require.NoError(t, err)
	require.Equal(t, dek2, unwrapped)
	require.NoError(t, keystorepkg.VerifyKeyStoreDirAtPath(keystoreDir, dek2))
	require.NoError(t, b.Logout())
	requireDBOpensWith(t, walletDBPath, dek2, sqlite.ReducedKDFIterationsNumber)
}

// TestRepairRekeyInterruptedAfterCommit simulates a crash after the rekey committed the
// main envelope under the NEW password but before removing the pending envelope. A login
// with the OLD password must still succeed (via the pending envelope) and repair must
// re-commit the main envelope under that password.
func TestRepairRekeyInterruptedAfterCommit(t *testing.T) {
	testContext, password, dek1 := migratedFixture(t)
	b := testContext.backend
	keyUID := testContext.profileKeypair.KeyUID
	appDBPath, walletDBPath, keystoreDir := profilePaths(t, b, keyUID)

	const newPassword = "committed-new-password"
	dek2, err := envelope.Generate()
	require.NoError(t, err)
	reEncryptDBFileForTest(t, appDBPath, dek1, sqlite.ReducedKDFIterationsNumber, dek2, sqlite.ReducedKDFIterationsNumber)
	reEncryptDBFileForTest(t, walletDBPath, dek1, sqlite.ReducedKDFIterationsNumber, dek2, sqlite.ReducedKDFIterationsNumber)
	require.NoError(t, keystorepkg.ReEncryptKeyStoreDirAtPath(keystoreDir, dek1, dek2))
	require.NoError(t, envelope.Write(b.rootDataDir, keyUID, dek2, newPassword, sqlite.ReducedKDFIterationsNumber))
	require.NoError(t, envelope.WritePending(b.rootDataDir, keyUID, dek2, password, sqlite.ReducedKDFIterationsNumber))

	// Login with the OLD password: resolution succeeds via the pending envelope.
	require.NoError(t, b.ensureDBsOpened(*testContext.multiAcc, password))
	appSecret, _ := b.dbOpenSecrets()
	require.Equal(t, dek2, appSecret)

	require.NoError(t, b.repairProfileEncryption(keyUID, password, testContext.multiAcc.KDFIterations))

	// The old password is committed as the KEK again; pending is gone.
	require.False(t, envelope.PendingExists(b.rootDataDir, keyUID))
	unwrapped, _, err := envelope.Unwrap(b.rootDataDir, keyUID, password)
	require.NoError(t, err)
	require.Equal(t, dek2, unwrapped)
	require.NoError(t, b.Logout())
}

// demigrateProfileForTest converts a DEK-native profile (logged out) back to the legacy
// scheme, as if it had been created by an older app version.
func demigrateProfileForTest(t *testing.T, b *StatusBackend, keyUID, password string) {
	t.Helper()
	dek, dekIter, err := envelope.Unwrap(b.rootDataDir, keyUID, password)
	require.NoError(t, err)
	appDBPath, walletDBPath, keystoreDir := profilePaths(t, b, keyUID)
	reEncryptDBFileForTest(t, appDBPath, dek, dekIter, password, sqlite.ReducedKDFIterationsNumber)
	reEncryptDBFileForTest(t, walletDBPath, dek, dekIter, password, sqlite.ReducedKDFIterationsNumber)
	require.NoError(t, keystorepkg.ReEncryptKeyStoreDirAtPath(keystoreDir, dek, password))
	require.NoError(t, envelope.Remove(b.rootDataDir, keyUID))
	b.clearProfileSecretCache()
}

// TestConvertAccountLegacyProfile verifies the keycard conversion of a LEGACY profile: the
// conversion performs the one-time migration to the DEK scheme, wrapped with the keycard
// encryption key. (TestConvertAccount covers the DEK-native profile: both of its
// conversions take the fast re-wrap path.)
func TestConvertAccountLegacyProfile(t *testing.T) {
	testContext := setupTestContext(t, testPassword, false, false, true)
	b := testContext.backend

	request := &requests.CreateAccount{
		RootDataDir:   testContext.config.RootDataDir,
		Password:      testPassword,
		KdfIterations: 1,
	}
	_, err := b.StartNodeWithChatKeyOrMnemonic(request, testContext.mnemonic, nil, false)
	require.NoError(t, err)

	accountsList, err := b.GetAccounts()
	require.NoError(t, err)
	require.Len(t, accountsList, 1)
	multiAcc := accountsList[0]
	keyUID := multiAcc.KeyUID

	// Bring the profile back to the legacy scheme, as if created by an older version.
	require.NoError(t, b.Logout())
	require.NoError(t, b.StopNode())
	demigrateProfileForTest(t, b, keyUID, testPassword)
	require.False(t, b.ProfileEncryptionInfo(keyUID))

	chatPrivKey := strings.TrimPrefix(testContext.chatPrivateKey, "0x")
	require.NoError(t, b.StartNodeWithKey(multiAcc, testPassword, chatPrivKey, testContext.config))
	defer func() {
		_ = b.Logout()
		_ = b.StopNode()
	}()

	keycardAccount := multiAcc
	keycardAccount.KeycardPairing = "pairing"
	const keycardPassword = "222222"
	require.NoError(t, b.ConvertToKeycardAccount(keycardAccount, settings.Settings{}, keyUID, testPassword, keycardPassword))

	// The conversion migrated the profile to the DEK scheme, wrapped with the keycard key.
	require.True(t, b.ProfileEncryptionInfo(keyUID))
	dek, dekIter, err := envelope.Unwrap(b.rootDataDir, keyUID, keycardPassword)
	require.NoError(t, err)
	require.Equal(t, sqlite.ReducedKDFIterationsNumber, dekIter)
	_, _, err = envelope.Unwrap(b.rootDataDir, keyUID, testPassword)
	require.ErrorIs(t, err, envelope.ErrInvalidKEK)

	appDBPath, walletDBPath, _ := profilePaths(t, b, keyUID)
	requireDBOpensWith(t, appDBPath, dek, sqlite.ReducedKDFIterationsNumber)
	requireDBOpensWith(t, walletDBPath, dek, sqlite.ReducedKDFIterationsNumber)
}

// TestChangeDatabasePasswordCrashInjection drives the REAL migration/rekey flows and
// simulates a process crash at every mutation boundary via reEncryptionCrashHook, then
// verifies that the recovery a fresh app start performs (open databases with fallbacks +
// login-time repair) always converges to a consistent state that the OLD password opens.
func TestChangeDatabasePasswordCrashInjection(t *testing.T) {
	cases := []struct {
		stage string
		rekey bool
		// wantLegacy: recovery rolls the profile back to a clean legacy state
		wantLegacy bool
		// wantNewDek: recovery converges forward onto a rotated DEK (rekey post-swap stages)
		wantNewDek bool
	}{
		{stage: "migrate:envelope-written", wantLegacy: true},
		{stage: "migrate:keystore-reencrypted", wantLegacy: true},
		{stage: "migrate:dbs-exported", wantLegacy: true},
		{stage: "migrate:app-swapped"},
		{stage: "migrate:dbs-swapped"},
		{stage: "rekey:pending-written", rekey: true},
		{stage: "rekey:dbs-exported", rekey: true},
		{stage: "rekey:keystore-reencrypted", rekey: true},
		{stage: "rekey:app-swapped", rekey: true, wantNewDek: true},
		{stage: "rekey:dbs-swapped", rekey: true, wantNewDek: true},
		{stage: "rekey:committed", rekey: true, wantNewDek: true},
	}

	for _, tc := range cases {
		t.Run(tc.stage, func(t *testing.T) {
			testContext := setupTestContext(t, testPassword, true, true, false)
			b := testContext.backend
			keyUID := testContext.profileKeypair.KeyUID
			appDBPath, walletDBPath, keystoreDir := profilePaths(t, b, keyUID)

			require.NoError(t, b.StartNode(testContext.config))
			defer func() {
				_ = b.StopNode()
			}()

			oldPassword := testPassword
			dekBefore := ""
			if tc.rekey {
				// bring the profile onto the DEK scheme first
				oldPassword = "rekey-old-password"
				require.NoError(t, b.ChangeDatabasePassword(keyUID, testPassword, oldPassword, false))
				b.UpdateRootDataDir(testContext.config.RootDataDir)
				var err error
				dekBefore, _, err = envelope.Unwrap(b.rootDataDir, keyUID, oldPassword)
				require.NoError(t, err)
			}

			reEncryptionCrashHook = func(stage string) error {
				if stage == tc.stage {
					return errors.New("injected crash at " + stage)
				}
				return nil
			}
			t.Cleanup(func() { reEncryptionCrashHook = nil })

			err := b.ChangeDatabasePassword(keyUID, oldPassword, "crash-new-password", tc.rekey)
			require.ErrorContains(t, err, "injected crash")
			reEncryptionCrashHook = nil

			// Recovery, as a fresh app start would do it: open + repair with the OLD password.
			_ = b.Logout()
			require.NoError(t, b.ensureDBsOpened(*testContext.multiAcc, oldPassword))
			require.NoError(t, b.repairProfileEncryption(keyUID, oldPassword, testContext.multiAcc.KDFIterations))
			require.NoError(t, b.Logout())

			if tc.wantLegacy {
				require.False(t, envelope.Exists(b.rootDataDir, keyUID))
				require.NoError(t, keystorepkg.VerifyKeyStoreDirAtPath(keystoreDir, oldPassword))
				requireDBOpensWith(t, appDBPath, oldPassword, testContext.multiAcc.KDFIterations)
				requireDBOpensWith(t, walletDBPath, oldPassword, testContext.multiAcc.KDFIterations)
				return
			}

			require.False(t, envelope.PendingExists(b.rootDataDir, keyUID))
			dekNow, dekIter, err := envelope.Unwrap(b.rootDataDir, keyUID, oldPassword)
			require.NoError(t, err)
			require.Equal(t, sqlite.ReducedKDFIterationsNumber, dekIter)
			if tc.rekey {
				if tc.wantNewDek {
					require.NotEqual(t, dekBefore, dekNow)
				} else {
					require.Equal(t, dekBefore, dekNow)
				}
			}
			require.NoError(t, keystorepkg.VerifyKeyStoreDirAtPath(keystoreDir, dekNow))
			requireDBOpensWith(t, appDBPath, dekNow, sqlite.ReducedKDFIterationsNumber)
			requireDBOpensWith(t, walletDBPath, dekNow, sqlite.ReducedKDFIterationsNumber)
		})
	}
}

// TestRepairInterruptedMigration simulates a crash right after the migration wrote the
// envelope (databases and keystore still on the raw password) and verifies that login
// falls back and repair rolls the profile back to a clean legacy state.
func TestRepairInterruptedMigration(t *testing.T) {
	testContext := setupTestContext(t, testPassword, true, true, false)
	b := testContext.backend
	keyUID := testContext.profileKeypair.KeyUID

	// Simulate the crash state: envelope present, everything else untouched.
	dek, err := envelope.Generate()
	require.NoError(t, err)
	require.NoError(t, envelope.Write(b.rootDataDir, keyUID, dek, testPassword, sqlite.ReducedKDFIterationsNumber))

	require.NoError(t, b.Logout())

	// Login path: databases open via the raw-password fallback.
	require.NoError(t, b.ensureDBsOpened(*testContext.multiAcc, testPassword))
	appSecret, walletSecret := b.dbOpenSecrets()
	require.Equal(t, testPassword, appSecret)
	require.Equal(t, testPassword, walletSecret)

	require.NoError(t, b.repairProfileEncryption(keyUID, testPassword, testContext.multiAcc.KDFIterations))

	// Rolled back to a clean legacy profile.
	require.False(t, envelope.Exists(b.rootDataDir, keyUID))

	// Keystore still decrypts with the raw password.
	_, keystoreDir := DefaultKeystorePath(b.rootDataDir, keyUID)
	require.NoError(t, keystorepkg.ReEncryptKeyStoreDirAtPath(keystoreDir, testPassword, testPassword))

	require.NoError(t, b.Logout())
}

// TestRepairInterruptedRekey simulates a crash after a rekey swapped the databases and
// re-encrypted the keystore to the new DEK, but before the new envelope was committed.
// Login must succeed via the pending envelope and repair must commit it.
func TestRepairInterruptedRekey(t *testing.T) {
	testContext := setupTestContext(t, testPassword, true, true, false)
	b := testContext.backend
	keyUID := testContext.profileKeypair.KeyUID

	// Migrate to the DEK scheme first.
	const password1 = "new-password-1"
	require.NoError(t, b.ChangeDatabasePassword(keyUID, testPassword, password1, false))
	b.UpdateRootDataDir(testContext.config.RootDataDir)
	dek1, _, err := envelope.Unwrap(b.rootDataDir, keyUID, password1)
	require.NoError(t, err)

	require.NoError(t, b.Logout())

	// Simulate the rekey crash state: pending envelope + databases + keystore on a new
	// DEK, while the main envelope still holds the old one.
	dek2, err := envelope.Generate()
	require.NoError(t, err)
	require.NoError(t, envelope.WritePending(b.rootDataDir, keyUID, dek2, password1, sqlite.ReducedKDFIterationsNumber))

	reEncryptInPlace := func(path string) {
		tmpPath := path + ".rekeyed"
		require.NoError(t, sqlite.ExportDBWithKDFChange(path, dek1, sqlite.ReducedKDFIterationsNumber,
			tmpPath, dek2, sqlite.ReducedKDFIterationsNumber, nil, nil))
		require.NoError(t, os.Rename(tmpPath, path))
		_ = os.Remove(path + "-wal")
		_ = os.Remove(path + "-shm")
	}
	appDBPath, err := b.getAppDBPath(keyUID)
	require.NoError(t, err)
	walletDBPath, err := b.getWalletDBPath(keyUID)
	require.NoError(t, err)
	reEncryptInPlace(appDBPath)
	reEncryptInPlace(walletDBPath)

	_, keystoreDir := DefaultKeystorePath(b.rootDataDir, keyUID)
	require.NoError(t, keystorepkg.ReEncryptKeyStoreDirAtPath(keystoreDir, dek1, dek2))

	// Login with the (old) password: primary resolve yields dek1, databases open via the
	// pending-DEK fallback.
	require.NoError(t, b.ensureDBsOpened(*testContext.multiAcc, password1))
	appSecret, walletSecret := b.dbOpenSecrets()
	require.Equal(t, dek2, appSecret)
	require.Equal(t, dek2, walletSecret)

	require.NoError(t, b.repairProfileEncryption(keyUID, password1, testContext.multiAcc.KDFIterations))

	// The pending envelope was committed: dek2 is now the main DEK, wrapped with the password.
	require.False(t, envelope.PendingExists(b.rootDataDir, keyUID))
	dekNow, _, err := envelope.Unwrap(b.rootDataDir, keyUID, password1)
	require.NoError(t, err)
	require.Equal(t, dek2, dekNow)

	// Keystore is on dek2 (probe: re-encrypt dek2 → dek2 succeeds).
	require.NoError(t, keystorepkg.ReEncryptKeyStoreDirAtPath(keystoreDir, dek2, dek2))

	require.NoError(t, b.Logout())
}
