package pathprocessor

import (
	"context"
	"math/big"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/account"
	statusErrors "github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty/paraswap"
	walletToken "github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/transactions"
)

type SwapParaswapTxArgs struct {
	transactions.SendTxArgs
	ChainID            uint64  `json:"chainId"`
	ChainIDTo          uint64  `json:"chainIdTo"`
	TokenIDFrom        string  `json:"tokenIdFrom"`
	TokenIDTo          string  `json:"tokenIdTo"`
	SlippagePercentage float32 `json:"slippagePercentage"`
}

type SwapParaswapProcessor struct {
	paraswapClient *paraswap.ClientV5
	transactor     transactions.TransactorIface
	priceRoute     sync.Map // [fromChainName-toChainName-fromTokenSymbol-toTokenSymbol, paraswap.Route]
}

func NewSwapParaswapProcessor(rpcClient *rpc.Client, transactor transactions.TransactorIface, tokenManager *walletToken.Manager) *SwapParaswapProcessor {
	return &SwapParaswapProcessor{
		paraswapClient: paraswap.NewClientV5(walletCommon.EthereumMainnet),
		transactor:     transactor,
		priceRoute:     sync.Map{},
	}
}

func (s *SwapParaswapProcessor) Name() string {
	return ProcessorSwapParaswapName
}

func (s *SwapParaswapProcessor) Clear() {
	s.priceRoute = sync.Map{}
}

func (s *SwapParaswapProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	if params.FromChain == nil || params.ToChain == nil {
		return false, ErrNoChainSet
	}
	if params.FromToken == nil || params.ToToken == nil {
		return false, ErrToAndFromTokensMustBeSet
	}

	if params.FromChain.ChainID != params.ToChain.ChainID {
		return false, ErrFromAndToChainsMustBeSame
	}

	if params.FromToken.Symbol == params.ToToken.Symbol {
		return false, ErrFromAndToTokensMustBeDifferent
	}

	s.paraswapClient.SetChainID(params.FromChain.ChainID)

	searchForToken := params.FromToken.Address == ZeroAddress
	searchForToToken := params.ToToken.Address == ZeroAddress
	if searchForToToken || searchForToken {
		tokensList, err := s.paraswapClient.FetchTokensList(context.Background())
		if err != nil {
			return false, statusErrors.CreateErrorResponseFromError(err)
		}

		for _, t := range tokensList {
			if searchForToken && t.Symbol == params.FromToken.Symbol {
				params.FromToken.Address = common.HexToAddress(t.Address)
				params.FromToken.Decimals = t.Decimals
				if !searchForToToken {
					break
				}
			}

			if searchForToToken && t.Symbol == params.ToToken.Symbol {
				params.ToToken.Address = common.HexToAddress(t.Address)
				params.ToToken.Decimals = t.Decimals
				if !searchForToken {
					break
				}
			}
		}
	}

	if params.FromToken.Address == ZeroAddress || params.ToToken.Address == ZeroAddress {
		return false, ErrCannotResolveTokens
	}

	return true, nil
}

func (s *SwapParaswapProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return ZeroBigIntValue, ZeroBigIntValue, nil
}

func (s *SwapParaswapProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	// not sure what we can do here since we're using the api to build the transaction
	return []byte{}, nil
}

func (s *SwapParaswapProcessor) EstimateGas(params ProcessorInputParams) (uint64, error) {
	if params.TestsMode {
		if params.TestEstimationMap != nil {
			if val, ok := params.TestEstimationMap[s.Name()]; ok {
				return val, nil
			}
		}
		return 0, ErrNoEstimationFound
	}

	swapSide := paraswap.SellSide
	if params.AmountOut != nil && params.AmountOut.Cmp(ZeroBigIntValue) > 0 {
		swapSide = paraswap.BuySide
	}

	priceRoute, err := s.paraswapClient.FetchPriceRoute(context.Background(), params.FromToken.Address, params.FromToken.Decimals,
		params.ToToken.Address, params.ToToken.Decimals, params.AmountIn, params.FromAddr, params.ToAddr, swapSide)
	if err != nil {
		return 0, statusErrors.CreateErrorResponseFromError(err)
	}

	key := makeKey(params.FromChain.ChainID, params.ToChain.ChainID, params.FromToken.Symbol, params.ToToken.Symbol)
	s.priceRoute.Store(key, &priceRoute)

	return priceRoute.GasCost.Uint64(), nil
}

