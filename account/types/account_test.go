package types

import (
	"testing"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/stretchr/testify/assert"
)

func TestAddressToAccount(t *testing.T) {

	tests := []struct {
		name        string
		address     string
		expectError bool
	}{
		{
			name:        "valid address",
			address:     "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
			expectError: false,
		},
		{
			name:        "invalid address",
			address:     "0xinvalid",
			expectError: true,
		},
		{
			name:        "empty address",
			address:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account, err := AddressToAccount(tt.address)
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, ErrInvalidAddress, err)
				assert.Equal(t, Account{}, account)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, types.HexToAddress(tt.address), account.Address)
			}
		})
	}
}
