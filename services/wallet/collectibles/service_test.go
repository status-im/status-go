package collectibles

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/services/wallet/collectibles/ownership"
	mock_ownership "github.com/status-im/status-go/services/wallet/collectibles/ownership/mock"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

func TestServiceShouldTriggerLoadFetchIfNotCachedSkipsWhenLoaderExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chainID := walletCommon.ChainID(1)
	address := common.HexToAddress("0x123")

	storage := mock_ownership.NewMockOwnershipStorage(ctrl)
	storage.EXPECT().GetOwnershipUpdateTimestamp(gomock.Any(), gomock.Any()).Return(ownership.InvalidTimestamp, nil).AnyTimes()
	storage.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil, nil, nil).AnyTimes()

	accountsProvider := mock_ownership.NewMockAccountsProvider(ctrl)
	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{types.Address(address)}, nil).AnyTimes()

	networksProvider := mock_ownership.NewMockNetworksProvider(ctrl)
	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{{ChainID: uint64(chainID), IsActive: true}}, nil).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(pubsub.NewPublisher()).AnyTimes()

	fetcher := mock_ownership.NewMockCollectibleOwnershipFetcher(ctrl)
	fetcher.EXPECT().FetchCollectibleOwnershipByOwner(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&thirdparty.CollectibleOwnershipContainer{
			Items:          []thirdparty.CollectibleIDBalance{},
			NextCursor:     "",
			PreviousCursor: "",
			Provider:       "mock",
		}, nil).AnyTimes()

	ownershipController := ownership.NewController(
		storage,
		accountsProvider,
		pubsub.NewPublisher(),
		networksProvider,
		nil,
		nil,
		nil,
		fetcher,
		pubsub.NewPublisher(),
		zap.NewNop(),
	)
	ownershipController.StartWithLoaderParams(ownership.PeriodicalLoaderParams{
		StartDelay:   0,
		LoadInterval: time.Hour,
		LoaderParams: ownership.LoaderParams{
			LoadDelay:  0,
			FetchLimit: 10,
		},
	})
	defer ownershipController.Stop()

	require.Eventually(t, func() bool {
		return ownershipController.GetLoaderState(chainID, address) != ownership.LoaderStateNotAvailable
	}, time.Second, 25*time.Millisecond)

	s := &Service{
		ownershipDB:         storage,
		ownershipController: ownershipController,
	}

	trigger, err := s.shouldTriggerLoad(chainID, address, FetchCriteria{
		FetchType: FetchTypeFetchIfNotCached,
	})
	require.NoError(t, err)
	require.False(t, trigger)
}

func TestServiceShouldTriggerLoadFetchIfCacheOldRespectsAge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chainID := walletCommon.ChainID(1)
	address := common.HexToAddress("0x123")

	storage := mock_ownership.NewMockOwnershipStorage(ctrl)
	now := time.Now().Unix()
	gomock.InOrder(
		storage.EXPECT().GetOwnershipUpdateTimestamp(address, chainID).Return(now, nil),
		storage.EXPECT().GetOwnershipUpdateTimestamp(address, chainID).Return(now-10, nil),
	)

	s := &Service{
		ownershipDB: storage,
	}

	trigger, err := s.shouldTriggerLoad(chainID, address, FetchCriteria{
		FetchType:          FetchTypeFetchIfCacheOld,
		MaxCacheAgeSeconds: 3600,
	})
	require.NoError(t, err)
	require.False(t, trigger)

	trigger, err = s.shouldTriggerLoad(chainID, address, FetchCriteria{
		FetchType:          FetchTypeFetchIfCacheOld,
		MaxCacheAgeSeconds: 1,
	})
	require.NoError(t, err)
	require.True(t, trigger)
}

func TestServiceFetchOwnedCollectiblesIfNeededReturnsTriggerLoadError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chainID := walletCommon.ChainID(1)
	address := common.HexToAddress("0x123")
	expectedErr := errors.New("fetch failed")

	storage := mock_ownership.NewMockOwnershipStorage(ctrl)
	storage.EXPECT().GetOwnershipUpdateTimestamp(gomock.Any(), gomock.Any()).Return(ownership.InvalidTimestamp, nil).AnyTimes()

	fetcher := mock_ownership.NewMockCollectibleOwnershipFetcher(ctrl)
	fetcher.EXPECT().FetchCollectibleOwnershipByOwner(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, expectedErr).AnyTimes()

	ownershipController := ownership.NewController(
		storage,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		fetcher,
		pubsub.NewPublisher(),
		zaptest.NewLogger(t),
	)

	s := &Service{
		ownershipController: ownershipController,
		ownershipDB:         storage,
	}

	err := s.fetchOwnedCollectiblesIfNeeded(
		context.Background(),
		[]walletCommon.ChainID{chainID},
		[]common.Address{address},
		FetchCriteria{FetchType: FetchTypeAlwaysFetch},
	)
	require.ErrorIs(t, err, expectedErr)
}
