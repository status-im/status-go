package fees

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.uber.org/mock/gomock"

	mock_client "github.com/status-im/status-go/rpc/chain/mock/client"
	mock_rpcclient "github.com/status-im/status-go/rpc/mock/client"
)

type testState struct {
	ctx        context.Context
	mockCtrl   *gomock.Controller
	rpcClient  *mock_rpcclient.MockClientInterface
	feeManager *FeeManager
}

func setupTest(t *testing.T) (state testState) {
	state.ctx = context.Background()
	state.mockCtrl = gomock.NewController(t)
	state.rpcClient = mock_rpcclient.NewMockClientInterface(state.mockCtrl)
	state.feeManager = &FeeManager{
		RPCClient: state.rpcClient,
	}
	return state
}

func TestEstimatedTime(t *testing.T) {
	state := setupTest(t)
	// no fee history
	feeHistory := &FeeHistory{}
	state.rpcClient.EXPECT().Call(feeHistory, uint64(1), "eth_feeHistory", uint64(10), "latest", nil).Times(1).Return(nil)

	maxFeesPerGas := big.NewInt(2e9)
	estimation := state.feeManager.TransactionEstimatedTime(context.Background(), uint64(1), maxFeesPerGas)

	assert.Equal(t, Unknown, estimation)

	// there is fee history
	state.rpcClient.EXPECT().Call(feeHistory, uint64(1), "eth_feeHistory", uint64(10), "latest", nil).Times(1).Return(nil).
		Do(func(feeHistory, chainID, method any, args ...any) {
			feeHistoryResponse := &FeeHistory{
				BaseFeePerGas: []string{
					"0x12f0e070b",
					"0x13f10da8b",
					"0x126c30d5e",
					"0x136e4fe51",
					"0x134180d5a",
					"0x134e32c33",
					"0x137da8d22",
				},
			}
			*feeHistory.(*FeeHistory) = *feeHistoryResponse
		})

	maxFeesPerGas = big.NewInt(100e9)
	estimation = state.feeManager.TransactionEstimatedTime(context.Background(), uint64(1), maxFeesPerGas)

	assert.Equal(t, LessThanOneMinute, estimation)
}

