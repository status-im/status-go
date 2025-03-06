package collectibles

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/circuitbreaker"
	mock_client "github.com/status-im/status-go/rpc/chain/mock/client"
	mock_rpcclient "github.com/status-im/status-go/rpc/mock/client"
	"github.com/status-im/status-go/services/wallet/bigint"
	mock_collectibles "github.com/status-im/status-go/services/wallet/collectibles/mock"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	mock_community "github.com/status-im/status-go/services/wallet/community/mock"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	mock_thirdparty "github.com/status-im/status-go/services/wallet/thirdparty/mock"
)

type CopyableMockChainClient struct {
	*mock_client.MockClientInterface
	cb *circuitbreaker.CircuitBreaker
}

func (c *CopyableMockChainClient) Copy() interface{} {
	return &CopyableMockChainClient{
		MockClientInterface: c.MockClientInterface,
	}
}

func (c *CopyableMockChainClient) GetCircuitBreaker() *circuitbreaker.CircuitBreaker {
	return c.cb
}

func (c *CopyableMockChainClient) SetCircuitBreaker(cb *circuitbreaker.CircuitBreaker) {
	c.cb = cb
}

func TestManager_FetchAllAssetsByOwner(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.TODO()
	chainID := walletCommon.ChainID(1)
	owner := common.HexToAddress("0x1234567890abcdef")
	cursor := ""
	limit := 1
	timestamp := time.Now().Nanosecond()
	providerID := fmt.Sprintf("circuit_%d", timestamp)

	chainClient := &CopyableMockChainClient{
		MockClientInterface: mock_client.NewMockClientInterface(mockCtrl),
	}
	chainClient.EXPECT().CallContract(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).Times(limit)
	chainClient.EXPECT().CodeAt(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).Times(limit)
	rpcClient := mock_rpcclient.NewMockClientInterface(mockCtrl)
	rpcClient.EXPECT().EthClient(gomock.Any()).Return(chainClient, nil).AnyTimes()
	mockProvider1 := mock_thirdparty.NewMockCollectibleAccountOwnershipProvider(mockCtrl)
	// We use 2 providers as the last one is not using hystrix
	mockProvider2 := mock_thirdparty.NewMockCollectibleAccountOwnershipProvider(mockCtrl)

	mockProviders := thirdparty.CollectibleProviders{
		AccountOwnershipProviders: []thirdparty.CollectibleAccountOwnershipProvider{mockProvider1, mockProvider2},
	}

	// Generate many collectibles, but none support toeknURI method, but circuit must not be tripped
	var items []thirdparty.FullCollectibleData
	for i := 0; i < limit; i++ {
		items = append(items, thirdparty.FullCollectibleData{
			CollectibleData: thirdparty.CollectibleData{
				ID: thirdparty.CollectibleUniqueID{
					ContractID: thirdparty.ContractID{
						Address: common.HexToAddress(fmt.Sprintf("0x%064x", i)),
					},
					TokenID: &bigint.BigInt{
						Int: big.NewInt(int64(i)),
					},
				},
			},
		})
	}
	mockAssetContainer := &thirdparty.FullCollectibleDataContainer{
		Items: items,
	}

	mockProvider1.EXPECT().IsChainSupported(chainID).Return(true).AnyTimes()
	mockProvider1.EXPECT().IsConnected().Return(true).AnyTimes()
	mockProvider1.EXPECT().ID().Return(providerID).AnyTimes()
	mockProvider1.EXPECT().FetchAllAssetsByOwner(gomock.Any(), chainID, owner, cursor, limit).Return(mockAssetContainer, nil)

	mockProvider2.EXPECT().IsChainSupported(chainID).Return(true).AnyTimes()
	mockProvider2.EXPECT().IsConnected().Return(true).AnyTimes()
	mockProvider2.EXPECT().ID().Return(providerID).AnyTimes()

	manager := NewManager(nil, rpcClient, nil, mockProviders, nil, nil)
	manager.statuses = &sync.Map{}
	collectiblesDataDB := mock_collectibles.NewMockCollectibleDataStorage(mockCtrl)
	collectiblesDataDB.EXPECT().SetData(gomock.Any(), gomock.Any()).Return(nil)
	collectiblesDataDB.EXPECT().GetData(gomock.Any()).DoAndReturn(func(ids []thirdparty.CollectibleUniqueID) (map[string]thirdparty.CollectibleData, error) {
		ret := make(map[string]thirdparty.CollectibleData)
		for _, id := range ids {
			ret[id.HashKey()] = thirdparty.CollectibleData{
				ID: id,
			}
		}
		return ret, nil
	})
	collectiblesDataDB.EXPECT().GetCommunityInfo(gomock.Any()).Return(&thirdparty.CollectibleCommunityInfo{}, nil).AnyTimes()

	collectionsDataDB := mock_collectibles.NewMockCollectionDataStorage(mockCtrl)
	collectionsDataDB.EXPECT().SetData(gomock.Any(), gomock.Any()).Return(nil)
	collectionsDataDB.EXPECT().GetData(gomock.Any()).DoAndReturn(func(ids []thirdparty.ContractID) (map[string]thirdparty.CollectionData, error) {
		ret := make(map[string]thirdparty.CollectionData)
		for _, id := range ids {
			ret[id.HashKey()] = thirdparty.CollectionData{
				ID: id,
			}
		}
		return ret, nil
	})

	communityManager := mock_community.NewMockCommunityManagerInterface(mockCtrl)
	communityManager.EXPECT().GetCommunityID(gomock.Any()).Return(providerID).AnyTimes()
	communityManager.EXPECT().FillCollectiblesMetadata(gomock.Any(), gomock.Any()).Return(true, nil).AnyTimes()
	communityManager.EXPECT().GetCommunityInfo(gomock.Any()).Return(&thirdparty.CommunityInfo{}, nil, nil).AnyTimes()

	ownershipDB := mock_collectibles.NewMockOwnershipStorage(mockCtrl)
	ownershipDB.EXPECT().GetLatestOwnershipUpdateTimestamp(gomock.Any()).Return(int64(0), nil).AnyTimes()
	ownershipDB.EXPECT().GetOwnership(gomock.Any()).Return([]thirdparty.AccountBalance{}, nil).AnyTimes()

	manager.collectiblesDataDB = collectiblesDataDB
	manager.collectionsDataDB = collectionsDataDB
	manager.communityManager = communityManager
	manager.ownershipDB = ownershipDB
	assetContainer, err := manager.FetchAllAssetsByOwner(ctx, chainID, owner, cursor, limit, providerID)

	assert.NoError(t, err)
	assert.Equal(t, mockAssetContainer, assetContainer)

	// Make sure the main circuit is not tripped
	circuitName := getCircuitName(mockProvider1, chainID)
	assert.True(t, circuitbreaker.CircuitExists(circuitName))
	assert.False(t, circuitbreaker.IsCircuitOpen(circuitName))
}
