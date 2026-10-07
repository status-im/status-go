package wallet

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/status-im/status-go/pkg/services/wallet/following"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/efp"
	mock_efp "github.com/status-im/status-go/pkg/services/wallet/thirdparty/efp/mock"
)

func TestAPIGetFollowingWithoutManager(t *testing.T) {
	api := NewAPI(&Service{})
	userAddress := common.HexToAddress("0x742d35cc6cf4c7c7")
	ctx := t.Context()

	addresses, err := api.GetFollowingAddresses(ctx, userAddress, "", 10, 0)
	require.Error(t, err)
	require.Nil(t, addresses)
	require.Contains(t, err.Error(), "following manager not initialized")

	count, err := api.GetFollowingStats(ctx, userAddress)
	require.Error(t, err)
	require.Equal(t, 0, count)
	require.Contains(t, err.Error(), "following manager not initialized")
}

func TestAPIGetFollowingDelegatesToManager(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := t.Context()
	userAddress := common.HexToAddress("0x742d35cc6cf4c7c7")
	expected := []efp.FollowingAddress{{
		Address: common.HexToAddress("0x983110309620D911731Ac0932219af06091b6744"),
		ENSName: "vitalik.eth",
		Tags:    []string{"ens"},
	}}

	mockProvider := mock_efp.NewMockFollowingDataProvider(mockCtrl)
	mockProvider.EXPECT().ID().Return("efp").AnyTimes()
	mockProvider.EXPECT().IsConnected().Return(true).Times(2)
	mockProvider.EXPECT().FetchFollowingAddresses(ctx, userAddress, "vitalik", 5, 10).Return(expected, nil)
	mockProvider.EXPECT().FetchFollowingStats(ctx, userAddress).Return(42, nil)

	api := NewAPI(&Service{
		followingManager: following.NewManager(mockProvider, zap.NewNop()),
	})

	addresses, err := api.GetFollowingAddresses(ctx, userAddress, "vitalik", 5, 10)
	require.NoError(t, err)
	require.Equal(t, expected, addresses)

	count, err := api.GetFollowingStats(ctx, userAddress)
	require.NoError(t, err)
	require.Equal(t, 42, count)
}
