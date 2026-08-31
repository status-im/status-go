package backend

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/accounts-management/keystore/envelope"
	"github.com/status-im/status-go/internal/db/sqlite"
	"github.com/status-im/status-go/internal/protocol/requests"
)

// TestDEKLoginPrepare verifies the happy path of a biometric (DEK) login: the request is
// turned into an ordinary password login riding on the primed session cache, the databases
// open with the DEK, and the login-time encryption repair is a no-op (the envelope keeps
// unwrapping with the real password).
func TestDEKLoginPrepare(t *testing.T) {
	testContext, password, dek := migratedFixture(t)
	b := testContext.backend
	keyUID := testContext.profileKeypair.KeyUID

	request := &requests.Login{KeyUID: keyUID, DEK: dek}
	require.NoError(t, request.Validate())
	require.NoError(t, b.prepareDEKLogin(request))
	require.Equal(t, dek, request.Password)

	// The primed cache serves the DEK and its client-hashed form, with no KEK known.
	resolved, err := b.resolveProfileSecret(keyUID, dek, 0)
	require.NoError(t, err)
	require.True(t, resolved.migrated)
	require.Equal(t, dek, resolved.secret)
	resolved, err = b.resolveProfileSecret(keyUID, clientHashOf(dek), 0)
	require.NoError(t, err)
	require.Equal(t, dek, resolved.secret)

	require.NoError(t, b.ensureDBsOpened(*testContext.multiAcc, request.Password))
	require.NoError(t, b.repairProfileEncryption(keyUID, request.Password, testContext.multiAcc.KDFIterations))

	require.True(t, envelope.Exists(b.rootDataDir, keyUID))
	unwrapped, _, err := envelope.Unwrap(b.rootDataDir, keyUID, password)
	require.NoError(t, err)
	require.Equal(t, dek, unwrapped)

	require.NoError(t, b.Logout())
}

// TestDEKLoginWrongDEK verifies that a wrong (but well-formed) DEK fails like a wrong
// password: the databases simply do not open.
func TestDEKLoginWrongDEK(t *testing.T) {
	testContext, password, dek := migratedFixture(t)
	b := testContext.backend
	keyUID := testContext.profileKeypair.KeyUID

	wrongDEK, err := envelope.Generate()
	require.NoError(t, err)
	require.NotEqual(t, dek, wrongDEK)

	request := &requests.Login{KeyUID: keyUID, DEK: wrongDEK}
	require.NoError(t, b.prepareDEKLogin(request))
	require.Error(t, b.ensureDBsOpened(*testContext.multiAcc, request.Password))

	// The envelope is untouched and the real password still works.
	b.clearProfileSecretCache()
	unwrapped, _, err := envelope.Unwrap(b.rootDataDir, keyUID, password)
	require.NoError(t, err)
	require.Equal(t, dek, unwrapped)
}

func TestDEKLoginRejectedForLegacyProfile(t *testing.T) {
	b := newSecretTestBackend(t)

	dek, err := envelope.Generate()
	require.NoError(t, err)

	request := &requests.Login{KeyUID: secretTestKeyUID, DEK: dek}
	require.ErrorContains(t, b.prepareDEKLogin(request), "not on the DEK encryption scheme")
}

// TestDEKLoginRejectedWhenRekeyPending: a DEK login must never run the interrupted-rekey
// repair — with the DEK as "password" the repair would remove the envelope and permanently
// lock the real KEK out. Such a state requires a password login.
func TestDEKLoginRejectedWhenRekeyPending(t *testing.T) {
	testContext, password, dek := migratedFixture(t)
	b := testContext.backend
	keyUID := testContext.profileKeypair.KeyUID

	pendingDEK, err := envelope.Generate()
	require.NoError(t, err)
	require.NoError(t, envelope.WritePending(b.rootDataDir, keyUID, pendingDEK, password, sqlite.ReducedKDFIterationsNumber))

	request := &requests.Login{KeyUID: keyUID, DEK: dek}
	require.ErrorContains(t, b.prepareDEKLogin(request), "interrupted rekey pending")
	require.True(t, envelope.PendingExists(b.rootDataDir, keyUID))
}

// TestFastPasswordChangeWithSessionCredential verifies the fast-path fallback: a caller
// holding only a session credential (the client-hashed DEK, e.g. from biometrics) can still
// re-wrap the envelope to a new password, consistent with the endpoint's session-based
// password pre-check.
func TestFastPasswordChangeWithSessionCredential(t *testing.T) {
	testContext, password, dek := migratedFixture(t)
	b := testContext.backend
	keyUID := testContext.profileKeypair.KeyUID

	// Warm the session as a login would.
	_, err := b.resolveProfileSecret(keyUID, password, testContext.multiAcc.KDFIterations)
	require.NoError(t, err)

	const newPassword = "new-password-after-biometric-auth"
	require.NoError(t, b.ChangeDatabasePassword(keyUID, clientHashOf(dek), newPassword, false))

	// The envelope unwraps with the new KEK only; the DEK is unchanged.
	unwrapped, _, err := envelope.Unwrap(b.rootDataDir, keyUID, newPassword)
	require.NoError(t, err)
	require.Equal(t, dek, unwrapped)
	_, _, err = envelope.Unwrap(b.rootDataDir, keyUID, password)
	require.ErrorIs(t, err, envelope.ErrInvalidKEK)

	// The refreshed KEK fingerprint accepts the new password in-session.
	resolved, err := b.resolveProfileSecret(keyUID, newPassword, 0)
	require.NoError(t, err)
	require.Equal(t, dek, resolved.secret)
}
