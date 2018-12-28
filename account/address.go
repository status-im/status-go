package account

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

func CreateAddress() (address, pubKey, privKey string, err error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return "", "", "", err
	}

	privKeyBytes := crypto.FromECDSA(key)
	pubKeyBytes := crypto.FromECDSAPub(&key.PublicKey)
	addressBytes := crypto.PubkeyToAddress(key.PublicKey)

	privKey = hexutil.Encode(privKeyBytes)
	pubKey = hexutil.Encode(pubKeyBytes)
	address = addressBytes.Hex()

	return
}
