package backend

import (
	"errors"

	"github.com/status-im/status-go/internal/accounts-management/keystore/envelope"
	"github.com/status-im/status-go/internal/protocol/requests"
)

var (
	ErrZeroOnboardedProfileNotAllowed = errors.New("zero-onboarded profile can only be created as the first, non-keycard profile")
	ErrNotZeroOnboardedProfile        = errors.New("profile is not zero-onboarded")
	ErrZeroOnboardedProfileNotActive  = errors.New("zero-onboarded profile is not the logged-in profile")
	ErrZeroOnboardedNoSessionSecret   = errors.New("zero-onboarded profile has no unlocked session secret")
	ErrZeroOnboardedRekeyPending      = errors.New("zero-onboarded profile has an interrupted rekey pending")
)

func (b *StatusBackend) validateZeroOnboardedCreation(request *requests.CreateAccount, keycardData *requests.KeycardData,
	restoreAccount bool) error {
	if !request.ZeroOnboarded {
		return nil
	}
	if restoreAccount || keycardData != nil || request.KeycardInstanceUID != "" {
		return ErrZeroOnboardedProfileNotAllowed
	}
	if err := b.OpenAccounts(); err != nil {
		return err
	}
	count, err := b.multiaccountsDB.GetAccountsCount()
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrZeroOnboardedProfileNotAllowed
	}
	return nil
}

// SetZeroOnboardedProfilePassword re-wraps the session's DEK with newPassword; nothing is re-encrypted.
// The envelope is written before the mark is cleared, so a crash in between is recovered by calling it again.
func (b *StatusBackend) SetZeroOnboardedProfilePassword(keyUID, newPassword string) error {
	b.reEncryptionMu.Lock()
	defer b.reEncryptionMu.Unlock()

	// Security-critical: without both checks this would replace any profile's password without the old one.
	b.mu.Lock()
	isActive := b.account != nil && b.account.KeyUID == keyUID
	b.mu.Unlock()
	if !isActive {
		return ErrZeroOnboardedProfileNotActive
	}
	acc, err := b.multiaccountsDB.GetAccount(keyUID)
	if err != nil {
		return err
	}
	if !acc.ZeroOnboarded {
		return ErrNotZeroOnboardedProfile
	}

	dek, ok := b.sessionSecret(keyUID)
	if !ok {
		return ErrZeroOnboardedNoSessionSecret
	}
	if envelope.PendingExists(b.rootDataDir, keyUID) {
		return ErrZeroOnboardedRekeyPending
	}
	dbKdfIterations, err := envelope.ReadDBKdfIterations(b.rootDataDir, keyUID)
	if err != nil {
		return err
	}

	if err := envelope.Write(b.rootDataDir, keyUID, dek, newPassword, dbKdfIterations); err != nil {
		return err
	}
	b.refreshSecretCacheKEK(keyUID, newPassword)
	return b.clearZeroOnboarded(keyUID)
}

func (b *StatusBackend) clearZeroOnboarded(keyUID string) error {
	if err := b.multiaccountsDB.ClearZeroOnboarded(keyUID); err != nil {
		return err
	}
	// GetActiveAccount (Android resume) reads the session copy.
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.account != nil && b.account.KeyUID == keyUID {
		b.account.ZeroOnboarded = false
	}
	return nil
}
