package fees

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/common"
)

const (
	RewardPercentiles1 = 10
	RewardPercentiles2 = 45
	RewardPercentiles3 = 90
)

type GasFeeMode int

const (
	GasFeeLow GasFeeMode = iota
	GasFeeMedium
	GasFeeHigh
	GasFeeCustom
)

var (
	ErrCustomFeeModeNotAvailableInSuggestedFees = &errors.ErrorResponse{Code: errors.ErrorCode("WRF-001"), Details: "custom fee mode is not available in suggested fees"}
	ErrEIP1559IncompaibleChain                  = &errors.ErrorResponse{Code: errors.ErrorCode("WRF-002"), Details: "EIP-1559 is not supported on this chain"}
	ErrInvalidRewardData                        = &errors.ErrorResponse{Code: errors.ErrorCode("WRF-003"), Details: "invalid reward data"}
)

type MaxFeesLevels struct {
	Low                 *hexutil.Big `json:"low"`                 // Low max fee per gas in WEI
	LowPriority         *hexutil.Big `json:"lowPriority"`         // Low priority fee in WEI
	LowEstimatedTime    uint         `json:"lowEstimatedTime"`    // Estimated time for low fees in seconds
	Medium              *hexutil.Big `json:"medium"`              // Medium max fee per gas in WEI
	MediumPriority      *hexutil.Big `json:"mediumPriority"`      // Medium priority fee in WEI
	MediumEstimatedTime uint         `json:"mediumEstimatedTime"` // Estimated time for medium fees in seconds
	High                *hexutil.Big `json:"high"`                // High max fee per gas in WEI
	HighPriority        *hexutil.Big `json:"highPriority"`        // High priority fee in WEI
	HighEstimatedTime   uint         `json:"highEstimatedTime"`   // Estimated time for high fees in seconds
}

type MaxPriorityFeesSuggestedBounds struct {
	Lower *big.Int // Lower bound for priority fee per gas in WEI
	Upper *big.Int // Upper bound for priority fee per gas in WEI
}

type SuggestedFees struct {
	GasPrice                      *big.Int                        // TODO: remove once clients stop using this field, used for EIP-1559 incompatible chains, not in use anymore
	BaseFee                       *big.Int                        // TODO: remove once clients stop using this field, current network base fee (in ETH WEI), kept for backward compatibility
	CurrentBaseFee                *big.Int                        // Current network base fee (in ETH WEI)
	MaxFeesLevels                 *MaxFeesLevels                  // Max fees levels for low, medium and high fee modes
	MaxPriorityFeePerGas          *big.Int                        // TODO: remove once clients stop using this field, kept for backward compatibility
	MaxPriorityFeeSuggestedBounds *MaxPriorityFeesSuggestedBounds // Lower and upper bounds for priority fee per gas in WEI
	L1GasFee                      *big.Float                      // TODO: remove once clients stop using this field, not in use anymore
	EIP1559Enabled                bool                            // TODO: remove it since all chains we have support EIP-1559
}

// //////////////////////////////////////////////////////////////////////////////
// TODO: remove `SuggestedFeesGwei` struct once new router is in place
// //////////////////////////////////////////////////////////////////////////////
type SuggestedFeesGwei struct {
	GasPrice             *big.Float `json:"gasPrice"`
	BaseFee              *big.Float `json:"baseFee"`
	MaxPriorityFeePerGas *big.Float `json:"maxPriorityFeePerGas"`
	MaxFeePerGasLow      *big.Float `json:"maxFeePerGasLow"`
	MaxFeePerGasMedium   *big.Float `json:"maxFeePerGasMedium"`
	MaxFeePerGasHigh     *big.Float `json:"maxFeePerGasHigh"`
	MaxFeePerGasCustom   *big.Float `json:"maxFeePerGasCustom"`
	L1GasFee             *big.Float `json:"l1GasFee,omitempty"`
	EIP1559Enabled       bool       `json:"eip1559Enabled"`
}

func (m *MaxFeesLevels) FeeFor(mode GasFeeMode) (*big.Int, *big.Int, uint, error) {
	if mode == GasFeeCustom {
		return nil, nil, 0, ErrCustomFeeModeNotAvailableInSuggestedFees
	}

	if mode == GasFeeLow {
		return m.Low.ToInt(), m.LowPriority.ToInt(), m.LowEstimatedTime, nil
	}

	if mode == GasFeeHigh {
		return m.High.ToInt(), m.HighPriority.ToInt(), m.MediumEstimatedTime, nil
	}

	return m.Medium.ToInt(), m.MediumPriority.ToInt(), m.HighEstimatedTime, nil
}

func (s *SuggestedFees) FeeFor(mode GasFeeMode) (*big.Int, *big.Int, uint, error) {
	return s.MaxFeesLevels.FeeFor(mode)
}

type FeeManager struct {
	RPCClient rpc.ClientInterface
}

