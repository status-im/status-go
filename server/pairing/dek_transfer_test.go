package pairing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	accsmanagementcommon "github.com/status-im/status-go/internal/accounts-management/common"
	keystorepkg "github.com/status-im/status-go/internal/accounts-management/keystore"
	"github.com/status-im/status-go/internal/accounts-management/keystore/envelope"
	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/internal/db/sqlite"
	"github.com/status-im/status-go/pkg/backend"
	"github.com/status-im/status-go/protocol/requests"
)

const (
	dekTransferKeyUID   = "0x1122334455667788990011223344556677889900112233445566778899001122"
	dekTransferPassword = "0xtransfer-password"
)

// prepareSenderProfileOnDisk creates a sender profile on disk: a keystore dir with one key
// file encrypted with keystoreSecret, and (when migrated) an envelope wrapping keystoreSecret
// as the DEK. Returns the sender config for an AccountPayloadLoader.
func prepareSenderProfileOnDisk(t *testing.T, rootDataDir, keystoreSecret string, migrated bool) *SenderConfig {
	t.Helper()

	keystorePath := filepath.Join(rootDataDir, backend.DefaultKeystoreRelativePath, dekTransferKeyUID)
	adapter, err := keystorepkg.NewGethKeystoreAdapter(keystorePath)
	require.NoError(t, err)

	privateKey, err := ethcrypto.GenerateKey()
	require.NoError(t, err)
	_, err = adapter.ImportECDSA(privateKey, keystoreSecret)
	require.NoError(t, err)

	if migrated {
		require.NoError(t, envelope.Write(rootDataDir, dekTransferKeyUID, keystoreSecret, dekTransferPassword, sqlite.ReducedKDFIterationsNumber))
	}

	db, err := multiaccounts.InitializeDB(filepath.Join(rootDataDir, "accounts.sql"))
	require.NoError(t, err)
	require.NoError(t, db.SaveAccount(multiaccounts.Account{
		Name:          "sender",
		KeyUID:        dekTransferKeyUID,
		KDFIterations: sqlite.ReducedKDFIterationsNumber,
	}))

	return &SenderConfig{
		KeystorePath: keystorePath,
		KeyUID:       dekTransferKeyUID,
		Password:     dekTransferPassword,
		DeviceType:   "desktop",
		DB:           db,
	}
}

// storePayloadOnReceiver runs the receiver side (AccountPayloadStorer) on the payload and
// returns the receiver's profile keystore path and multiaccounts DB.
func storePayloadOnReceiver(t *testing.T, p *AccountPayload, rootDataDir string) (string, *multiaccounts.Database) {
	t.Helper()

	db, err := multiaccounts.InitializeDB(filepath.Join(rootDataDir, "accounts.sql"))
	require.NoError(t, err)

	storer, err := NewAccountPayloadStorer(p, &ReceiverConfig{
		CreateAccount: &requests.CreateAccount{
			RootDataDir:   rootDataDir,
			KdfIterations: 256000, // legacy default coming from an (old) client
		},
		DB: db,
	})
	require.NoError(t, err)
	require.NoError(t, storer.Store())

	return filepath.Join(rootDataDir, backend.DefaultKeystoreRelativePath, dekTransferKeyUID), db
}

func requireReceiverAdoptedDEK(t *testing.T, rootDataDir, receiverKeystorePath string, db *multiaccounts.Database, senderSecret string) {
	t.Helper()

	// The receiver generated its own DEK, wrapped with the transfer password.
	require.True(t, envelope.Exists(rootDataDir, dekTransferKeyUID))
	receiverDek, dekIter, err := envelope.Unwrap(rootDataDir, dekTransferKeyUID, dekTransferPassword)
	require.NoError(t, err)
	require.Equal(t, sqlite.ReducedKDFIterationsNumber, dekIter)
	require.NotEqual(t, senderSecret, receiverDek)

	// The stored keystore files are encrypted with the receiver's DEK — not the password.
	require.NoError(t, keystorepkg.VerifyKeyStoreDirAtPath(receiverKeystorePath, receiverDek))
	require.Error(t, keystorepkg.VerifyKeyStoreDirAtPath(receiverKeystorePath, dekTransferPassword))

	// The saved account uses reduced kdf iterations (high-entropy DEK).
	kdfIterations, err := db.GetAccountKDFIterationsNumber(dekTransferKeyUID)
	require.NoError(t, err)
	require.Equal(t, sqlite.ReducedKDFIterationsNumber, kdfIterations)
}

