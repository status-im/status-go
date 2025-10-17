package fees

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	mock_client "github.com/status-im/status-go/rpc/chain/mock/client"
	mock_rpcclient "github.com/status-im/status-go/rpc/mock/client"
	walletCommon "github.com/status-im/status-go/services/wallet/common"

	geth "github.com/ethereum/go-ethereum"
)

type testState struct {
	ctx             context.Context
	mockCtrl        *gomock.Controller
	ethClientGetter *mock_rpcclient.MockEthClientGetter
	ethClient       *mock_client.MockClientInterface
	feeManager      *FeeManager
}

func setupTest(t *testing.T) (state testState) {
	state.ctx = context.Background()
	state.mockCtrl = gomock.NewController(t)
	state.ethClient = mock_client.NewMockClientInterface(state.mockCtrl)
	state.ethClientGetter = mock_rpcclient.NewMockEthClientGetter(state.mockCtrl)
	state.ethClientGetter.EXPECT().EthClient(uint64(1)).Return(state.ethClient, nil)
	state.feeManager = NewFeeManager(state.ethClientGetter, zap.NewNop())
	return state
}

func TestEstimatedTimeV2(t *testing.T) {
	state := setupTest(t)
	// there is fee history and rewards
	feeHistory := &geth.FeeHistory{
		BaseFee: []*big.Int{
			big.NewInt(0x12f0e070b),
			big.NewInt(0x13f10da8b),
			big.NewInt(0x126c30d5e),
			big.NewInt(0x136e4fe51),
			big.NewInt(0x134180d5a),
			big.NewInt(0x134e32c33),
			big.NewInt(0x137da8d22),
		},
		GasUsedRatio: []float64{
			0.7113286209349903,
			0.19531163333333335,
			0.7189235666666667,
			0.4639678021079083,
			0.5103012666666666,
			0.538413,
			0.16543626666666666,
		},
		OldestBlock: big.NewInt(0x1497d4b),
		Reward: [][]*big.Int{
			{
				big.NewInt(0x2faf080),
				big.NewInt(0x39d10680),
				big.NewInt(0x722d7ef5),
			},
			{
				big.NewInt(0x342e4a2),
				big.NewInt(0x39d10680),
				big.NewInt(0x77359400),
			},
			{
				big.NewInt(0x14a22237),
				big.NewInt(0x40170350),
				big.NewInt(0x77359400),
			},
			{
				big.NewInt(0x9134860),
				big.NewInt(0x39d10680),
				big.NewInt(0x618400ad),
			},
			{
				big.NewInt(0x2faf080),
				big.NewInt(0x39d10680),
				big.NewInt(0x77359400),
			},
			{
				big.NewInt(0x1ed69035),
				big.NewInt(0x39d10680),
				big.NewInt(0x41d0a8d6),
			},
		},
	}
	state.ethClient.EXPECT().FeeHistory(context.Background(), gomock.Any(), nil, gomock.Any()).Times(1).Return(feeHistory, nil)

	maxFeesPerGas := big.NewInt(100e9)
	priorityFeesPerGas := big.NewInt(10e9)
	estimation, err := state.feeManager.EstimatedTime(context.Background(), uint64(1), maxFeesPerGas, priorityFeesPerGas)
	assert.NoError(t, err)

	assert.Less(t, estimation, uint(20))
	assert.Greater(t, estimation, uint(10))
}

func TestSuggestedFeesForEIP1559CompatibleChains(t *testing.T) {
	state := setupTest(t)

	feeHistoryResponse := &geth.FeeHistory{
		BaseFee: []*big.Int{
			big.NewInt(0x12f0e070b),
			big.NewInt(0x13f10da8b),
			big.NewInt(0x126c30d5e),
			big.NewInt(0x136e4fe51),
			big.NewInt(0x134180d5a),
			big.NewInt(0x134e32c33),
			big.NewInt(0x137da8d22),
		},
		GasUsedRatio: []float64{
			0.7113286209349903,
			0.19531163333333335,
			0.7189235666666667,
			0.4639678021079083,
			0.5103012666666666,
			0.538413,
			0.16543626666666666,
		},
		OldestBlock: big.NewInt(0x1497d4b),
		Reward: [][]*big.Int{
			{
				big.NewInt(0x2faf080),
				big.NewInt(0x39d10680),
				big.NewInt(0x722d7ef5),
			},
			{
				big.NewInt(0x5f5e100),
				big.NewInt(0x3b9aca00),
				big.NewInt(0x59682f00),
			},
			{
				big.NewInt(0x342e4a2),
				big.NewInt(0x39d10680),
				big.NewInt(0x77359400),
			},
			{
				big.NewInt(0x14a22237),
				big.NewInt(0x40170350),
				big.NewInt(0x77359400),
			},
			{
				big.NewInt(0x9134860),
				big.NewInt(0x39d10680),
				big.NewInt(0x618400ad),
			},
			{
				big.NewInt(0x2faf080),
				big.NewInt(0x39d10680),
				big.NewInt(0x77359400),
			},
			{
				big.NewInt(0x1ed69035),
				big.NewInt(0x39d10680),
				big.NewInt(0x41d0a8d6),
			},
		},
	}

	chainID := uint64(1)
	state.ethClient.EXPECT().FeeHistory(context.Background(), uint64(10), nil, gomock.Any()).Times(1).Return(feeHistoryResponse, nil)

	suggestedFees, noBaseFee, noPriorityFee, err := state.feeManager.SuggestedFees(context.Background(), chainID, walletCommon.ZeroAddress())
	assert.NoError(t, err)
	assert.NotNil(t, suggestedFees)
	assert.False(t, noBaseFee)
	assert.False(t, noPriorityFee)

	assert.Equal(t, suggestedFees.GasPrice.Sign(), 0)
	assert.Greater(t, suggestedFees.BaseFee.Sign(), 0)
	assert.Greater(t, suggestedFees.CurrentBaseFee.Sign(), 0)
	assert.Greater(t, suggestedFees.MaxFeesLevels.Low.ToInt().Sign(), 0)
	assert.Greater(t, suggestedFees.MaxFeesLevels.LowPriority.ToInt().Sign(), 0)
	assert.Greater(t, suggestedFees.MaxFeesLevels.Medium.ToInt().Sign(), 0)
	assert.Greater(t, suggestedFees.MaxFeesLevels.MediumPriority.ToInt().Sign(), 0)
	assert.Greater(t, suggestedFees.MaxFeesLevels.High.ToInt().Sign(), 0)
	assert.Greater(t, suggestedFees.MaxFeesLevels.HighPriority.ToInt().Sign(), 0)

	assert.Equal(t, suggestedFees.MaxFeesLevels.MediumPriority.ToInt(), suggestedFees.MaxPriorityFeePerGas)

	assert.Greater(t, suggestedFees.MaxPriorityFeeSuggestedBounds.Lower.Sign(), 0)
	assert.Greater(t, suggestedFees.MaxPriorityFeeSuggestedBounds.Upper.Sign(), 0)

	assert.Greater(t, suggestedFees.MaxFeesLevels.Low.ToInt().Sign(), 0)
	assert.Greater(t, suggestedFees.MaxFeesLevels.Low.ToInt().Sign(), 0)

	assert.True(t, suggestedFees.EIP1559Enabled)

	assert.Greater(t, suggestedFees.MaxFeesLevels.LowEstimatedTime, uint(0))
	assert.Greater(t, suggestedFees.MaxFeesLevels.MediumEstimatedTime, uint(0))
	assert.Greater(t, suggestedFees.MaxFeesLevels.HighEstimatedTime, uint(0))
}
