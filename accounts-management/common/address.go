package common

import (
	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/crypto/types"
)

func CreateAddress() (address, pubKey, privKey string, err error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return "", "", "", err
	}

	privKeyBytes := crypto.FromECDSA(key)
	pubKeyBytes := crypto.FromECDSAPub(&key.PublicKey)
	addressBytes := crypto.PubkeyToAddress(key.PublicKey)

	privKey = types.EncodeHex(privKeyBytes)
	pubKey = types.EncodeHex(pubKeyBytes)
	address = addressBytes.Hex()

	return
}