// TestAccountPayloadTransferFromMigratedSender covers the migrated-sender → new-receiver
// account transfer: the sender re-encrypts its DEK-encrypted keystore files to the transfer
// password (the wire format, understood by receivers of ANY version — this is the
// new-sender → old-receiver compatibility guarantee), and the receiver adopts a fresh
// device-local DEK.
func TestAccountPayloadTransferFromMigratedSender(t *testing.T) {
	senderRoot := t.TempDir()
	senderDek, err := envelope.Generate()
	require.NoError(t, err)
	senderConfig := prepareSenderProfileOnDisk(t, senderRoot, senderDek, true)

	p := new(AccountPayload)
	loader, err := NewAccountPayloadLoader(p, senderConfig)
	require.NoError(t, err)
	require.NoError(t, loader.Load())

	// Wire format: every transferred key decrypts with the plain transfer password.
	require.NotEmpty(t, p.keys)
	for _, rawKey := range p.keys {
		_, err := accsmanagementcommon.DecryptKey(rawKey, dekTransferPassword)
		require.NoError(t, err)
	}

	// Sender's on-disk keystore is untouched (still DEK-encrypted).
	require.NoError(t, keystorepkg.VerifyKeyStoreDirAtPath(senderConfig.KeystorePath, senderDek))

	receiverRoot := t.TempDir()
	receiverKeystorePath, receiverDB := storePayloadOnReceiver(t, p, receiverRoot)
	requireReceiverAdoptedDEK(t, receiverRoot, receiverKeystorePath, receiverDB, senderDek)
}

// TestAccountPayloadTransferFromLegacySender covers the legacy-sender → new-receiver
// account transfer (an old app pairing to a new one): the wire format is unchanged and the
// receiver still adopts the DEK scheme.
func TestAccountPayloadTransferFromLegacySender(t *testing.T) {
	senderRoot := t.TempDir()
	// legacy: keystore files are encrypted directly with the password, no envelope
	senderConfig := prepareSenderProfileOnDisk(t, senderRoot, dekTransferPassword, false)

	p := new(AccountPayload)
	loader, err := NewAccountPayloadLoader(p, senderConfig)
	require.NoError(t, err)
	require.NoError(t, loader.Load())

	require.NotEmpty(t, p.keys)
	for _, rawKey := range p.keys {
		_, err := accsmanagementcommon.DecryptKey(rawKey, dekTransferPassword)
		require.NoError(t, err)
	}

	receiverRoot := t.TempDir()
	receiverKeystorePath, receiverDB := storePayloadOnReceiver(t, p, receiverRoot)
	requireReceiverAdoptedDEK(t, receiverRoot, receiverKeystorePath, receiverDB, dekTransferPassword)
}

