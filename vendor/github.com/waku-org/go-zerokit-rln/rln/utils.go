package rln

import (
	"encoding/hex"
)

func ToIdentityCredentials(groupKeys [][]string) ([]IdentityCredential, error) {
	// groupKeys is  sequence of membership key tuples in the form of (identity key, identity commitment) all in the hexadecimal format
	// the toIdentityCredentials proc populates a sequence of IdentityCredentials using the supplied groupKeys
	// Returns an error if the conversion fails

	var groupIdCredentials []IdentityCredential

	for _, gk := range groupKeys {
		idTrapdoor, err := ToBytes32LE(gk[0])
		if err != nil {
			return nil, err
		}

		idNullifier, err := ToBytes32LE(gk[1])
		if err != nil {
			return nil, err
		}

		idSecretHash, err := ToBytes32LE(gk[2])
		if err != nil {
			return nil, err
		}

		idCommitment, err := ToBytes32LE(gk[3])
		if err != nil {
			return nil, err
		}

		groupIdCredentials = append(groupIdCredentials, IdentityCredential{
			IDTrapdoor:   idTrapdoor,
			IDNullifier:  idNullifier,
			IDSecretHash: idSecretHash,
			IDCommitment: idCommitment,
		})
	}

	return groupIdCredentials, nil
}

func Bytes32(b []byte) [32]byte {
	var result [32]byte
	copy(result[32-len(b):], b)
	return result
}

func Bytes128(b []byte) [128]byte {
	var result [128]byte
	copy(result[128-len(b):], b)
	return result
}

func ToBytes32LE(hexStr string) ([32]byte, error) {

	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return [32]byte{}, err
	}

	for i := 0; i < len(b)/2; i++ {
		b[i], b[len(b)-i-1] = b[len(b)-i-1], b[i]
	}

	return Bytes32(b), nil
}
