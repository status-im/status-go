package contactrequests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/crypto"
)

func TestValidateSignature(t *testing.T) {
	pk1, err := crypto.GenerateKey()
	require.NoError(t, err)

	pk2, err := crypto.GenerateKey()
	require.NoError(t, err)

	var timestamp uint64 = 10

	// Build signature
	signature, err := BuildSignature(&pk1.PublicKey, pk2, timestamp)
	require.NoError(t, err)

	// And verify

	err = VerifySignature(signature, &pk1.PublicKey, &pk2.PublicKey, timestamp)
	require.NoError(t, err)
}
