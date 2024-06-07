package bridge

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

type ERC721TransferTxArgs struct {
	transactions.SendTxArgs
	TokenID   *hexutil.Big   `json:"tokenId"`
	Recipient common.Address `json:"recipient"`
}

type ERC721TransferBridge struct {
	rpcClient  *rpc.Client
	transactor transactions.TransactorIface
}

func NewERC721TransferBridge(rpcClient *rpc.Client, transactor transactions.TransactorIface) *ERC721TransferBridge {
	return &ERC721TransferBridge{rpcClient: rpcClient, transactor: transactor}
}

func (s *ERC721TransferBridge) Name() string {
	return ERC721TransferName
}

func (s *ERC721TransferBridge) AvailableFor(params BridgeParams) (bool, error) {
	return params.FromChain.ChainID == params.ToChain.ChainID && params.ToToken == nil, nil
}

func (s *ERC721TransferBridge) CalculateFees(params BridgeParams) (*big.Int, *big.Int, error) {
	return big.NewInt(0), big.NewInt(0), nil
}

func (s *ERC721TransferBridge) PackTxInputData(params BridgeParams, contractType string) ([]byte, error) {
	abi, err := abi.JSON(strings.NewReader(collectibles.CollectiblesMetaData.ABI))
	if err != nil {
		return []byte{}, err
	}

	id, success := big.NewInt(0).SetString(params.FromToken.Symbol, 0)
	if !success {
		return []byte{}, fmt.Errorf("failed to convert %s to big.Int", params.FromToken.Symbol)
	}

	return abi.Pack("safeTransferFrom",
		params.FromAddr,
		params.ToAddr,
		id,
	)
}

func (s *ERC721TransferBridge) EstimateGas(params BridgeParams) (uint64, error) {
	ethClient, err := s.rpcClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, err
	}

	value := new(big.Int)

	input, err := s.PackTxInputData(params, "")
	if err != nil {
		return 0, err
	}

	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &params.FromToken.Address,
		Value: value,
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, err
	}

	increasedEstimation := float64(estimation) * IncreaseEstimatedGasFactor
	return uint64(increasedEstimation), nil
}

func (s *ERC721TransferBridge) BuildTx(params BridgeParams) (*ethTypes.Transaction, error) {
	contractAddress := types.Address(params.FromToken.Address)

	// We store ERC721 Token ID using big.Int.String() in token.Symbol
	tokenID, success := new(big.Int).SetString(params.FromToken.Symbol, 10)
	if !success {
		return nil, fmt.Errorf("failed to convert ERC721's Symbol %s to big.Int", params.FromToken.Symbol)
	}

	sendArgs := &TransactionBridge{
		ERC721TransferTx: &ERC721TransferTxArgs{
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

func (s *ERC721TransferBridge) sendOrBuild(sendArgs *TransactionBridge, signerFn bind.SignerFn) (tx *ethTypes.Transaction, err error) {
	ethClient, err := s.rpcClient.EthClient(sendArgs.ChainID)
	if err != nil {
		return tx, err
	}

	contract, err := collectibles.NewCollectibles(common.Address(*sendArgs.ERC721TransferTx.To), ethClient)
	if err != nil {
		return tx, err
	}

	nonce, err := s.transactor.NextNonce(s.rpcClient, sendArgs.ChainID, sendArgs.ERC721TransferTx.From)
	if err != nil {
		return tx, err
	}

	argNonce := hexutil.Uint64(nonce)
	sendArgs.ERC721TransferTx.Nonce = &argNonce
	txOpts := sendArgs.ERC721TransferTx.ToTransactOpts(signerFn)

	tx, err = contract.SafeTransferFrom(txOpts, common.Address(sendArgs.ERC721TransferTx.From),
		sendArgs.ERC721TransferTx.Recipient,
		sendArgs.ERC721TransferTx.TokenID.ToInt())
	return tx, err
}

func (s *ERC721TransferBridge) Send(sendArgs *TransactionBridge, verifiedAccount *account.SelectedExtKey) (hash types.Hash, err error) {
	tx, err := s.sendOrBuild(sendArgs, getSigner(sendArgs.ChainID, sendArgs.ERC721TransferTx.From, verifiedAccount))
	if err != nil {
		return hash, err
	}
	return types.Hash(tx.Hash()), nil
}

func (s *ERC721TransferBridge) BuildTransaction(sendArgs *TransactionBridge) (*ethTypes.Transaction, error) {
	return s.sendOrBuild(sendArgs, nil)
}

func (s *ERC721TransferBridge) CalculateAmountOut(params BridgeParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *ERC721TransferBridge) GetContractAddress(params BridgeParams) (common.Address, error) {
	return params.FromToken.Address, nil
}
