package chain

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/core/vm"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	healthManager "github.com/status-im/status-go/healthmanager"
	"github.com/status-im/status-go/healthmanager/rpcstatus"
	"github.com/status-im/status-go/rpc/chain/ethclient"
	"github.com/status-im/status-go/rpc/chain/rpclimiter"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	mockEthclient "github.com/status-im/status-go/rpc/chain/ethclient/mock/client/ethclient"
)

type ClientWithFallbackSuite struct {
	suite.Suite
	client                 *ClientWithFallback
	mockEthClients         []*mockEthclient.MockRPSLimitedEthClientInterface
	providersHealthManager *healthManager.ProvidersHealthManager
	mockCtrl               *gomock.Controller
}

func (s *ClientWithFallbackSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *ClientWithFallbackSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ClientWithFallbackSuite) setupClients(numClients int) {
	s.mockEthClients = make([]*mockEthclient.MockRPSLimitedEthClientInterface, 0)
	ethClients := make([]ethclient.RPSLimitedEthClientInterface, 0)

	for i := 0; i < numClients; i++ {
		ethClient := mockEthclient.NewMockRPSLimitedEthClientInterface(s.mockCtrl)
		ethClient.EXPECT().GetName().AnyTimes().Return("test" + strconv.Itoa(i))
		ethClient.EXPECT().GetLimiter().AnyTimes().Return(nil)

		s.mockEthClients = append(s.mockEthClients, ethClient)
		ethClients = append(ethClients, ethClient)
	}
	var chainID uint64 = 0
	s.providersHealthManager = healthManager.NewProvidersHealthManager(chainID)
	s.client = NewClient(ethClients, chainID, s.providersHealthManager)
}

func (s *ClientWithFallbackSuite) TestSingleClientSuccess() {
	s.setupClients(1)
	ctx := context.Background()
	hash := common.HexToHash("0x1234")
	block := &types.Block{}

	// GIVEN
	s.mockEthClients[0].EXPECT().BlockByHash(ctx, hash).Return(block, nil).Times(1)

	// WHEN
	result, err := s.client.BlockByHash(ctx, hash)
	require.NoError(s.T(), err)
	require.Equal(s.T(), block, result)

	// THEN
	chainStatus := s.providersHealthManager.Status()
	require.Equal(s.T(), rpcstatus.StatusUp, chainStatus.Status)
	providerStatuses := s.providersHealthManager.GetStatuses()
	require.Len(s.T(), providerStatuses, 1)
	require.Equal(s.T(), providerStatuses["test0"].Status, rpcstatus.StatusUp)
}

func (s *ClientWithFallbackSuite) TestSingleClientConnectionError() {
	s.setupClients(1)
	ctx := context.Background()
	hash := common.HexToHash("0x1234")

	// GIVEN
	s.mockEthClients[0].EXPECT().BlockByHash(ctx, hash).Return(nil, errors.New("connection error")).Times(1)

	// WHEN
	_, err := s.client.BlockByHash(ctx, hash)
	require.Error(s.T(), err)

	// THEN
	chainStatus := s.providersHealthManager.Status()
	require.Equal(s.T(), rpcstatus.StatusDown, chainStatus.Status)
	providerStatuses := s.providersHealthManager.GetStatuses()
	require.Len(s.T(), providerStatuses, 1)
	require.Equal(s.T(), providerStatuses["test0"].Status, rpcstatus.StatusDown)
}

func (s *ClientWithFallbackSuite) TestRPSLimitErrorDoesNotMarkChainDown() {
	s.setupClients(1)

	ctx := context.Background()
	hash := common.HexToHash("0x1234")

	// WHEN
	s.mockEthClients[0].EXPECT().BlockByHash(ctx, hash).Return(nil, rpclimiter.ErrRequestsOverLimit).Times(1)

	_, err := s.client.BlockByHash(ctx, hash)
	require.Error(s.T(), err)

	// THEN

	chainStatus := s.providersHealthManager.Status()
	require.Equal(s.T(), rpcstatus.StatusUp, chainStatus.Status)
	providerStatuses := s.providersHealthManager.GetStatuses()
	require.Len(s.T(), providerStatuses, 1)
	require.Equal(s.T(), providerStatuses["test0"].Status, rpcstatus.StatusUp)

	status := providerStatuses["test0"]
	require.Equal(s.T(), status.Status, rpcstatus.StatusUp, "provider shouldn't be DOWN on RPS limit")
}

