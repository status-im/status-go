package typeddata

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	// x19 to avoid collision with rlp encode. x01 version byte defined in EIP-191
	messagePadding = []byte{0x19, 0x01}
)

func encodeData(typed TypedData) (rst common.Hash, err error) {
	domainSeparator, err := hashStruct(eip712Domain, typed.Domain, typed.Types)
	if err != nil {
		return rst, err
	}
	primary, err := hashStruct(typed.PrimaryType, typed.Message, typed.Types)
	if err != nil {
		return rst, err
	}
	return crypto.Keccak256Hash(messagePadding, domainSeparator[:], primary[:]), nil
}

// Sign TypedData with a given private key. Verify that chainId in the typed data matches currently selected chain.
func Sign(typed TypedData, prv *ecdsa.PrivateKey, chain *big.Int) ([]byte, error) {
	hash, err := Hash(typed, chain)
	if err != nil {
		return nil, err
	}
	sig, err := crypto.Sign(hash[:], prv)
	if err != nil {
		return nil, err
	}
	sig[64] += 27
	return sig, nil
}
