package backend

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/accounts-management/keystore/envelope"
	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/internal/testutils"
)

const (
	zeroOnboardedGeneratedPassword = "0x20756cad9b728c8225fd8cedb6badaf8731e174506950219ea657cd54f35f46c"
	zeroOnboardedNewPassword       = "0x8a1bd4a7dd2a3f4d6c7b10de1c7ddf6b1f6f1a2c9e7b2e5d9a7c3b1f0e2d4c6b"
)

func newCreateAccountRequest(t *testing.T, rootDir string, zeroOnboarded bool) *requests.CreateAccount {
	t.Helper()
	requestJSON := fmt.Sprintf(`{
		"rootDataDir": %q,
		"password": %q,
		"displayName": "",
		"customizationColor": "blue",
		"wakuV2Nameserver": "1.1.1.1",
		"logFilePath": %q,
		"logLevel": "INFO",
		"networkId": 1
	}`, rootDir, zeroOnboardedGeneratedPassword, rootDir)
	var request requests.CreateAccount
	require.NoError(t, json.Unmarshal([]byte(requestJSON), &request))
	request.ZeroOnboarded = zeroOnboarded
	return &request
}

func requireStoredZeroOnboarded(t *testing.T, b *StatusBackend, keyUID string, expected bool) {
	t.Helper()
	acc, err := b.multiaccountsDB.GetAccount(keyUID)
	require.NoError(t, err)
	require.Equal(t, expected, acc.ZeroOnboarded)
}

// zeroOnboardedProfileLoggedInWithDEK logs in with the DEK, as the app does on every start of such a profile.
func zeroOnboardedProfileLoggedInWithDEK(t *testing.T) (b *StatusBackend, keyUID, dek string) {
	t.Helper()
	b = NewStatusBackend(testutils.MustCreateTestLogger())
	acc, err := b.CreateAccountAndLogin(newCreateAccountRequest(t, t.TempDir(), true))
	require.NoError(t, err)
	keyUID = acc.KeyUID
	require.True(t, acc.ZeroOnboarded)
	requireStoredZeroOnboarded(t, b, keyUID, true)

	dek, err = b.ExportProfileDEK(keyUID, zeroOnboardedGeneratedPassword)
	require.NoError(t, err)
	require.NoError(t, b.Logout())

	require.NoError(t, b.LoginAccount(&requests.Login{KeyUID: keyUID, DEK: dek}))
	active, err := b.GetActiveAccount()
	require.NoError(t, err)
	require.True(t, active.ZeroOnboarded)
	return b, keyUID, dek
}

func TestZeroOnboardedProfileSetPassword(t *testing.T) {
	b, keyUID, dek := zeroOnboardedProfileLoggedInWithDEK(t)

	require.NoError(t, b.SetZeroOnboardedProfilePassword(keyUID, zeroOnboardedNewPassword))

	requireStoredZeroOnboarded(t, b, keyUID, false)
	active, err := b.GetActiveAccount()
	require.NoError(t, err)
	require.False(t, active.ZeroOnboarded)

	unwrapped, _, err := envelope.Unwrap(b.rootDataDir, keyUID, zeroOnboardedNewPassword)
	require.NoError(t, err)
	require.Equal(t, dek, unwrapped)
	_, _, err = envelope.Unwrap(b.rootDataDir, keyUID, zeroOnboardedGeneratedPassword)
	require.ErrorIs(t, err, envelope.ErrInvalidKEK)

	ok, err := b.VerifyProfilePassword(zeroOnboardedNewPassword)
	require.NoError(t, err)
	require.True(t, ok)

	require.Error(t, b.SetZeroOnboardedProfilePassword(keyUID, "0xanother"))

	require.NoError(t, b.Logout())
	require.NoError(t, b.LoginAccount(&requests.Login{KeyUID: keyUID, Password: zeroOnboardedNewPassword}))
	active, err = b.GetActiveAccount()
	require.NoError(t, err)
	require.False(t, active.ZeroOnboarded)
	require.NoError(t, b.Logout())
	require.NoError(t, b.LoginAccount(&requests.Login{KeyUID: keyUID, DEK: dek}))
	require.NoError(t, b.Logout())
}

func TestZeroOnboardedSetPasswordRefusedForWrongKeyUID(t *testing.T) {
	b, keyUID, dek := zeroOnboardedProfileLoggedInWithDEK(t)
	defer func() { require.NoError(t, b.Logout()) }()

	require.Error(t, b.SetZeroOnboardedProfilePassword("0xnot-the-active-profile", zeroOnboardedNewPassword))
	requireStoredZeroOnboarded(t, b, keyUID, true)
	unwrapped, _, err := envelope.Unwrap(b.rootDataDir, keyUID, zeroOnboardedGeneratedPassword)
	require.NoError(t, err)
	require.Equal(t, dek, unwrapped)
}

