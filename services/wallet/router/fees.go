package router

import (
	"context"
	"math"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/params"
	gaspriceoracle "github.com/status-im/status-go/contracts/gas-price-oracle"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/common"
)

type GasFeeMode int

const (
	GasFeeLow GasFeeMode = iota
	GasFeeMedium
	GasFeeHigh
)

type MaxFeesLevels struct {
	Low    *hexutil.Big `json:"low"`
	Medium *hexutil.Big `json:"medium"`
	High   *hexutil.Big `json:"high"`
}

type SuggestedFees struct {
	GasPrice             *big.Int       `json:"gasPrice"`
	BaseFee              *big.Int       `json:"baseFee"`
	MaxFeesLevels        *MaxFeesLevels `json:"maxFeesLevels"`
	MaxPriorityFeePerGas *big.Int       `json:"maxPriorityFeePerGas"`
	L1GasFee             *big.Float     `json:"l1GasFee,omitempty"`
	EIP1559Enabled       bool           `json:"eip1559Enabled"`
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
	L1GasFee             *big.Float `json:"l1GasFee,omitempty"`
	EIP1559Enabled       bool       `json:"eip1559Enabled"`
}

func (m *MaxFeesLevels) feeFor(mode GasFeeMode) *big.Int {
	if mode == GasFeeLow {
		return m.Low.ToInt()
	}

	if mode == GasFeeHigh {
		return m.High.ToInt()
	}

	return m.Medium.ToInt()
}

func (s *SuggestedFees) feeFor(mode GasFeeMode) *big.Int {
	return s.MaxFeesLevels.feeFor(mode)
}

func (s *SuggestedFeesGwei) feeFor(mode GasFeeMode) *big.Float {
	if mode == GasFeeLow {
		return s.MaxFeePerGasLow
	}

	if mode == GasFeeHigh {
		return s.MaxFeePerGasHigh
	}

	return s.MaxFeePerGasMedium
}

const inclusionThreshold = 0.95

type TransactionEstimation int

const (
	Unknown TransactionEstimation = iota
	LessThanOneMinute
	LessThanThreeMinutes
	LessThanFiveMinutes
	MoreThanFiveMinutes
)

type FeeHistory struct {
	BaseFeePerGas []string `json:"baseFeePerGas"`
}

type FeeManager struct {
	RPCClient *rpc.Client
}

func weiToGwei(val *big.Int) *big.Float {
	result := new(big.Float)
	result.SetInt(val)

	unit := new(big.Int)
	unit.SetInt64(params.GWei)

	return result.Quo(result, new(big.Float).SetInt(unit))
}

func gweiToEth(val *big.Float) *big.Float {
	return new(big.Float).Quo(val, big.NewFloat(1000000000))
}

func gweiToWei(val *big.Float) *big.Int {
	res, _ := new(big.Float).Mul(val, big.NewFloat(1000000000)).Int(nil)
	return res
}

func (f *FeeManager) SuggestedFees(ctx context.Context, chainID uint64) (*SuggestedFees, error) {
	backend, err := f.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	gasPrice, err := backend.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}
	maxPriorityFeePerGas, err := backend.SuggestGasTipCap(ctx)
	if err != nil {
		return &SuggestedFees{
			GasPrice:             gasPrice,
			BaseFee:              big.NewInt(0),
			MaxPriorityFeePerGas: big.NewInt(0),
			MaxFeesLevels: &MaxFeesLevels{
				Low:    (*hexutil.Big)(gasPrice),
				Medium: (*hexutil.Big)(gasPrice),
				High:   (*hexutil.Big)(gasPrice),
			},
			EIP1559Enabled: false,
		}, nil
	}
	baseFee, err := f.getBaseFee(ctx, backend)
	if err != nil {
		return nil, err
	}

	return &SuggestedFees{
		GasPrice:             gasPrice,
		BaseFee:              baseFee,
		MaxPriorityFeePerGas: maxPriorityFeePerGas,
		MaxFeesLevels: &MaxFeesLevels{
			Low:    (*hexutil.Big)(new(big.Int).Add(baseFee, maxPriorityFeePerGas)),
			Medium: (*hexutil.Big)(new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(2)), maxPriorityFeePerGas)),
			High:   (*hexutil.Big)(new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(3)), maxPriorityFeePerGas)),
		},
		EIP1559Enabled: true,
	}, nil
}

func (f *FeeManager) SuggestedFeesGwei(ctx context.Context, chainID uint64) (*SuggestedFeesGwei, error) {
	fees, err := f.SuggestedFees(ctx, chainID)
	if err != nil {
		return nil, err
	}
	return &SuggestedFeesGwei{
		GasPrice:             weiToGwei(fees.GasPrice),
		BaseFee:              weiToGwei(fees.BaseFee),
		MaxPriorityFeePerGas: weiToGwei(fees.MaxPriorityFeePerGas),
		MaxFeePerGasLow:      weiToGwei(fees.MaxFeesLevels.Low.ToInt()),
		MaxFeePerGasMedium:   weiToGwei(fees.MaxFeesLevels.Medium.ToInt()),
		MaxFeePerGasHigh:     weiToGwei(fees.MaxFeesLevels.High.ToInt()),
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
	case common.EthereumGoerli,
		common.OptimismGoerli,
		common.ArbitrumGoerli:
		config = params.GoerliChainConfig
	}
	baseFee := misc.CalcBaseFee(config, header)
	return baseFee, nil
}

