package backend

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"sync"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-go/internal/accounts-management/keystore"
	"github.com/status-im/status-go/internal/accounts-management/keystore/envelope"
	"github.com/status-im/status-go/internal/protocol/requests"
)

var ErrProfileNotMigratedToDEK = errors.New("profile is not migrated to the DEK encryption scheme")

// profileSecretCache remembers the unwrapped DEK of the profile bound to the current session.
// The KEK itself is not stored — only its sha256 fingerprint, enough to re-verify an incoming password.
type profileSecretCache struct {
	mu             sync.Mutex
	keyUID         string
	kekFingerprint []byte
	dekHex         string
	dekClientHash  string // clientHashOf(dekHex); lets a client-hashed DEK act as the session credential, the only need to have a client-hashed DEK
	// on the stauts-go side is to avoid updating those ~20 places on the client side and continue using the already established mechanism that is being used for passwords
	dbKdfIter int

	primaryFromPending bool // true when dekHex came from the pending envelope

	pendingDekHex    string
	pendingDbKdfIter int

	dbAppOpenedWith    string
	dbWalletOpenedWith string
}

// dbFallbackCredential is an alternative credential for opening a database (when the primary one fails).
type dbFallbackCredential struct {
	secret  string
	kdfIter int
}

// resolvedProfileSecret is the outcome of translating a client-provided password into the secret.
type resolvedProfileSecret struct {
	secret             string
	dbKdfIter          int
	migrated           bool                   // true when the profile uses the DEK encryption scheme.
	primaryFromPending bool                   // true when the main envelope rejected the password and the secret came from the pending envelope instead.
	fallbacks          []dbFallbackCredential // tried in order when a database fails to open with secret
}

// resolveProfileSecret translates (keyUID, password) into the secret used for the databases and keystore files.
// It returns either the secret or password if the profile is not migrated and an error if the password is wrong.
func (b *StatusBackend) resolveProfileSecret(keyUID, password string, kdfIterationsFallback int) (resolvedProfileSecret, error) {
	c := &b.secretCache
	c.mu.Lock()
	if c.keyUID == keyUID && c.dekHex != "" {
		fingerprint := sha256.Sum256([]byte(password))
		if subtle.ConstantTimeCompare(fingerprint[:], c.kekFingerprint) == 1 ||
			subtle.ConstantTimeCompare([]byte(password), []byte(c.dekHex)) == 1 ||
			(c.dekClientHash != "" && subtle.ConstantTimeCompare([]byte(password), []byte(c.dekClientHash)) == 1) {
			resolved := resolvedProfileSecret{
				secret:             c.dekHex,
				dbKdfIter:          c.dbKdfIter,
				migrated:           true,
				primaryFromPending: c.primaryFromPending,
				fallbacks:          []dbFallbackCredential{{secret: password, kdfIter: kdfIterationsFallback}},
			}
			if c.pendingDekHex != "" && c.pendingDekHex != c.dekHex {
				resolved.fallbacks = append(resolved.fallbacks,
					dbFallbackCredential{secret: c.pendingDekHex, kdfIter: c.pendingDbKdfIter})
			}
			c.mu.Unlock()
			return resolved, nil
		}
	}
	c.mu.Unlock()

	if !envelope.Exists(b.rootDataDir, keyUID) {
		return resolvedProfileSecret{
			secret:    password,
			dbKdfIter: kdfIterationsFallback,
		}, nil
	}

	pendingDek, pendingIter := "", 0
	if envelope.PendingExists(b.rootDataDir, keyUID) {
		pendingDek, pendingIter, _ = envelope.UnwrapPending(b.rootDataDir, keyUID, password)
	}

	primary, primaryIter := "", 0
	primaryFromPending := false
	dekHex, dbKdfIter, err := envelope.Unwrap(b.rootDataDir, keyUID, password)
	if err == nil {
		primary, primaryIter = dekHex, dbKdfIter
	} else if pendingDek != "" {
		// The main envelope rejects this password but the pending one accepts it, so the pending DEK is what the databases
		// are encrypted with (mostly).
		primary, primaryIter = pendingDek, pendingIter
		primaryFromPending = true
		pendingDek, pendingIter = "", 0
	} else {
		return resolvedProfileSecret{}, err
	}

	fingerprint := sha256.Sum256([]byte(password))
	c.mu.Lock()
	c.keyUID = keyUID
	c.kekFingerprint = fingerprint[:]
	c.dekHex = primary
	c.dekClientHash = clientHashOf(primary)
	c.dbKdfIter = primaryIter
	c.primaryFromPending = primaryFromPending
	c.pendingDekHex = pendingDek
	c.pendingDbKdfIter = pendingIter
	c.mu.Unlock()

	resolved := resolvedProfileSecret{
		secret:             primary,
		dbKdfIter:          primaryIter,
		migrated:           true,
		primaryFromPending: primaryFromPending,
		fallbacks:          []dbFallbackCredential{{secret: password, kdfIter: kdfIterationsFallback}},
	}
	if pendingDek != "" && pendingDek != primary {
		resolved.fallbacks = append(resolved.fallbacks,
			dbFallbackCredential{secret: pendingDek, kdfIter: pendingIter})
	}
	return resolved, nil
}

// clearProfileSecretCache clears the cached secret.
func (b *StatusBackend) clearProfileSecretCache() {
	c := &b.secretCache
	c.mu.Lock()
	c.keyUID = ""
	c.kekFingerprint = nil
	c.dekHex = ""
	c.dekClientHash = ""
	c.dbKdfIter = 0
	c.primaryFromPending = false
	c.pendingDekHex = ""
	c.pendingDbKdfIter = 0
	c.dbAppOpenedWith = ""
	c.dbWalletOpenedWith = ""
	c.mu.Unlock()
}

