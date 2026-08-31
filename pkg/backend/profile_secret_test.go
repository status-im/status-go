package backend

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/accounts-management/keystore"
	"github.com/status-im/status-go/internal/accounts-management/keystore/envelope"
)

const (
	secretTestKeyUID   = "0xaabbccddeeff00112233445566778899aabbccddeeff001122334455667788aa"
	secretTestPassword = "0x8a9f7d2b6c1e4f3a5d8b9c0e2f4a6b8d0c2e4f6a8b0d2c4e6f8a0b2d4c6e8f0a"
)

func newSecretTestBackend(t *testing.T) *StatusBackend {
	t.Helper()
	return &StatusBackend{
		rootDataDir: t.TempDir(),
		logger:      zap.NewNop(),
	}
}

func TestResolveProfileSecretLegacyProfile(t *testing.T) {
	b := newSecretTestBackend(t)

	// No wrapped-DEK file: the password is the secret, untouched.
	resolved, err := b.resolveProfileSecret(secretTestKeyUID, secretTestPassword, 256000)
	require.NoError(t, err)
	require.False(t, resolved.migrated)
	require.Equal(t, secretTestPassword, resolved.secret)
	require.Equal(t, 256000, resolved.dbKdfIter)
}

func TestResolveProfileSecretMigratedProfile(t *testing.T) {
	b := newSecretTestBackend(t)

	dek, err := envelope.Generate()
	require.NoError(t, err)
	require.NoError(t, envelope.Write(b.rootDataDir, secretTestKeyUID, dek, secretTestPassword, 3200))

	resolved, err := b.resolveProfileSecret(secretTestKeyUID, secretTestPassword, 256000)
	require.NoError(t, err)
	require.True(t, resolved.migrated)
	require.Equal(t, dek, resolved.secret)
	require.Equal(t, 3200, resolved.dbKdfIter)
	require.Equal(t, []dbFallbackCredential{{secret: secretTestPassword, kdfIter: 256000}}, resolved.fallbacks)
}

func TestResolveProfileSecretWrongPassword(t *testing.T) {
	b := newSecretTestBackend(t)

	dek, err := envelope.Generate()
	require.NoError(t, err)
	require.NoError(t, envelope.Write(b.rootDataDir, secretTestKeyUID, dek, secretTestPassword, 3200))

	_, err = b.resolveProfileSecret(secretTestKeyUID, "0xwrong", 256000)
	require.ErrorIs(t, err, envelope.ErrInvalidKEK)

	// A wrong password after a successful (cached) resolution must still fail.
	_, err = b.resolveProfileSecret(secretTestKeyUID, secretTestPassword, 256000)
	require.NoError(t, err)
	_, err = b.resolveProfileSecret(secretTestKeyUID, "0xwrong", 256000)
	require.ErrorIs(t, err, envelope.ErrInvalidKEK)
}

func TestResolveProfileSecretCache(t *testing.T) {
	b := newSecretTestBackend(t)

	dek, err := envelope.Generate()
	require.NoError(t, err)
	require.NoError(t, envelope.Write(b.rootDataDir, secretTestKeyUID, dek, secretTestPassword, 3200))

	resolved, err := b.resolveProfileSecret(secretTestKeyUID, secretTestPassword, 256000)
	require.NoError(t, err)
	require.Equal(t, dek, resolved.secret)

	// Remove the file: a cached resolution must still succeed (no disk access).
	require.NoError(t, envelope.Remove(b.rootDataDir, secretTestKeyUID))
	resolved, err = b.resolveProfileSecret(secretTestKeyUID, secretTestPassword, 256000)
	require.NoError(t, err)
	require.True(t, resolved.migrated)
	require.Equal(t, dek, resolved.secret)

	// Resolution is idempotent: passing the resolved secret back in returns it.
	resolved, err = b.resolveProfileSecret(secretTestKeyUID, dek, 256000)
	require.NoError(t, err)
	require.Equal(t, dek, resolved.secret)

	// After clearing the cache the profile is treated as legacy again
	// (file was removed above).
	b.clearProfileSecretCache()
	resolved, err = b.resolveProfileSecret(secretTestKeyUID, secretTestPassword, 256000)
	require.NoError(t, err)
	require.False(t, resolved.migrated)
	require.Equal(t, secretTestPassword, resolved.secret)
}

func TestResolveProfileSecretCacheIsPerProfile(t *testing.T) {
	b := newSecretTestBackend(t)

	dek, err := envelope.Generate()
	require.NoError(t, err)
	require.NoError(t, envelope.Write(b.rootDataDir, secretTestKeyUID, dek, secretTestPassword, 3200))

	_, err = b.resolveProfileSecret(secretTestKeyUID, secretTestPassword, 256000)
	require.NoError(t, err)

	// Same password, different profile: must not hit the cache.
	otherKeyUID := "0x99" + secretTestKeyUID[4:]
	resolved, err := b.resolveProfileSecret(otherKeyUID, secretTestPassword, 256000)
	require.NoError(t, err)
	require.False(t, resolved.migrated)
	require.Equal(t, secretTestPassword, resolved.secret)
}

