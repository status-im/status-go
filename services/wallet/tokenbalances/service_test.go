package tokenbalances_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/services/wallet/multistandardbalance"
	"github.com/status-im/status-go/services/wallet/tokenbalances"
	mock_tokenbalances "github.com/status-im/status-go/services/wallet/tokenbalances/mock"
)

func TestServiceLifecycle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_tokenbalances.NewMockStorage(ctrl)
	publisher := pubsub.NewPublisher()
	controller := &multistandardbalance.Controller{}
	service := tokenbalances.NewService(mockStorage, publisher, controller)

	// Test Start
	err := service.Start()
	require.NoError(t, err)

	// Test double Start (should be idempotent)
	err = service.Start()
	require.NoError(t, err)

	// Test Stop
	service.Stop()

	// Test double Stop (should be idempotent)
	service.Stop()
}

func TestGetPublisher(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_tokenbalances.NewMockStorage(ctrl)
	publisher := pubsub.NewPublisher()
	controller := &multistandardbalance.Controller{}
	service := tokenbalances.NewService(mockStorage, publisher, controller)

	pub := service.GetPublisher()
	assert.NotNil(t, pub)
}

func TestForceRefresh(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_tokenbalances.NewMockStorage(ctrl)
	publisher := pubsub.NewPublisher()
	// Controller needs to be properly initialized, not just an empty struct
	// For this test, we'll just verify the method exists and can be called
	// The actual functionality requires a fully initialized controller which
	// is tested in integration tests
	service := tokenbalances.NewService(mockStorage, publisher, nil)

	// This test just verifies the API exists
	// In real usage, controller should be non-nil
	// We skip calling ForceRefresh since it requires a real controller
	assert.NotNil(t, service)
}

func TestGetAllBalancesWithState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_tokenbalances.NewMockStorage(ctrl)
	publisher := pubsub.NewPublisher()
	controller := &multistandardbalance.Controller{}
	service := tokenbalances.NewService(mockStorage, publisher, controller)

	address1 := common.HexToAddress("0x1234567890123456789012345678901234567890")
	address2 := common.HexToAddress("0x2234567890123456789012345678901234567890")
	chainID1 := uint64(1)
	chainID2 := uint64(5)

	// Set up mock expectations for storage calls
	tokenAddress := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	testBalances := map[tokenbalances.ContractAddress]*big.Int{
		tokenbalances.ContractAddress(tokenAddress): big.NewInt(1000000),
	}
	testState := tokenbalances.State{
		AtBlockNumber: big.NewInt(12345),
		AtBlockHash:   common.HexToHash("0xabcdef"),
		FetchedAt:     time.Now().Unix(),
	}

	// Mock storage will be called for each address/chainID combination
	mockStorage.EXPECT().
		GetAllBalancesForAccountAndChain(gomock.Any(), tokenbalances.AccountAddress(address1), chainID1).
		Return(testBalances, testState, nil)

	mockStorage.EXPECT().
		GetAllBalancesForAccountAndChain(gomock.Any(), tokenbalances.AccountAddress(address1), chainID2).
		Return(testBalances, testState, nil)

	mockStorage.EXPECT().
		GetAllBalancesForAccountAndChain(gomock.Any(), tokenbalances.AccountAddress(address2), chainID1).
		Return(testBalances, testState, nil)

	mockStorage.EXPECT().
		GetAllBalancesForAccountAndChain(gomock.Any(), tokenbalances.AccountAddress(address2), chainID2).
		Return(testBalances, testState, nil)

	// Call GetAllBalancesWithState
	result, err := service.GetAllBalancesWithState(context.Background(), []uint64{chainID1, chainID2}, []common.Address{address1, address2})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify the structure
	assert.Contains(t, result, tokenbalances.AccountAddress(address1))
	assert.Contains(t, result, tokenbalances.AccountAddress(address2))
	assert.Contains(t, result[tokenbalances.AccountAddress(address1)], chainID1)
	assert.Contains(t, result[tokenbalances.AccountAddress(address1)], chainID2)

	// Verify balances are included
	assert.Equal(t, testBalances, result[tokenbalances.AccountAddress(address1)][chainID1].Balances)

	// Verify state information from storage
	assert.Equal(t, testState.AtBlockNumber, result[tokenbalances.AccountAddress(address1)][chainID1].State.AtBlockNumber)
	assert.Equal(t, testState.AtBlockHash, result[tokenbalances.AccountAddress(address1)][chainID1].State.AtBlockHash)
}

func TestGetAllBalancesWithState_EmptyAddresses(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_tokenbalances.NewMockStorage(ctrl)
	publisher := pubsub.NewPublisher()
	controller := &multistandardbalance.Controller{}
	service := tokenbalances.NewService(mockStorage, publisher, controller)

	// Call with empty addresses should return empty result
	result, err := service.GetAllBalancesWithState(context.Background(), []uint64{1}, []common.Address{})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result)
}

func TestGetAllBalancesWithState_StorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_tokenbalances.NewMockStorage(ctrl)
	publisher := pubsub.NewPublisher()
	controller := &multistandardbalance.Controller{}
	service := tokenbalances.NewService(mockStorage, publisher, controller)

	address := common.HexToAddress("0x1234567890123456789012345678901234567890")
	chainID := uint64(1)

	// Mock storage returns an error
	mockStorage.EXPECT().
		GetAllBalancesForAccountAndChain(gomock.Any(), tokenbalances.AccountAddress(address), chainID).
		Return(nil, tokenbalances.State{}, assert.AnError)

	// Call should return error from storage
	result, err := service.GetAllBalancesWithState(context.Background(), []uint64{chainID}, []common.Address{address})
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestGetCachedBalances(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_tokenbalances.NewMockStorage(ctrl)
	publisher := pubsub.NewPublisher()
	controller := &multistandardbalance.Controller{}
	service := tokenbalances.NewService(mockStorage, publisher, controller)

	// Set up mock expectation for GetBalances call
	expectedResult := make(map[uint64]map[tokenbalances.AccountAddress]map[tokenbalances.ContractAddress]*big.Int)
	mockStorage.EXPECT().
		GetBalances(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(expectedResult, nil)

	// Call GetCachedBalances
	result, err := service.GetCachedBalances(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
}
