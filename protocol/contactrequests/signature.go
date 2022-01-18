package contactrequests

import (
	"crypto/ecdsa"
	"encoding/binary"
	"errors"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/common"
)

var contactSignaturePrefix = []byte{0x12, 0x13}

func VerifySignature(signature []byte, nonSigningKey *ecdsa.PublicKey, signingKey *ecdsa.PublicKey, timestamp uint64) error {

	signatureMaterial, err := buildSignatureMaterial(nonSigningKey, signingKey, timestamp)
	if err != nil {
		return err
	}

	recoveredKey, err := crypto.SigToPub(
		signatureMaterial,
		signature,
	)

	if err != nil {
		return err
	}

	if !common.IsPubKeyEqual(signingKey, recoveredKey) {
		return errors.New("signature not matching")
	}

	return nil
}

func buildSignatureMaterial(theirKey *ecdsa.PublicKey, myKey *ecdsa.PublicKey, timestamp uint64) ([]byte, error) {
	var first, last *ecdsa.PublicKey
	// compare X
	switch theirKey.X.Cmp(myKey.X) {
	case 0:
		// compare Y
		switch theirKey.Y.Cmp(myKey.Y) {
		case 0:
			return nil, errors.New("keys can't be the same")
		case -1:
			first = theirKey
			last = myKey
		case 1:
			first = myKey
			last = theirKey
		}

	case -1:
		first = theirKey
		last = myKey
	case 1:
		first = myKey
		last = theirKey
	}

	firstKeyBytes := crypto.FromECDSAPub(first)
	lastKeyBytes := crypto.FromECDSAPub(last)

	signatureMaterial := append(contactSignaturePrefix, firstKeyBytes...)
	signatureMaterial = append(signatureMaterial, lastKeyBytes...)

	timestampBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(timestampBytes, timestamp)

	signatureMaterial = append(signatureMaterial, timestampBytes...)
	return crypto.Keccak256(signatureMaterial), nil
}

func BuildSignature(theirKey *ecdsa.PublicKey, myKey *ecdsa.PrivateKey, timestamp uint64) ([]byte, error) {
	signatureMaterial, err := buildSignatureMaterial(theirKey, &myKey.PublicKey, timestamp)
	if err != nil {
		return nil, err
	}

	signature, err := crypto.Sign(signatureMaterial, myKey)

	if err != nil {
		return nil, err
	}

	return signature, nil
}
