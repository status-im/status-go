package fees

import (
	"context"
	"math"
	"math/big"
	"sort"
	"strings"
)

const inclusionThreshold = 0.95

type TransactionEstimation int

const (
	Unknown TransactionEstimation = iota
	LessThanOneMinute
	LessThanThreeMinutes
	LessThanFiveMinutes
	MoreThanFiveMinutes
)

func (f *FeeManager) TransactionEstimatedTime(ctx context.Context, chainID uint64, maxFeePerGas *big.Int) TransactionEstimation {
	feeHistory, err := f.getFeeHistory(ctx, chainID, 100, "latest", nil)
	if err != nil {
		return Unknown
	}

	return f.estimatedTime(feeHistory, maxFeePerGas)
}

func (f *FeeManager) estimatedTime(feeHistory *FeeHistory, maxFeePerGas *big.Int) TransactionEstimation {
	fees, err := f.getFeeHistorySorted(feeHistory)
	if err != nil || len(fees) == 0 {
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

func (f *FeeManager) getFeeHistorySorted(feeHistory *FeeHistory) ([]*big.Int, error) {
	fees := []*big.Int{}
	for _, fee := range feeHistory.BaseFeePerGas {
		i := new(big.Int)
		i.SetString(strings.Replace(fee, "0x", "", 1), 16)
		fees = append(fees, i)
	}

	sort.Slice(fees, func(i, j int) bool { return fees[i].Cmp(fees[j]) < 0 })
	return fees, nil
}
