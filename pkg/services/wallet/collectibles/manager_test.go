package collectibles

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/common"

	mock_client "github.com/status-im/status-go/internal/rpc/chain/mock/client"
	mock_rpcclient "github.com/status-im/status-go/internal/rpc/mock/client"
	mock_collectibles "github.com/status-im/status-go/pkg/services/wallet/collectibles/mock"
	mock_ownership "github.com/status-im/status-go/pkg/services/wallet/collectibles/ownership/mock"
	mock_community "github.com/status-im/status-go/pkg/services/wallet/community/mock"
	mock_thirdparty "github.com/status-im/status-go/pkg/services/wallet/thirdparty/mock"

	"github.com/status-im/status-go/internal/circuitbreaker"
	"github.com/status-im/status-go/pkg/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
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

	ownershipDB := mock_ownership.NewMockOwnershipStorage(mockCtrl)
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

func TestManager_FillAnimationMediatype(t *testing.T) {
	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "video/webm")
	}))
	defer server.Close()

	unreachable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	unreachableURL := unreachable.URL
	unreachable.Close()

	manager := &Manager{httpClient: server.Client()}

	assetWith := func(url string, mediaType string) *thirdparty.FullCollectibleData {
		return &thirdparty.FullCollectibleData{
			CollectibleData: thirdparty.CollectibleData{
				AnimationURL:       url,
				AnimationMediaType: mediaType,
			},
		}
	}

	t.Run("keeps what the provider reported without asking the network", func(t *testing.T) {
		requests.Store(0)
		asset := assetWith(server.URL, "video/mp4")

		manager.fillAnimationMediatype(context.Background(), asset)

		assert.Equal(t, "video/mp4", asset.CollectibleData.AnimationMediaType)
		assert.Equal(t, server.URL, asset.CollectibleData.AnimationURL)
		assert.Zero(t, requests.Load())
	})

	t.Run("resolves a media type the provider did not report", func(t *testing.T) {
		requests.Store(0)
		asset := assetWith(server.URL, "")

		manager.fillAnimationMediatype(context.Background(), asset)

		assert.Equal(t, "video/webm", asset.CollectibleData.AnimationMediaType)
		assert.Equal(t, server.URL, asset.CollectibleData.AnimationURL)
		assert.Equal(t, int32(1), requests.Load())
	})

	t.Run("asks nothing when there is no animation", func(t *testing.T) {
		requests.Store(0)
		asset := assetWith("", "")

		manager.fillAnimationMediatype(context.Background(), asset)

		assert.Zero(t, requests.Load())
	})

	t.Run("drops an animation whose media type cannot be resolved", func(t *testing.T) {
		asset := assetWith(unreachableURL, "")

		manager.fillAnimationMediatype(context.Background(), asset)

		assert.Empty(t, asset.CollectibleData.AnimationURL)
		assert.Empty(t, asset.CollectibleData.AnimationMediaType)
	})
}
