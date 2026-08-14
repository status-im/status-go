package chain

import (
	"context"
	"errors"
	"math/big"
	"runtime"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	sdkethclient "github.com/status-im/go-wallet-sdk/pkg/ethclient"

	"github.com/stretchr/testify/require"

	"go.uber.org/mock/gomock"

	ethclient "github.com/status-im/status-go/internal/rpc/chain/ethclient"
	mock_ethclient "github.com/status-im/status-go/internal/rpc/chain/ethclient/mock/client/ethclient"
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
	block := &sdkethclient.BlockWithTxHashes{}

	// Expect the first client to be called, others should not be called, should succeed
	ethClients[0].EXPECT().EthGetBlockByHashWithTxHashes(ctx, hash).Return(block, nil).Times(1)
	ethClients[1].EXPECT().EthGetBlockByHashWithTxHashes(ctx, hash).Return(nil, nil).Times(0)
	ethClients[2].EXPECT().EthGetBlockByHashWithTxHashes(ctx, hash).Return(nil, nil).Times(0)
	_, err := client.EthGetBlockByHashWithTxHashes(ctx, hash)
	require.NoError(t, err)

	// Expect the first and second client to be called, others should not be called, should succeed
	ethClients[0].EXPECT().EthGetBlockByHashWithTxHashes(ctx, hash).Return(nil, errors.New("some error")).Times(1)
	ethClients[1].EXPECT().EthGetBlockByHashWithTxHashes(ctx, hash).Return(block, nil).Times(1)
	ethClients[2].EXPECT().EthGetBlockByHashWithTxHashes(ctx, hash).Return(nil, nil).Times(0)
	_, err = client.EthGetBlockByHashWithTxHashes(ctx, hash)
	require.NoError(t, err)

	// Expect the all client to be called, should succeed
	ethClients[0].EXPECT().EthGetBlockByHashWithTxHashes(ctx, hash).Return(nil, errors.New("some error")).Times(1)
	ethClients[1].EXPECT().EthGetBlockByHashWithTxHashes(ctx, hash).Return(nil, errors.New("some other error")).Times(1)
	ethClients[2].EXPECT().EthGetBlockByHashWithTxHashes(ctx, hash).Return(block, nil).Times(1)
	_, err = client.EthGetBlockByHashWithTxHashes(ctx, hash)
	require.NoError(t, err)

	// Expect the all client to be called, should fail
	ethClients[0].EXPECT().EthGetBlockByHashWithTxHashes(ctx, hash).Return(nil, errors.New("some error")).Times(1)
	ethClients[1].EXPECT().EthGetBlockByHashWithTxHashes(ctx, hash).Return(nil, errors.New("some other error")).Times(1)
	ethClients[2].EXPECT().EthGetBlockByHashWithTxHashes(ctx, hash).Return(nil, errors.New("some other other error")).Times(1)
	_, err = client.EthGetBlockByHashWithTxHashes(ctx, hash)
	require.Error(t, err)
}

// TestClientWithFallback_CloseStopsOperations tests that closing the client
// properly stops all ongoing operations
func TestClientWithFallback_CloseStopsOperations(t *testing.T) {
	client, ethClients, cleanup := setupClientTest(t)
	defer cleanup()

	ctx := context.Background()
	addr := common.HexToAddress("0x1234")

	// Create channels to coordinate the test
	done := make(chan struct{})
	operationStarted := make(chan struct{})

	// Set up the first client to block for a short time
	ethClients[0].EXPECT().CodeAt(ctx, addr, nil).DoAndReturn(
		func(ctx context.Context, addr common.Address, blockNumber *big.Int) ([]byte, error) {
			close(operationStarted) // Signal that operation has started

			// Wait for context cancellation
			<-ctx.Done()
			return nil, ctx.Err()
		}).Times(1)

	// Set up expectations for other clients - they should not be called
	// because the operation should be cancelled after the first client
	ethClients[1].EXPECT().CodeAt(ctx, addr, nil).Times(0)
	ethClients[2].EXPECT().CodeAt(ctx, addr, nil).Times(0)

	// Set up expectations for Close on all clients
	for _, ethClient := range ethClients {
		ethClient.EXPECT().Close().Times(1)
	}

	// Start the operation in a goroutine
	go func() {
		defer close(done)
		_, err := client.CodeAt(ctx, addr, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "context canceled")
	}()

	// Wait for operation to start
	<-operationStarted

	// Close the client while operation is running
	client.Close()

	// Wait for the operation to complete
	<-done

	// Verify that subsequent calls fail immediately
	_, err := client.CodeAt(ctx, addr, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "client is closed")
}

