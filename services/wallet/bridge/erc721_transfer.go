package bridge

import (
	"math/big"

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

func (s *ERC721TransferBridge) EstimateGas(from, to *params.Network, account common.Address, token *token.Token, amountIn *big.Int) (uint64, error) {
	// ethClient, err := s.rpcClient.EthClient(from.ChainID)
	// if err != nil {
	// 	return 0, err
	// }
	// collectiblesABI, err := abi.JSON(strings.NewReader(collectibles.CollectiblesABI))
	// if err != nil {
	// 	return 0, err
	// }

	// toAddress := common.HexToAddress("0x0")
	// tokenID, success := new(big.Int).SetString(token.Symbol, 10)
	// if !success {
	// 	return 0, err
	// }

	// data, err := collectiblesABI.Pack("safeTransferFrom", account, toAddress, tokenID)
	// if err != nil {
	// 	return 0, err
	// }
	// estimate, err := ethClient.EstimateGas(context.Background(), ethereum.CallMsg{
	// 	From:  account,
	// 	To:    &toAddress,
	// 	Value: big.NewInt(0),
	// 	Data:  data,
	// })
	// if err != nil {
	// 	return 0, err
	// }
	// return estimate + 1000, nil
	return 80000, nil
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
