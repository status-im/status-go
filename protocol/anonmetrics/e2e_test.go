package anonmetrics

import (
	"encoding/hex"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/status-im/status-go/eth-node/crypto"
)

func TestKeyGen(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Error(err)
	}

	keyBytes := crypto.FromECDSA(key)
	keyString := hex.EncodeToString(keyBytes)

	keyPubBytes := crypto.FromECDSAPub(&key.PublicKey)
	keyPubString := hex.EncodeToString(keyPubBytes)

	spew.Dump(keyString, keyPubString)
}
