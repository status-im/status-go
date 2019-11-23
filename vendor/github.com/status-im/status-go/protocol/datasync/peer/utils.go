package peer

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/eth-node/crypto"

	"github.com/vacp2p/mvds/state"
)

func PublicKeyToPeerID(k ecdsa.PublicKey) state.PeerID {
	var p state.PeerID
	copy(p[:], crypto.FromECDSAPub(&k))
	return p
}

func PeerIDToPublicKey(p state.PeerID) (*ecdsa.PublicKey, error) {
	return crypto.UnmarshalPubkey(p[:])
}
