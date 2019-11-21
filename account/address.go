package account

import (
	"github.com/ethereum/go-ethereum/crypto"
	protocol "github.com/status-im/status-go/protocol/types"
)

func CreateAddress() (address, pubKey, privKey string, err error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return "", "", "", err
	}

	privKeyBytes := crypto.FromECDSA(key)
	pubKeyBytes := crypto.FromECDSAPub(&key.PublicKey)
	addressBytes := crypto.PubkeyToAddress(key.PublicKey)

	privKey = protocol.EncodeHex(privKeyBytes)
	pubKey = protocol.EncodeHex(pubKeyBytes)
	address = addressBytes.Hex()

	return
}
