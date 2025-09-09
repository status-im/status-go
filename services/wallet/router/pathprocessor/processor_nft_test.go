package pathprocessor

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/params"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
)

func TestNFTProcessor_Name(t *testing.T) {
	processor := NewNFTProcessor(nil, nil)
	assert.Equal(t, "ERC721Transfer", processor.Name())
}

func TestNFTProcessor_AvailableFor(t *testing.T) {
	processor := NewNFTProcessor(nil, nil)

	// Test with CryptoKitties contract
	cryptoKittiesParams := ProcessorInputParams{
		FromChain: &params.Network{ChainID: walletCommon.EthereumMainnet},
		ToChain:   &params.Network{ChainID: walletCommon.EthereumMainnet},
		FromToken: &tokenTypes.Token{
			Address: CryptoKittiesContractID.Address,
		},
		ToToken: nil, // NFT transfers don't have destination token
	}

	available, err := processor.AvailableFor(cryptoKittiesParams)
	require.NoError(t, err)
	assert.True(t, available, "Should be available for CryptoKitties contract")

	// Test with CryptoPunks contract
	cryptoPunksParams := ProcessorInputParams{
		FromChain: &params.Network{ChainID: walletCommon.EthereumMainnet},
		ToChain:   &params.Network{ChainID: walletCommon.EthereumMainnet},
		FromToken: &tokenTypes.Token{
			Address: CryptoPunksContractID.Address,
		},
		ToToken: nil,
	}

	available, err = processor.AvailableFor(cryptoPunksParams)
	require.NoError(t, err)
	assert.True(t, available, "Should be available for CryptoPunks contract")

	// Test with generic ERC721 contract
	genericERC721Params := ProcessorInputParams{
		FromChain: &params.Network{ChainID: walletCommon.EthereumMainnet},
		ToChain:   &params.Network{ChainID: walletCommon.EthereumMainnet},
		FromToken: &tokenTypes.Token{
			Address: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		},
		ToToken: nil,
	}

	available, err = processor.AvailableFor(genericERC721Params)
	require.NoError(t, err)
	assert.True(t, available, "Should be available for generic ERC721 contract")

	// Test with cross-chain transfer (should not be available)
	crossChainParams := ProcessorInputParams{
		FromChain: &params.Network{ChainID: walletCommon.EthereumMainnet},
		ToChain:   &params.Network{ChainID: walletCommon.OptimismMainnet},
		FromToken: &tokenTypes.Token{
			Address: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		},
		ToToken: nil,
	}

	available, err = processor.AvailableFor(crossChainParams)
	require.NoError(t, err)
	assert.False(t, available, "Should not be available for cross-chain transfers")

	// Test with destination token (should not be available for NFT transfers)
	withToTokenParams := ProcessorInputParams{
		FromChain: &params.Network{ChainID: walletCommon.EthereumMainnet},
		ToChain:   &params.Network{ChainID: walletCommon.EthereumMainnet},
		FromToken: &tokenTypes.Token{
			Address: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		},
		ToToken: &tokenTypes.Token{
			Address: common.HexToAddress("0x0987654321098765432109876543210987654321"),
		},
	}

	available, err = processor.AvailableFor(withToTokenParams)
	require.NoError(t, err)
	assert.False(t, available, "Should not be available when ToToken is specified")
}

func TestNFTProcessor_GetHandlerForContract(t *testing.T) {
	processor := NewNFTProcessor(nil, nil)

	// Test CryptoKitties handler selection
	cryptoKittiesParams := ProcessorInputParams{
		FromChain: &params.Network{ChainID: walletCommon.EthereumMainnet},
		FromToken: &tokenTypes.Token{
			Address: CryptoKittiesContractID.Address,
		},
	}

	handler := processor.getHandlerForContract(cryptoKittiesParams)
	require.NotNil(t, handler)
	assert.Equal(t, "CryptoKittiesTransfer", handler.Name())

	// Test CryptoPunks handler selection
	cryptoPunksParams := ProcessorInputParams{
		FromChain: &params.Network{ChainID: walletCommon.EthereumMainnet},
		FromToken: &tokenTypes.Token{
			Address: CryptoPunksContractID.Address,
		},
	}

	handler = processor.getHandlerForContract(cryptoPunksParams)
	require.NotNil(t, handler)
	assert.Equal(t, "CryptoPunksTransfer", handler.Name())

	// Test generic ERC721 handler selection
	genericParams := ProcessorInputParams{
		FromChain: &params.Network{ChainID: walletCommon.EthereumMainnet},
		FromToken: &tokenTypes.Token{
			Address: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		},
	}

	handler = processor.getHandlerForContract(genericParams)
	require.NotNil(t, handler)
	assert.Equal(t, "ERC721Transfer", handler.Name())
}

func TestNFTProcessor_CalculateFees(t *testing.T) {
	processor := NewNFTProcessor(nil, nil)

	params := ProcessorInputParams{}
	bonderFee, tokenFee, err := processor.CalculateFees(params)

	require.NoError(t, err)
	assert.Equal(t, "0", bonderFee.String())
	assert.Equal(t, "0", tokenFee.String())
}

func TestNFTProcessor_CalculateAmountOut(t *testing.T) {
	processor := NewNFTProcessor(nil, nil)

	params := ProcessorInputParams{
		AmountIn: big.NewInt(1),
	}

	amountOut, err := processor.CalculateAmountOut(params)
	require.NoError(t, err)
	assert.Equal(t, params.AmountIn, amountOut)
}
