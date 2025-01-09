package fees

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/params"
	"github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/common"
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
	Low    *hexutil.Big `json:"low"`
	Medium *hexutil.Big `json:"medium"`
	High   *hexutil.Big `json:"high"`
}

type MaxPriorityFeesSuggestedBounds struct {
	Lower *big.Int
	Upper *big.Int
}

type SuggestedFees struct {
	GasPrice                      *big.Int
	BaseFee                       *big.Int
	CurrentBaseFee                *big.Int // Current network base fee (in ETH WEI)
	MaxFeesLevels                 *MaxFeesLevels
	MaxPriorityFeePerGas          *big.Int
	MaxPriorityFeeSuggestedBounds *MaxPriorityFeesSuggestedBounds
	L1GasFee                      *big.Float
	EIP1559Enabled                bool
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

func (m *MaxFeesLevels) FeeFor(mode GasFeeMode) (*big.Int, error) {
	if mode == GasFeeCustom {
		return nil, ErrCustomFeeModeNotAvailableInSuggestedFees
	}

	if mode == GasFeeLow {
		return m.Low.ToInt(), nil
	}

	if mode == GasFeeHigh {
		return m.High.ToInt(), nil
	}

	return m.Medium.ToInt(), nil
}

func (s *SuggestedFees) FeeFor(mode GasFeeMode) (*big.Int, error) {
	return s.MaxFeesLevels.FeeFor(mode)
}

type FeeManager struct {
	RPCClient rpc.ClientInterface
}

func (f *FeeManager) SuggestedFees(ctx context.Context, chainID uint64) (*SuggestedFees, error) {
	feeHistory, err := f.getFeeHistory(ctx, chainID, 300, "latest", []int{25, 50, 75})
	if err != nil {
		return f.getNonEIP1559SuggestedFees(ctx, chainID)
	}

	maxPriorityFeePerGasLowerBound, maxPriorityFeePerGas, maxPriorityFeePerGasUpperBound, baseFee, err := getEIP1559SuggestedFees(feeHistory)
	if err != nil {
		return f.getNonEIP1559SuggestedFees(ctx, chainID)
	}

	return &SuggestedFees{
		BaseFee:              baseFee,
		CurrentBaseFee:       baseFee,
		MaxPriorityFeePerGas: maxPriorityFeePerGas,
		MaxPriorityFeeSuggestedBounds: &MaxPriorityFeesSuggestedBounds{
			Lower: maxPriorityFeePerGasLowerBound,
			Upper: maxPriorityFeePerGasUpperBound,
		},
		MaxFeesLevels: &MaxFeesLevels{
			Low:    (*hexutil.Big)(new(big.Int).Add(baseFee, maxPriorityFeePerGas)),
			Medium: (*hexutil.Big)(new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(2)), maxPriorityFeePerGas)),
			High:   (*hexutil.Big)(new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(3)), maxPriorityFeePerGas)),
		},
		EIP1559Enabled: true,
	}, nil
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

func (f *FeeManager) getBaseFee(ctx context.Context, client chain.ClientInterface) (*big.Int, error) {
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}

	chainID := client.NetworkID()
	config := params.MainnetChainConfig
	switch chainID {
	case common.EthereumSepolia,
		common.OptimismSepolia,
		common.ArbitrumSepolia:
		config = params.SepoliaChainConfig
	}
	baseFee := misc.CalcBaseFee(config, header)
	return baseFee, nil
}