// TestAccountPayloadStoreRetryAfterFailure verifies that a failure after the keystore
// directory was created (here: saving the account row into a closed multiaccounts DB)
// cleans up the new-profile state, so the pairing can be retried from scratch.
func TestAccountPayloadStoreRetryAfterFailure(t *testing.T) {
	senderRoot := t.TempDir()
	senderConfig := prepareSenderProfileOnDisk(t, senderRoot, dekTransferPassword, false)

	p := new(AccountPayload)
	loader, err := NewAccountPayloadLoader(p, senderConfig)
	require.NoError(t, err)
	require.NoError(t, loader.Load())

	receiverRoot := t.TempDir()
	brokenDB, err := multiaccounts.InitializeDB(filepath.Join(receiverRoot, "accounts.sql"))
	require.NoError(t, err)
	require.NoError(t, brokenDB.Close())

	storer, err := NewAccountPayloadStorer(p, &ReceiverConfig{
		CreateAccount: &requests.CreateAccount{
			RootDataDir:   receiverRoot,
			KdfIterations: 256000,
		},
		DB: brokenDB,
	})
	require.NoError(t, err)
	require.Error(t, storer.Store(), "storing the account row must fail on the closed DB")

	// The new-profile state was cleaned up: no keystore dir, no envelope.
	profileKeystorePath := filepath.Join(receiverRoot, backend.DefaultKeystoreRelativePath, dekTransferKeyUID)
	_, statErr := os.Stat(profileKeystorePath)
	require.True(t, os.IsNotExist(statErr))
	require.False(t, envelope.Exists(receiverRoot, dekTransferKeyUID))

	// A retry with a working DB succeeds end to end.
	receiverKeystorePath, receiverDB := storePayloadOnReceiver(t, p, receiverRoot)
	requireReceiverAdoptedDEK(t, receiverRoot, receiverKeystorePath, receiverDB, dekTransferPassword)
}

// TestAccountPayloadStoreRetryAfterKeyWriteFailure verifies that a key-file write failure
// inside storeKeys removes the profile directory this attempt created (storeKeys itself only
// empties it), keeping the pairing retryable.
func TestAccountPayloadStoreRetryAfterKeyWriteFailure(t *testing.T) {
	senderRoot := t.TempDir()
	senderConfig := prepareSenderProfileOnDisk(t, senderRoot, dekTransferPassword, false)

	p := new(AccountPayload)
	loader, err := NewAccountPayloadLoader(p, senderConfig)
	require.NoError(t, err)
	require.NoError(t, loader.Load())

	// Inject a valid key under an unwritable name (nested path) to make its write fail.
	for name, data := range p.keys {
		p.keys["unwritable/"+name] = data
		break
	}

	receiverRoot := t.TempDir()
	db, err := multiaccounts.InitializeDB(filepath.Join(receiverRoot, "accounts.sql"))
	require.NoError(t, err)

	storer, err := NewAccountPayloadStorer(p, &ReceiverConfig{
		CreateAccount: &requests.CreateAccount{
			RootDataDir:   receiverRoot,
			KdfIterations: 256000,
		},
		DB: db,
	})
	require.NoError(t, err)
	require.Error(t, storer.Store(), "writing the injected key file must fail")

	// The directory created by this attempt is gone, so the retry starts from scratch.
	profileKeystorePath := filepath.Join(receiverRoot, backend.DefaultKeystoreRelativePath, dekTransferKeyUID)
	_, statErr := os.Stat(profileKeystorePath)
	require.True(t, os.IsNotExist(statErr))
	require.False(t, envelope.Exists(receiverRoot, dekTransferKeyUID))

	// A retry without the bad entry succeeds end to end.
	for name := range p.keys {
		if strings.HasPrefix(name, "unwritable/") {
			delete(p.keys, name)
		}
	}
	receiverKeystorePath, receiverDB := storePayloadOnReceiver(t, p, receiverRoot)
	requireReceiverAdoptedDEK(t, receiverRoot, receiverKeystorePath, receiverDB, dekTransferPassword)
}

// TestValidateAndVerifyPasswordMigratedSender ensures the sender-side config validation
// accepts a migrated profile (whose keystore no longer decrypts with the raw password).
func TestValidateAndVerifyPasswordMigratedSender(t *testing.T) {
	senderRoot := t.TempDir()
	senderDek, err := envelope.Generate()
	require.NoError(t, err)
	senderConfig := prepareSenderProfileOnDisk(t, senderRoot, senderDek, true)

	err = validateAndVerifyPassword(&SenderServerConfig{SenderConfig: senderConfig, ServerConfig: new(ServerConfig)}, senderConfig)
	require.NoError(t, err)

	// A wrong password must still be rejected.
	wrongConfig := *senderConfig
	wrongConfig.Password = "0xwrong-password"
	err = validateAndVerifyPassword(&SenderServerConfig{SenderConfig: &wrongConfig, ServerConfig: new(ServerConfig)}, &wrongConfig)
	require.Error(t, err)
}
