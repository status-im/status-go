package typeddata

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	chainIDKey = "chainId"
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

// Sign TypedData with a given private key. Verify that chainId in the typed data matches currently selected chain.
func Sign(typed TypedData, prv *ecdsa.PrivateKey, chain *big.Int) ([]byte, error) {
	if _, exist := typed.Domain[chainIDKey]; !exist {
		return nil, fmt.Errorf("domain misses chain key %s", chainIDKey)
	}
	chainID, ok := typed.Domain[chainIDKey].(int)
	if !ok {
		return nil, errors.New("chainId is not an int")
	}
	if int64(chainID) != chain.Int64() {
		return nil, fmt.Errorf("chainId %d doesn't match selected chain %s", chainID, chain)
	}
	hash, err := hashToSign(typed)
	if err != nil {
		return nil, err
	}
	return crypto.Sign(hash[:], prv)
}
