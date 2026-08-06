// Package envelope implements the per-profile wrapped database-encryption-key (DEK).
//
// Goal: Changing the password only re-wraps the DEK in <keyUID>-profile.kek file, the databases and keystore files are untouched.
package envelope

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	geth "github.com/status-im/status-go/internal/accounts-management/keystore/internal/geth"
)

const (
	fileSuffix  = "-profile.kek"
	fileVersion = 1

	pendingSuffix = ".pending" // used for deep rekey (when a new DEK is generated)

	dekLength = 32 // DEK size in bytes

	scryptN = 1 << 18
	scryptP = 1
)

var ErrInvalidKEK = errors.New("envelope: invalid key-encryption key")

type wrappedKeyFile struct {
	Version         int             `json:"version"`
	KeyUID          string          `json:"keyUid"`
	DBKdfIterations int             `json:"dbKdfIterations"`
	Crypto          geth.CryptoJSON `json:"crypto"`
}

// Path returns the wrapped-DEK file path for the given profile.
func Path(rootDataDir, keyUID string) string {
	return filepath.Join(rootDataDir, keyUID+fileSuffix)
}

// PendingPath returns the path of the pending (rekey-in-progress) envelope.
func PendingPath(rootDataDir, keyUID string) string {
	return Path(rootDataDir, keyUID) + pendingSuffix
}

// Exists reports whether the profile has a wrapped-DEK file.
func Exists(rootDataDir, keyUID string) bool {
	_, err := os.Stat(Path(rootDataDir, keyUID))
	return err == nil
}

// PendingExists reports whether an interrupted rekey left a pending envelope.
func PendingExists(rootDataDir, keyUID string) bool {
	_, err := os.Stat(PendingPath(rootDataDir, keyUID))
	return err == nil
}

// Generate creates a new random DEK and returns it as a lowercase hex string.
func Generate() (string, error) {
	dek := make([]byte, dekLength)
	if _, err := rand.Read(dek); err != nil {
		return "", err
	}
	return hex.EncodeToString(dek), nil
}

// Write wraps dekHex with kek and atomically writes the wrapped-DEK file, replacing any existing one.
func Write(rootDataDir, keyUID, dekHex, kek string, dbKdfIterations int) error {
	return writeToPath(Path(rootDataDir, keyUID), keyUID, dekHex, kek, dbKdfIterations)
}

// WritePending writes the pending (rekey-in-progress) envelope.
func WritePending(rootDataDir, keyUID, dekHex, kek string, dbKdfIterations int) error {
	return writeToPath(PendingPath(rootDataDir, keyUID), keyUID, dekHex, kek, dbKdfIterations)
}

func writeToPath(path, keyUID, dekHex, kek string, dbKdfIterations int) error {
	dek, err := hex.DecodeString(dekHex)
	if err != nil || len(dek) != dekLength {
		return fmt.Errorf("envelope: DEK must be %d hex-encoded bytes", dekLength)
	}

	cryptoJSON, err := geth.EncryptDataV3(dek, []byte(kek), scryptN, scryptP)
	if err != nil {
		return err
	}

	content, err := json.Marshal(wrappedKeyFile{
		Version:         fileVersion,
		KeyUID:          keyUID,
		DBKdfIterations: dbKdfIterations,
		Crypto:          cryptoJSON,
	})
	if err != nil {
		return err
	}

	return atomicWrite(path, content)
}

// Unwrap reads the profile's wrapped-DEK file and decrypts the DEK with kek.
func Unwrap(rootDataDir, keyUID, kek string) (dekHex string, dbKdfIterations int, err error) {
	return unwrapFromPath(Path(rootDataDir, keyUID), kek)
}

// UnwrapPending reads and decrypts the pending (rekey-in-progress) envelope.
func UnwrapPending(rootDataDir, keyUID, kek string) (dekHex string, dbKdfIterations int, err error) {
	return unwrapFromPath(PendingPath(rootDataDir, keyUID), kek)
}

func unwrapFromPath(path, kek string) (dekHex string, dbKdfIterations int, err error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}

	var file wrappedKeyFile
	if err := json.Unmarshal(content, &file); err != nil {
		return "", 0, fmt.Errorf("envelope: malformed wrapped-DEK file: %w", err)
	}
	if file.Version != fileVersion {
		return "", 0, fmt.Errorf("envelope: unsupported wrapped-DEK file version %d", file.Version)
	}

	dek, err := geth.DecryptDataV3(file.Crypto, kek)
	if err != nil {
		// MAC mismatch (or any decrypt failure) means the KEK is wrong.
		return "", 0, ErrInvalidKEK
	}
	if len(dek) != dekLength {
		return "", 0, fmt.Errorf("envelope: unexpected DEK length %d", len(dek))
	}

	return hex.EncodeToString(dek), file.DBKdfIterations, nil
}

// Rewrap re-encrypts the profile's DEK with a new KEK, atomically replacing the wrapped-DEK file.
func Rewrap(rootDataDir, keyUID, oldKEK, newKEK string) error {
	dekHex, dbKdfIterations, err := Unwrap(rootDataDir, keyUID, oldKEK)
	if err != nil {
		return err
	}
	return Write(rootDataDir, keyUID, dekHex, newKEK, dbKdfIterations)
}

// Remove deletes the profile's wrapped-DEK file, its pending sidecar and any leftover temp files from interrupted writes.
func Remove(rootDataDir, keyUID string) error {
	if err := RemovePending(rootDataDir, keyUID); err != nil {
		return err
	}
	return removePath(Path(rootDataDir, keyUID))
}

// RemovePending deletes the pending (rekey-in-progress) envelope, if any.
func RemovePending(rootDataDir, keyUID string) error {
	return removePath(PendingPath(rootDataDir, keyUID))
}

func removePath(path string) error {
	if matches, globErr := filepath.Glob(path + ".*.tmp"); globErr == nil {
		for _, match := range matches {
			_ = os.Remove(match)
		}
	}
	err := os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// atomicWrite writes content to path via a unique same-directory temp file + fsync + rename, so a crash never leaves
// a partially-written (unrecoverable) key file behind and concurrent writers cannot corrupt each other's temp content.
// The last completed rename wins with a self-consistent file.
func atomicWrite(path string, content []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := f.Name()

	if err = f.Chmod(0600); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if _, err = f.Write(content); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err = f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	if err = os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}