func TestZeroOnboardedSetPasswordRefusedWithoutSession(t *testing.T) {
	b, keyUID, dek := zeroOnboardedProfileLoggedInWithDEK(t)
	require.NoError(t, b.Logout())

	require.Error(t, b.SetZeroOnboardedProfilePassword(keyUID, zeroOnboardedNewPassword))
	requireStoredZeroOnboarded(t, b, keyUID, true)
	unwrapped, _, err := envelope.Unwrap(b.rootDataDir, keyUID, zeroOnboardedGeneratedPassword)
	require.NoError(t, err)
	require.Equal(t, dek, unwrapped)
}

func TestZeroOnboardedSetPasswordRefusedForRegularProfile(t *testing.T) {
	b := NewStatusBackend(testutils.MustCreateTestLogger())
	acc, err := b.CreateAccountAndLogin(newCreateAccountRequest(t, t.TempDir(), false))
	require.NoError(t, err)
	defer func() { require.NoError(t, b.Logout()) }()
	require.False(t, acc.ZeroOnboarded)

	require.Error(t, b.SetZeroOnboardedProfilePassword(acc.KeyUID, zeroOnboardedNewPassword))
	_, _, err = envelope.Unwrap(b.rootDataDir, acc.KeyUID, zeroOnboardedGeneratedPassword)
	require.NoError(t, err)
}

func TestCreateZeroOnboardedRejectedWhenAccountExists(t *testing.T) {
	rootDir := t.TempDir()
	b := NewStatusBackend(testutils.MustCreateTestLogger())
	_, err := b.CreateAccountAndLogin(newCreateAccountRequest(t, rootDir, false))
	require.NoError(t, err)
	require.NoError(t, b.Logout())

	_, err = b.CreateAccountAndLogin(newCreateAccountRequest(t, rootDir, true))
	require.ErrorIs(t, err, ErrZeroOnboardedProfileNotAllowed)
	accounts, err := b.GetAccounts()
	require.NoError(t, err)
	require.Len(t, accounts, 1)
}

func TestCreateZeroOnboardedRejectedForKeycard(t *testing.T) {
	b := NewStatusBackend(testutils.MustCreateTestLogger())
	request := newCreateAccountRequest(t, t.TempDir(), true)
	request.KeycardInstanceUID = "keycard-instance"
	_, err := b.CreateAccountAndLogin(request)
	require.ErrorIs(t, err, ErrZeroOnboardedProfileNotAllowed)
}

func TestRestoreZeroOnboardedRejected(t *testing.T) {
	b := NewStatusBackend(testutils.MustCreateTestLogger())
	request := &requests.RestoreAccount{
		Mnemonic:      "test test test test test test test test test test test junk",
		CreateAccount: *newCreateAccountRequest(t, t.TempDir(), true),
	}
	_, err := b.RestoreAccountAndLogin(request)
	require.ErrorIs(t, err, ErrZeroOnboardedProfileNotAllowed)
	require.NoError(t, b.OpenAccounts())
	accounts, err := b.GetAccounts()
	require.NoError(t, err)
	require.Empty(t, accounts)
}

// A kept mark would let SetZeroOnboardedProfilePassword replace the user's password (keycard conversion included).
func TestChangeDatabasePasswordClearsZeroOnboarded(t *testing.T) {
	b, keyUID, dek := zeroOnboardedProfileLoggedInWithDEK(t)
	defer func() { require.NoError(t, b.Logout()) }()

	require.NoError(t, b.ChangeDatabasePassword(keyUID, clientHashOf(dek), zeroOnboardedNewPassword, false))

	requireStoredZeroOnboarded(t, b, keyUID, false)
	active, err := b.GetActiveAccount()
	require.NoError(t, err)
	require.False(t, active.ZeroOnboarded)
	require.ErrorIs(t, b.SetZeroOnboardedProfilePassword(keyUID, "0xanother"), ErrNotZeroOnboardedProfile)
}

func TestChangeDatabasePasswordLeavesRegularProfileMarkAlone(t *testing.T) {
	b := NewStatusBackend(testutils.MustCreateTestLogger())
	acc, err := b.CreateAccountAndLogin(newCreateAccountRequest(t, t.TempDir(), false))
	require.NoError(t, err)
	defer func() { require.NoError(t, b.Logout()) }()

	_, err = b.multiaccountsDB.DB().Exec(`CREATE TRIGGER block_mark_write BEFORE UPDATE OF zeroOnboarded ON accounts
		BEGIN SELECT RAISE(ABORT, 'mark write blocked'); END`)
	require.NoError(t, err)

	require.NoError(t, b.ChangeDatabasePassword(acc.KeyUID, zeroOnboardedGeneratedPassword, zeroOnboardedNewPassword, false))
	_, _, err = envelope.Unwrap(b.rootDataDir, acc.KeyUID, zeroOnboardedNewPassword)
	require.NoError(t, err)
}
