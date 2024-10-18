package routes

import (
	"math"
	"math/big"

	"github.com/status-im/status-go/services/wallet/common"
)

type Route []*Path

func (r Route) Copy() Route {
	newRoute := make(Route, len(r))
	for i, path := range r {
		newRoute[i] = path.Copy()
	}
	return newRoute
}

func FindBestRoute(routes []Route, tokenPrice float64, nativeTokenPrice float64) Route {
	var best Route
	bestCost := big.NewFloat(math.Inf(1))
	for _, route := range routes {
		currentCost := big.NewFloat(0)
		for _, path := range route {
			tokenDenominator := big.NewFloat(math.Pow(10, float64(path.FromToken.Decimals)))

			// calculate the cost of the path
			nativeTokenPrice := new(big.Float).SetFloat64(nativeTokenPrice)

			// tx fee
			txFeeInEth := common.GweiToEth(common.WeiToGwei(path.TxFee.ToInt()))
			pathCost := new(big.Float).Mul(txFeeInEth, nativeTokenPrice)

			if path.TxL1Fee.ToInt().Cmp(common.ZeroBigIntValue()) > 0 {
				txL1FeeInEth := common.GweiToEth(common.WeiToGwei(path.TxL1Fee.ToInt()))
				pathCost.Add(pathCost, new(big.Float).Mul(txL1FeeInEth, nativeTokenPrice))
			}

			if path.TxBonderFees != nil && path.TxBonderFees.ToInt().Cmp(common.ZeroBigIntValue()) > 0 {
				pathCost.Add(pathCost, new(big.Float).Mul(
					new(big.Float).Quo(new(big.Float).SetInt(path.TxBonderFees.ToInt()), tokenDenominator),
					new(big.Float).SetFloat64(tokenPrice)))

			}

			if path.TxTokenFees != nil && path.TxTokenFees.ToInt().Cmp(common.ZeroBigIntValue()) > 0 && path.FromToken != nil {
				pathCost.Add(pathCost, new(big.Float).Mul(
					new(big.Float).Quo(new(big.Float).SetInt(path.TxTokenFees.ToInt()), tokenDenominator),
					new(big.Float).SetFloat64(tokenPrice)))
			}

			if path.ApprovalRequired {
				// tx approval fee
				approvalFeeInEth := common.GweiToEth(common.WeiToGwei(path.ApprovalFee.ToInt()))
				pathCost.Add(pathCost, new(big.Float).Mul(approvalFeeInEth, nativeTokenPrice))

				if path.ApprovalL1Fee.ToInt().Cmp(common.ZeroBigIntValue()) > 0 {
					approvalL1FeeInEth := common.GweiToEth(common.WeiToGwei(path.ApprovalL1Fee.ToInt()))
					pathCost.Add(pathCost, new(big.Float).Mul(approvalL1FeeInEth, nativeTokenPrice))
				}
			}

			currentCost = new(big.Float).Add(currentCost, pathCost)
		}

		if currentCost.Cmp(bestCost) == -1 {
			best = route
			bestCost = currentCost
		}
	}

	return best
}