func TestEstimatedTimeV2(t *testing.T) {
	state := setupTest(t)
	// no fee history
	feeHistory := &FeeHistory{}
	state.rpcClient.EXPECT().Call(feeHistory, uint64(1), "eth_feeHistory", uint64(10), "latest", []int{RewardPercentiles2}).Times(1).Return(nil)

	maxFeesPerGas := big.NewInt(2e9)
	priorityFeesPerGas := big.NewInt(2e8)
	estimation := state.feeManager.TransactionEstimatedTimeV2(context.Background(), uint64(1), maxFeesPerGas, priorityFeesPerGas)

	assert.Equal(t, uint(0), estimation)

	// there is fee history
	state.rpcClient.EXPECT().Call(feeHistory, uint64(1), "eth_feeHistory", uint64(10), "latest", []int{RewardPercentiles2}).Times(1).Return(nil).
		Do(func(feeHistory, chainID, method any, args ...any) {
			feeHistoryResponse := &FeeHistory{
				BaseFeePerGas: []string{
					"0x12f0e070b",
					"0x13f10da8b",
					"0x126c30d5e",
					"0x136e4fe51",
					"0x134180d5a",
					"0x134e32c33",
					"0x137da8d22",
				},
			}
			*feeHistory.(*FeeHistory) = *feeHistoryResponse
		})

	maxFeesPerGas = big.NewInt(100e9)
	priorityFeesPerGas = big.NewInt(10e9)
	estimation = state.feeManager.TransactionEstimatedTimeV2(context.Background(), uint64(1), maxFeesPerGas, priorityFeesPerGas)

	assert.Equal(t, uint(0), estimation)

	// there is fee history and rewards
	state.rpcClient.EXPECT().Call(feeHistory, uint64(1), "eth_feeHistory", uint64(10), "latest", []int{RewardPercentiles2}).Times(1).Return(nil).
		Do(func(feeHistory, chainID, method any, args ...any) {
			feeHistoryResponse := &FeeHistory{
				BaseFeePerGas: []string{
					"0x12f0e070b",
					"0x13f10da8b",
					"0x126c30d5e",
					"0x136e4fe51",
					"0x134180d5a",
					"0x134e32c33",
					"0x137da8d22",
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
				OldestBlock: "0x1497d4b",
				Reward: [][]string{
					{
						"0x2faf080",
						"0x39d10680",
						"0x722d7ef5",
					},
					{
						"0x5f5e100",
						"0x3b9aca00",
						"0x59682f00",
					},
					{
						"0x342e4a2",
						"0x39d10680",
						"0x77359400",
					},
					{
						"0x14a22237",
						"0x40170350",
						"0x77359400",
					},
					{
						"0x9134860",
						"0x39d10680",
						"0x618400ad",
					},
					{
						"0x2faf080",
						"0x39d10680",
						"0x77359400",
					},
					{
						"0x1ed69035",
						"0x39d10680",
						"0x41d0a8d6",
					},
				},
			}
			*feeHistory.(*FeeHistory) = *feeHistoryResponse
		})

	estimation = state.feeManager.TransactionEstimatedTimeV2(context.Background(), uint64(1), maxFeesPerGas, priorityFeesPerGas)

	assert.Equal(t, uint(15), estimation)
}

func TestSuggestedFeesForNotEIP1559CompatibleChains(t *testing.T) {
	state := setupTest(t)

	chainID := uint64(1)
	gasPrice := big.NewInt(1)
	feeHistory := &FeeHistory{}
	percentiles := []int{RewardPercentiles1, RewardPercentiles2, RewardPercentiles3}
	state.rpcClient.EXPECT().Call(feeHistory, chainID, "eth_feeHistory", uint64(10), "latest", percentiles).Times(1).Return(nil)
	mockedChainClient := mock_client.NewMockClientInterface(state.mockCtrl)
	state.rpcClient.EXPECT().EthClient(chainID).Times(1).Return(mockedChainClient, nil)
	mockedChainClient.EXPECT().SuggestGasPrice(state.ctx).Times(1).Return(gasPrice, nil)

	suggestedFees, err := state.feeManager.SuggestedFees(context.Background(), chainID)
	assert.NoError(t, err)
	assert.NotNil(t, suggestedFees)
	assert.Equal(t, gasPrice, suggestedFees.GasPrice)
	assert.False(t, suggestedFees.EIP1559Enabled)
}

func TestSuggestedFeesForEIP1559CompatibleChains(t *testing.T) {
	state := setupTest(t)

	feeHistoryResponse := &FeeHistory{
		BaseFeePerGas: []string{
			"0x12f0e070b",
			"0x13f10da8b",
			"0x126c30d5e",
			"0x136e4fe51",
			"0x134180d5a",
			"0x134e32c33",
			"0x137da8d22",
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
		OldestBlock: "0x1497d4b",
		Reward: [][]string{
			{
				"0x2faf080",
				"0x39d10680",
				"0x722d7ef5",
			},
			{
				"0x5f5e100",
				"0x3b9aca00",
				"0x59682f00",
			},
			{
				"0x342e4a2",
				"0x39d10680",
				"0x77359400",
			},
			{
				"0x14a22237",
				"0x40170350",
				"0x77359400",
			},
			{
				"0x9134860",
				"0x39d10680",
				"0x618400ad",
			},
			{
				"0x2faf080",
				"0x39d10680",
				"0x77359400",
			},
			{
				"0x1ed69035",
				"0x39d10680",
				"0x41d0a8d6",
			},
		},
	}

	chainID := uint64(1)
	feeHistory := &FeeHistory{}
	percentiles := []int{RewardPercentiles1, RewardPercentiles2, RewardPercentiles3}
	state.rpcClient.EXPECT().Call(feeHistory, chainID, "eth_feeHistory", uint64(10), "latest", percentiles).Times(1).Return(nil).
		Do(func(feeHistory, chainID, method any, args ...any) {
			*feeHistory.(*FeeHistory) = *feeHistoryResponse
		})

	suggestedFees, err := state.feeManager.SuggestedFees(context.Background(), chainID)
	assert.NoError(t, err)
	assert.NotNil(t, suggestedFees)

	variadicFee1 := big.NewInt(0).Sub(suggestedFees.MaxFeesLevels.High.ToInt(), suggestedFees.MaxFeesLevels.HighPriority.ToInt())
	variadicFee1.Sub(variadicFee1, big.NewInt(0).Mul(big.NewInt(2), suggestedFees.BaseFee))

	variadicFee2 := big.NewInt(0).Sub(suggestedFees.MaxFeesLevels.Medium.ToInt(), suggestedFees.MaxFeesLevels.MediumPriority.ToInt())
	variadicFee2.Sub(variadicFee2, suggestedFees.BaseFee)

	assert.Equal(t, variadicFee1, variadicFee2)

	assert.Equal(t, big.NewInt(0), suggestedFees.GasPrice)
	assert.Equal(t, big.NewInt(5362838082), suggestedFees.BaseFee)
	assert.Equal(t, big.NewInt(5362838082), suggestedFees.CurrentBaseFee)
	assert.Equal(t, big.NewInt(5462838082), suggestedFees.MaxFeesLevels.Low.ToInt())
	assert.Equal(t, big.NewInt(100000000), suggestedFees.MaxFeesLevels.LowPriority.ToInt())
	assert.Equal(t, big.NewInt(7795989313), suggestedFees.MaxFeesLevels.Medium.ToInt())
	assert.Equal(t, big.NewInt(970000000), suggestedFees.MaxFeesLevels.MediumPriority.ToInt())
	assert.Equal(t, big.NewInt(14104411640), suggestedFees.MaxFeesLevels.High.ToInt())
	assert.Equal(t, big.NewInt(1915584245), suggestedFees.MaxFeesLevels.HighPriority.ToInt())
	assert.Equal(t, suggestedFees.MaxFeesLevels.MediumPriority.ToInt(), suggestedFees.MaxPriorityFeePerGas)
	assert.Equal(t, big.NewInt(100000000), suggestedFees.MaxPriorityFeeSuggestedBounds.Lower)
	assert.Equal(t, big.NewInt(1915584245), suggestedFees.MaxPriorityFeeSuggestedBounds.Upper)
	assert.True(t, suggestedFees.EIP1559Enabled)
	assert.Equal(t, uint(65), suggestedFees.MaxFeesLevels.LowEstimatedTime)
	assert.Equal(t, uint(15), suggestedFees.MaxFeesLevels.MediumEstimatedTime)
	assert.Equal(t, uint(15), suggestedFees.MaxFeesLevels.HighEstimatedTime)
}
