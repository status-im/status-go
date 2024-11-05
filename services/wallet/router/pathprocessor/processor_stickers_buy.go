package pathprocessor

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/contracts"
	"github.com/status-im/status-go/contracts/snt"
	stickersContracts "github.com/status-im/status-go/contracts/stickers"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/transactions"
)

type StickersBuyProcessor struct {
	contractMaker *contracts.ContractMaker
	transactor    transactions.TransactorIface
}

func NewStickersBuyProcessor(rpcClient *rpc.Client, transactor transactions.TransactorIface) *StickersBuyProcessor {
	return &StickersBuyProcessor{
		contractMaker: &contracts.ContractMaker{
			RPCClient: rpcClient,
		},
		transactor: transactor,
	}
}

func createStickersBuyErrorResponse(err error) error {
	return createErrorResponse(walletCommon.ProcessorStickersBuyName, err)
}

func (s *StickersBuyProcessor) Name() string {
	return walletCommon.ProcessorStickersBuyName
}

func (s *StickersBuyProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	return params.FromChain.ChainID == walletCommon.EthereumMainnet || params.FromChain.ChainID == walletCommon.EthereumSepolia, nil
}

func (s *StickersBuyProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return walletCommon.ZeroBigIntValue(), walletCommon.ZeroBigIntValue(), nil
}

func (s *StickersBuyProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	stickerType, err := s.contractMaker.NewStickerType(params.FromChain.ChainID)
	if err != nil {
		return []byte{}, createStickersBuyErrorResponse(err)
	}

	callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}

	packInfo, err := stickerType.GetPackData(callOpts, params.PackID)
	if err != nil {
		return []byte{}, createStickersBuyErrorResponse(err)
	}

	stickerMarketABI, err := abi.JSON(strings.NewReader(stickersContracts.StickerMarketABI))
	if err != nil {
		return []byte{}, createStickersBuyErrorResponse(err)
	}

	extraData, err := stickerMarketABI.Pack("buyToken", params.PackID, params.FromAddr, packInfo.Price)
	if err != nil {
		return []byte{}, createStickersBuyErrorResponse(err)
	}

	sntABI, err := abi.JSON(strings.NewReader(snt.SNTABI))
	if err != nil {
		return []byte{}, createStickersBuyErrorResponse(err)
	}

	stickerMarketAddress, err := stickersContracts.StickerMarketContractAddress(params.FromChain.ChainID)
	if err != nil {
		return []byte{}, createStickersBuyErrorResponse(err)
	}

	return sntABI.Pack("approveAndCall", stickerMarketAddress, packInfo.Price, extraData)
}

func (s *StickersBuyProcessor) EstimateGas(params ProcessorInputParams) (uint64, error) {
	if params.TestsMode {
		if params.TestEstimationMap != nil {
			if val, ok := params.TestEstimationMap[s.Name()]; ok {
				return val.Value, val.Err
			}
		}
		return 0, ErrNoEstimationFound
	}

	contractAddress, err := s.GetContractAddress(params)
	if err != nil {
		return 0, createStickersBuyErrorResponse(err)
	}

	input, err := s.PackTxInputData(params)
	if err != nil {
		return 0, createStickersBuyErrorResponse(err)
	}

	ethClient, err := s.contractMaker.RPCClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, createStickersBuyErrorResponse(err)
	}

	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &contractAddress,
		Value: walletCommon.ZeroBigIntValue(),
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, createStickersBuyErrorResponse(err)
	}

	increasedEstimation := float64(estimation) * IncreaseEstimatedGasFactor

	return uint64(increasedEstimation), nil
}

func (s *StickersBuyProcessor) Send(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64, verifiedAccount *account.SelectedExtKey) (hash types.Hash, usedNonce uint64, err error) {
	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce, verifiedAccount)
}

func (s *StickersBuyProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce)
}

func (s *StickersBuyProcessor) BuildTransactionV2(sendArgs *transactions.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, *sendArgs, lastUsedNonce)
}

func (s *StickersBuyProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *StickersBuyProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	return snt.ContractAddress(params.FromChain.ChainID)
}
