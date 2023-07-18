package collectibles

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

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
