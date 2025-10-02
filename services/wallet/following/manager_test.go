package following

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/services/wallet/thirdparty/efp"
	mock_efp "github.com/status-im/status-go/services/wallet/thirdparty/efp/mock"
)

func TestFetchFollowingAddressesSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.TODO()
	userAddress := common.HexToAddress("0x742d35cc6cf4c7c7")

	expected := []efp.FollowingAddress{
		{
			Address: common.HexToAddress("0x983110309620D911731Ac0932219af06091b6744"),
			ENSName: "vitalik.eth",
		},
	}

	mockProvider := mock_efp.NewMockFollowingDataProvider(mockCtrl)
	mockProvider.EXPECT().ID().Return("efp").AnyTimes()
	mockProvider.EXPECT().IsConnected().Return(true)
	mockProvider.EXPECT().FetchFollowingAddresses(ctx, userAddress, "", 10, 0).Return(expected, nil)

	manager := NewManager([]efp.FollowingDataProvider{mockProvider})

	result, err := manager.FetchFollowingAddresses(ctx, userAddress, "", 10, 0)

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, expected[0].Address, result[0].Address)
	require.Equal(t, expected[0].ENSName, result[0].ENSName)
}

func TestFetchFollowingStatsSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.TODO()
	userAddress := common.HexToAddress("0x742d35cc6cf4c7c7")
	expectedCount := 150

	mockProvider := mock_efp.NewMockFollowingDataProvider(mockCtrl)
	mockProvider.EXPECT().IsConnected().Return(true)
	mockProvider.EXPECT().FetchFollowingStats(ctx, userAddress).Return(expectedCount, nil)

	manager := NewManager([]efp.FollowingDataProvider{mockProvider})

	result, err := manager.FetchFollowingStats(ctx, userAddress)

	require.NoError(t, err)
	require.Equal(t, expectedCount, result)
}

func TestFetchFollowingAddressesProviderNotConnected(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.TODO()
	userAddress := common.HexToAddress("0x742d35cc6cf4c7c7")

	mockProvider := mock_efp.NewMockFollowingDataProvider(mockCtrl)
	mockProvider.EXPECT().IsConnected().Return(false)
	mockProvider.EXPECT().ID().Return("efp").AnyTimes()

	manager := NewManager([]efp.FollowingDataProvider{mockProvider})

	result, err := manager.FetchFollowingAddresses(ctx, userAddress, "", 10, 0)

	require.NoError(t, err)
	require.Len(t, result, 0)
}

func TestFetchFollowingAddressesProviderError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.TODO()
	userAddress := common.HexToAddress("0x742d35cc6cf4c7c7")
	expectedError := errors.New("provider error")

	mockProvider := mock_efp.NewMockFollowingDataProvider(mockCtrl)
	mockProvider.EXPECT().ID().Return("efp").AnyTimes()
	mockProvider.EXPECT().IsConnected().Return(true)
	mockProvider.EXPECT().FetchFollowingAddresses(ctx, userAddress, "", 10, 0).Return(nil, expectedError)

	manager := NewManager([]efp.FollowingDataProvider{mockProvider})

	result, err := manager.FetchFollowingAddresses(ctx, userAddress, "", 10, 0)

	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, expectedError, err)
}

func TestFetchFollowingAddressesNoProviders(t *testing.T) {
	ctx := context.TODO()
	userAddress := common.HexToAddress("0x742d35cc6cf4c7c7")

	manager := NewManager([]efp.FollowingDataProvider{})

	result, err := manager.FetchFollowingAddresses(ctx, userAddress, "", 10, 0)

	require.NoError(t, err)
	require.Len(t, result, 0)
}
