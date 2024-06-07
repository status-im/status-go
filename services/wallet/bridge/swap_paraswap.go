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
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty/paraswap"
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
	transactor     transactions.TransactorIface
}

func NewSwapParaswap(rpcClient *rpc.Client, transactor transactions.TransactorIface, tokenManager *walletToken.Manager) *SwapParaswap {
	return &SwapParaswap{
		paraswapClient: paraswap.NewClientV5(walletCommon.EthereumMainnet),
		transactor:     transactor,
	}
}

func (s *SwapParaswap) Name() string {
	return SwapParaswapName
}

func (s *SwapParaswap) AvailableFor(params BridgeParams) (bool, error) {
	if params.FromToken == nil || params.ToToken == nil {
		return false, errors.New("token and toToken cannot be nil")
	}

	if params.FromChain.ChainID != params.ToChain.ChainID {
		return false, nil
	}

	s.paraswapClient.SetChainID(params.FromChain.ChainID)

	searchForToken := params.FromToken.Address == ZeroAddress
	searchForToToken := params.ToToken.Address == ZeroAddress
	if searchForToToken || searchForToken {
		tokensList, err := s.paraswapClient.FetchTokensList(context.Background())
		if err != nil {
			return false, err
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
		return false, errors.New("cannot resolve token/s")
	}

	return true, nil
}

func (s *SwapParaswap) CalculateFees(params BridgeParams) (*big.Int, *big.Int, error) {
	return big.NewInt(0), big.NewInt(0), nil
}

func (s *SwapParaswap) PackTxInputData(params BridgeParams, contractType string) ([]byte, error) {
	// not sure what we can do here since we're using the api to build the transaction
	return []byte{}, nil
}

func (s *SwapParaswap) EstimateGas(params BridgeParams) (uint64, error) {
	priceRoute, err := s.paraswapClient.FetchPriceRoute(context.Background(), params.FromToken.Address, params.FromToken.Decimals,
		params.ToToken.Address, params.ToToken.Decimals, params.AmountIn, params.FromAddr, params.ToAddr)
	if err != nil {
		return 0, err
	}

	s.priceRoute = priceRoute

	return priceRoute.GasCost.Uint64(), nil
}

func (s *SwapParaswap) GetContractAddress(params BridgeParams) (address common.Address, err error) {
	if params.FromChain.ChainID == walletCommon.EthereumMainnet {
		address = common.HexToAddress("0x216b4b4ba9f3e719726886d34a177484278bfcae")
	} else if params.FromChain.ChainID == walletCommon.ArbitrumMainnet {
		address = common.HexToAddress("0x216b4b4ba9f3e719726886d34a177484278bfcae")
	} else if params.FromChain.ChainID == walletCommon.OptimismMainnet {
		address = common.HexToAddress("0x216b4b4ba9f3e719726886d34a177484278bfcae")
	} else {
		err = errors.New("unsupported network")
	}
	return
}

func (s *SwapParaswap) BuildTx(params BridgeParams) (*ethTypes.Transaction, error) {
	toAddr := types.Address(params.ToAddr)
	sendArgs := &TransactionBridge{
		SwapTx: &SwapTxArgs{
			SendTxArgs: transactions.SendTxArgs{
				From:   types.Address(params.FromAddr),
				To:     &toAddr,
				Value:  (*hexutil.Big)(params.AmountIn),
				Data:   types.HexBytes("0x0"),
				Symbol: params.FromToken.Symbol,
			},
			ChainID: params.FromChain.ChainID,
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

func (s *SwapParaswap) CalculateAmountOut(params BridgeParams) (*big.Int, error) {
	return s.priceRoute.DestAmount.Int, nil
}
