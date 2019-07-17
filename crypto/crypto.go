package crypto

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
)

// Sign signs the hash of an arbitrary string
func Sign(content string, identity *ecdsa.PrivateKey) (string, error) {
	signature, err := crypto.Sign(crypto.Keccak256([]byte(content)), identity)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(signature), nil
}

// VerifySignatures verifys tuples of signatures content/hash/public key
func VerifySignatures(signaturePairs [][3]string) error {
	for _, signaturePair := range signaturePairs {
		content := crypto.Keccak256([]byte(signaturePair[0]))

		signature, err := hex.DecodeString(signaturePair[1])
		if err != nil {
			return err
		}

		publicKeyBytes, err := hex.DecodeString(signaturePair[2])
		if err != nil {
			return err
		}

		publicKey, err := crypto.UnmarshalPubkey(publicKeyBytes)
		if err != nil {
			return err
		}

		recoveredKey, err := crypto.SigToPub(
			content,
			signature,
		)
		if err != nil {
			return err
		}

		if crypto.PubkeyToAddress(*recoveredKey) != crypto.PubkeyToAddress(*publicKey) {
			return errors.New("identity key and signature mismatch")
		}
	}

	return nil
}

// ExtractSignatures extract from tuples of signatures content a public key
func ExtractSignatures(signaturePairs [][2]string) ([]string, error) {
	response := make([]string, len(signaturePairs))
	for i, signaturePair := range signaturePairs {
		content := crypto.Keccak256([]byte(signaturePair[0]))

		signature, err := hex.DecodeString(signaturePair[1])
		if err != nil {
			return nil, err
		}

		recoveredKey, err := crypto.SigToPub(
			content,
			signature,
		)
		if err != nil {
			return nil, err
		}

		response[i] = fmt.Sprintf("%x", crypto.FromECDSAPub(recoveredKey))
	}

	return response, nil
}
