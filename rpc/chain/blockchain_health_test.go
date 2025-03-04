package chain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/status-im/status-go/healthmanager"
	"github.com/status-im/status-go/healthmanager/rpcstatus"
	mockEthclient "github.com/status-im/status-go/rpc/chain/ethclient/mock/client/ethclient"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/rpc/chain/ethclient"
)

type BlockchainHealthSuite struct {
	suite.Suite
	blockchainHealthManager *healthmanager.BlockchainHealthManager
	mockProviders           map[uint64]*healthmanager.ProvidersHealthManager
	mockEthClients          map[uint64]*mockEthclient.MockRPSLimitedEthClientInterface
	clients                 map[uint64]*ClientWithFallback
	mockCtrl                *gomock.Controller
}

func (s *BlockchainHealthSuite) SetupTest() {
	s.blockchainHealthManager = healthmanager.NewBlockchainHealthManager()
	s.mockProviders = make(map[uint64]*healthmanager.ProvidersHealthManager)
	s.mockEthClients = make(map[uint64]*mockEthclient.MockRPSLimitedEthClientInterface)
	s.clients = make(map[uint64]*ClientWithFallback)
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *BlockchainHealthSuite) TearDownTest() {
	s.blockchainHealthManager.Stop()
	s.mockCtrl.Finish()
}

func (s *BlockchainHealthSuite) setupClients(chainIDs []uint64) {
	ctx := context.Background()

	for _, chainID := range chainIDs {
		mockEthClient := mockEthclient.NewMockRPSLimitedEthClientInterface(s.mockCtrl)
		mockEthClient.EXPECT().GetProviderName().AnyTimes().Return(fmt.Sprintf("test_client_chain_%d_provider", chainID))
		mockEthClient.EXPECT().GetCircuitName().AnyTimes().Return(fmt.Sprintf("test_client_chain_%d_circuit", chainID))
		mockEthClient.EXPECT().GetLimiter().AnyTimes().Return(nil)
		mockEthClient.EXPECT().ExecuteWithRPSLimit(gomock.Any()).DoAndReturn(func(f func(client ethclient.EthClientInterface) (interface{}, error)) (interface{}, error) {
			return f(mockEthClient)
		}).AnyTimes()

		phm := healthmanager.NewProvidersHealthManager(chainID)
		client := NewClient([]ethclient.RPSLimitedEthClientInterface{mockEthClient}, chainID, phm)

		err := s.blockchainHealthManager.RegisterProvidersHealthManager(ctx, phm)
		require.NoError(s.T(), err)

		s.mockProviders[chainID] = phm
		s.mockEthClients[chainID] = mockEthClient
		s.clients[chainID] = client
	}
}

func (s *BlockchainHealthSuite) simulateChainStatus(chainID uint64, up bool) {
	client, exists := s.clients[chainID]
	require.True(s.T(), exists, "Client for chainID %d not found", chainID)

	mockEthClient := s.mockEthClients[chainID]
	ctx := context.Background()
	hash := common.HexToHash("0x1234")

	if up {
		block := &types.Block{}
		mockEthClient.EXPECT().BlockByHash(ctx, hash).Return(block, nil).Times(1)
		_, err := client.BlockByHash(ctx, hash)
		require.NoError(s.T(), err)
	} else {
		mockEthClient.EXPECT().BlockByHash(ctx, hash).Return(nil, errors.New("no such host")).Times(1)
		_, err := client.BlockByHash(ctx, hash)
		require.Error(s.T(), err)
	}
}

func (s *BlockchainHealthSuite) waitForStatus(statusCh chan struct{}, expectedStatus rpcstatus.StatusType) {
	timeout := time.After(2 * time.Second)
	for {
		select {
		case <-statusCh:
			status := s.blockchainHealthManager.Status()
			if status.Status == expectedStatus {
				return
			}
		case <-timeout:
			s.T().Errorf("Did not receive expected blockchain status update in time")
			return
		}
	}
}

