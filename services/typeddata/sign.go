package typeddata

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	// x19 to avoid collision with rlp encode. x01 version byte defined in EIP-191
	messagePadding = []byte{0x19, 0x01}
)

func hashToSign(typed TypedData) (rst common.Hash, err error) {
	domainSeparator, err := encodeData(eip712Domain, typed.Domain, typed.Types)
	if err != nil {
		return rst, err
	}
	primary, err := encodeData(typed.PrimaryType, typed.Message, typed.Types)
	if err != nil {
		return rst, err
	}
	return crypto.Keccak256Hash(messagePadding, domainSeparator[:], primary[:]), nil
}

func Sign(typed TypedData, prv *ecdsa.PrivateKey, chain *big.Int) ([]byte, error) {
	chainID := typed.Message["chainId"].(int64)
	if chainID != chain.Int64() {
		return nil, fmt.Errorf("chainId %d doesn't match selected chain %d", chainID, chain.Int64())
	}
	hash, err := hashToSign(typed)
	if err != nil {
		return nil, err
	}
	return crypto.Sign(hash[:], prv)
}
