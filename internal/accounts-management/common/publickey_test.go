package common

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	identityUtils "github.com/status-im/status-go/internal/protocol/identity"
	"github.com/status-im/status-go/internal/protocol/identity/alias"
	"github.com/status-im/status-go/internal/protocol/identity/emojihash"
	"github.com/status-im/status-go/pkg/multiformat"
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
				assert.NotEmpty(t, data.Alias, "alias should be derived for a valid pubkey")
				assert.GreaterOrEqual(t, data.ColorID, int64(0))
				assert.LessOrEqual(t, data.ColorID, int64(10))
			}
		})
	}
}

// TestGetPublicKeyDataDeterministic asserts the pure-function contract: the
// same pubkey always yields the same identity. Catches accidental state
// leaks (e.g. introducing a non-thread-safe cache).
func TestGetPublicKeyDataDeterministic(t *testing.T) {
	pk := "0x0498593ff6c560f5dad6e947cd9c8aa41d687126fa253eb5d917f1f5911519c570651268c40f818b98ec18b00d2b59e16075a162f1bba6b568b753444b4c5a6a79"

	first, err := GetPublicKeyData(pk)
	assert.NoError(t, err)
	assert.NotNil(t, first)

	second, err := GetPublicKeyData(pk)
	assert.NoError(t, err)
	assert.NotNil(t, second)

	assert.Equal(t, first.CompressedKey, second.CompressedKey)
	assert.Equal(t, first.EmojiHash, second.EmojiHash)
	assert.Equal(t, first.Alias, second.Alias)
	assert.Equal(t, first.ColorID, second.ColorID)
}

// TestGetPublicKeyDataGolden is a fixed-input/fixed-output regression test.
func TestGetPublicKeyDataGolden(t *testing.T) {
	const pk = "0x04eedbaafd6adf4a9233a13e7b1c3c14461fffeba2e9054b8d456ce5f6ebeafadcbf3dce3716253fbc391277fa5a086b60b283daf61fb5b1f26895f456c2f31ae3"

	pkd, err := GetPublicKeyData(pk)
	assert.NoError(t, err)
	assert.NotNil(t, pkd)

	assert.Equal(t, "zQ3shviWSAwB9X9Kz7aZHdUi6yU9S3BEpCsxM94XhFd2k2bWB", pkd.CompressedKey,
		"compressedKey should match multiformat.SerializeLegacyKey output")
	assert.Equal(t, "Darkorange Blue Bubblefish", pkd.Alias,
		"alias should match alias.GenerateFromPublicKeyString output (also asserted in protocol/identity/alias)")
	assert.Equal(t, int64(9), pkd.ColorID,
		"colorId should be (pubkey mod 11)")
	assert.Equal(t, []string{
		"🛁", "🧎🏿‍♀️", "😱", "👳‍♀️", "🎍",
		"🕺🏾", "🦸🏼‍♀️", "🕖", "🧏🏻‍♀️", "🤸‍♀️",
		"🦗", "🚴", "🍄", "👩🏻‍🦰",
	}, pkd.EmojiHash, "emojiHash should match emojihash.GenerateFor output")
}

// TestGetPublicKeyDataMatchesPrimitives proves the composition: every field
// in the result equals what the underlying primitive would produce on its
// own.
func TestGetPublicKeyDataMatchesPrimitives(t *testing.T) {
	pubkeys := []string{
		"0x0498593ff6c560f5dad6e947cd9c8aa41d687126fa253eb5d917f1f5911519c570651268c40f818b98ec18b00d2b59e16075a162f1bba6b568b753444b4c5a6a79",
		"0x04eedbaafd6adf4a9233a13e7b1c3c14461fffeba2e9054b8d456ce5f6ebeafadcbf3dce3716253fbc391277fa5a086b60b283daf61fb5b1f26895f456c2f31ae3",
		"0x04e25da6994ea2dc4ac70727e07eca153ae92bf7609db7befb7ebdceaad348f4fc55bbe90abf9501176301db5aa103fc0eb3bc3750272a26c424a10887db2a7ea8",
	}

	for _, pk := range pubkeys {
		t.Run(pk[:10], func(t *testing.T) {
			pkd, err := GetPublicKeyData(pk)
			assert.NoError(t, err)
			assert.NotNil(t, pkd)

			expCompressed, err := multiformat.SerializeLegacyKey(pk)
			assert.NoError(t, err)
			assert.Equal(t, expCompressed, pkd.CompressedKey)

			expEmoji, err := emojihash.GenerateFor(pk)
			assert.NoError(t, err)
			assert.Equal(t, expEmoji, pkd.EmojiHash)

			expAlias, err := alias.GenerateFromPublicKeyString(pk)
			assert.NoError(t, err)
			assert.Equal(t, expAlias, pkd.Alias)

			expColor, err := identityUtils.ToColorID(pk)
			assert.NoError(t, err)
			assert.Equal(t, expColor, pkd.ColorID)
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
