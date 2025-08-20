package common

import (
	"github.com/status-im/status-go/accounts-management/keystore"
	"github.com/status-im/status-go/accounts-management/types"
)

// This function is exposed from here just to be used for validating transferred keystore files while local pairing.
// The rest of the code should use the keystore package, via KeyStore interface.
func DecryptKey(keyjson []byte, auth string) (*types.Key, error) {
	return keystore.DecryptKey(keyjson, auth)
}
