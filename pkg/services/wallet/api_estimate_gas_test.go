package wallet

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/pkg/services/wallet/wallettypes"
)

// a tx's from and to, as JSON fields
const txAddrs = `"from":"0x0000000000000000000000000000000000000001","to":"0x0000000000000000000000000000000000000002"`

func TestEstimateTransactionGas(t *testing.T) {
	for _, tc := range []struct {
		name     string
		args     string
		raw      uint64
		expected hexutil.Uint64
	}{
		{"buffered once", "{" + txAddrs + "}", 200000, 210000},
		{"dApp gas above the buffered estimate is kept", "{" + txAddrs + `,"gas":"0x493e0"}`, 200000, 300000},
		// e.g. a dApp's 21000 for ETH to a new account after Glamsterdam
		{"dApp gas below the buffered estimate is raised", "{" + txAddrs + `,"gas":"0x5208"}`, 200000, 210000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var estimatedArgs wallettypes.SendTxArgs
			estimate := func(chainID uint64, args wallettypes.SendTxArgs) (uint64, error) {
				require.Equal(t, uint64(1), chainID)
				estimatedArgs = args
				return tc.raw, nil
			}
			got, err := estimateTransactionGas(estimate, 1, tc.args)
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
			require.Equal(t, "0x0000000000000000000000000000000000000002", estimatedArgs.To.Hex())
		})
	}

	t.Run("bad JSON isn't estimated", func(t *testing.T) {
		_, err := estimateTransactionGas(func(uint64, wallettypes.SendTxArgs) (uint64, error) {
			t.Fatal("estimated a bad request")
			return 0, nil
		}, 1, "{")
		require.Error(t, err)
	})

	t.Run("a failed estimate gives no gas, not the dApp's", func(t *testing.T) {
		_, err := estimateTransactionGas(func(uint64, wallettypes.SendTxArgs) (uint64, error) {
			return 0, errors.New("execution reverted")
		}, 1, "{"+txAddrs+`,"gas":"0x5208"}`)
		require.Error(t, err)
	})
}
