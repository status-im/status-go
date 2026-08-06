package backend

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

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
	require.Equal(t, secretTestPassword, resolved.legacySecret)
	require.Equal(t, 256000, resolved.legacyKdfIter)
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

func TestOpenDBWithCredsFallback(t *testing.T) {
	b := newSecretTestBackend(t)

	openErr := errors.New("file is not a database")

	// Primary succeeds: fallback never attempted.
	attempts := []string{}
	db, err := b.openDBWithCredsFallback("app", dbCredentials{secret: "dek", kdfIter: 3200, fallbackSecret: "pw", fallbackKdfIter: 256000},
		func(secret string, kdfIter int) (*sql.DB, error) {
			attempts = append(attempts, secret)
			return nil, nil
		})
	require.NoError(t, err)
	require.Nil(t, db)
	require.Equal(t, []string{"dek"}, attempts)

	// Primary fails, fallback succeeds.
	attempts = nil
	_, err = b.openDBWithCredsFallback("app", dbCredentials{secret: "dek", kdfIter: 3200, fallbackSecret: "pw", fallbackKdfIter: 256000},
		func(secret string, kdfIter int) (*sql.DB, error) {
			attempts = append(attempts, secret)
			if secret == "dek" {
				return nil, openErr
			}
			return nil, nil
		})
	require.NoError(t, err)
	require.Equal(t, []string{"dek", "pw"}, attempts)

	// Both fail: the primary error is reported.
	attempts = nil
	fallbackErr := errors.New("also wrong")
	_, err = b.openDBWithCredsFallback("app", dbCredentials{secret: "dek", kdfIter: 3200, fallbackSecret: "pw", fallbackKdfIter: 256000},
		func(secret string, kdfIter int) (*sql.DB, error) {
			attempts = append(attempts, secret)
			if secret == "dek" {
				return nil, openErr
			}
			return nil, fallbackErr
		})
	require.ErrorIs(t, err, openErr)
	require.Equal(t, []string{"dek", "pw"}, attempts)

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
