package pathprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mock_client "github.com/status-im/status-go/internal/rpc/chain/mock/client"
	mock_rpcclient "github.com/status-im/status-go/internal/rpc/mock/client"
	"github.com/status-im/status-go/params"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

func TestStickersBuyProcessorEstimateGas(t *testing.T) {
	ctrl := gomock.NewController(t)
	ethClientGetter := mock_rpcclient.NewMockEthClientGetter(ctrl)
	ethClient := mock_client.NewMockClientInterface(ctrl)

	ethClientGetter.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(ethClient, nil)
	ethClient.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(uint64(1000), nil)

	processor := NewStickersBuyProcessor(ethClientGetter, nil)
	gas, err := processor.EstimateGas(ProcessorInputParams{
		FromChain: &params.Network{ChainID: walletCommon.EthereumMainnet},
	}, nil)

	require.NoError(t, err)
	require.Equal(t, uint64(1050), gas)
}
