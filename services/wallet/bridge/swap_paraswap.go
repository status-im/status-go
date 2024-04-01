package bridge

import (
	"context"
	"errors"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty/paraswap"
	"github.com/status-im/status-go/services/wallet/token"
	walletToken "github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/transactions"
)

type SwapTxArgs struct {
	transactions.SendTxArgs
	ChainID uint64 `json:"chainId"`
}

type SwapParaswap struct {
	paraswapClient *paraswap.ClientV5
	priceRoute     paraswap.Route
	transactor     *transactions.Transactor
}

func NewSwapParaswap(rpcClient *rpc.Client, transactor *transactions.Transactor, tokenManager *walletToken.Manager) *SwapParaswap {
	return &SwapParaswap{
		paraswapClient: paraswap.NewClientV5(walletCommon.EthereumMainnet),
		transactor:     transactor,
	}
}

func (s *SwapParaswap) Name() string {
	return "Paraswap"
}

func (s *SwapParaswap) Can(from, to *params.Network, token *walletToken.Token, toToken *walletToken.Token, balance *big.Int) (bool, error) {
	if token == nil || toToken == nil {
		return false, errors.New("token and toToken cannot be nil")
	}

	if from.ChainID != to.ChainID {
		return false, nil
	}

	s.paraswapClient.SetChainID(from.ChainID)

	searchForToken := token.Address == ZeroAddress
	searchForToToken := toToken.Address == ZeroAddress
	if searchForToToken || searchForToken {
		tokensList, err := s.paraswapClient.FetchTokensList(context.Background())
		if err != nil {
			return false, err
		}

		for _, t := range tokensList {
			if searchForToken && t.Symbol == token.Symbol {
				token.Address = common.HexToAddress(t.Address)
				token.Decimals = t.Decimals
				if !searchForToToken {
					break
				}
			}

			if searchForToToken && t.Symbol == toToken.Symbol {
				toToken.Address = common.HexToAddress(t.Address)
				toToken.Decimals = t.Decimals
				if !searchForToken {
					break
				}
			}
		}
	}

	if token.Address == ZeroAddress || toToken.Address == ZeroAddress {
		return false, errors.New("cannot resolve token/s")
	}

	return true, nil
}

func (s *SwapParaswap) CalculateFees(from, to *params.Network, token *token.Token, amountIn *big.Int, nativeTokenPrice, tokenPrice float64, gasPrice *big.Float) (*big.Int, *big.Int, error) {
	return big.NewInt(0), big.NewInt(0), nil
}

func (s *SwapParaswap) EstimateGas(fromNetwork *params.Network, toNetwork *params.Network, from common.Address, to common.Address, token *token.Token, toToken *token.Token, amountIn *big.Int) (uint64, error) {
	priceRoute, err := s.paraswapClient.FetchPriceRoute(context.Background(), token.Address, token.Decimals, toToken.Address, toToken.Decimals, amountIn, from, to)
	if err != nil {
		return 0, err
	}

	s.priceRoute = priceRoute

	return priceRoute.GasCost.Uint64(), nil
}

func (s *SwapParaswap) GetContractAddress(network *params.Network, token *token.Token) *common.Address {
	var address common.Address
	if network.ChainID == walletCommon.EthereumMainnet {
		address = common.HexToAddress("0x216b4b4ba9f3e719726886d34a177484278bfcae")
	} else if network.ChainID == walletCommon.ArbitrumMainnet {
		address = common.HexToAddress("0x216b4b4ba9f3e719726886d34a177484278bfcae")
	} else if network.ChainID == walletCommon.OptimismMainnet {
		address = common.HexToAddress("0x216b4b4ba9f3e719726886d34a177484278bfcae")
	}

	return &address
}

func (s *SwapParaswap) BuildTx(network *params.Network, fromAddress common.Address, toAddress common.Address, token *token.Token, amountIn *big.Int) (*ethTypes.Transaction, error) {
	toAddr := types.Address(toAddress)
	sendArgs := &TransactionBridge{
		SwapTx: &SwapTxArgs{
			SendTxArgs: transactions.SendTxArgs{
				From:   types.Address(fromAddress),
				To:     &toAddr,
				Value:  (*hexutil.Big)(amountIn),
				Data:   types.HexBytes("0x0"),
				Symbol: token.Symbol,
			},
			ChainID: network.ChainID,
		},
	}

	return s.BuildTransaction(sendArgs)
}

func (s *SwapParaswap) prepareTransaction(sendArgs *TransactionBridge) error {
	tx, err := s.paraswapClient.BuildTransaction(context.Background(), s.priceRoute.SrcTokenAddress, s.priceRoute.SrcTokenDecimals, s.priceRoute.SrcAmount.Int,
		s.priceRoute.DestTokenAddress, s.priceRoute.DestTokenDecimals, s.priceRoute.DestAmount.Int, common.Address(sendArgs.SwapTx.From), common.Address(*sendArgs.SwapTx.To), s.priceRoute.RawPriceRoute)
	if err != nil {
		return err
	}

	value, ok := new(big.Int).SetString(tx.Value, 10)
	if !ok {
		return errors.New("error converting amount to big.Int")
	}

	gas, err := strconv.ParseUint(tx.Gas, 10, 64)
	if err != nil {
		return err
	}

	gasPrice, ok := new(big.Int).SetString(tx.GasPrice, 10)
	if !ok {
		return errors.New("error converting amount to big.Int")
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

func (s *SwapParaswap) BuildTransaction(sendArgs *TransactionBridge) (*ethTypes.Transaction, error) {
	err := s.prepareTransaction(sendArgs)
	if err != nil {
		return nil, err
	}
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, sendArgs.SwapTx.SendTxArgs)
}

func (s *SwapParaswap) Send(sendArgs *TransactionBridge, verifiedAccount *account.SelectedExtKey) (types.Hash, error) {

	txBridgeArgs := &TransactionBridge{
		SwapTx: &SwapTxArgs{
			SendTxArgs: transactions.SendTxArgs{
				From:               sendArgs.SwapTx.From,
				To:                 sendArgs.SwapTx.To,
				MultiTransactionID: sendArgs.SwapTx.MultiTransactionID,
				Symbol:             sendArgs.SwapTx.Symbol,
			},
		},
	}

	err := s.prepareTransaction(txBridgeArgs)
	if err != nil {
		return types.Hash{}, err
	}

	return s.transactor.SendTransactionWithChainID(txBridgeArgs.ChainID, txBridgeArgs.SwapTx.SendTxArgs, verifiedAccount)
}

func (s *SwapParaswap) CalculateAmountOut(from, to *params.Network, amountIn *big.Int, symbol string) (*big.Int, error) {
	return s.priceRoute.DestAmount.Int, nil
}
