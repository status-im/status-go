package rln

import "encoding/hex"

func toMembershipKeyPairs(groupKeys [][]string) ([]MembershipKeyPair, error) {
	// groupKeys is  sequence of membership key tuples in the form of (identity key, identity commitment) all in the hexadecimal format
	// the toMembershipKeyPairs proc populates a sequence of MembershipKeyPairs using the supplied groupKeys

	groupKeyPairs := []MembershipKeyPair{}
	for _, pair := range groupKeys {
		idKey, err := hex.DecodeString(pair[0])
		if err != nil {
			return nil, err
		}
		idCommitment, err := hex.DecodeString(pair[1])
		if err != nil {
			return nil, err
		}

		groupKeyPairs = append(groupKeyPairs, MembershipKeyPair{IDKey: IDKey(Bytes32(idKey)), IDCommitment: IDCommitment(Bytes32(idCommitment))})
	}

	return groupKeyPairs, nil
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