func (f *FeeManager) SuggestedFees(ctx context.Context, chainID uint64) (*SuggestedFees, error) {
	feeHistory, err := f.getFeeHistory(ctx, chainID, "latest", []int{RewardPercentiles1, RewardPercentiles2, RewardPercentiles3})
	if err != nil {
		return f.getNonEIP1559SuggestedFees(ctx, chainID)
	}

	lowPriorityFeePerGasLowerBound, mediumPriorityFeePerGas, maxPriorityFeePerGasUpperBound, baseFee, err := getEIP1559SuggestedFees(feeHistory)
	if err != nil {
		return f.getNonEIP1559SuggestedFees(ctx, chainID)
	}

	suggestedFees := &SuggestedFees{
		GasPrice:             big.NewInt(0),
		BaseFee:              baseFee,
		CurrentBaseFee:       baseFee,
		MaxPriorityFeePerGas: mediumPriorityFeePerGas,
		MaxPriorityFeeSuggestedBounds: &MaxPriorityFeesSuggestedBounds{
			Lower: lowPriorityFeePerGasLowerBound,
			Upper: maxPriorityFeePerGasUpperBound,
		},
		EIP1559Enabled: true,
	}

	if chainID == common.EthereumMainnet || chainID == common.EthereumSepolia || chainID == common.AnvilMainnet {
		networkCongestion := calculateNetworkCongestion(feeHistory)

		baseFeeFloat := new(big.Float).SetUint64(baseFee.Uint64())
		baseFeeFloat.Mul(baseFeeFloat, big.NewFloat(networkCongestion))
		additionBasedOnCongestion := new(big.Int)
		baseFeeFloat.Int(additionBasedOnCongestion)

		mediumBaseFee := new(big.Int).Add(baseFee, additionBasedOnCongestion)

		highBaseFee := new(big.Int).Mul(baseFee, big.NewInt(2))
		highBaseFee.Add(highBaseFee, additionBasedOnCongestion)

		suggestedFees.MaxFeesLevels = &MaxFeesLevels{
			Low:            (*hexutil.Big)(new(big.Int).Add(baseFee, lowPriorityFeePerGasLowerBound)),
			LowPriority:    (*hexutil.Big)(lowPriorityFeePerGasLowerBound),
			Medium:         (*hexutil.Big)(new(big.Int).Add(mediumBaseFee, mediumPriorityFeePerGas)),
			MediumPriority: (*hexutil.Big)(mediumPriorityFeePerGas),
			High:           (*hexutil.Big)(new(big.Int).Add(highBaseFee, maxPriorityFeePerGasUpperBound)),
			HighPriority:   (*hexutil.Big)(maxPriorityFeePerGasUpperBound),
		}
	} else {
		suggestedFees.MaxFeesLevels = &MaxFeesLevels{
			Low:            (*hexutil.Big)(new(big.Int).Add(baseFee, lowPriorityFeePerGasLowerBound)),
			LowPriority:    (*hexutil.Big)(lowPriorityFeePerGasLowerBound),
			Medium:         (*hexutil.Big)(new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(4)), mediumPriorityFeePerGas)),
			MediumPriority: (*hexutil.Big)(mediumPriorityFeePerGas),
			High:           (*hexutil.Big)(new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(10)), maxPriorityFeePerGasUpperBound)),
			HighPriority:   (*hexutil.Big)(maxPriorityFeePerGasUpperBound),
		}
	}

	suggestedFees.MaxFeesLevels.LowEstimatedTime = estimatedTimeV2(feeHistory, suggestedFees.MaxFeesLevels.Low.ToInt(), suggestedFees.MaxFeesLevels.LowPriority.ToInt(), chainID)
	suggestedFees.MaxFeesLevels.MediumEstimatedTime = estimatedTimeV2(feeHistory, suggestedFees.MaxFeesLevels.Medium.ToInt(), suggestedFees.MaxFeesLevels.MediumPriority.ToInt(), chainID)
	suggestedFees.MaxFeesLevels.HighEstimatedTime = estimatedTimeV2(feeHistory, suggestedFees.MaxFeesLevels.High.ToInt(), suggestedFees.MaxFeesLevels.HighPriority.ToInt(), chainID)

	return suggestedFees, nil
}

// //////////////////////////////////////////////////////////////////////////////
// TODO: remove `SuggestedFeesGwei` once mobile app fully switched to router, this function should not be exposed via api
// //////////////////////////////////////////////////////////////////////////////
func (f *FeeManager) SuggestedFeesGwei(ctx context.Context, chainID uint64) (*SuggestedFeesGwei, error) {
	fees, err := f.SuggestedFees(ctx, chainID)
	if err != nil {
		return nil, err
	}
	return &SuggestedFeesGwei{
		GasPrice:             common.WeiToGwei(fees.GasPrice),
		BaseFee:              common.WeiToGwei(fees.BaseFee),
		MaxPriorityFeePerGas: common.WeiToGwei(fees.MaxPriorityFeePerGas),
		MaxFeePerGasLow:      common.WeiToGwei(fees.MaxFeesLevels.Low.ToInt()),
		MaxFeePerGasMedium:   common.WeiToGwei(fees.MaxFeesLevels.Medium.ToInt()),
		MaxFeePerGasHigh:     common.WeiToGwei(fees.MaxFeesLevels.High.ToInt()),
		EIP1559Enabled:       fees.EIP1559Enabled,
	}, nil
}
