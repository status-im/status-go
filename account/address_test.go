package account

import (
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestCreateAddress(t *testing.T) {
	addr, pub, priv, err := CreateAddress()
	require.NoError(t, err)
	require.Equal(t, gethcommon.IsHexAddress(addr), true)

	privECDSA, err := crypto.HexToECDSA(priv[2:])
	require.NoError(t, err)

	pubECDSA := privECDSA.PublicKey
	expectedPubStr := hexutil.Encode(crypto.FromECDSAPub(&pubECDSA))

	require.Equal(t, expectedPubStr, pub)
}
