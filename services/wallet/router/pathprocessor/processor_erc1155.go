package pathprocessor

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/contracts/ierc1155"
	statusErrors "github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/transactions"
)

type ERC1155TxArgs struct {
	transactions.SendTxArgs
	TokenID   *hexutil.Big   `json:"tokenId"`
	Recipient common.Address `json:"recipient"`
	Amount    *hexutil.Big   `json:"amount"`
}

type ERC1155Processor struct {
	rpcClient  *rpc.Client
	transactor transactions.TransactorIface
}

func NewERC1155Processor(rpcClient *rpc.Client, transactor transactions.TransactorIface) *ERC1155Processor {
	return &ERC1155Processor{rpcClient: rpcClient, transactor: transactor}
}

func (s *ERC1155Processor) Name() string {
	return ProcessorERC1155Name
}

func (s *ERC1155Processor) AvailableFor(params ProcessorInputParams) (bool, error) {
	return params.FromChain.ChainID == params.ToChain.ChainID && params.ToToken == nil, nil
}

func (s *ERC1155Processor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return ZeroBigIntValue, ZeroBigIntValue, nil
}

func (s *ERC1155Processor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	abi, err := abi.JSON(strings.NewReader(ierc1155.Ierc1155ABI))
	if err != nil {
		return []byte{}, statusErrors.CreateErrorResponseFromError(err)
	}

	id, success := big.NewInt(0).SetString(params.FromToken.Symbol, 0)
	if !success {
		return []byte{}, statusErrors.CreateErrorResponseFromError(fmt.Errorf("failed to convert %s to big.Int", params.FromToken.Symbol))
	}

	return abi.Pack("safeTransferFrom",
		params.FromAddr,
		params.ToAddr,
		id,
		params.AmountIn,
		[]byte{},
	)
}

func (s *ERC1155Processor) EstimateGas(params ProcessorInputParams) (uint64, error) {
	if params.TestsMode {
		if params.TestEstimationMap != nil {
			if val, ok := params.TestEstimationMap[s.Name()]; ok {
				return val, nil
			}
		}
		return 0, ErrNoEstimationFound
	}

	ethClient, err := s.rpcClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, statusErrors.CreateErrorResponseFromError(err)
	}

	value := new(big.Int)

	input, err := s.PackTxInputData(params)
	if err != nil {
		return 0, statusErrors.CreateErrorResponseFromError(err)
	}

	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &params.FromToken.Address,
		Value: value,
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, statusErrors.CreateErrorResponseFromError(err)
	}
	increasedEstimation := float64(estimation) * IncreaseEstimatedGasFactor
	return uint64(increasedEstimation), nil
}

func (s *ERC1155Processor) BuildTx(params ProcessorInputParams) (*ethTypes.Transaction, error) {
	contractAddress := types.Address(params.FromToken.Address)

	// We store ERC1155 Token ID using big.Int.String() in token.Symbol
	tokenID, success := new(big.Int).SetString(params.FromToken.Symbol, 10)
	if !success {
		return nil, statusErrors.CreateErrorResponseFromError(fmt.Errorf("failed to convert ERC1155's Symbol %s to big.Int", params.FromToken.Symbol))
	}

	sendArgs := &MultipathProcessorTxArgs{
		ERC1155TransferTx: &ERC1155TxArgs{
			SendTxArgs: transactions.SendTxArgs{
				From:  types.Address(params.FromAddr),
				To:    &contractAddress,
				Value: (*hexutil.Big)(params.AmountIn),
				Data:  types.HexBytes("0x0"),
			},
			TokenID:   (*hexutil.Big)(tokenID),
			Recipient: params.ToAddr,
			Amount:    (*hexutil.Big)(params.AmountIn),
		},
		ChainID: params.FromChain.ChainID,
	}

	return s.BuildTransaction(sendArgs)
}

func (s *ERC1155Processor) sendOrBuild(sendArgs *MultipathProcessorTxArgs, signerFn bind.SignerFn) (tx *ethTypes.Transaction, err error) {
	ethClient, err := s.rpcClient.EthClient(sendArgs.ChainID)
	if err != nil {
		return tx, statusErrors.CreateErrorResponseFromError(err)
	}

	contract, err := ierc1155.NewIerc1155(common.Address(*sendArgs.ERC1155TransferTx.To), ethClient)
	if err != nil {
		return tx, statusErrors.CreateErrorResponseFromError(err)
	}

	nonce, err := s.transactor.NextNonce(s.rpcClient, sendArgs.ChainID, sendArgs.ERC1155TransferTx.From)
	if err != nil {
		return tx, statusErrors.CreateErrorResponseFromError(err)
	}

	argNonce := hexutil.Uint64(nonce)
	sendArgs.ERC1155TransferTx.Nonce = &argNonce
	txOpts := sendArgs.ERC1155TransferTx.ToTransactOpts(signerFn)
	tx, err = contract.SafeTransferFrom(
		txOpts, common.Address(sendArgs.ERC1155TransferTx.From),
		sendArgs.ERC1155TransferTx.Recipient,
		sendArgs.ERC1155TransferTx.TokenID.ToInt(),
		sendArgs.ERC1155TransferTx.Amount.ToInt(),
		[]byte{},
	)
	if err != nil {
		return tx, statusErrors.CreateErrorResponseFromError(err)
	}

	return tx, nil
}

func (s *ERC1155Processor) Send(sendArgs *MultipathProcessorTxArgs, verifiedAccount *account.SelectedExtKey) (hash types.Hash, err error) {
	tx, err := s.sendOrBuild(sendArgs, getSigner(sendArgs.ChainID, sendArgs.ERC1155TransferTx.From, verifiedAccount))
	if err != nil {
		return hash, statusErrors.CreateErrorResponseFromError(err)
	}
	return types.Hash(tx.Hash()), nil
}

func (s *ERC1155Processor) BuildTransaction(sendArgs *MultipathProcessorTxArgs) (*ethTypes.Transaction, error) {
	return s.sendOrBuild(sendArgs, nil)
}

func (s *ERC1155Processor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *ERC1155Processor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	return params.FromToken.Address, nil
}
