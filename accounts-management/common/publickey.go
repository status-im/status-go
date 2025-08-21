package common

import (
	"reflect"

	"github.com/status-im/status-go/accounts-management/types"
	"github.com/status-im/status-go/api/multiformat"
	"github.com/status-im/status-go/protocol/identity/emojihash"
)

// GetPublicKeyData processes a public key and returns its compressed form and emoji hash
func GetPublicKeyData(publicKey string) (*types.PublicKeyData, error) {
	if publicKey == "" {
		return nil, nil
	}

	compressedKey, err := multiformat.SerializeLegacyKey(publicKey)
	if err != nil {
		return nil, err
	}

	emojiHash, err := emojihash.GenerateFor(publicKey)
	if err != nil {
		return nil, err
	}

	return &types.PublicKeyData{
		CompressedKey: compressedKey,
		EmojiHash:     emojiHash,
	}, nil
}

func ExtendStructWithPubKeyData(publicKey string, item any) (any, error) {
	// If the public key is empty, do not attempt to extend the incoming item
	if publicKey == "" {
		return item, nil
	}

	pkd, err := GetPublicKeyData(publicKey)
	if err != nil {
		return nil, err
	}

	// Create a struct with 2 embedded substruct fields in order to circumvent
	// "embedded field type cannot be a (pointer to a) type parameter"
	// compiler error that arises if we were to use a generic function instead
	typ := reflect.StructOf([]reflect.StructField{
		{
			Name:      "Item",
			Anonymous: true,
			Type:      reflect.TypeOf(item),
		},
		{
			Name:      "Pkd",
			Anonymous: true,
			Type:      reflect.TypeOf(pkd),
		},
	})

	v := reflect.New(typ).Elem()
	v.Field(0).Set(reflect.ValueOf(item))
	v.Field(1).Set(reflect.ValueOf(pkd))
	s := v.Addr().Interface()

	return s, nil
}