func TestResolveProfileSecretAcceptsClientHashedDEKWhenWarm(t *testing.T) {
	b := newSecretTestBackend(t)

	dek, err := envelope.Generate()
	require.NoError(t, err)
	require.NoError(t, envelope.Write(b.rootDataDir, secretTestKeyUID, dek, secretTestPassword, 3200))

	// Cold: the client-hashed DEK is not a valid KEK.
	_, err = b.resolveProfileSecret(secretTestKeyUID, clientHashOf(dek), 0)
	require.ErrorIs(t, err, envelope.ErrInvalidKEK)

	// Warm the cache with the real password; the client-hashed DEK then resolves.
	_, err = b.resolveProfileSecret(secretTestKeyUID, secretTestPassword, 0)
	require.NoError(t, err)
	resolved, err := b.resolveProfileSecret(secretTestKeyUID, clientHashOf(dek), 0)
	require.NoError(t, err)
	require.True(t, resolved.migrated)
	require.Equal(t, dek, resolved.secret)

	// Cleared again: rejected again.
	b.clearProfileSecretCache()
	_, err = b.resolveProfileSecret(secretTestKeyUID, clientHashOf(dek), 0)
	require.ErrorIs(t, err, envelope.ErrInvalidKEK)
}

func TestPrimeSecretCacheWithDEK(t *testing.T) {
	b := newSecretTestBackend(t)

	dek, err := envelope.Generate()
	require.NoError(t, err)
	require.NoError(t, envelope.Write(b.rootDataDir, secretTestKeyUID, dek, secretTestPassword, 3200))

	b.primeSecretCacheWithDEK(secretTestKeyUID, dek, 3200)

	// The raw DEK and its client hash both resolve from the primed cache.
	resolved, err := b.resolveProfileSecret(secretTestKeyUID, dek, 0)
	require.NoError(t, err)
	require.True(t, resolved.migrated)
	require.Equal(t, dek, resolved.secret)
	require.Equal(t, 3200, resolved.dbKdfIter)

	resolved, err = b.resolveProfileSecret(secretTestKeyUID, clientHashOf(dek), 0)
	require.NoError(t, err)
	require.Equal(t, dek, resolved.secret)

	// No KEK is known for a primed session, so the real password takes the cold path,
	// re-verifies against the envelope and restores the KEK fingerprint.
	resolved, err = b.resolveProfileSecret(secretTestKeyUID, secretTestPassword, 0)
	require.NoError(t, err)
	require.Equal(t, dek, resolved.secret)

	// Wrong values still fail.
	_, err = b.resolveProfileSecret(secretTestKeyUID, "0xwrong", 0)
	require.ErrorIs(t, err, envelope.ErrInvalidKEK)
}

func TestExportProfileDEK(t *testing.T) {
	b := newSecretTestBackend(t)

	// Legacy profile: the password must never be echoed back as a "DEK".
	_, err := b.ExportProfileDEK(secretTestKeyUID, secretTestPassword)
	require.ErrorIs(t, err, ErrProfileNotMigratedToDEK)

	dek, err := envelope.Generate()
	require.NoError(t, err)
	require.NoError(t, envelope.Write(b.rootDataDir, secretTestKeyUID, dek, secretTestPassword, 3200))

	// Correct KEK, cold.
	exported, err := b.ExportProfileDEK(secretTestKeyUID, secretTestPassword)
	require.NoError(t, err)
	require.Equal(t, dek, exported)

	// Session credential (client-hashed DEK), warm.
	exported, err = b.ExportProfileDEK(secretTestKeyUID, clientHashOf(dek))
	require.NoError(t, err)
	require.Equal(t, dek, exported)

	// Wrong KEK, cold: reported as an incorrect password.
	b.clearProfileSecretCache()
	_, err = b.ExportProfileDEK(secretTestKeyUID, "0xwrong")
	require.ErrorIs(t, err, keystore.ErrIncorrectPasswordProvided)
}

func TestOpenDBWithCredsFallback(t *testing.T) {
	b := newSecretTestBackend(t)

	openErr := errors.New("file is not a database")
	credsWithFallbacks := dbCredentials{secret: "dek", kdfIter: 3200, fallbacks: []dbFallbackCredential{
		{secret: "pw", kdfIter: 256000},
		{secret: "pendingdek", kdfIter: 3200},
	}}

	// Primary succeeds: fallbacks never attempted.
	attempts := []string{}
	db, err := b.openDBWithCredsFallback("app", credsWithFallbacks,
		func(secret string, kdfIter int) (*sql.DB, error) {
			attempts = append(attempts, secret)
			return nil, nil
		})
	require.NoError(t, err)
	require.Nil(t, db)
	require.Equal(t, []string{"dek"}, attempts)
	usedApp, _ := b.dbOpenSecrets()
	require.Equal(t, "dek", usedApp)

	// Primary fails, second fallback succeeds; the winning secret is recorded.
	attempts = nil
	_, err = b.openDBWithCredsFallback("app", credsWithFallbacks,
		func(secret string, kdfIter int) (*sql.DB, error) {
			attempts = append(attempts, secret)
			if secret == "pendingdek" {
				return nil, nil
			}
			return nil, openErr
		})
	require.NoError(t, err)
	require.Equal(t, []string{"dek", "pw", "pendingdek"}, attempts)
	usedApp, _ = b.dbOpenSecrets()
	require.Equal(t, "pendingdek", usedApp)

	// All fail: the primary error is reported.
	attempts = nil
	_, err = b.openDBWithCredsFallback("app", credsWithFallbacks,
		func(secret string, kdfIter int) (*sql.DB, error) {
			attempts = append(attempts, secret)
			return nil, errors.New("wrong key: " + secret)
		})
	require.ErrorContains(t, err, "wrong key: dek")
	require.Equal(t, []string{"dek", "pw", "pendingdek"}, attempts)

	// No fallback credentials (legacy profile): single attempt.
	attempts = nil
	_, err = b.openDBWithCredsFallback("app", dbCredentials{secret: "pw", kdfIter: 256000},
		func(secret string, kdfIter int) (*sql.DB, error) {
			attempts = append(attempts, secret)
			return nil, openErr
		})
	require.ErrorIs(t, err, openErr)
	require.Equal(t, []string{"pw"}, attempts)
}
