package pathprocessor

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/internal/transactions"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
	"github.com/status-im/status-go/pkg/services/wallet/wallettypes"
)

type ERC721TxArgs struct {
	wallettypes.SendTxArgs
	TokenID   *hexutil.Big   `json:"tokenId"`
	Recipient common.Address `json:"recipient"`
}

// NFTProcessor handles NFT transfers using strategy pattern
type NFTProcessor struct {
	ethClientGetter rpc.EthClientGetter
	transactor      transactions.TransactorIface
	handlers        []NFTHandler
}

func NewNFTProcessor(ethClientGetter rpc.EthClientGetter, transactor transactions.TransactorIface) *NFTProcessor {
	processor := &NFTProcessor{
		ethClientGetter: ethClientGetter,
		transactor:      transactor,
		handlers:        make([]NFTHandler, 0),
	}

	// Register handlers in order of priority
	// Specialized handlers first, then generic ERC721
	processor.handlers = append(processor.handlers, NewCryptoKittiesHandler(ethClientGetter, transactor))
	processor.handlers = append(processor.handlers, NewCryptoPunksHandler(ethClientGetter, transactor))
	processor.handlers = append(processor.handlers, NewERC721Handler(ethClientGetter, transactor))

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
	if params.FromChain.ChainID != params.ToChain.ChainID {
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
