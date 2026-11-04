package dynamic

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager"
)

func TestMetadata(t *testing.T) {

	metadata := &RLNMetadata{
		LastProcessedBlock: 128,
		ChainID:            big.NewInt(1155511),
		ContractAddress:    common.HexToAddress("0x9c09146844c1326c2dbc41c451766c7138f88155"),
		ValidRootsPerBlock: []group_manager.RootsPerBlock{{Root: [32]byte{1}, BlockNumber: 100}, {Root: [32]byte{2}, BlockNumber: 200}},
	}

	serializedMetadata, err := metadata.Serialize()
	require.NoError(t, err)
	unserializedMetadata, err := DeserializeMetadata(serializedMetadata)
	require.NoError(t, err)
	require.Equal(t, metadata.ChainID.Uint64(), unserializedMetadata.ChainID.Uint64())
	require.Equal(t, metadata.LastProcessedBlock, unserializedMetadata.LastProcessedBlock)
	require.Equal(t, metadata.ContractAddress.Hex(), unserializedMetadata.ContractAddress.Hex())
	require.Len(t, unserializedMetadata.ValidRootsPerBlock, len(metadata.ValidRootsPerBlock))
	require.Equal(t, metadata.ValidRootsPerBlock[0].BlockNumber, unserializedMetadata.ValidRootsPerBlock[0].BlockNumber)
	require.Equal(t, metadata.ValidRootsPerBlock[0].Root, unserializedMetadata.ValidRootsPerBlock[0].Root)
	require.Equal(t, metadata.ValidRootsPerBlock[1].BlockNumber, unserializedMetadata.ValidRootsPerBlock[1].BlockNumber)
	require.Equal(t, metadata.ValidRootsPerBlock[1].Root, unserializedMetadata.ValidRootsPerBlock[1].Root)

	// Handle cases where the chainId is not specified (for some reason?) or no valid roots were specified
	metadata.ChainID = nil
	metadata.ValidRootsPerBlock = nil
	_, err = metadata.Serialize()
	require.Error(t, err)

	metadata.ChainID = big.NewInt(1)
	serializedMetadata, err = metadata.Serialize()
	require.NoError(t, err)

	unserializedMetadata, err = DeserializeMetadata(serializedMetadata)
	require.NoError(t, err)
	require.Equal(t, uint64(1), unserializedMetadata.ChainID.Uint64())
}
