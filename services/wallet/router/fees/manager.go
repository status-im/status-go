package fees

import (
	"context"
	"fmt"
	"math/big"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/errors"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/services/wallet/common"

	"github.com/status-im/go-wallet-sdk/pkg/gas"
)

type GasFeeMode int

const (
	GasFeeLow GasFeeMode = iota
	GasFeeMedium
	GasFeeHigh
	GasFeeCustom
)

type TransactionEstimation int

const (
	Unknown TransactionEstimation = iota
	LessThanOneMinute
	LessThanThreeMinutes
	LessThanFiveMinutes
	MoreThanFiveMinutes
)

var (
	ErrCustomFeeModeNotAvailableInSuggestedFees = &errors.ErrorResponse{Code: errors.ErrorCode("WRF-001"), Details: "custom fee mode is not available in suggested fees"}
	ErrEIP1559IncompaibleChain                  = &errors.ErrorResponse{Code: errors.ErrorCode("WRF-002"), Details: "EIP-1559 is not supported on this chain"}
	ErrInvalidRewardData                        = &errors.ErrorResponse{Code: errors.ErrorCode("WRF-003"), Details: "invalid reward data"}
)

// NonEIP1559Fees represents the fees for non EIP-1559 compatible chains
type NonEIP1559Fees struct {
	GasPrice      *hexutil.Big `json:"gasPrice"`      // Gas price for the transaction used for non EIP-1559 compatible chains (in base unit of the chain eg. WEI for ETH or BNB)
	EstimatedTime uint         `json:"estimatedTime"` // Estimated time for the transaction in seconds, used for non EIP-1559 compatible chains
}

// MaxFeesLevels represents the max fees levels for low, medium and high fee modes and should be used for EIP-1559 compatible chains
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
	// Fields that need to be removed once clients stop using them
	GasPrice             *big.Int   // TODO: remove once clients stop using this field, used for EIP-1559 incompatible chains, not in use anymore
	BaseFee              *big.Int   // TODO: remove once clients stop using this field, current network base fee (in ETH WEI), kept for backward compatibility
	MaxPriorityFeePerGas *big.Int   // TODO: remove once clients stop using this field, kept for backward compatibility
	L1GasFee             *big.Float // TODO: remove once clients stop using this field, not in use anymore

	// Fields in use
	NonEIP1559Fees                *NonEIP1559Fees                 // Fees for non EIP-1559 compatible chains
	MaxFeesLevels                 *MaxFeesLevels                  // Max fees levels for low, medium and high fee modes, should be used for EIP-1559 compatible chains
	MaxPriorityFeeSuggestedBounds *MaxPriorityFeesSuggestedBounds // Lower and upper bounds for priority fee per gas in WEI
	CurrentBaseFee                *big.Int                        // Current network base fee (in ETH WEI)
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
		return m.High.ToInt(), m.HighPriority.ToInt(), m.HighEstimatedTime, nil
	}

	return m.Medium.ToInt(), m.MediumPriority.ToInt(), m.MediumEstimatedTime, nil
}

func (s *SuggestedFees) FeeFor(mode GasFeeMode) (*big.Int, *big.Int, uint, error) {
	return s.MaxFeesLevels.FeeFor(mode)
}

type FeeManager struct {
	ethClientGetter rpc.EthClientGetter
	logger          *zap.Logger
}

func NewFeeManager(ethClientGetter rpc.EthClientGetter, logger *zap.Logger) *FeeManager {
	return &FeeManager{
		ethClientGetter: ethClientGetter,
		logger:          logger,
	}
}

func chainIDToClass(chainID uint64) (gas.ChainClass, error) {
	switch chainID {
	case common.EthereumMainnet, common.EthereumSepolia, common.AnvilMainnet, common.BSCMainnet, common.BSCTestnet:
		return gas.ChainClassL1, nil
	case common.ArbitrumMainnet, common.ArbitrumSepolia:
		return gas.ChainClassArbStack, nil
	case common.OptimismMainnet, common.OptimismSepolia, common.BaseMainnet, common.BaseSepolia:
		return gas.ChainClassOPStack, nil
	case common.StatusNetworkSepolia, common.LineaMainnet, common.LineaSepolia:
		return gas.ChainClassLineaStack, nil
	}
	return "", fmt.Errorf("chainID class identification not handled for chainID: %d", chainID)
}

func buildConfig(chainID uint64) (gas.ChainParameters, gas.SuggestionsConfig, error) {
	class, err := chainIDToClass(chainID)
	if err != nil {
		return gas.ChainParameters{}, gas.SuggestionsConfig{}, err
	}

	config := gas.DefaultConfig(class)
	params := gas.ChainParameters{
		ChainClass:       class,
		NetworkBlockTime: common.GetBlockCreationTimeForChain(chainID).Seconds(),
	}

	return params, config, nil
}

