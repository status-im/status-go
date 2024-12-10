package communitytokensv2

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

func TestTypedDataHash(t *testing.T) {
	sigHash := common.Hex2Bytes("dd91c30357aafeb2792b5f0facbd83995943c1ea113a906ebbeb58bfeb27dfc2")
	domainSep := common.Hex2Bytes("4a672b5a08e88d37f7239165a0c9e03a01196587d52c638c0c99cbee5ba527c8")
	contractAddr := "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"
	signer := "0x54e3922e97e334905fb489be7c5df1f83cb1ce58"
	deployer := "0x7c8999dC9a822c1f0Df42023113EDB4FDd543266"
	goodHashResult := "0xccbb375343347491706cf4b43796f7b96ccc89c9e191a8b78679daeba1684ec7"

	typedHash, err := typedStructuredDataHash(domainSep, signer, deployer, contractAddr, 420)
	require.NoError(t, err, "creating typed structured data hash")
	require.Equal(t, goodHashResult, typedHash.String())

	customTypedHash := customTypedStructuredDataHash(domainSep, sigHash, signer, deployer)
	require.Equal(t, goodHashResult, customTypedHash.String())
}

func TestCompressedKeyToEthAddress(t *testing.T) {
	ethAddr, err := convert33BytesPubKeyToEthAddress("0x02bcbe39785b55a22383f82ac631ea7500e204627369c4ea01d9296af0ea573f57")
	require.NoError(t, err, "converting pub key to address")
	require.Equal(t, "0x0A1ec0002dDB927B03049F1aD8D589aBEA4Ba4b3", ethAddr.Hex())
}
