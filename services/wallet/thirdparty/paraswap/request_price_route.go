package paraswap

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	netUrl "net/url"
	"strconv"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/services/wallet/bigint"
)

const pricesURL = "https://api.paraswap.io/prices"

type Route struct {
	GasCost            *bigint.BigInt  `json:"gasCost"`
	SrcAmount          *bigint.BigInt  `json:"srcAmount"`
	SrcTokenAddress    common.Address  `json:"srcToken"`
	SrcTokenDecimals   uint            `json:"srcDecimals"`
	DestAmount         *bigint.BigInt  `json:"destAmount"`
	DestTokenAddress   common.Address  `json:"destToken"`
	DestTokenDecimals  uint            `json:"destDecimals"`
	RawPriceRoute      json.RawMessage `json:"rawPriceRoute"`
	Side               SwapSide        `json:"side"`
	ContractAddress    common.Address  `json:"contractAddress"`
	TokenTransferProxy common.Address  `json:"tokenTransferProxy"`
}

func (r *Route) Copy() *Route {
	gasCost := new(bigint.BigInt)
	if r.GasCost != nil {
		var ok bool
		gasCost.Int, ok = new(big.Int).SetString(r.GasCost.String(), 10)
		if !ok {
			gasCost.Int = big.NewInt(0)
		}
	}
	srcAmount := new(bigint.BigInt)
	if r.SrcAmount != nil {
		var ok bool
		srcAmount.Int, ok = new(big.Int).SetString(r.SrcAmount.String(), 10)
		if !ok {
			srcAmount.Int = big.NewInt(0)
		}
	}
	destAmount := new(bigint.BigInt)
	if r.DestAmount != nil {
		var ok bool
		destAmount.Int, ok = new(big.Int).SetString(r.DestAmount.String(), 10)
		if !ok {
			destAmount.Int = big.NewInt(0)
		}
	}

	return &Route{
		GasCost:            gasCost,
		SrcAmount:          srcAmount,
		SrcTokenAddress:    r.SrcTokenAddress,
		SrcTokenDecimals:   r.SrcTokenDecimals,
		DestAmount:         destAmount,
		DestTokenAddress:   r.DestTokenAddress,
		DestTokenDecimals:  r.DestTokenDecimals,
		RawPriceRoute:      r.RawPriceRoute,
		Side:               r.Side,
		ContractAddress:    r.ContractAddress,
		TokenTransferProxy: r.TokenTransferProxy,
	}
}

type PriceRouteResponse struct {
	PriceRoute json.RawMessage `json:"priceRoute"`
	Error      string          `json:"error"`
}

func (c *ClientV5) FetchPriceRoute(ctx context.Context, srcTokenAddress common.Address, srcTokenDecimals uint,
	destTokenAddress common.Address, destTokenDecimals uint, amountWei *big.Int, addressFrom common.Address,
	addressTo common.Address, side SwapSide) (Route, error) {

	params := netUrl.Values{}
	params.Add("srcToken", srcTokenAddress.Hex())
	params.Add("srcDecimals", strconv.Itoa(int(srcTokenDecimals)))
	params.Add("destToken", destTokenAddress.Hex())
	params.Add("destDecimals", strconv.Itoa(int(destTokenDecimals)))
	params.Add("userAddress", addressFrom.Hex())
	// params.Add("receiver", addressTo.Hex())  // at this point paraswap doesn't allow swap and transfer transaction
	params.Add("network", strconv.FormatUint(c.chainID, 10))
	params.Add("amount", amountWei.String())
	params.Add("side", string(side))
	params.Add("partner", c.partnerID)
	params.Add("excludeContractMethodsWithoutFeeModel", "true")
	params.Add("version", "6.2")

	url := pricesURL
	response, err := c.httpClient.DoGetRequest(ctx, url, params)
	if err != nil {
		return Route{}, err
	}

	return handlePriceRouteResponse(response)
}

func handlePriceRouteResponse(response []byte) (Route, error) {
	var priceRouteResponse PriceRouteResponse
	err := json.Unmarshal(response, &priceRouteResponse)
	if err != nil {
		return Route{}, err
	}

	if priceRouteResponse.Error != "" {
		return Route{}, errors.New(priceRouteResponse.Error)
	}

	var route Route
	err = json.Unmarshal(priceRouteResponse.PriceRoute, &route)
	if err != nil {
		return Route{}, err
	}

	route.RawPriceRoute = priceRouteResponse.PriceRoute

	return route, nil
}