func (s *ClientWithFallbackSuite) TestContextCanceledDoesNotMarkChainDown() {
	s.setupClients(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	hash := common.HexToHash("0x1234")

	// WHEN
	s.mockEthClients[0].EXPECT().BlockByHash(ctx, hash).Return(nil, context.Canceled).Times(1)

	_, err := s.client.BlockByHash(ctx, hash)
	require.Error(s.T(), err)
	require.True(s.T(), errors.Is(err, context.Canceled))

	// THEN
	chainStatus := s.providersHealthManager.Status()
	require.Equal(s.T(), rpcstatus.StatusUp, chainStatus.Status)
	providerStatuses := s.providersHealthManager.GetStatuses()
	require.Len(s.T(), providerStatuses, 1)
	require.Equal(s.T(), providerStatuses["test0"].Status, rpcstatus.StatusUp)
}

func (s *ClientWithFallbackSuite) TestVMErrorDoesNotMarkChainDown() {
	s.setupClients(1)
	ctx := context.Background()
	hash := common.HexToHash("0x1234")
	vmError := vm.ErrOutOfGas

	// GIVEN
	s.mockEthClients[0].EXPECT().BlockByHash(ctx, hash).Return(nil, vmError).Times(1)

	// WHEN
	_, err := s.client.BlockByHash(ctx, hash)
	require.Error(s.T(), err)
	require.True(s.T(), errors.Is(err, vm.ErrOutOfGas))

	// THEN
	chainStatus := s.providersHealthManager.Status()
	require.Equal(s.T(), rpcstatus.StatusUp, chainStatus.Status)
	providerStatuses := s.providersHealthManager.GetStatuses()
	require.Len(s.T(), providerStatuses, 1)
	require.Equal(s.T(), providerStatuses["test0"].Status, rpcstatus.StatusUp)
}

func (s *ClientWithFallbackSuite) TestNoClientsChainUnknown() {
	s.setupClients(0)

	ctx := context.Background()
	hash := common.HexToHash("0x1234")

	// WHEN
	_, err := s.client.BlockByHash(ctx, hash)
	require.Error(s.T(), err)

	// THEN
	chainStatus := s.providersHealthManager.Status()
	require.Equal(s.T(), rpcstatus.StatusUnknown, chainStatus.Status)
}

func (s *ClientWithFallbackSuite) TestAllClientsDifferentErrors() {
	s.setupClients(3)
	ctx := context.Background()
	hash := common.HexToHash("0x1234")

	// GIVEN
	s.mockEthClients[0].EXPECT().BlockByHash(ctx, hash).Return(nil, errors.New("no such host")).Times(1)
	s.mockEthClients[1].EXPECT().BlockByHash(ctx, hash).Return(nil, rpclimiter.ErrRequestsOverLimit).Times(1)
	s.mockEthClients[2].EXPECT().BlockByHash(ctx, hash).Return(nil, vm.ErrOutOfGas).Times(1)

	// WHEN
	_, err := s.client.BlockByHash(ctx, hash)
	require.Error(s.T(), err)

	// THEN
	chainStatus := s.providersHealthManager.Status()
	require.Equal(s.T(), rpcstatus.StatusUp, chainStatus.Status)

	providerStatuses := s.providersHealthManager.GetStatuses()
	require.Len(s.T(), providerStatuses, 3)

	require.Equal(s.T(), providerStatuses["test0"].Status, rpcstatus.StatusDown, "provider test0 should be DOWN due to a connection error")
	require.Equal(s.T(), providerStatuses["test1"].Status, rpcstatus.StatusUp, "provider test1 should not be marked DOWN due to RPS limit error")
	require.Equal(s.T(), providerStatuses["test2"].Status, rpcstatus.StatusUp, "provider test2 should not be labelled DOWN due to a VM error")
}

func (s *ClientWithFallbackSuite) TestAllClientsNetworkErrors() {
	s.setupClients(3)
	ctx := context.Background()
	hash := common.HexToHash("0x1234")

	// GIVEN
	s.mockEthClients[0].EXPECT().BlockByHash(ctx, hash).Return(nil, errors.New("no such host")).Times(1)
	s.mockEthClients[1].EXPECT().BlockByHash(ctx, hash).Return(nil, errors.New("no such host")).Times(1)
	s.mockEthClients[2].EXPECT().BlockByHash(ctx, hash).Return(nil, errors.New("no such host")).Times(1)

	// WHEN
	_, err := s.client.BlockByHash(ctx, hash)
	require.Error(s.T(), err)

	// THEN
	chainStatus := s.providersHealthManager.Status()
	require.Equal(s.T(), rpcstatus.StatusDown, chainStatus.Status)

	providerStatuses := s.providersHealthManager.GetStatuses()
	require.Len(s.T(), providerStatuses, 3)
	require.Equal(s.T(), providerStatuses["test0"].Status, rpcstatus.StatusDown)
	require.Equal(s.T(), providerStatuses["test1"].Status, rpcstatus.StatusDown)
	require.Equal(s.T(), providerStatuses["test2"].Status, rpcstatus.StatusDown)
}

func (s *ClientWithFallbackSuite) TestChainStatusUnknownWhenAllProvidersUnknown() {
	s.setupClients(2)

	chainStatus := s.providersHealthManager.Status()
	require.Equal(s.T(), rpcstatus.StatusUnknown, chainStatus.Status)
}

func TestClientWithFallbackSuite(t *testing.T) {
	suite.Run(t, new(ClientWithFallbackSuite))
}
