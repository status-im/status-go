package common

import (
	"testing"

	"github.com/status-im/extkeys"
	"github.com/stretchr/testify/assert"

	ethtypes "github.com/status-im/status-go/eth-node/types"
)

func generateTestKey(t *testing.T) *extkeys.ExtendedKey {
	mnemonic, err := CreateRandomMnemonicWithDefaultLength()
	assert.NoError(t, err)

	extendedKey, err := CreateExtendedKeyFromMnemonic(mnemonic, "")
	assert.NoError(t, err)

	return extendedKey
}

func TestValidateExtendedKey(t *testing.T) {
	extendedKey1 := generateTestKey(t)
	extendedKey2 := generateTestKey(t)

	// Create a valid key
	validKey := &ethtypes.Key{
		PrivateKey:  extendedKey1.ToECDSA(),
		ExtendedKey: extendedKey1,
	}

	// Create an invalid key with different private key
	invalidKey := &ethtypes.Key{
		PrivateKey:  extendedKey1.ToECDSA(),
		ExtendedKey: extendedKey2,
	}

	// Create a zeroed key
	zeroedKey := &ethtypes.Key{
		PrivateKey:  extendedKey1.ToECDSA(),
		ExtendedKey: &extkeys.ExtendedKey{},
	}

	tests := []struct {
		name        string
		key         *ethtypes.Key
		expectError bool
	}{
		{
			name:        "valid key",
			key:         validKey,
			expectError: false,
		},
		{
			name:        "invalid key",
			key:         invalidKey,
			expectError: true,
		},
		{
			name:        "zeroed key",
			key:         zeroedKey,
			expectError: false,
		},
		{
			name:        "nil key",
			key:         nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExtendedKey(tt.key)
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, ErrInvalidKeystoreExtendedKey, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
