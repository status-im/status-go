package chain

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/rpc/chain/ethclient"
	mock_ethclient "github.com/status-im/status-go/rpc/chain/ethclient/mock/client/ethclient"

	"github.com/stretchr/testify/require"

	gomock "go.uber.org/mock/gomock"
)

func setupClientTest(t *testing.T) (*ClientWithFallback, []*mock_ethclient.MockRPSLimitedEthClientInterface, func()) {
	mockCtrl := gomock.NewController(t)

	mockEthClients := make([]*mock_ethclient.MockRPSLimitedEthClientInterface, 0)
	ethClients := make([]ethclient.EthClientInterface, 0)

	for i := 0; i < 3; i++ {
		ethCl := mock_ethclient.NewMockRPSLimitedEthClientInterface(mockCtrl)
		ethCl.EXPECT().GetName().AnyTimes().Return("test" + strconv.Itoa(i))

		mockEthClients = append(mockEthClients, ethCl)
		ethClients = append(ethClients, ethCl)
	}

	client := NewClient(ethClients, 0, nil)

	cleanup := func() {
		mockCtrl.Finish()
	}
	return client, mockEthClients, cleanup
}

// Basic test, just make sure
func TestClient_Fallbacks(t *testing.T) {
	client, ethClients, cleanup := setupClientTest(t)
	defer cleanup()

	ctx := context.Background()
	hash := common.HexToHash("0x1234")
	block := &types.Block{}

	// Expect the first client to be called, others should not be called, should succeed
	ethClients[0].EXPECT().BlockByHash(ctx, hash).Return(block, nil).Times(1)
	ethClients[1].EXPECT().BlockByHash(ctx, hash).Return(nil, nil).Times(0)
	ethClients[2].EXPECT().BlockByHash(ctx, hash).Return(nil, nil).Times(0)
	_, err := client.BlockByHash(ctx, hash)
	require.NoError(t, err)

	// Expect the first and second client to be called, others should not be called, should succeed
	ethClients[0].EXPECT().BlockByHash(ctx, hash).Return(nil, errors.New("some error")).Times(1)
	ethClients[1].EXPECT().BlockByHash(ctx, hash).Return(block, nil).Times(1)
	ethClients[2].EXPECT().BlockByHash(ctx, hash).Return(nil, nil).Times(0)
	_, err = client.BlockByHash(ctx, hash)
	require.NoError(t, err)

	// Expect the all client to be called, should succeed
	ethClients[0].EXPECT().BlockByHash(ctx, hash).Return(nil, errors.New("some error")).Times(1)
	ethClients[1].EXPECT().BlockByHash(ctx, hash).Return(nil, errors.New("some other error")).Times(1)
	ethClients[2].EXPECT().BlockByHash(ctx, hash).Return(block, nil).Times(1)
	_, err = client.BlockByHash(ctx, hash)
	require.NoError(t, err)

	// Expect the all client to be called, should fail
	ethClients[0].EXPECT().BlockByHash(ctx, hash).Return(nil, errors.New("some error")).Times(1)
	ethClients[1].EXPECT().BlockByHash(ctx, hash).Return(nil, errors.New("some other error")).Times(1)
	ethClients[2].EXPECT().BlockByHash(ctx, hash).Return(nil, errors.New("some other other error")).Times(1)
	_, err = client.BlockByHash(ctx, hash)
	require.Error(t, err)
}
