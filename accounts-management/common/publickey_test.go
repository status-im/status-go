package common

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPublicKeyData(t *testing.T) {
	tests := []struct {
		name        string
		publicKey   string
		expectError bool
	}{
		{
			name:        "valid public key",
			publicKey:   "0x0498593ff6c560f5dad6e947cd9c8aa41d687126fa253eb5d917f1f5911519c570651268c40f818b98ec18b00d2b59e16075a162f1bba6b568b753444b4c5a6a79",
			expectError: false,
		},
		{
			name:        "empty public key",
			publicKey:   "",
			expectError: false,
		},
		{
			name:        "invalid public key",
			publicKey:   "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := GetPublicKeyData(tt.publicKey)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, data)
			} else if tt.publicKey == "" {
				assert.NoError(t, err)
				assert.Nil(t, data)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, data)
				assert.NotEmpty(t, data.CompressedKey)
				assert.NotEmpty(t, data.EmojiHash)
			}
		})
	}
}

func TestExtendStructWithPubKeyData(t *testing.T) {
	tests := []struct {
		name        string
		publicKey   string
		item        any
		expectError bool
	}{
		{
			name:      "valid public key",
			publicKey: "0x0498593ff6c560f5dad6e947cd9c8aa41d687126fa253eb5d917f1f5911519c570651268c40f818b98ec18b00d2b59e16075a162f1bba6b568b753444b4c5a6a79",
			item: struct {
				ID string `json:"id"`
			}{
				ID: "1234",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Printf("item: %+v\n", tt.item)
			fmt.Printf("publicKey: %s\n", tt.publicKey)

			item, err := ExtendStructWithPubKeyData(tt.publicKey, tt.item)
			fmt.Printf("item: %+v\n", item)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, item)
			}
		})
	}
}
