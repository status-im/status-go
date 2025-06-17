package generator

import (
	"testing"

	"github.com/status-im/extkeys"
	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/accounts-management/common"
)

func generateTestKey(t *testing.T) *Account {
	mnemonic, err := common.CreateRandomMnemonicWithDefaultLength()
	assert.NoError(t, err)

	extendedKey, err := common.CreateExtendedKeyFromMnemonic(mnemonic, "")
	assert.NoError(t, err)

	return NewAccount(extendedKey.ToECDSA(), extendedKey)
}

func TestValidateExtendedKey(t *testing.T) {
	acc1 := generateTestKey(t)
	acc2 := generateTestKey(t)

	// Create a valid account
	validAccount := NewAccount(acc1.PrivateKey(), acc1.ExtendedKey())

	// Create an invalid account with different private key
	invalidAccount := NewAccount(acc1.PrivateKey(), acc2.ExtendedKey())

	// Create a zeroed account
	zeroedAccount := NewAccount(acc1.PrivateKey(), &extkeys.ExtendedKey{})

	tests := []struct {
		name        string
		account     *Account
		expectError bool
	}{
		{
			name:        "valid account",
			account:     validAccount,
			expectError: false,
		},
		{
			name:        "invalid account",
			account:     invalidAccount,
			expectError: true,
		},
		{
			name:        "zeroed account",
			account:     zeroedAccount,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.account.ValidateExtendedKey()
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, ErrInvalidKeystoreExtendedKey, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