// TestClientWithFallback_CloseStopsMultipleOperations tests that closing the client
// properly stops multiple concurrent operations
func TestClientWithFallback_CloseStopsMultipleOperations(t *testing.T) {
	client, ethClients, cleanup := setupClientTest(t)
	defer cleanup()

	ctx := context.Background()
	addr1 := common.HexToAddress("0x1234")
	addr2 := common.HexToAddress("0x5678")
	hash := common.HexToHash("0xabcd")
	blockNumber := big.NewInt(100)

	// Create channels to coordinate the test
	operation1Started := make(chan struct{})
	operation2Started := make(chan struct{})
	operation3Started := make(chan struct{})

	operation1Done := make(chan struct{})
	operation2Done := make(chan struct{})
	operation3Done := make(chan struct{})

	allOperationsStarted := make(chan struct{})

	// Helper function to create operation handlers with common logic
	createOperationHandler := func(operationStarted chan struct{}) func(context.Context) error {
		return func(ctx context.Context) error {
			close(operationStarted)

			// Wait for all operations to start or context to be cancelled
			select {
			case <-allOperationsStarted:
				// Continue with normal processing
			case <-ctx.Done():
				return ctx.Err()
			}

			// Now wait for context cancellation
			<-ctx.Done()
			return ctx.Err()
		}
	}

	// Set up the mock responses for operation 1
	ethClients[0].EXPECT().CodeAt(ctx, addr1, nil).DoAndReturn(
		func(ctx context.Context, addr common.Address, blockNumber *big.Int) ([]byte, error) {
			err := createOperationHandler(operation1Started)(ctx)
			return nil, err
		}).Times(1)

	// Set up the mock responses for operation 2
	ethClients[0].EXPECT().BalanceAt(ctx, addr2, blockNumber).DoAndReturn(
		func(ctx context.Context, addr common.Address, blockNumber *big.Int) (*big.Int, error) {
			err := createOperationHandler(operation2Started)(ctx)
			return nil, err
		}).Times(1)

	// Set up the mock responses for operation 3
	ethClients[0].EXPECT().EthGetBlockByHashWithTxHashes(ctx, hash).DoAndReturn(
		func(ctx context.Context, hash common.Hash) (*sdkethclient.BlockWithTxHashes, error) {
			err := createOperationHandler(operation3Started)(ctx)
			return nil, err
		}).Times(1)

	// Set up expectations for Close on all clients
	for _, ethClient := range ethClients {
		ethClient.EXPECT().Close().Times(1)
	}

	// Start operation 1 in a goroutine
	go func() {
		defer close(operation1Done)
		_, err := client.CodeAt(ctx, addr1, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "context canceled")
	}()

	// Start operation 2 in a goroutine
	go func() {
		defer close(operation2Done)
		_, err := client.BalanceAt(ctx, addr2, blockNumber)
		require.Error(t, err)
		require.Contains(t, err.Error(), "context canceled")
	}()

	// Start operation 3 in a goroutine
	go func() {
		defer close(operation3Done)
		_, err := client.EthGetBlockByHashWithTxHashes(ctx, hash)
		require.Error(t, err)
		require.Contains(t, err.Error(), "context canceled")
	}()

	// Wait for all operations to start
	<-operation1Started
	<-operation2Started
	<-operation3Started

	// Signal all operations that they can proceed past their initial state
	close(allOperationsStarted)

	// Wait a small amount of time for operations to reach the waiting state
	runtime.Gosched()

	// Close the client while operations are running
	client.Close()

	// All operations should complete with cancellation errors
	<-operation1Done
	<-operation2Done
	<-operation3Done
}
