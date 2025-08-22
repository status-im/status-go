package pathprocessor

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/accounts-management/generator"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/utils"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

type ERC721TxArgs struct {
	wallettypes.SendTxArgs
	TokenID   *hexutil.Big   `json:"tokenId"`
	Recipient common.Address `json:"recipient"`
}

// NFTProcessor handles NFT transfers using strategy pattern
type NFTProcessor struct {
	rpcClient  rpc.ClientInterface
	transactor transactions.TransactorIface
	handlers   []NFTHandler
}

func NewNFTProcessor(rpcClient rpc.ClientInterface, transactor transactions.TransactorIface) *NFTProcessor {
	processor := &NFTProcessor{
		rpcClient:  rpcClient,
		transactor: transactor,
		handlers:   make([]NFTHandler, 0),
	}

	// Register handlers in order of priority
	// Specialized handlers first, then generic ERC721
	processor.handlers = append(processor.handlers, NewCryptoKittiesHandler(rpcClient, transactor))
	processor.handlers = append(processor.handlers, NewCryptoPunksHandler(rpcClient, transactor))
	processor.handlers = append(processor.handlers, NewERC721Handler(rpcClient, transactor))

	return processor
}

func createNFTErrorResponse(err error) error {
	return createErrorResponse(pathProcessorCommon.ProcessorERC721Name, err)
}

func (s *NFTProcessor) Name() string {
	return pathProcessorCommon.ProcessorERC721Name
}

func (s *NFTProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	if params.FromChain == nil || params.ToChain == nil {
		return false, ErrNoChainSet
	}
	if params.FromToken == nil {
		return false, ErrNoTokenSet
	}

	// Only handle same-chain transfers with no destination token (NFT transfers)
	if params.FromChain.ChainID != params.ToChain.ChainID || params.ToToken != nil {
		return false, nil
	}

	return s.getHandlerForContract(params) != nil, nil
}

func (s *NFTProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return walletCommon.ZeroBigIntValue(), walletCommon.ZeroBigIntValue(), nil
}

func (s *NFTProcessor) getHandlerForContractID(contractID thirdparty.ContractID) NFTHandler {
	for _, handler := range s.handlers {
		if handler.CanHandle(contractID) {
			return handler
		}
	}

	return nil
}

func (s *NFTProcessor) getHandlerForContract(params ProcessorInputParams) NFTHandler {
	contractID := thirdparty.ContractID{
		ChainID: walletCommon.ChainID(params.FromChain.ChainID),
		Address: params.FromToken.Address,
	}

	return s.getHandlerForContractID(contractID)
}

func (s *NFTProcessor) getHandlerForMultipathTx(sendArgs *MultipathProcessorTxArgs) NFTHandler {
	contractID := thirdparty.ContractID{
		ChainID: walletCommon.ChainID(sendArgs.ChainID),
		Address: common.Address(*sendArgs.ERC721TransferTx.To),
	}

	return s.getHandlerForContractID(contractID)
}

func (s *NFTProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	handler := s.getHandlerForContract(params)
	if handler == nil {
		return nil, createNFTErrorResponse(ErrNoTokenSet)
	}

	data, err := handler.PackTxInputData(params)
	if err != nil {
		return nil, createNFTErrorResponse(err)
	}

	return data, nil
}

func (s *NFTProcessor) EstimateGas(params ProcessorInputParams, input []byte) (uint64, error) {
	handler := s.getHandlerForContract(params)
	if handler == nil {
		return 0, createNFTErrorResponse(ErrNoTokenSet)
	}

	estimation, err := handler.EstimateGas(params, input, handler.Name())
	if err != nil {
		return 0, createNFTErrorResponse(err)
	}
	return estimation, nil
}

func (s *NFTProcessor) Send(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64, verifiedAccount *generator.Account) (types.Hash, uint64, error) {
	handler := s.getHandlerForMultipathTx(sendArgs)
	if handler == nil {
		return types.Hash{}, 0, createNFTErrorResponse(ErrNoTokenSet)
	}

	tx, err := handler.SendOrBuild(
		s.transactor,
		s.rpcClient,
		sendArgs,
		utils.GetSigner(sendArgs.ChainID, sendArgs.ERC721TransferTx.From, verifiedAccount.PrivateKey()),
		lastUsedNonce,
	)
	if err != nil {
		return types.Hash{}, 0, createNFTErrorResponse(err)
	}
	return types.Hash(tx.Hash()), tx.Nonce(), nil
}

func (s *NFTProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	handler := s.getHandlerForMultipathTx(sendArgs)
	if handler == nil {
		return nil, 0, createNFTErrorResponse(ErrNoTokenSet)
	}

	tx, err := handler.SendOrBuild(
		s.transactor,
		s.rpcClient,
		sendArgs,
		nil,
		lastUsedNonce,
	)
	if err != nil {
		return nil, 0, createNFTErrorResponse(err)
	}
	return tx, tx.Nonce(), nil
}

func (s *NFTProcessor) BuildTransactionV2(sendArgs *wallettypes.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	if sendArgs.To == nil {
		return nil, 0, createNFTErrorResponse(ErrNoTokenSet)
	}

	contractID := thirdparty.ContractID{
		ChainID: walletCommon.ChainID(sendArgs.FromChainID),
		Address: common.Address(*sendArgs.To),
	}
	handler := s.getHandlerForContractID(contractID)
	if handler == nil {
		return nil, 0, createNFTErrorResponse(ErrNoTokenSet)
	}

	return handler.BuildTransactionV2(s.transactor, sendArgs, lastUsedNonce)
}

func (s *NFTProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *NFTProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	handler := s.getHandlerForContract(params)
	if handler == nil {
		return common.Address{}, createNFTErrorResponse(ErrNoTokenSet)
	}

	return handler.GetContractAddress(params)
}