func (s *SwapParaswapProcessor) GetContractAddress(params ProcessorInputParams) (address common.Address, err error) {
	if params.FromChain.ChainID == walletCommon.EthereumMainnet {
		address = common.HexToAddress("0x216b4b4ba9f3e719726886d34a177484278bfcae")
	} else if params.FromChain.ChainID == walletCommon.ArbitrumMainnet {
		address = common.HexToAddress("0x216b4b4ba9f3e719726886d34a177484278bfcae")
	} else if params.FromChain.ChainID == walletCommon.OptimismMainnet {
		address = common.HexToAddress("0x216b4b4ba9f3e719726886d34a177484278bfcae")
	} else {
		err = ErrContractNotFound
	}
	return
}

func (s *SwapParaswapProcessor) BuildTx(params ProcessorInputParams) (*ethTypes.Transaction, error) {
	toAddr := types.Address(params.ToAddr)
	sendArgs := &MultipathProcessorTxArgs{
		SwapTx: &SwapParaswapTxArgs{
			SendTxArgs: transactions.SendTxArgs{
				From:   types.Address(params.FromAddr),
				To:     &toAddr,
				Value:  (*hexutil.Big)(params.AmountIn),
				Data:   types.HexBytes("0x0"),
				Symbol: params.FromToken.Symbol,
			},
			ChainID:     params.FromChain.ChainID,
			ChainIDTo:   params.ToChain.ChainID,
			TokenIDFrom: params.FromToken.Symbol,
			TokenIDTo:   params.ToToken.Symbol,
		},
	}

	return s.BuildTransaction(sendArgs)
}

func (s *SwapParaswapProcessor) prepareTransaction(sendArgs *MultipathProcessorTxArgs) error {
	slippageBP := uint(sendArgs.SwapTx.SlippagePercentage * 100) // convert to basis points

	key := makeKey(sendArgs.SwapTx.ChainID, sendArgs.SwapTx.ChainIDTo, sendArgs.SwapTx.TokenIDFrom, sendArgs.SwapTx.TokenIDTo)
	priceRouteIns, ok := s.priceRoute.Load(key)
	if !ok {
		return ErrPriceRouteNotFound
	}
	priceRoute := priceRouteIns.(*paraswap.Route)

	tx, err := s.paraswapClient.BuildTransaction(context.Background(), priceRoute.SrcTokenAddress, priceRoute.SrcTokenDecimals, priceRoute.SrcAmount.Int,
		priceRoute.DestTokenAddress, priceRoute.DestTokenDecimals, priceRoute.DestAmount.Int, slippageBP,
		common.Address(sendArgs.SwapTx.From), common.Address(*sendArgs.SwapTx.To),
		priceRoute.RawPriceRoute, priceRoute.Side)
	if err != nil {
		return statusErrors.CreateErrorResponseFromError(err)
	}

	value, ok := new(big.Int).SetString(tx.Value, 10)
	if !ok {
		return ErrConvertingAmountToBigInt
	}

	gas, err := strconv.ParseUint(tx.Gas, 10, 64)
	if err != nil {
		return statusErrors.CreateErrorResponseFromError(err)
	}

	gasPrice, ok := new(big.Int).SetString(tx.GasPrice, 10)
	if !ok {
		return ErrConvertingAmountToBigInt
	}

	sendArgs.ChainID = tx.ChainID
	sendArgs.SwapTx.ChainID = tx.ChainID
	toAddr := types.HexToAddress(tx.To)
	sendArgs.SwapTx.From = types.HexToAddress(tx.From)
	sendArgs.SwapTx.To = &toAddr
	sendArgs.SwapTx.Value = (*hexutil.Big)(value)
	sendArgs.SwapTx.Gas = (*hexutil.Uint64)(&gas)
	sendArgs.SwapTx.GasPrice = (*hexutil.Big)(gasPrice)
	sendArgs.SwapTx.Data = types.Hex2Bytes(tx.Data)

	return nil
}

func (s *SwapParaswapProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs) (*ethTypes.Transaction, error) {
	err := s.prepareTransaction(sendArgs)
	if err != nil {
		return nil, statusErrors.CreateErrorResponseFromError(err)
	}
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, sendArgs.SwapTx.SendTxArgs)
}

func (s *SwapParaswapProcessor) Send(sendArgs *MultipathProcessorTxArgs, verifiedAccount *account.SelectedExtKey) (types.Hash, error) {
	err := s.prepareTransaction(sendArgs)
	if err != nil {
		return types.Hash{}, statusErrors.CreateErrorResponseFromError(err)
	}

	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, sendArgs.SwapTx.SendTxArgs, verifiedAccount)
}

func (s *SwapParaswapProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	key := makeKey(params.FromChain.ChainID, params.ToChain.ChainID, params.FromToken.Symbol, params.ToToken.Symbol)
	priceRouteIns, ok := s.priceRoute.Load(key)
	if !ok {
		return nil, ErrPriceRouteNotFound
	}
	priceRoute := priceRouteIns.(*paraswap.Route)

	return priceRoute.DestAmount.Int, nil
}
