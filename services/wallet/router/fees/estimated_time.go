package fees

import (
	"context"
	"math"
	"math/big"
	"sort"

	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/services/wallet/common"
)

const (
	inclusionThreshold = 0.95

	priorityFeePercentileHigh   = 0.3
	priorityFeePercentileMedium = 0.2
	priorityFeePercentileLow    = 0.1

	baseFeePercentileFirstBlock  = 0.55
	baseFeePercentileSecondBlock = 0.45
	baseFeePercentileThirdBlock  = 0.35
	baseFeePercentileFourthBlock = 0.3
	baseFeePercentileFifthBlock  = 0.2
	baseFeePercentileSixthBlock  = 0.1
)

type TransactionEstimation int

const (
	Unknown TransactionEstimation = iota
	LessThanOneMinute
	LessThanThreeMinutes
	LessThanFiveMinutes
	MoreThanFiveMinutes
)

func (f *FeeManager) TransactionEstimatedTime(ctx context.Context, chainID uint64, maxFeePerGas *big.Int) TransactionEstimation {
	feeHistory, err := f.getFeeHistory(ctx, chainID, "latest", nil)
	if err != nil {
		return Unknown
	}

	return f.estimatedTime(feeHistory, maxFeePerGas)
}

func (f *FeeManager) estimatedTime(feeHistory *FeeHistory, maxFeePerGas *big.Int) TransactionEstimation {
	fees := convertToBigIntAndSort(feeHistory.BaseFeePerGas)
	if len(fees) == 0 {
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

func convertToBigIntAndSort(hexArray []string) []*big.Int {
	values := []*big.Int{}
	for _, sValue := range hexArray {
		iValue := new(big.Int)
		_, ok := iValue.SetString(sValue, 0)
		if !ok {
			continue
		}
		values = append(values, iValue)
	}

	sort.Slice(values, func(i, j int) bool { return values[i].Cmp(values[j]) < 0 })
	return values
}

func removeDuplicatesFromSortedArray(array []*big.Int) []*big.Int {
	if len(array) == 0 {
		return array
	}

	uniqueArray := []*big.Int{array[0]}
	for i := 1; i < len(array); i++ {
		if array[i].Cmp(array[i-1]) != 0 {
			uniqueArray = append(uniqueArray, array[i])
		}
	}
	return uniqueArray
}

func calculateTimeForInclusion(chainID uint64, expectedInclusionInBlock int) uint {
	blockCreationTime := common.GetBlockCreationTimeForChain(chainID)
	blockCreationTimeInSeconds := uint(blockCreationTime.Seconds())

	// the client will decide how to display estimated times, status-go sends it in the steps of 5 (for example the client
	// should display "more than 1 minute" if the expected inclusion time is 60 seconds or more.
	expectedInclusionTime := uint(expectedInclusionInBlock) * blockCreationTimeInSeconds
	return (expectedInclusionTime/5 + 1) * 5
}

func getBaseFeePercentileIndex(sortedBaseFees []*big.Int, percentile float64, networkCongestion float64) int {
	// calculate the index of the base fee for the given percentile corrected by the network congestion
	index := int(float64(len(sortedBaseFees)) * percentile * networkCongestion)
	if index >= len(sortedBaseFees) {
		return len(sortedBaseFees) - 1
	}
	return index
}

// TransactionEstimatedTimeV2 returns the estimated time in seconds for a transaction to be included in a block
func (f *FeeManager) TransactionEstimatedTimeV2(ctx context.Context, chainID uint64, maxFeePerGas *big.Int, priorityFee *big.Int) uint {
	feeHistory, err := f.getFeeHistory(ctx, chainID, "latest", []int{RewardPercentiles2})
	if err != nil {
		return 0
	}

	return estimatedTimeV2(feeHistory, maxFeePerGas, priorityFee, chainID, 0)
}

func estimatedTimeV2(feeHistory *FeeHistory, txMaxFeePerGas *big.Int, txPriorityFee *big.Int, chainID uint64, rewardIndex int) uint {
	sortedBaseFees := convertToBigIntAndSort(feeHistory.BaseFeePerGas)
	if len(sortedBaseFees) == 0 {
		return 0
	}
	uniqueBaseFees := removeDuplicatesFromSortedArray(sortedBaseFees)

	var mediumPriorityFees []string // based on 50th percentile in the last 100 blocks
	for _, fee := range feeHistory.Reward {
		mediumPriorityFees = append(mediumPriorityFees, fee[rewardIndex])
	}
	mediumPriorityFeesSorted := convertToBigIntAndSort(mediumPriorityFees)
	if len(mediumPriorityFeesSorted) == 0 {
		return 0
	}
	uniqueMediumPriorityFees := removeDuplicatesFromSortedArray(mediumPriorityFeesSorted)

	txBaseFee := new(big.Int).Sub(txMaxFeePerGas, txPriorityFee)

	return estimateV2(txBaseFee, txPriorityFee, uniqueBaseFees, uniqueMediumPriorityFees, chainID)
}

func (f *FeeManager) TransactionEstimatedTimeV2Legacy(ctx context.Context, chainID uint64, gasPrice *big.Int) uint {
	backend, err := f.RPCClient.EthClient(chainID)
	if err != nil {
		return 0
	}
	latestBlockNum, err := backend.BlockNumber(ctx)
	if err != nil {
		return 0
	}

	gasPrices := []*big.Int{}
	for i := uint64(0); i < uint64(blocksToCheck); i++ {
		blockNum := big.NewInt(0).SetUint64(latestBlockNum - i)
		block, err := backend.BlockByNumber(ctx, blockNum)
		if err != nil {
			return 0
		}

		for _, tx := range block.Transactions() {
			if tx.Type() != gethtypes.LegacyTxType && tx.Type() != gethtypes.AccessListTxType {
				continue
			}
			gasPrices = append(gasPrices, tx.GasPrice())
		}
	}

	sort.Slice(gasPrices, func(i, j int) bool { return gasPrices[i].Cmp(gasPrices[j]) < 0 })

	uniqueGasPrices := removeDuplicatesFromSortedArray(gasPrices)

	return estimateV2(gasPrice, nil, uniqueGasPrices, nil, chainID)
}

func estimateV2(txBaseFee *big.Int, txPriorityFee *big.Int, uniqueBaseFees []*big.Int, uniqueMediumPriorityFees []*big.Int, chainID uint64) uint {
	// results are not good if we include the network congestion, cause we reduced the number of blocks we are looking at
	// and also removed duplicates from the arrays, thus for now we will ignore it and use `1.0` for the network congestion
	networkCongestion := 1.0 // calculateNetworkCongestion(feeHistory)

	priorityFeeForFirstTwoBlock := new(big.Int)
	priorityFeeForSecondTwoBlocks := new(big.Int)
	priorityFeeForThirdTwoBlocks := new(big.Int)
	if len(uniqueMediumPriorityFees) > 0 {
		// Priority fee for the first two blocks has to be higher than 60th percentile of the mediumPriorityFeesSorted
		priorityFeePercentileIndex := int(float64(len(uniqueMediumPriorityFees)) * priorityFeePercentileHigh)
		priorityFeeForFirstTwoBlock = uniqueMediumPriorityFees[priorityFeePercentileIndex]

		// Priority fee for the second two blocks has to be higher than 50th percentile of the mediumPriorityFeesSorted
		priorityFeePercentileIndex = int(float64(len(uniqueMediumPriorityFees)) * priorityFeePercentileMedium)
		priorityFeeForSecondTwoBlocks = uniqueMediumPriorityFees[priorityFeePercentileIndex]
		// Priority fee for the third two blocks has to be higher than 40th percentile of the mediumPriorityFeesSorted
		priorityFeePercentileIndex = int(float64(len(uniqueMediumPriorityFees)) * priorityFeePercentileLow)
		priorityFeeForThirdTwoBlocks = uniqueMediumPriorityFees[priorityFeePercentileIndex]
	}

	// To include the transaction in the block `inclusionInBlock` its base fee has to be in a higher than `baseFeePercentile`
	// and its priority fee has to be higher than the `priorityFee`
	inclusions := []struct {
		inclusionInBlock  int
		baseFeePercentile float64
		priorityFee       *big.Int
	}{
		{1, baseFeePercentileFirstBlock, priorityFeeForFirstTwoBlock},
		{2, baseFeePercentileSecondBlock, priorityFeeForFirstTwoBlock},
		{3, baseFeePercentileThirdBlock, priorityFeeForSecondTwoBlocks},
		{4, baseFeePercentileFourthBlock, priorityFeeForSecondTwoBlocks},
		{5, baseFeePercentileFifthBlock, priorityFeeForThirdTwoBlocks},
		{6, baseFeePercentileSixthBlock, priorityFeeForThirdTwoBlocks},
	}

	// check priority fee for L1 chains only
	checkPriorityFee := chainID == common.EthereumMainnet || chainID == common.EthereumSepolia || chainID == common.AnvilMainnet
	for _, p := range inclusions {
		baseFeePercentileIndex := getBaseFeePercentileIndex(uniqueBaseFees, p.baseFeePercentile, networkCongestion)
		if txBaseFee.Cmp(uniqueBaseFees[baseFeePercentileIndex]) >= 0 && (!checkPriorityFee || txPriorityFee.Cmp(p.priorityFee) >= 0) {
			return calculateTimeForInclusion(chainID, p.inclusionInBlock)
		}
	}

	return calculateTimeForInclusion(chainID, 10)
}
