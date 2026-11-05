package router

import (
	"context"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common/hexutil"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor"
	pathprocessorcommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
)

// newRecipientRaisesGas tells whether the recipient is new and the transfer costs more than the
// same transfer to the sender (e.g. EIP-8037 state gas). Errors just mean false.
func newRecipientRaisesGas(processor pathprocessor.PathProcessor, params pathprocessor.ProcessorInputParams,
	estimatedGas uint64, recipientIsNew func() (bool, error), logger *zap.Logger) bool {
	if processor.Name() != pathprocessorcommon.ProcessorTransferName || estimatedGas == 0 ||
		params.FromToken == nil || params.ToAddr == params.FromAddr {
		return false
	}

	isNew, err := recipientIsNew()
	if err != nil {
		logger.Debug("newRecipientRaisesGas: recipient state unknown, no hint", zap.Error(err))
		return false
	}
	if !isNew {
		return false
	}

	toSelf := params
	toSelf.ToAddr = params.FromAddr
	packedData, err := processor.PackTxInputData(toSelf)
	if err != nil {
		logger.Debug("newRecipientRaisesGas: packing the transfer to self failed, no hint", zap.Error(err))
		return false
	}
	existingRecipientGas, err := processor.EstimateGas(toSelf, packedData)
	if err != nil {
		logger.Debug("newRecipientRaisesGas: estimating the transfer to self failed, no hint", zap.Error(err))
		return false
	}
	logger.Debug("newRecipientRaisesGas: new recipient",
		zap.Uint64("estimatedGas", estimatedGas),
		zap.Uint64("existingRecipientGas", existingRecipientGas))
	return estimatedGas > existingRecipientGas
}

// recipientIsNew tells whether the recipient has no balance of the token and, for the native
// token, no nonce or code either.
func (r *Router) recipientIsNew(ctx context.Context, params pathprocessor.ProcessorInputParams) (bool, error) {
	chainID := params.FromChain.ChainID
	balance, err := r.getBalance(ctx, chainID, params.FromToken, params.ToAddr)
	if err != nil {
		return false, err
	}
	if balance.Sign() > 0 {
		return false, nil
	}
	if !params.FromToken.IsNative() {
		return true, nil
	}

	ethClient, err := r.rpcClient.EthClient(chainID)
	if err != nil {
		return false, err
	}
	var (
		nonce hexutil.Uint64
		code  hexutil.Bytes
	)
	batch := []gethrpc.BatchElem{
		{Method: "eth_getTransactionCount", Args: []interface{}{params.ToAddr, "latest"}, Result: &nonce},
		{Method: "eth_getCode", Args: []interface{}{params.ToAddr, "latest"}, Result: &code},
	}
	if err := ethClient.BatchCallContext(ctx, batch); err != nil {
		return false, err
	}
	for _, elem := range batch {
		if elem.Error != nil {
			return false, elem.Error
		}
	}
	return nonce == 0 && len(code) == 0, nil
}
