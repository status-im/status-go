package onramp

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"
)

func TestCryptoOnRamps_MercuryoSignature(t *testing.T) {
	address := common.HexToAddress("0x1234567890123456789012345678901234567890")
	key := "asdbnm,asdb,mnabs=qweqwrhiuasdkj"

	signature := getMercuryoSignature(address, key)
	require.Equal(t, "76e386d5957353e2ce51d9960540979e36472cb754cbd8dcee164b9b4300bdafaa04e9370a4fa47165600b6c15f30f444ec69b2a227741e34189d6c73231f391", signature)
}
