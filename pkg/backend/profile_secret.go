package backend

import (
	"crypto/sha256"
	"crypto/subtle"
	"sync"

	"github.com/status-im/status-go/internal/accounts-management/keystore/envelope"
)

// profileSecretCache remembers the unwrapped DEK of the profile bound to the current session.
// The KEK itself is not stored — only its sha256 fingerprint, enough to re-verify an incoming password.
type profileSecretCache struct {
	mu             sync.Mutex
	keyUID         string
	kekFingerprint []byte
	dekHex         string
	dbKdfIter      int
}

// resolvedProfileSecret is the outcome of translating a client-provided password into the secret.
type resolvedProfileSecret struct {
	secret    string
	dbKdfIter int
	migrated  bool // true when the profile uses the DEK encryption scheme.
	// legacySecret/legacyKdfIter are here for the crash-recovery fallback
	legacySecret  string
	legacyKdfIter int
}

// resolveProfileSecret translates (keyUID, password) into the secret used for the databases and keystore files.
// It returns either the secret or password if the profile is not migrated and an error if the password is wrong.
func (b *StatusBackend) resolveProfileSecret(keyUID, password string, kdfIterationsFallback int) (resolvedProfileSecret, error) {
	legacy := resolvedProfileSecret{
		secret:        password,
		dbKdfIter:     kdfIterationsFallback,
		legacySecret:  password,
		legacyKdfIter: kdfIterationsFallback,
	}

	c := &b.secretCache
	c.mu.Lock()
	if c.keyUID == keyUID && c.dekHex != "" {
		fingerprint := sha256.Sum256([]byte(password))
		if subtle.ConstantTimeCompare(fingerprint[:], c.kekFingerprint) == 1 ||
			subtle.ConstantTimeCompare([]byte(password), []byte(c.dekHex)) == 1 {
			resolved := resolvedProfileSecret{
				secret:        c.dekHex,
				dbKdfIter:     c.dbKdfIter,
				migrated:      true,
				legacySecret:  password,
				legacyKdfIter: kdfIterationsFallback,
			}
			c.mu.Unlock()
			return resolved, nil
		}
	}
	c.mu.Unlock()

	if !envelope.Exists(b.rootDataDir, keyUID) {
		return legacy, nil
	}

	dekHex, dbKdfIter, err := envelope.Unwrap(b.rootDataDir, keyUID, password)
	if err != nil {
		return resolvedProfileSecret{}, err
	}

	fingerprint := sha256.Sum256([]byte(password))
	c.mu.Lock()
	c.keyUID = keyUID
	c.kekFingerprint = fingerprint[:]
	c.dekHex = dekHex
	c.dbKdfIter = dbKdfIter
	c.mu.Unlock()

	return resolvedProfileSecret{
		secret:        dekHex,
		dbKdfIter:     dbKdfIter,
		migrated:      true,
		legacySecret:  password,
		legacyKdfIter: kdfIterationsFallback,
	}, nil
}

// clearProfileSecretCache clears the cached secret.
func (b *StatusBackend) clearProfileSecretCache() {
	c := &b.secretCache
	c.mu.Lock()
	c.keyUID = ""
	c.kekFingerprint = nil
	c.dekHex = ""
	c.dbKdfIter = 0
	c.mu.Unlock()
}

// setSecretResolverForProfile points the accounts manager's keystore operations at this profile's secret resolution.
func (b *StatusBackend) setSecretResolverForProfile(keyUID string, kdfIterations int) {
	b.accountsManager.SetSecretResolver(func(password string) (string, error) {
		resolved, err := b.resolveProfileSecret(keyUID, password, kdfIterations)
		if err != nil {
			return "", err
		}
		return resolved.secret, nil
	})
}
