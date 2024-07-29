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
	"github.com/status-im/status-go/contracts/community-tokens/collectibles"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/transactions"
)

type ERC721TxArgs struct {
	transactions.SendTxArgs
	TokenID   *hexutil.Big   `json:"tokenId"`
	Recipient common.Address `json:"recipient"`
}

type ERC721Processor struct {
	rpcClient  *rpc.Client
	transactor transactions.TransactorIface
}

func NewERC721Processor(rpcClient *rpc.Client, transactor transactions.TransactorIface) *ERC721Processor {
	return &ERC721Processor{rpcClient: rpcClient, transactor: transactor}
}

func createERC721ErrorResponse(err error) error {
	return createErrorResponse(ProcessorERC721Name, err)
}

func (s *ERC721Processor) Name() string {
	return ProcessorERC721Name
}

func (s *ERC721Processor) AvailableFor(params ProcessorInputParams) (bool, error) {
	return params.FromChain.ChainID == params.ToChain.ChainID && params.ToToken == nil, nil
}

func (s *ERC721Processor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return ZeroBigIntValue, ZeroBigIntValue, nil
}

func (s *ERC721Processor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	abi, err := abi.JSON(strings.NewReader(collectibles.CollectiblesMetaData.ABI))
	if err != nil {
		return []byte{}, createERC721ErrorResponse(err)
	}

	id, success := big.NewInt(0).SetString(params.FromToken.Symbol, 0)
	if !success {
		return []byte{}, createERC721ErrorResponse(fmt.Errorf("failed to convert %s to big.Int", params.FromToken.Symbol))
	}

	return abi.Pack("safeTransferFrom",
		params.FromAddr,
		params.ToAddr,
		id,
	)
}

func (s *ERC721Processor) EstimateGas(params ProcessorInputParams) (uint64, error) {
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
		return 0, createERC721ErrorResponse(err)
	}

	value := new(big.Int)

	input, err := s.PackTxInputData(params)
	if err != nil {
		return 0, createERC721ErrorResponse(err)
	}

	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &params.FromToken.Address,
		Value: value,
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, createERC721ErrorResponse(err)
	}

	increasedEstimation := float64(estimation) * IncreaseEstimatedGasFactor
	return uint64(increasedEstimation), nil
}

func (s *ERC721Processor) BuildTx(params ProcessorInputParams) (*ethTypes.Transaction, error) {
	contractAddress := types.Address(params.FromToken.Address)

	// We store ERC721 Token ID using big.Int.String() in token.Symbol
	tokenID, success := new(big.Int).SetString(params.FromToken.Symbol, 10)
	if !success {
		return nil, createERC721ErrorResponse(fmt.Errorf("failed to convert ERC721's Symbol %s to big.Int", params.FromToken.Symbol))
	}

	sendArgs := &MultipathProcessorTxArgs{
		ERC721TransferTx: &ERC721TxArgs{
			SendTxArgs: transactions.SendTxArgs{
				From:  types.Address(params.FromAddr),
				To:    &contractAddress,
				Value: (*hexutil.Big)(params.AmountIn),
				Data:  types.HexBytes("0x0"),
			},
			TokenID:   (*hexutil.Big)(tokenID),
			Recipient: params.ToAddr,
		},
		ChainID: params.FromChain.ChainID,
	}

	return s.BuildTransaction(sendArgs)
}

func (s *ERC721Processor) sendOrBuild(sendArgs *MultipathProcessorTxArgs, signerFn bind.SignerFn) (tx *ethTypes.Transaction, err error) {
	ethClient, err := s.rpcClient.EthClient(sendArgs.ChainID)
	if err != nil {
		return tx, createERC721ErrorResponse(err)
	}

	contract, err := collectibles.NewCollectibles(common.Address(*sendArgs.ERC721TransferTx.To), ethClient)
	if err != nil {
		return tx, createERC721ErrorResponse(err)
	}

	nonce, err := s.transactor.NextNonce(s.rpcClient, sendArgs.ChainID, sendArgs.ERC721TransferTx.From)
	if err != nil {
		return tx, createERC721ErrorResponse(err)
	}

	argNonce := hexutil.Uint64(nonce)
	sendArgs.ERC721TransferTx.Nonce = &argNonce
	txOpts := sendArgs.ERC721TransferTx.ToTransactOpts(signerFn)
	from := common.Address(sendArgs.ERC721TransferTx.From)
	tx, err = contract.SafeTransferFrom(txOpts, from,
		sendArgs.ERC721TransferTx.Recipient,
		sendArgs.ERC721TransferTx.TokenID.ToInt())
	if err != nil {
		return tx, createERC721ErrorResponse(err)
	}
	fmt.Printf("erc721StoreAndTrackPendingTxArgs0: %v\n", from)
	fmt.Printf("erc721StoreAndTrackPendingTxArgs1: %v\n", sendArgs.ERC721TransferTx.Symbol)
	fmt.Printf("erc721StoreAndTrackPendingTxArgs2: %v\n", sendArgs.ChainID)
	fmt.Printf("erc721StoreAndTrackPendingTxArgs3: %v\n", sendArgs.ERC721TransferTx.MultiTransactionID)
	fmt.Printf("erc721StoreAndTrackPendingTxArgs4: %v\n", tx)
	//err = s.transactor.StoreAndTrackPendingTx(from, sendArgs.ERC721TransferTx.Symbol, sendArgs.ChainID, sendArgs.ERC721TransferTx.MultiTransactionID, tx)
	//if err != nil {
	//	return tx, createERC721ErrorResponse(err)
	//}
	return tx, nil
}

func (s *ERC721Processor) Send(sendArgs *MultipathProcessorTxArgs, verifiedAccount *account.SelectedExtKey) (hash types.Hash, err error) {
	tx, err := s.sendOrBuild(sendArgs, getSigner(sendArgs.ChainID, sendArgs.ERC721TransferTx.From, verifiedAccount))
	if err != nil {
		return hash, createERC721ErrorResponse(err)
	}
	return types.Hash(tx.Hash()), nil
}

func (s *ERC721Processor) BuildTransaction(sendArgs *MultipathProcessorTxArgs) (*ethTypes.Transaction, error) {
	return s.sendOrBuild(sendArgs, nil)
}

func (s *ERC721Processor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *ERC721Processor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	return params.FromToken.Address, nil
}
