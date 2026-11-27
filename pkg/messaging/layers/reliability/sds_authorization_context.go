package reliability

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/status-im/status-go/internal/crypto"
)

const uncompressedPublicKeySize = 65

// BuildSDSAuthorizationContext passes the verified Waku signer and, when
// present, the group used to decrypt the payload to the local SDS authorizer.
func BuildSDSAuthorizationContext(sender *ecdsa.PublicKey, hashRatchetGroupID []byte) []byte {
	if sender == nil {
		return nil
	}
	return append(crypto.FromECDSAPub(sender), hashRatchetGroupID...)
}

// ParseSDSAuthorizationContext reads the fixed-size signer followed by the
// optional hash-ratchet group. This context is local and never sent over Waku.
func ParseSDSAuthorizationContext(data []byte) (*ecdsa.PublicKey, []byte, error) {
	signerEnd := min(len(data), uncompressedPublicKeySize)
	sender, err := crypto.UnmarshalPubkey(data[:signerEnd])
	if err != nil {
		return nil, nil, fmt.Errorf("invalid SDS transport sender: %w", err)
	}
	return sender, data[signerEnd:], nil
}