// clientHashOf returns the client hashing convention of a secret: "0x" + lowercase hex keccak256.
func clientHashOf(secret string) string {
	return "0x" + hex.EncodeToString(ethcrypto.Keccak256([]byte(secret)))
}

// primeSecretCacheWithDEK caches the DEK directly from a client-supplied DEK (biometric login).
func (b *StatusBackend) primeSecretCacheWithDEK(keyUID, dekHex string, dbKdfIter int) {
	c := &b.secretCache
	c.mu.Lock()
	c.keyUID = keyUID
	c.kekFingerprint = nil
	c.dekHex = dekHex
	c.dekClientHash = clientHashOf(dekHex)
	c.dbKdfIter = dbKdfIter
	c.primaryFromPending = false
	c.pendingDekHex = ""
	c.pendingDbKdfIter = 0
	c.mu.Unlock()
}

// prepareDEKLogin prepares a DEK login request by caching the DEK and setting the password to the DEK.
func (b *StatusBackend) prepareDEKLogin(request *requests.Login) error {
	if !envelope.Exists(b.rootDataDir, request.KeyUID) {
		return errors.New("dek login: profile is not on the DEK encryption scheme")
	}
	if envelope.PendingExists(b.rootDataDir, request.KeyUID) {
		return errors.New("dek login: interrupted rekey pending, a password login is required")
	}
	iter, err := envelope.ReadDBKdfIterations(b.rootDataDir, request.KeyUID)
	if err != nil {
		return err
	}
	b.primeSecretCacheWithDEK(request.KeyUID, request.DEK, iter)
	request.Password = request.DEK
	return nil
}

// ExportProfileDEK returns the profile's DEK given a valid credential (the client-hashed password KEK, or the raw DEK or its client hash).
func (b *StatusBackend) ExportProfileDEK(keyUID, password string) (string, error) {
	resolved, err := b.resolveProfileSecret(keyUID, password, 0)
	if err != nil {
		return "", asKeystoreResolutionError(err)
	}
	if !resolved.migrated {
		return "", ErrProfileNotMigratedToDEK
	}
	return resolved.secret, nil
}

// refreshSecretCacheKEK updates the cached KEK fingerprint after the envelope was re-wrapped with
// a new KEK (fast password change).
func (b *StatusBackend) refreshSecretCacheKEK(keyUID, newKEK string) {
	c := &b.secretCache
	c.mu.Lock()
	if c.keyUID == keyUID && c.dekHex != "" {
		fingerprint := sha256.Sum256([]byte(newKEK))
		c.kekFingerprint = fingerprint[:]
	}
	c.mu.Unlock()
}

// recordDBOpenSecret notes which secret successfully opened a database.
func (b *StatusBackend) recordDBOpenSecret(dbName, secret string) {
	c := &b.secretCache
	c.mu.Lock()
	switch dbName {
	case "app":
		c.dbAppOpenedWith = secret
	case "wallet":
		c.dbWalletOpenedWith = secret
	}
	c.mu.Unlock()
}

// dbOpenSecrets returns which secrets opened the app and wallet databases.
func (b *StatusBackend) dbOpenSecrets() (app, wallet string) {
	c := &b.secretCache
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dbAppOpenedWith, c.dbWalletOpenedWith
}

// ResolveKeystoreSecret translates a client-provided password into the secret the profile's keystore files are encrypted with
func (b *StatusBackend) ResolveKeystoreSecret(keyUID, password string) (string, error) {
	resolved, err := b.resolveProfileSecret(keyUID, password, 0)
	if err != nil {
		return "", err
	}
	return resolved.secret, nil
}

// sessionSecret returns the cached profile secret (DEK) for keyUID, available while the profile is unlocked
// in this session.
func (b *StatusBackend) sessionSecret(keyUID string) (string, bool) {
	c := &b.secretCache
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.keyUID == keyUID && c.dekHex != "" {
		return c.dekHex, true
	}
	return "", false
}

// asKeystoreResolutionError keeps the pre-DEK error contract of the keystore operations: a password that
// cannot unwrap the profile envelope is simply an incorrect password.
func asKeystoreResolutionError(err error) error {
	if errors.Is(err, envelope.ErrInvalidKEK) {
		return keystore.ErrIncorrectPasswordProvided
	}
	return err
}

// setSecretResolverForProfile points the accounts manager's keystore operations at this profile's secret resolution.
func (b *StatusBackend) setSecretResolverForProfile(keyUID string, kdfIterations int) {
	b.accountsManager.SetSecretResolver(func(password string) (string, error) {
		resolved, err := b.resolveProfileSecret(keyUID, password, kdfIterations)
		if err != nil {
			return "", asKeystoreResolutionError(err)
		}
		return resolved.secret, nil
	})
	// Keystore writes must keep the directory uniformly encrypted with the profile secret. Clients may pass a
	// password that is not the login password, so fall back to the session secret instead of failing.
	b.accountsManager.SetWriteSecretResolver(func(password string) (string, error) {
		resolved, err := b.resolveProfileSecret(keyUID, password, kdfIterations)
		if err == nil {
			return resolved.secret, nil
		}
		if secret, ok := b.sessionSecret(keyUID); ok {
			return secret, nil
		}
		return "", asKeystoreResolutionError(err)
	})
}
