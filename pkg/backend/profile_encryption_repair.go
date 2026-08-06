package backend

import (
	"fmt"

	"go.uber.org/zap"

	keystorepkg "github.com/status-im/status-go/internal/accounts-management/keystore"
	"github.com/status-im/status-go/internal/accounts-management/keystore/envelope"
	"github.com/status-im/status-go/internal/db/sqlite"
	"github.com/status-im/status-go/internal/db/walletdb"
)

// repairProfileEncryption reconciles a profile whose encryption-scheme migration or rekey was interrupted by a crash.
// It must run after the databases were opened (possibly via fallback credentials) and before the keystore is used.
func (b *StatusBackend) repairProfileEncryption(keyUID, password string, kdfIterations int) error {
	b.reEncryptionMu.Lock()
	defer b.reEncryptionMu.Unlock()

	if !envelope.Exists(b.rootDataDir, keyUID) {
		return nil
	}

	appSecret, walletSecret := b.dbOpenSecrets()
	if appSecret == "" {
		// databases were not opened by this session, so nothing to align against
		return nil
	}

	resolved, err := b.resolveProfileSecret(keyUID, password, kdfIterations)
	if err != nil {
		return err
	}

	pendingExists := envelope.PendingExists(b.rootDataDir, keyUID)
	misaligned := pendingExists || appSecret != resolved.secret || (walletSecret != "" && walletSecret != appSecret)
	if !misaligned {
		return nil
	}

	b.logger.Warn("repairing interrupted encryption-scheme migration/rekey", zap.String("keyUID", keyUID))

	// kdf_iter per known candidate secret
	iterOf := map[string]int{resolved.secret: resolved.dbKdfIter}
	candidates := []string{resolved.secret}
	for _, fallback := range resolved.fallbacks {
		if _, known := iterOf[fallback.secret]; !known {
			iterOf[fallback.secret] = fallback.kdfIter
			candidates = append(candidates, fallback.secret)
		}
	}

	target := appSecret
	targetIter, known := iterOf[target]
	if !known {
		return fmt.Errorf("app database opened with an unknown secret; cannot repair profile encryption")
	}

	_, keystoreDir := DefaultKeystorePath(b.rootDataDir, keyUID)
	if err := ensureKeystoreEncryptedWith(keystoreDir, target, candidates); err != nil {
		return fmt.Errorf("failed to reconcile keystore encryption: %w", err)
	}

	if walletSecret != "" && walletSecret != target {
		if err := b.reEncryptWalletDBInPlace(keyUID, walletSecret, iterOf[walletSecret], target, targetIter); err != nil {
			return err
		}
	}

	if target == password {
		// The app DB is still on the raw password, so an interrupted migration is rolled back to a clean legacy state.
		if err := envelope.Remove(b.rootDataDir, keyUID); err != nil {
			return err
		}
		b.clearProfileSecretCache()
		return nil
	}

	// Commit the envelope to the DEK that actually encrypts the data.
	if resolved.secret != target || resolved.primaryFromPending {
		if err := envelope.Write(b.rootDataDir, keyUID, target, password, targetIter); err != nil {
			return err
		}
	}
	if err := envelope.RemovePending(b.rootDataDir, keyUID); err != nil {
		return err
	}
	b.clearProfileSecretCache()
	return nil
}

// ensureKeystoreEncryptedWith re-encrypts the keystore directory to target if not already encrypted with it.
func ensureKeystoreEncryptedWith(keystoreDir, target string, candidates []string) error {
	if err := keystorepkg.VerifyKeyStoreDirAtPath(keystoreDir, target); err == nil {
		return nil
	}

	for _, candidate := range candidates {
		if candidate == "" || candidate == target {
			continue
		}
		if err := keystorepkg.ReEncryptKeyStoreDirAtPath(keystoreDir, candidate, target); err == nil {
			return nil
		}
	}

	// A previous re-encryption may have failed after installing the files, so re-verify.
	return keystorepkg.VerifyKeyStoreDirAtPath(keystoreDir, target)
}

// reEncryptWalletDBInPlace re-encrypts the (currently open) wallet database from oldSecret to newSecret.
func (b *StatusBackend) reEncryptWalletDBInPlace(keyUID, oldSecret string, oldIter int, newSecret string, newIter int) error {
	walletDBPath, err := b.getWalletDBPath(keyUID)
	if err != nil {
		return err
	}

	tmpPath, cleanup, err := b.createTempDBFile("wallet.db")
	if err != nil {
		return err
	}
	defer cleanup()

	if err := sqlite.ExportDBWithKDFChange(walletDBPath, oldSecret, oldIter, tmpPath, newSecret, newIter, nil, nil); err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.closeWalletDB(); err != nil {
		return err
	}

	// Run the wal/shm cleanup before reopening, so it cannot delete the fresh WAL files.
	replaceCleanup, err := replaceDBFile(walletDBPath, tmpPath)
	if replaceCleanup != nil {
		replaceCleanup()
	}
	if err != nil {
		return err
	}

	b.walletDB, err = walletdb.InitializeDB(walletDBPath, newSecret, newIter)
	if err != nil {
		return err
	}
	b.statusNode.SetWalletDB(b.walletDB)
	b.recordDBOpenSecret("wallet", newSecret)
	return nil
}
