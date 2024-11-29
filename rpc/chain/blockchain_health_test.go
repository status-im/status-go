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
		mockEthClient.EXPECT().GetName().AnyTimes().Return(fmt.Sprintf("test_client_chain_%d", chainID))
		mockEthClient.EXPECT().GetLimiter().AnyTimes().Return(nil)

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

	// Simulate provider statuses for chain 1
	providerCallStatusesChain1 := []rpcstatus.RpcProviderCallStatus{
		{
			Name:      "provider1_chain1",
			Timestamp: time.Now(),
			Err:       nil, // Up
		},
		{
			Name:      "provider2_chain1",
			Timestamp: time.Now(),
			Err:       errors.New("connection error"), // Down
		},
	}
	ctx := context.Background()
	s.mockProviders[1].Update(ctx, providerCallStatusesChain1)

	// Simulate provider statuses for chain 2
	providerCallStatusesChain2 := []rpcstatus.RpcProviderCallStatus{
		{
			Name:      "provider1_chain2",
			Timestamp: time.Now(),
			Err:       nil, // Up
		},
		{
			Name:      "provider2_chain2",
			Timestamp: time.Now(),
			Err:       nil, // Up
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

	provider2Chain1Status := providerStatusesChain1["provider2_chain1"]
	require.Equal(s.T(), rpcstatus.StatusDown, provider2Chain1Status.Status)

	// Provider statuses for chain 2
	providerStatusesChain2 := fullStatus.StatusPerChainPerProvider[2]
	require.Contains(s.T(), providerStatusesChain2, "provider1_chain2")
	require.Contains(s.T(), providerStatusesChain2, "provider2_chain2")

	provider1Chain2Status := providerStatusesChain2["provider1_chain2"]
	require.Equal(s.T(), rpcstatus.StatusUp, provider1Chain2Status.Status)

	provider2Chain2Status := providerStatusesChain2["provider2_chain2"]
	require.Equal(s.T(), rpcstatus.StatusUp, provider2Chain2Status.Status)

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

	// Simulate provider statuses for chain 1
	providerCallStatusesChain1 := []rpcstatus.RpcProviderCallStatus{
		{
			Name:      "provider1_chain1",
			Timestamp: time.Now(),
			Err:       nil, // Up
		},
		{
			Name:      "provider2_chain1",
			Timestamp: time.Now(),
			Err:       errors.New("connection error"), // Down
		},
	}
	ctx := context.Background()
	s.mockProviders[1].Update(ctx, providerCallStatusesChain1)

	// Simulate provider statuses for chain 2
	providerCallStatusesChain2 := []rpcstatus.RpcProviderCallStatus{
		{
			Name:      "provider1_chain2",
			Timestamp: time.Now(),
			Err:       nil, // Up
		},
		{
			Name:      "provider2_chain2",
			Timestamp: time.Now(),
			Err:       nil, // Up
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

	require.Equal(s.T(), rpcstatus.StatusUp, shortStatus.StatusPerChain[1].Status)
	require.Equal(s.T(), rpcstatus.StatusUp, shortStatus.StatusPerChain[2].Status)

	// Serialization to JSON works without errors
	jsonData, err := json.MarshalIndent(shortStatus, "", "  ")
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), jsonData)
}

func TestBlockchainHealthSuite(t *testing.T) {
	suite.Run(t, new(BlockchainHealthSuite))
}