func (f *FeeManager) SuggestedFees(ctx context.Context, chainID uint64, address ethCommon.Address) (suggestedFees *SuggestedFees, noBaseFee bool, noPriorityFee bool, err error) {
	params, config, err := buildConfig(chainID)
	if err != nil {
		return nil, false, false, err
	}

	f.logger.Debug("Getting Tx Suggestions",
		zap.String("chainID", fmt.Sprintf("%d", chainID)),
		zap.Any("params", params),
		zap.Any("config", config),
	)

	ethClient, err := f.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, false, false, err
	}

	feeSuggestions, err := gas.GetChainSuggestions(ctx, ethClient, params, config, address)
	if err != nil {
		f.logger.Error("Failed to get chain suggestions", zap.Error(err))
		return nil, false, false, err
	}

	f.logger.Debug("Got Tx Suggestions",
		zap.String("chainID", fmt.Sprintf("%d", chainID)),
		zap.Any("feeSuggestions", feeSuggestions),
	)

	suggestedFees = &SuggestedFees{
		GasPrice:             big.NewInt(0),
		BaseFee:              feeSuggestions.EstimatedBaseFee,
		MaxPriorityFeePerGas: feeSuggestions.Medium.MaxPriorityFeePerGas,
		L1GasFee:             big.NewFloat(0),

		NonEIP1559Fees: nil,
		MaxFeesLevels: &MaxFeesLevels{
			Low:                 (*hexutil.Big)(feeSuggestions.Low.MaxFeePerGas),
			LowPriority:         (*hexutil.Big)(feeSuggestions.Low.MaxPriorityFeePerGas),
			LowEstimatedTime:    uint(feeSuggestions.LowInclusion.MinTimeUntilInclusion),
			Medium:              (*hexutil.Big)(feeSuggestions.Medium.MaxFeePerGas),
			MediumPriority:      (*hexutil.Big)(feeSuggestions.Medium.MaxPriorityFeePerGas),
			MediumEstimatedTime: uint(feeSuggestions.MediumInclusion.MinTimeUntilInclusion),
			High:                (*hexutil.Big)(feeSuggestions.High.MaxFeePerGas),
			HighPriority:        (*hexutil.Big)(feeSuggestions.High.MaxPriorityFeePerGas),
			HighEstimatedTime:   uint(feeSuggestions.HighInclusion.MinTimeUntilInclusion),
		},
		MaxPriorityFeeSuggestedBounds: &MaxPriorityFeesSuggestedBounds{
			Lower: feeSuggestions.PriorityFeeLowerBound,
			Upper: feeSuggestions.PriorityFeeUpperBound,
		},
		CurrentBaseFee: feeSuggestions.EstimatedBaseFee,
		EIP1559Enabled: true,
	}

	noBaseFee = false
	estimatedBaseFee := feeSuggestions.EstimatedBaseFee
	if estimatedBaseFee != nil && estimatedBaseFee.Sign() == 0 {
		noBaseFee = true
	}

	noPriorityFee = false
	estimatedPriorityFeeLowerBound := feeSuggestions.PriorityFeeLowerBound
	if estimatedPriorityFeeLowerBound != nil && estimatedPriorityFeeLowerBound.Sign() == 0 {
		noPriorityFee = true
	}

	f.logger.Debug("Suggested fees",
		zap.String("chainID", fmt.Sprintf("%d", chainID)),
		zap.Any("suggestedFees", suggestedFees),
		zap.Bool("noBaseFee", noBaseFee),
		zap.Bool("noPriorityFee", noPriorityFee),
	)

	return
}

func (f *FeeManager) EstimatedTime(ctx context.Context, chainID uint64, maxFeePerGas *big.Int, priorityFee *big.Int) (uint, error) {
	params, config, err := buildConfig(chainID)
	if err != nil {
		return 0, err
	}

	ethClient, err := f.ethClientGetter.EthClient(chainID)
	if err != nil {
		return 0, err
	}

	estimatedTime, err := gas.EstimateInclusion(ctx, ethClient, params, config, gas.Fee{
		MaxFeePerGas:         maxFeePerGas,
		MaxPriorityFeePerGas: priorityFee,
	})
	if err != nil {
		return 0, err
	}

	time := uint(0)
	if estimatedTime.MinTimeUntilInclusion < 0 || estimatedTime.MaxTimeUntilInclusion < 0 {
		// Some value is unbounded, take whichever is valid
		time = uint(max(estimatedTime.MinTimeUntilInclusion, estimatedTime.MaxTimeUntilInclusion))
	} else {
		// Take the average
		time = uint((estimatedTime.MinTimeUntilInclusion + estimatedTime.MaxTimeUntilInclusion) / 2)
	}

	return time, nil
}

// Remove when WalletConnect is reworked to use the router and API method `GetTransactionEstimatedTime` is removed
func (f *FeeManager) EstimatedTimeLevel(ctx context.Context, chainID uint64, gasPrice *big.Int) (TransactionEstimation, error) {
	return LessThanOneMinute, nil
}

// //////////////////////////////////////////////////////////////////////////////
// TODO: remove `SuggestedFeesGwei` once mobile app fully switched to router, this function should not be exposed via api
// //////////////////////////////////////////////////////////////////////////////
func (f *FeeManager) SuggestedFeesGwei(ctx context.Context, chainID uint64) (*SuggestedFeesGwei, error) {
	fees, _, _, err := f.SuggestedFees(ctx, chainID, common.ZeroAddress())
	if err != nil {
		return nil, err
	}

	if !fees.EIP1559Enabled {
		return nil, ErrEIP1559IncompaibleChain
	}

	feesGwei := &SuggestedFeesGwei{
		EIP1559Enabled: fees.EIP1559Enabled,
	}
	feesGwei.GasPrice = common.WeiToGwei(fees.GasPrice)
	feesGwei.BaseFee = common.WeiToGwei(fees.BaseFee)
	feesGwei.MaxPriorityFeePerGas = common.WeiToGwei(fees.MaxPriorityFeePerGas)
	feesGwei.MaxFeePerGasLow = common.WeiToGwei(fees.MaxFeesLevels.Low.ToInt())
	feesGwei.MaxFeePerGasMedium = common.WeiToGwei(fees.MaxFeesLevels.Medium.ToInt())
	feesGwei.MaxFeePerGasHigh = common.WeiToGwei(fees.MaxFeesLevels.High.ToInt())

	return feesGwei, nil
}
