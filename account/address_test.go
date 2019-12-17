package account

import (
	"testing"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/stretchr/testify/require"
)

func TestCreateAddress(t *testing.T) {
	addr, pub, priv, err := CreateAddress()
	require.NoError(t, err)
	require.Equal(t, types.IsHexAddress(addr), true)

	privECDSA, err := crypto.HexToECDSA(priv[2:])
	require.NoError(t, err)

	pubECDSA := privECDSA.PublicKey
	expectedPubStr := types.EncodeHex(crypto.FromECDSAPub(&pubECDSA))

	require.Equal(t, expectedPubStr, pub)
}