func (f *FeeManager) TransactionEstimatedTime(ctx context.Context, chainID uint64, maxFeePerGas *big.Int) TransactionEstimation {
	fees, err := f.getFeeHistorySorted(chainID)
	if err != nil {
		return Unknown
	}

	// pEvent represents the probability of the transaction being included in a block,
	// we assume this one is static over time, in reality it is not.
	pEvent := 0.0
	for idx, fee := range fees {
		if fee.Cmp(maxFeePerGas) == 1 || idx == len(fees)-1 {
			pEvent = float64(idx) / float64(len(fees))
			break
		}
	}

	// Probability of next 4 blocks including the transaction (less than 1 minute)
	// Generalising the formula: P(AUB) = P(A) + P(B) - P(A∩B) for 4 events and in our context P(A) == P(B) == pEvent
	// The factors are calculated using the combinations formula
	probability := pEvent*4 - 6*(math.Pow(pEvent, 2)) + 4*(math.Pow(pEvent, 3)) - (math.Pow(pEvent, 4))
	if probability >= inclusionThreshold {
		return LessThanOneMinute
	}

	// Probability of next 12 blocks including the transaction (less than 5 minutes)
	// Generalising the formula: P(AUB) = P(A) + P(B) - P(A∩B) for 20 events and in our context P(A) == P(B) == pEvent
	// The factors are calculated using the combinations formula
	probability = pEvent*12 -
		66*(math.Pow(pEvent, 2)) +
		220*(math.Pow(pEvent, 3)) -
		495*(math.Pow(pEvent, 4)) +
		792*(math.Pow(pEvent, 5)) -
		924*(math.Pow(pEvent, 6)) +
		792*(math.Pow(pEvent, 7)) -
		495*(math.Pow(pEvent, 8)) +
		220*(math.Pow(pEvent, 9)) -
		66*(math.Pow(pEvent, 10)) +
		12*(math.Pow(pEvent, 11)) -
		math.Pow(pEvent, 12)
	if probability >= inclusionThreshold {
		return LessThanThreeMinutes
	}

	// Probability of next 20 blocks including the transaction (less than 5 minutes)
	// Generalising the formula: P(AUB) = P(A) + P(B) - P(A∩B) for 20 events and in our context P(A) == P(B) == pEvent
	// The factors are calculated using the combinations formula
	probability = pEvent*20 -
		190*(math.Pow(pEvent, 2)) +
		1140*(math.Pow(pEvent, 3)) -
		4845*(math.Pow(pEvent, 4)) +
		15504*(math.Pow(pEvent, 5)) -
		38760*(math.Pow(pEvent, 6)) +
		77520*(math.Pow(pEvent, 7)) -
		125970*(math.Pow(pEvent, 8)) +
		167960*(math.Pow(pEvent, 9)) -
		184756*(math.Pow(pEvent, 10)) +
		167960*(math.Pow(pEvent, 11)) -
		125970*(math.Pow(pEvent, 12)) +
		77520*(math.Pow(pEvent, 13)) -
		38760*(math.Pow(pEvent, 14)) +
		15504*(math.Pow(pEvent, 15)) -
		4845*(math.Pow(pEvent, 16)) +
		1140*(math.Pow(pEvent, 17)) -
		190*(math.Pow(pEvent, 18)) +
		20*(math.Pow(pEvent, 19)) -
		math.Pow(pEvent, 20)
	if probability >= inclusionThreshold {
		return LessThanFiveMinutes
	}

	return MoreThanFiveMinutes
}

func (f *FeeManager) getFeeHistorySorted(chainID uint64) ([]*big.Int, error) {
	var feeHistory FeeHistory

	err := f.RPCClient.Call(&feeHistory, chainID, "eth_feeHistory", 101, "latest", nil)
	if err != nil {
		return nil, err
	}

	fees := []*big.Int{}
	for _, fee := range feeHistory.BaseFeePerGas {
		i := new(big.Int)
		i.SetString(strings.Replace(fee, "0x", "", 1), 16)
		fees = append(fees, i)
	}

	sort.Slice(fees, func(i, j int) bool { return fees[i].Cmp(fees[j]) < 0 })
	return fees, nil
}

// Returns L1 fee for placing a transaction to L1 chain, appicable only for txs made from L2.
func (f *FeeManager) GetL1Fee(ctx context.Context, chainID uint64, input []byte) (uint64, error) {
	if chainID == common.EthereumMainnet || chainID == common.EthereumSepolia && chainID != common.EthereumGoerli {
		return 0, nil
	}

	ethClient, err := f.RPCClient.EthClient(chainID)
	if err != nil {
		return 0, err
	}

	contractAddress, err := gaspriceoracle.ContractAddress(chainID)
	if err != nil {
		return 0, err
	}

	contract, err := gaspriceoracle.NewGaspriceoracleCaller(contractAddress, ethClient)
	if err != nil {
		return 0, err
	}

	callOpt := &bind.CallOpts{}

	result, err := contract.GetL1Fee(callOpt, input)
	if err != nil {
		return 0, err
	}

	return result.Uint64(), nil
}