func (s *BlockchainHealthSuite) TestAllChainsUp() {
	s.setupClients([]uint64{1, 2, 3})

	statusCh := s.blockchainHealthManager.Subscribe()
	defer s.blockchainHealthManager.Unsubscribe(statusCh)

	s.simulateChainStatus(1, true)
	s.simulateChainStatus(2, true)
	s.simulateChainStatus(3, true)

	s.waitForStatus(statusCh, rpcstatus.StatusUp)
}

func (s *BlockchainHealthSuite) TestSomeChainsDown() {
	s.setupClients([]uint64{1, 2, 3})

	statusCh := s.blockchainHealthManager.Subscribe()
	defer s.blockchainHealthManager.Unsubscribe(statusCh)

	s.simulateChainStatus(1, true)
	s.simulateChainStatus(2, false)
	s.simulateChainStatus(3, true)

	s.waitForStatus(statusCh, rpcstatus.StatusUp)
}

func (s *BlockchainHealthSuite) TestAllChainsDown() {
	s.setupClients([]uint64{1, 2})

	statusCh := s.blockchainHealthManager.Subscribe()
	defer s.blockchainHealthManager.Unsubscribe(statusCh)

	s.simulateChainStatus(1, false)
	s.simulateChainStatus(2, false)

	s.waitForStatus(statusCh, rpcstatus.StatusDown)
}

func (s *BlockchainHealthSuite) TestChainStatusChanges() {
	s.setupClients([]uint64{1, 2})

	statusCh := s.blockchainHealthManager.Subscribe()
	defer s.blockchainHealthManager.Unsubscribe(statusCh)

	s.simulateChainStatus(1, false)
	s.simulateChainStatus(2, false)
	s.waitForStatus(statusCh, rpcstatus.StatusDown)

	s.simulateChainStatus(1, true)
	s.waitForStatus(statusCh, rpcstatus.StatusUp)
}

