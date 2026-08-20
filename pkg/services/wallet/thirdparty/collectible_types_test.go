package thirdparty

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"

	w_common "github.com/status-im/status-go/pkg/services/wallet/common"
)

func TestContractID_Equality(t *testing.T) {
	// Test case 1: Same contract IDs should return true
	contractID1 := ContractID{
		ChainID: w_common.ChainID(w_common.EthereumMainnet),
		Address: common.HexToAddress("0x06012c8cf97bead5deae237070f9587f8e7a266d"),
	}
	contractID2 := ContractID{
		ChainID: w_common.ChainID(w_common.EthereumMainnet),
		Address: common.HexToAddress("0x06012c8cf97bead5deae237070f9587f8e7a266d"),
	}

	assert.True(t, contractID1 == contractID2, "Same contract IDs should return true")
	assert.True(t, contractID2 == contractID1, "Same contract IDs should be symmetric")

	// Test case 2: Different chain IDs should return false
	contractID3 := ContractID{
		ChainID: w_common.ChainID(w_common.OptimismMainnet),
		Address: common.HexToAddress("0x06012c8cf97bead5deae237070f9587f8e7a266d"),
	}

	assert.False(t, contractID1 == contractID3, "Different chain IDs should return false")

	// Test case 3: Different addresses should return false
	contractID4 := ContractID{
		ChainID: w_common.ChainID(w_common.EthereumMainnet),
		Address: common.HexToAddress("0xb47e3cd837dDF8e4c57F05d70Ab865de6e193BBB"),
	}

	assert.False(t, contractID1 == contractID4, "Different addresses should return false")

	// Test case 4: Both different chain ID and address should return false
	contractID5 := ContractID{
		ChainID: w_common.ChainID(w_common.ArbitrumMainnet),
		Address: common.HexToAddress("0x1111111111111111111111111111111111111111"),
	}

	assert.False(t, contractID1 == contractID5, "Different chain ID and address should return false")
}

func TestContractID_HashKey(t *testing.T) {
	contractID := ContractID{
		ChainID: w_common.ChainID(w_common.EthereumMainnet),
		Address: common.HexToAddress("0x06012c8cf97bead5deae237070f9587f8e7a266d"),
	}

	hashKey := contractID.HashKey()
	expected := "1+0x06012c8cf97BEaD5deAe237070F9587f8E7A266d"
	assert.Equal(t, expected, hashKey, "HashKey should return correct format")

	// Test that same contract IDs produce same hash keys
	contractID2 := ContractID{
		ChainID: w_common.ChainID(w_common.EthereumMainnet),
		Address: common.HexToAddress("0x06012c8cf97bead5deae237070f9587f8e7a266d"),
	}

	assert.Equal(t, contractID.HashKey(), contractID2.HashKey(), "Same contract IDs should have same hash keys")
}
