package bridge

import (
	"context"
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
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/transactions"
)

type ERC721TransferTxArgs struct {
	transactions.SendTxArgs
	TokenID   *hexutil.Big   `json:"tokenId"`
	Recipient common.Address `json:"recipient"`
}

type ERC721TransferBridge struct {
	rpcClient  *rpc.Client
	transactor *transactions.Transactor
}

func NewERC721TransferBridge(rpcClient *rpc.Client, transactor *transactions.Transactor) *ERC721TransferBridge {
	return &ERC721TransferBridge{rpcClient: rpcClient, transactor: transactor}
}

func (s *ERC721TransferBridge) Name() string {
	return "ERC721Transfer"
}

func (s *ERC721TransferBridge) Can(from, to *params.Network, token *token.Token, balance *big.Int) (bool, error) {
	return from.ChainID == to.ChainID, nil
}

func (s *ERC721TransferBridge) CalculateFees(from, to *params.Network, token *token.Token, amountIn *big.Int, nativeTokenPrice, tokenPrice float64, gasPrice *big.Float) (*big.Int, *big.Int, error) {
	return big.NewInt(0), big.NewInt(0), nil
}

func (s *ERC721TransferBridge) EstimateGas(fromNetwork *params.Network, toNetwork *params.Network, from common.Address, to common.Address, token *token.Token, amountIn *big.Int) (uint64, error) {
	ethClient, err := s.rpcClient.EthClient(fromNetwork.ChainID)
	if err != nil {
		return 0, err
	}

	var input []byte
	value := new(big.Int)

	contractAddress := to

	abi, err := abi.JSON(strings.NewReader(collectibles.CollectiblesMetaData.ABI))
	if err != nil {
		return 0, err
	}

	input, err = abi.Pack("safeTransferFrom",
		from,
		to,
		new(big.Int))

	if err != nil {
		return 0, err
	}

	ctx := context.Background()

	if code, err := ethClient.PendingCodeAt(ctx, contractAddress); err != nil {
		return 0, err
	} else if len(code) == 0 {
		return 0, bind.ErrNoCode
	}

	msg := ethereum.CallMsg{
		From:  from,
		To:    &contractAddress,
		Value: value,
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(ctx, msg)
	if err != nil {
		return 0, err
	}
	increasedEstimation := float64(estimation) * IncreaseEstimatedGasFactor
	return uint64(increasedEstimation), nil
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

	nonce, unlock, err := s.transactor.NextNonce(s.rpcClient, sendArgs.ChainID, sendArgs.ERC721TransferTx.From)
	if err != nil {
		return tx, err
	}
	defer func() {
		unlock(err == nil, nonce)
	}()
	argNonce := hexutil.Uint64(nonce)
	sendArgs.ERC721TransferTx.Nonce = &argNonce
	txOpts := sendArgs.ERC721TransferTx.ToTransactOpts(signerFn)
	return contract.SafeTransferFrom(txOpts, common.Address(sendArgs.ERC721TransferTx.From), sendArgs.ERC721TransferTx.Recipient,
		sendArgs.ERC721TransferTx.TokenID.ToInt())
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

func (s *ERC721TransferBridge) CalculateAmountOut(from, to *params.Network, amountIn *big.Int, symbol string) (*big.Int, error) {
	return amountIn, nil
}

func (s *ERC721TransferBridge) GetContractAddress(network *params.Network, token *token.Token) *common.Address {
	return &token.Address
}