func (s *BlockchainHealthSuite) TestGetFullStatus() {
	// Setup clients for chain IDs 1 and 2
	s.setupClients([]uint64{1, 2})

	// Subscribe to blockchain status updates
	statusCh := s.blockchainHealthManager.Subscribe()
	defer s.blockchainHealthManager.Unsubscribe(statusCh)

	now := time.Now()
	duration1 := 100 * time.Millisecond
	duration2 := 200 * time.Millisecond
	duration3 := 150 * time.Millisecond
	duration4 := 250 * time.Millisecond

	// Simulate provider statuses for chain 1 with metrics
	providerCallStatusesChain1 := []rpcstatus.RpcProviderCallStatus{
		{
			Name:      "provider1_chain1",
			Timestamp: now,
			Err:       nil, // Up
			StartTime: now.Add(-duration1),
		},
		{
			Name:      "provider2_chain1",
			Timestamp: now,
			Err:       context.DeadlineExceeded, // Down
			StartTime: now.Add(-duration2),
		},
	}
	ctx := context.Background()
	s.mockProviders[1].Update(ctx, providerCallStatusesChain1)

	// Simulate provider statuses for chain 2 with metrics
	providerCallStatusesChain2 := []rpcstatus.RpcProviderCallStatus{
		{
			Name:      "provider1_chain2",
			Timestamp: now,
			Err:       nil, // Up
			StartTime: now.Add(-duration3),
		},
		{
			Name:      "provider2_chain2",
			Timestamp: now,
			Err:       context.DeadlineExceeded, // Down
			StartTime: now.Add(-duration4),
		},
	}
	s.mockProviders[2].Update(ctx, providerCallStatusesChain2)

	// Wait for status event to be triggered before getting full status
	s.waitForStatus(statusCh, rpcstatus.StatusUp)

	// Get the full status from the BlockchainHealthManager
	fullStatus := s.blockchainHealthManager.GetFullStatus()

	// Assert overall blockchain status
	require.Equal(s.T(), rpcstatus.StatusUp, fullStatus.Status.Status)

	// Assert provider statuses per chain
	require.Contains(s.T(), fullStatus.StatusPerChainPerProvider, uint64(1))
	require.Contains(s.T(), fullStatus.StatusPerChainPerProvider, uint64(2))

	// Provider statuses for chain 1
	providerStatusesChain1 := fullStatus.StatusPerChainPerProvider[1]
	require.Contains(s.T(), providerStatusesChain1, "provider1_chain1")
	require.Contains(s.T(), providerStatusesChain1, "provider2_chain1")

	provider1Chain1Status := providerStatusesChain1["provider1_chain1"]
	require.Equal(s.T(), rpcstatus.StatusUp, provider1Chain1Status.Status)
	require.Equal(s.T(), duration1, provider1Chain1Status.TotalDuration)
	require.Equal(s.T(), int64(1), provider1Chain1Status.TotalRequests)
	require.Equal(s.T(), int64(0), provider1Chain1Status.TotalTimeoutCount)

	provider2Chain1Status := providerStatusesChain1["provider2_chain1"]
	require.Equal(s.T(), rpcstatus.StatusDown, provider2Chain1Status.Status)
	require.Equal(s.T(), duration2, provider2Chain1Status.TotalDuration)
	require.Equal(s.T(), int64(1), provider2Chain1Status.TotalRequests)
	require.Equal(s.T(), int64(1), provider2Chain1Status.TotalTimeoutCount)

	// Provider statuses for chain 2
	providerStatusesChain2 := fullStatus.StatusPerChainPerProvider[2]
	require.Contains(s.T(), providerStatusesChain2, "provider1_chain2")
	require.Contains(s.T(), providerStatusesChain2, "provider2_chain2")

	provider1Chain2Status := providerStatusesChain2["provider1_chain2"]
	require.Equal(s.T(), rpcstatus.StatusUp, provider1Chain2Status.Status)
	require.Equal(s.T(), duration3, provider1Chain2Status.TotalDuration)
	require.Equal(s.T(), int64(1), provider1Chain2Status.TotalRequests)
	require.Equal(s.T(), int64(0), provider1Chain2Status.TotalTimeoutCount)

	provider2Chain2Status := providerStatusesChain2["provider2_chain2"]
	require.Equal(s.T(), rpcstatus.StatusDown, provider2Chain2Status.Status)
	require.Equal(s.T(), duration4, provider2Chain2Status.TotalDuration)
	require.Equal(s.T(), int64(1), provider2Chain2Status.TotalRequests)
	require.Equal(s.T(), int64(1), provider2Chain2Status.TotalTimeoutCount)

	// Verify aggregated status for chain 1
	chain1Status := fullStatus.StatusPerChain[1]
	require.Equal(s.T(), rpcstatus.StatusUp, chain1Status.Status)
	require.Equal(s.T(), duration1+duration2, chain1Status.TotalDuration)
	require.Equal(s.T(), int64(2), chain1Status.TotalRequests)
	require.Equal(s.T(), int64(1), chain1Status.TotalTimeoutCount)

	// Verify aggregated status for chain 2
	chain2Status := fullStatus.StatusPerChain[2]
	require.Equal(s.T(), rpcstatus.StatusUp, chain2Status.Status)
	require.Equal(s.T(), duration3+duration4, chain2Status.TotalDuration)
	require.Equal(s.T(), int64(2), chain2Status.TotalRequests)
	require.Equal(s.T(), int64(1), chain2Status.TotalTimeoutCount)

	// Verify overall aggregated status
	overallStatus := fullStatus.Status
	require.Equal(s.T(), rpcstatus.StatusUp, overallStatus.Status)
	require.Equal(s.T(), duration1+duration2+duration3+duration4, overallStatus.TotalDuration)
	require.Equal(s.T(), int64(4), overallStatus.TotalRequests)
	require.Equal(s.T(), int64(2), overallStatus.TotalTimeoutCount)

	// Serialization to JSON works without errors
	jsonData, err := json.MarshalIndent(fullStatus, "", "  ")
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), jsonData)
}

