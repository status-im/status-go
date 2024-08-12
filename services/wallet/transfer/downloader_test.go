package transfer

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/golang/mock/gomock"

	mock_client "github.com/status-im/status-go/rpc/chain/mock/client"
	walletCommon "github.com/status-im/status-go/services/wallet/common"

	"github.com/stretchr/testify/require"
)

func TestERC20Downloader_getHeadersInRange(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClientIface := mock_client.NewMockClientInterface(mockCtrl)

	accounts := []common.Address{
		common.HexToAddress("0x1"),
	}
	chainID := walletCommon.EthereumMainnet
	signer := types.LatestSignerForChainID(big.NewInt(int64(chainID)))
	mockClientIface.EXPECT().NetworkID().Return(chainID).AnyTimes()

	downloader := NewERC20TransfersDownloader(
		mockClientIface,
		accounts,
		signer,
		false,
	)

	ctx := context.Background()

	_, err := downloader.GetHeadersInRange(ctx, big.NewInt(10), big.NewInt(0))
	require.Error(t, err)

	mockClientIface.EXPECT().FilterLogs(ctx, gomock.Any()).Return(nil, nil).AnyTimes()
	_, err = downloader.GetHeadersInRange(ctx, big.NewInt(0), big.NewInt(10))
	require.NoError(t, err)
}
