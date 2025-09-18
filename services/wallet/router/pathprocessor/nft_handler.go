package pathprocessor

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/accounts-management/generator"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/utils"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

//go:generate go tool mockgen -source=nft_handler.go -destination=nft_handler_mock_test.go -package=pathprocessor NFTHandler

// NFTHandler handling different types of NFT transfers
type NFTHandler interface {
	Name() string

	CanHandle(contractID thirdparty.ContractID) bool

	PackTxInputData(params ProcessorInputParams) ([]byte, error)

	EstimateGas(params ProcessorInputParams, input []byte, handlerName string) (uint64, error)

	SendOrBuild(
		transactor transactions.TransactorIface,
		rpcClient rpc.ClientInterface,
		sendArgs *MultipathProcessorTxArgs,
		signerFn bind.SignerFn,
		lastUsedNonce int64,
	) (*ethTypes.Transaction, error)

	BuildTransactionV2(
		transactor transactions.TransactorIface,
		sendArgs *wallettypes.SendTxArgs,
		lastUsedNonce int64,
	) (*ethTypes.Transaction, uint64, error)

	GetContractAddress(params ProcessorInputParams) (common.Address, error)
}

type BaseNFTHandler struct {
	rpcClient  rpc.ClientInterface
	transactor transactions.TransactorIface
}

func NewBaseNFTHandler(rpcClient rpc.ClientInterface, transactor transactions.TransactorIface) *BaseNFTHandler {
	return &BaseNFTHandler{
		rpcClient:  rpcClient,
		transactor: transactor,
	}
}

func (h *BaseNFTHandler) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return big.NewInt(0), big.NewInt(0), nil
}

func (h *BaseNFTHandler) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (h *BaseNFTHandler) EstimateGas(params ProcessorInputParams, input []byte, handlerName string) (uint64, error) {
	if params.TestsMode {
		if params.TestEstimationMap != nil {
			if val, ok := params.TestEstimationMap[handlerName]; ok {
				return val.Value, val.Err
			}
		}
		return 0, ErrNoEstimationFound
	}

	ethClient, err := h.rpcClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, err
	}

	estimation, err := ethClient.EstimateGas(context.Background(), ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &params.FromToken.Address,
		Value: new(big.Int),
		Data:  input,
	})
	if err != nil {
		return 0, err
	}

	return uint64(float64(estimation) * pathProcessorCommon.IncreaseEstimatedGasFactor), nil
}

func (h *BaseNFTHandler) PrepareNonce(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64) (uint64, error) {
	var nonce uint64
	var err error
	if lastUsedNonce < 0 {
		nonce, err = h.transactor.NextNonce(context.Background(), h.rpcClient, sendArgs.ChainID, sendArgs.ERC721TransferTx.From)
		if err != nil {
			return 0, err
		}
	} else {
		nonce = uint64(lastUsedNonce) + 1
	}

	argNonce := hexutil.Uint64(nonce)
	sendArgs.ERC721TransferTx.Nonce = &argNonce
	return nonce, nil
}

func (h *BaseNFTHandler) SendOrBuildCollectible(
	sendArgs *MultipathProcessorTxArgs,
	lastUsedNonce int64,
	packDataFn func(ProcessorInputParams) ([]byte, error),
	targetContractID *thirdparty.ContractID, // nil for regular ERC721, specify for special contracts
) (*ethTypes.Transaction, error) {
	from := common.Address(sendArgs.ERC721TransferTx.From)

	inputParams := ProcessorInputParams{
		FromChain: &params.Network{ChainID: sendArgs.ChainID},
		FromAddr:  from,
		ToAddr:    sendArgs.ERC721TransferTx.Recipient,
		FromToken: &tokenTypes.Token{
			Symbol:  sendArgs.ERC721TransferTx.TokenID.String(),
			Address: common.Address(*sendArgs.ERC721TransferTx.To),
		},
	}

	nonce, err := h.PrepareNonce(sendArgs, lastUsedNonce)
	if err != nil {
		return nil, err
	}

	data, err := packDataFn(inputParams)
	if err != nil {
		return nil, err
	}

	var contractAddress *types.Address
	if targetContractID != nil {
		// For special contracts (CryptoKitties, CryptoPunks) force use their address
		addr := types.Address(targetContractID.Address)
		contractAddress = &addr
	} else {
		contractAddress = sendArgs.ERC721TransferTx.To
	}

	tx, _, err := h.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, wallettypes.SendTxArgs{
		From:     sendArgs.ERC721TransferTx.From,
		To:       contractAddress,
		Gas:      sendArgs.ERC721TransferTx.Gas,
		GasPrice: sendArgs.ERC721TransferTx.GasPrice,
		Value:    (*hexutil.Big)(big.NewInt(0)),
		Nonce:    sendArgs.ERC721TransferTx.Nonce,
		Data:     types.HexBytes(data),
	}, int64(nonce-1))

	if err != nil {
		return nil, err
	}

	err = h.transactor.StoreAndTrackPendingTx(from, sendArgs.ERC721TransferTx.Symbol, sendArgs.ChainID, tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (h *BaseNFTHandler) Send(
	sendArgs *MultipathProcessorTxArgs,
	lastUsedNonce int64,
	verifiedAccount *generator.Account,
	handler NFTHandler,
) (types.Hash, uint64, error) {
	tx, err := handler.SendOrBuild(
		h.transactor,
		h.rpcClient,
		sendArgs,
		utils.GetSigner(sendArgs.ChainID, sendArgs.ERC721TransferTx.From, verifiedAccount.PrivateKey()),
		lastUsedNonce,
	)
	if err != nil {
		return types.Hash{}, 0, err
	}
	return types.Hash(tx.Hash()), tx.Nonce(), nil
}

func (h *BaseNFTHandler) BuildTransaction(
	sendArgs *MultipathProcessorTxArgs,
	lastUsedNonce int64,
	handler NFTHandler,
) (*ethTypes.Transaction, uint64, error) {
	tx, err := handler.SendOrBuild(
		h.transactor,
		h.rpcClient,
		sendArgs,
		nil,
		lastUsedNonce,
	)
	return tx, tx.Nonce(), err
}
