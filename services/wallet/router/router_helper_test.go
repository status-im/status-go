package router_test

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	mock_client "github.com/status-im/status-go/rpc/chain/mock/client"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/router"
)

type CalculateFeesTestSuite struct {
	suite.Suite
	ethClient *mock_client.MockClientInterface // Mock client implementing ContractCaller
	mockCtrl  *gomock.Controller
	chainID   uint64
}

func (s *CalculateFeesTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.ethClient = mock_client.NewMockClientInterface(s.mockCtrl)
	s.chainID = walletCommon.OptimismMainnet
}

func (s *CalculateFeesTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *CalculateFeesTestSuite) TestCalculateApprovalL1Fee_Success() {
	// Test inputs
	approvalTxInputData := []byte("0x123456")
	expectedFee := big.NewInt(500)

	// Prepare mock return data
	expectedReturnData := expectedFee.FillBytes(make([]byte, 32)) // Mocked return as ABI encoded uint256

	// Mock CallContract to simulate contract interaction
	s.ethClient.EXPECT().
		CallContract(gomock.Any(), gomock.Any(), gomock.Nil()).
		DoAndReturn(func(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
			// Check that the call message matches expectations
			require.NotEmpty(s.T(), call.Data)

			// Return encoded data
			return expectedReturnData, nil
		})

	// Call the function
	fee, err := router.CalculateL1Fee(s.chainID, approvalTxInputData, s.ethClient)

	// Assertions
	require.NoError(s.T(), err)
	require.Equal(s.T(), expectedFee, fee)
}

func (s *CalculateFeesTestSuite) TestCalculateApprovalL1Fee_ZeroFeeOnContractCallError() {
	// Test inputs
	approvalTxInputData := []byte("0x123456")

	// Mock CallContract to return an error
	s.ethClient.EXPECT().
		CallContract(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return(nil, errors.New("contract call failed"))

	// Call the function
	fee, err := router.CalculateL1Fee(s.chainID, approvalTxInputData, s.ethClient)

	// Assertions
	require.NotNil(s.T(), err)
	require.Nil(s.T(), fee)
}

func TestCalculateFeesTestSuite(t *testing.T) {
	suite.Run(t, new(CalculateFeesTestSuite))
}

func TestParseCollectibleID(t *testing.T) {
	collectibleID := "RANDOM"
	_, _, success := router.ParseCollectibleID(collectibleID)
	require.False(t, success)

	collectibleID = "0xfC43ac5f309499385e91e059862bDb0Bfa2Cd16c:123"
	expectedContractAddress := common.HexToAddress("0xfC43ac5f309499385e91e059862bDb0Bfa2Cd16c")
	expectedCollectibleID := big.NewInt(123)
	parsedContractAddress, parsedCollectibleID, success := router.ParseCollectibleID(collectibleID)
	require.True(t, success)
	require.Equal(t, expectedContractAddress, parsedContractAddress)
	require.Equal(t, expectedCollectibleID, parsedCollectibleID)
}