func (s *BlockchainHealthSuite) TestGetShortStatus() {
	// Setup clients for chain IDs 1 and 2
	s.setupClients([]uint64{1, 2})

	// Subscribe to blockchain status updates
	statusCh := s.blockchainHealthManager.Subscribe()
	defer s.blockchainHealthManager.Unsubscribe(statusCh)

	now := time.Now()
	duration1 := 100 * time.Millisecond
	duration2 := 200 * time.Millisecond
	duration3 := 150 * time.Millisecond
	duration4 := 250 * time.Millisecond

	// Simulate provider statuses for chain 1 with metrics
	providerCallStatusesChain1 := []rpcstatus.RpcProviderCallStatus{
		{
			Name:      "provider1_chain1",
			Timestamp: now,
			Err:       nil, // Up
			StartTime: now.Add(-duration1),
		},
		{
			Name:      "provider2_chain1",
			Timestamp: now,
			Err:       context.DeadlineExceeded, // Down
			StartTime: now.Add(-duration2),
		},
	}
	ctx := context.Background()
	s.mockProviders[1].Update(ctx, providerCallStatusesChain1)

	// Simulate provider statuses for chain 2 with metrics
	providerCallStatusesChain2 := []rpcstatus.RpcProviderCallStatus{
		{
			Name:      "provider1_chain2",
			Timestamp: now,
			Err:       nil, // Up
			StartTime: now.Add(-duration3),
		},
		{
			Name:      "provider2_chain2",
			Timestamp: now,
			Err:       context.DeadlineExceeded, // Down
			StartTime: now.Add(-duration4),
		},
	}
	s.mockProviders[2].Update(ctx, providerCallStatusesChain2)

	// Wait for status event to be triggered before getting short status
	s.waitForStatus(statusCh, rpcstatus.StatusUp)

	// Get the short status from the BlockchainHealthManager
	shortStatus := s.blockchainHealthManager.GetStatusPerChain()

	// Assert overall blockchain status
	require.Equal(s.T(), rpcstatus.StatusUp, shortStatus.Status.Status)

	// Assert chain statuses
	require.Contains(s.T(), shortStatus.StatusPerChain, uint64(1))
	require.Contains(s.T(), shortStatus.StatusPerChain, uint64(2))

	// Verify metrics for chain 1
	chain1Status := shortStatus.StatusPerChain[1]
	require.Equal(s.T(), rpcstatus.StatusUp, chain1Status.Status)
	require.Equal(s.T(), duration1+duration2, chain1Status.TotalDuration)
	require.Equal(s.T(), int64(2), chain1Status.TotalRequests)
	require.Equal(s.T(), int64(1), chain1Status.TotalTimeoutCount)

	// Verify metrics for chain 2
	chain2Status := shortStatus.StatusPerChain[2]
	require.Equal(s.T(), rpcstatus.StatusUp, chain2Status.Status)
	require.Equal(s.T(), duration3+duration4, chain2Status.TotalDuration)
	require.Equal(s.T(), int64(2), chain2Status.TotalRequests)
	require.Equal(s.T(), int64(1), chain2Status.TotalTimeoutCount)

	// Verify overall aggregated status
	overallStatus := shortStatus.Status
	require.Equal(s.T(), rpcstatus.StatusUp, overallStatus.Status)
	require.Greater(s.T(), overallStatus.TotalDuration, time.Duration(0), "Overall should have non-zero duration")
	require.Equal(s.T(), int64(4), overallStatus.TotalRequests)
	require.Equal(s.T(), int64(2), overallStatus.TotalTimeoutCount)

	// Serialization to JSON works without errors
	jsonData, err := json.MarshalIndent(shortStatus, "", "  ")
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), jsonData)
}

func TestBlockchainHealthSuite(t *testing.T) {
	suite.Run(t, new(BlockchainHealthSuite))
}
