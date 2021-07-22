package wallet

import (
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

type FeeHistoryResult struct {
	OldestBlock  uint64           `json:"oldestBlock"`
	Reward       [][]*hexutil.Big `json:"reward,omitempty"`
	BaseFee      []*hexutil.Big   `json:"baseFeePerGas,omitempty"`
	GasUsedRatio []float64        `json:"gasUsedRatio"`
}

type FeeSuggestion struct {
	MaxFeePerGas         *big.Float `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *big.Float `json:"maxPriorityFeePerGas"`
}

type SuggestFeesResponse struct {
	Fees map[string]*FeeSuggestion `json:"fees"`
}

func (s *Service) SuggestFees() (*SuggestFeesResponse, error) {
	result := &FeeHistoryResult{}
	err := s.rpcClient.Call(&result, "eth_feeHistory", 100, "latest", nil)
	if err != nil {
		return nil, err
	}

	var baseFees []*big.Int
	var order []int
	for i, e := range result.BaseFee {
		baseFees = append(baseFees, (*big.Int)(e))
		order = append(order, i)
	}
	lastElement := baseFees[len(baseFees)-1]
	lastElement.Mul(lastElement, big.NewInt(9))
	lastElement.Div(lastElement, big.NewInt(8))

	for i, u := range result.GasUsedRatio {
		if u > 0.9 {
			baseFees[i] = baseFees[i+1]
		}
	}

	sort.Slice(order, func(i, j int) bool {
		return baseFees[i].Cmp(baseFees[j]) < 0
	})
	fmt.Println(result)
	fmt.Println(baseFees)
	fmt.Println(order)

	tip, err := s.suggestTip(result.OldestBlock, result.GasUsedRatio)
	if err != nil {
		return nil, err
	}

	maxBaseFee := big.NewFloat(0)

	var maxTimeFactor float64 = 15

	var extraTipRatio = big.NewFloat(0.25)

	response := &SuggestFeesResponse{
		Fees: make(map[string]*FeeSuggestion),
	}

	for timeFactor := int(maxTimeFactor); timeFactor >= 0; timeFactor-- {
		bf := suggestBaseFee(baseFees, order, float64(timeFactor))
		t := big.NewFloat(float64(tip.Int64()))
		if bf.Cmp(maxBaseFee) == 1 {
			maxBaseFee = bf
		} else {
			maxBaseFee.Sub(maxBaseFee, bf)
			maxBaseFee.Mul(maxBaseFee, extraTipRatio)
			t.Add(t, maxBaseFee)
			bf = maxBaseFee
		}
		response.Fees[strconv.Itoa(timeFactor)] = &FeeSuggestion{
			MaxFeePerGas:         big.NewFloat(0).Add(bf, t),
			MaxPriorityFeePerGas: t,
		}
	}

	return response, nil
}

func (s *Service) suggestTip(firstBlock uint64, gasUsedRatio []float64) (*big.Int, error) {
	ptr := len(gasUsedRatio) - 1
	needBlocks := 5
	var rewards []*big.Int
	for needBlocks > 0 && ptr >= 0 {
		blockCount := maxBlockCount(gasUsedRatio, ptr, needBlocks)
		if blockCount > 0 {
			feeHistory := &FeeHistoryResult{}
			err := s.rpcClient.Call(&feeHistory, "eth_feeHistory", blockCount, fmt.Sprintf("0x%x", firstBlock), []int{10})
			if err != nil {
				return big.NewInt(0), err
			}
			for i := range feeHistory.Reward {
				rewards = append(rewards, (*big.Int)(feeHistory.Reward[i][0]))
			}

			if len(feeHistory.Reward) < blockCount {
				break
			}
			needBlocks -= blockCount
		}
		ptr -= blockCount + 1
	}

	if len(rewards) == 0 {
		return big.NewInt(5e9), nil
	}

	sort.Slice(rewards, func(i, j int) bool {
		return rewards[i].Cmp(rewards[j]) < 0
	})

	return rewards[len(rewards)/2], nil
}

// maxBlockCount returns the number of consecutive blocks suitable for tip suggestion (gasUsedRatio between 0.1 and 0.9).
func maxBlockCount(gasUsedRatio []float64, ptr int, needBlocks int) int {
	blockCount := 0
	for needBlocks > 0 && ptr >= 0 {
		if gasUsedRatio[ptr] < 0.1 || gasUsedRatio[ptr] > 0.9 {
			break
		}
		ptr--
		needBlocks--
		blockCount++
	}
	return blockCount
}

func suggestBaseFee(baseFee []*big.Int, order []int, timeFactor float64) *big.Float {
	if timeFactor < 1e-6 {
		return big.NewFloat(float64(baseFee[len(baseFee)-1].Int64()))
	}
	pendingWeight := (1 - math.Exp(-1/timeFactor)) / (1 - math.Exp(-float64(len(baseFee))/timeFactor))
	var sumWeight float64
	result := big.NewFloat(0)
	var samplingCurveLast float64
	for i := 0; i < len(order); i++ {
		sumWeight += pendingWeight * math.Exp(float64(order[i]-len(baseFee)+1)/timeFactor)
		var samplingCurveValue = samplingCurve(sumWeight)
		result.Add(result, big.NewFloat((samplingCurveValue-samplingCurveLast)*(float64(baseFee[order[i]].Int64()))))
		if samplingCurveValue >= 1 {
			return result
		}
		samplingCurveLast = samplingCurveValue
	}
	return result
}

// samplingCurve is a helper function for the base fee percentile range calculation.
func samplingCurve(sumWeight float64) float64 {

	sampleMin := 0.1
	sampleMax := 0.3
	if sumWeight <= sampleMin {
		return 0
	}
	if sumWeight >= sampleMax {
		return 1
	}
	return (1 - math.Cos((sumWeight-sampleMin)*2*math.Pi/(sampleMax-sampleMin))) / 2
}
