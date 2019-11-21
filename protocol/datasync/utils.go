package datasync

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/vacp2p/mvds/state"
)

func ToGroupID(data []byte) state.GroupID {
	g := state.GroupID{}
	copy(g[:], data[:])
	return g
}

// ToOneToOneGroupID returns a groupID for a onetoonechat, which is taken by
// concatenating the bytes of the compressed keys, in ascending order by X
func ToOneToOneGroupID(key1 *ecdsa.PublicKey, key2 *ecdsa.PublicKey) state.GroupID {
	pk1 := crypto.CompressPubkey(key1)
	pk2 := crypto.CompressPubkey(key2)
	var groupID []byte
	if key1.X.Cmp(key2.X) == -1 {
		groupID = append(pk1, pk2...)
	} else {
		groupID = append(pk2, pk1...)
	}

	return ToGroupID(crypto.Keccak256(groupID))
}
