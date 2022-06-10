package server

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/protocol/common"
)

func makeEncryptionKey(key *ecdsa.PrivateKey) ([]byte, error) {
	return common.MakeECDHSharedKey(key, &key.PublicKey)
}
