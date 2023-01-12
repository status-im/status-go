package collectibles

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeploymentParameters(t *testing.T) {
	var testCases = []struct {
		name       string
		parameters DeploymentParameters
		isError    bool
	}{
		{
			name:       "emptyName",
			parameters: DeploymentParameters{"", "SYMBOL", 123, false, false, false, ""},
			isError:    true,
		},
		{
			name:       "emptySymbol",
			parameters: DeploymentParameters{"NAME", "", 123, false, false, false, ""},
			isError:    true,
		},
		{
			name:       "negativeSupply",
			parameters: DeploymentParameters{"NAME", "SYM", -123, false, false, false, ""},
			isError:    true,
		},
		{
			name:       "zeroSupply",
			parameters: DeploymentParameters{"NAME", "SYM", 0, false, false, false, ""},
			isError:    false,
		},
		{
			name:       "negativeSupplyAndInfinite",
			parameters: DeploymentParameters{"NAME", "SYM", -123, true, false, false, ""},
			isError:    false,
		},
		{
			name:       "supplyGreaterThanMax",
			parameters: DeploymentParameters{"NAME", "SYM", maxSupply + 1, false, false, false, ""},
			isError:    true,
		},
		{
			name:       "supplyIsMax",
			parameters: DeploymentParameters{"NAME", "SYM", maxSupply, false, false, false, ""},
			isError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.parameters.Validate()
			if tc.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}

	notInfiniteSupplyParams := DeploymentParameters{"NAME", "SYM", 123, false, false, false, ""}
	requiredSupply := big.NewInt(123)
	require.Equal(t, notInfiniteSupplyParams.GetSupply(), requiredSupply)
	infiniteSupplyParams := DeploymentParameters{"NAME", "SYM", 123, true, false, false, ""}
	requiredSupply = infiniteSupplyParams.GetInfiniteSupply()
	require.Equal(t, infiniteSupplyParams.GetSupply(), requiredSupply)
}
