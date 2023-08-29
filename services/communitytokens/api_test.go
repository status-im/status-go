package communitytokens

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/services/wallet/bigint"
)

func TestDeploymentParameters(t *testing.T) {
	var testCases = []struct {
		name       string
		parameters DeploymentParameters
		isError    bool
	}{
		{
			name:       "emptyName",
			parameters: DeploymentParameters{"", "SYMBOL", &bigint.BigInt{Int: big.NewInt(int64(123))}, false, false, false, "", "", ""},
			isError:    true,
		},
		{
			name:       "emptySymbol",
			parameters: DeploymentParameters{"NAME", "", &bigint.BigInt{Int: big.NewInt(123)}, false, false, false, "", "", ""},
			isError:    true,
		},
		{
			name:       "negativeSupply",
			parameters: DeploymentParameters{"NAME", "SYM", &bigint.BigInt{Int: big.NewInt(-123)}, false, false, false, "", "", ""},
			isError:    true,
		},
		{
			name:       "zeroSupply",
			parameters: DeploymentParameters{"NAME", "SYM", &bigint.BigInt{Int: big.NewInt(0)}, false, false, false, "", "", ""},
			isError:    false,
		},
		{
			name:       "negativeSupplyAndInfinite",
			parameters: DeploymentParameters{"NAME", "SYM", &bigint.BigInt{Int: big.NewInt(-123)}, true, false, false, "", "", ""},
			isError:    false,
		},
		{
			name:       "supplyGreaterThanMax",
			parameters: DeploymentParameters{"NAME", "SYM", &bigint.BigInt{Int: big.NewInt(maxSupply + 1)}, false, false, false, "", "", ""},
			isError:    true,
		},
		{
			name:       "supplyIsMax",
			parameters: DeploymentParameters{"NAME", "SYM", &bigint.BigInt{Int: big.NewInt(maxSupply)}, false, false, false, "", "", ""},
			isError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.parameters.Validate(false)
			if tc.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}

	notInfiniteSupplyParams := DeploymentParameters{"NAME", "SYM", &bigint.BigInt{Int: big.NewInt(123)}, false, false, false, "", "", ""}
	requiredSupply := big.NewInt(123)
	require.Equal(t, notInfiniteSupplyParams.GetSupply(), requiredSupply)
	infiniteSupplyParams := DeploymentParameters{"NAME", "SYM", &bigint.BigInt{Int: big.NewInt(123)}, true, false, false, "", "", ""}
	requiredSupply = infiniteSupplyParams.GetInfiniteSupply()
	require.Equal(t, infiniteSupplyParams.GetSupply(), requiredSupply)
}

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
