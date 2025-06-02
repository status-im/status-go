package chain

import (
	"context"
	"errors"
	"reflect"
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
	ethClients := make([]ethclient.RPSLimitedEthClientInterface, 0)

	for i := 0; i < 3; i++ {
		ethCl := mock_ethclient.NewMockRPSLimitedEthClientInterface(mockCtrl)
		ethCl.EXPECT().GetProviderName().AnyTimes().Return("test" + strconv.Itoa(i) + "_provider")
		ethCl.EXPECT().GetCircuitName().AnyTimes().Return("test" + strconv.Itoa(i) + "_circuit")
		ethCl.EXPECT().GetLimiter().AnyTimes().Return(nil)
		ethCl.EXPECT().ExecuteWithRPSLimit(gomock.Any()).DoAndReturn(func(f func(client ethclient.EthClientInterface) (interface{}, error)) (interface{}, error) {
			return f(ethCl)
		}).AnyTimes()

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

func TestClientWithFallback_Copy(t *testing.T) {
	client, _, cleanup := setupClientTest(t)
	defer cleanup()

	// Setup test values
	testTag := "test-tag"
	testGroupTag := "test-group-tag"
	testNotifier := func(chainId uint64, message string) {}

	// Set values on the original client
	client.tag = testTag
	client.groupTag = testGroupTag
	client.WalletNotifier = testNotifier

	// Copy the client
	clientCopy := client.Copy().(*ClientWithFallback)

	// Check that the copy has the same values
	require.Equal(t, client.ChainID, clientCopy.ChainID)
	require.Equal(t, client.tag, clientCopy.tag)
	require.Equal(t, client.groupTag, clientCopy.groupTag)
	require.Equal(t, client.LastCheckedAt, clientCopy.LastCheckedAt)

	// Verify that both clients have the same ethClients slice
	require.Equal(t, len(client.ethClients), len(clientCopy.ethClients))
	for i := 0; i < len(client.ethClients); i++ {
		require.Equal(t, client.ethClients[i], clientCopy.ethClients[i])
	}

	// Check that pointer values are the same (shallow copy)
	require.Same(t, client.isConnected, clientCopy.isConnected)
	require.Same(t, client.circuitbreaker, clientCopy.circuitbreaker)
	require.Same(t, client.providersHealthManager, clientCopy.providersHealthManager)

	// Verify that function references are the same
	clientFuncPtr := getFuncPtr(client.WalletNotifier)
	copyFuncPtr := getFuncPtr(clientCopy.WalletNotifier)
	require.Equal(t, clientFuncPtr, copyFuncPtr)

	// Modify the copy, ensure it doesn't affect the original
	clientCopy.tag = "new-tag"
	clientCopy.groupTag = "new-group-tag"
	require.Equal(t, testTag, client.tag)
	require.Equal(t, testGroupTag, client.groupTag)
}

// Helper function to get a comparable value for function pointers
func getFuncPtr(f func(uint64, string)) uintptr {
	if f == nil {
		return 0
	}
	return reflect.ValueOf(f).Pointer()
}
