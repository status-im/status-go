package account

import (
	"github.com/ethereum/go-ethereum/crypto"
	statusproto "github.com/status-im/status-protocol-go/types"
)

func CreateAddress() (address, pubKey, privKey string, err error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return "", "", "", err
	}

	privKeyBytes := crypto.FromECDSA(key)
	pubKeyBytes := crypto.FromECDSAPub(&key.PublicKey)
	addressBytes := crypto.PubkeyToAddress(key.PublicKey)

	privKey = statusproto.EncodeHex(privKeyBytes)
	pubKey = statusproto.EncodeHex(pubKeyBytes)
	address = addressBytes.Hex()

	return
}
