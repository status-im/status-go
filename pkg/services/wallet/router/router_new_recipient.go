package router

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor"
	pathprocessorcommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
)

// newRecipientCheckApplies tells whether the path is a plain transfer to another address.
func newRecipientCheckApplies(processor pathprocessor.PathProcessor, params pathprocessor.ProcessorInputParams) bool {
	return processor.Name() == pathprocessorcommon.ProcessorTransferName &&
		params.FromToken != nil && params.FromChain != nil && params.ToAddr != params.FromAddr
}

// checkRecipientIsNew starts the recipient check in the background and returns a func to wait for it.
func (r *Router) checkRecipientIsNew(ctx context.Context, processor pathprocessor.PathProcessor,
	params pathprocessor.ProcessorInputParams) func() (bool, error) {
	if !newRecipientCheckApplies(processor, params) {
		return func() (bool, error) { return false, nil }
	}
	type result struct {
		isNew bool
		err   error
	}
	done := make(chan result, 1)
	go func() {
		defer panics.LogOnPanic()
		isNew, err := r.recipientIsNew(ctx, params)
		done <- result{isNew, err}
	}()
	return func() (bool, error) {
		res := <-done
		return res.isNew, res.err
	}
}

// newRecipientRaisesGas tells whether the recipient is new and the transfer costs more than the
// same transfer to the sender (e.g. EIP-8037 state gas). Errors just mean false.
func newRecipientRaisesGas(processor pathprocessor.PathProcessor, params pathprocessor.ProcessorInputParams,
	estimatedGas uint64, recipientIsNew func() (bool, error), logger *zap.Logger) bool {
	// skipping the wait is safe: the channel is buffered
	if estimatedGas == 0 || !newRecipientCheckApplies(processor, params) {
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
// token, no nonce or code either. Cached for recipientStateTTL.
func (r *Router) recipientIsNew(ctx context.Context, params pathprocessor.ProcessorInputParams) (bool, error) {
	key := recipientStateKey{chainID: params.FromChain.ChainID, token: params.FromToken.Address, recipient: params.ToAddr}
	if isNew, ok := r.newRecipients.get(key, time.Now()); ok {
		return isNew, nil
	}
	isNew, err := r.fetchRecipientIsNew(ctx, params)
	if err != nil {
		return false, err
	}
	r.newRecipients.set(key, isNew, time.Now())
	return isNew, nil
}

func (r *Router) fetchRecipientIsNew(ctx context.Context, params pathprocessor.ProcessorInputParams) (bool, error) {
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

const recipientStateTTL = time.Minute

type recipientStateKey struct {
	chainID   uint64
	token     common.Address
	recipient common.Address
}

type recipientState struct {
	isNew     bool
	fetchedAt time.Time
}

// recipientStateCache caches recipientIsNew answers; the zero value is ready to use.
type recipientStateCache struct {
	mu      sync.Mutex
	entries map[recipientStateKey]recipientState
}

func (c *recipientStateCache) get(key recipientStateKey, now time.Time) (bool, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state, ok := c.entries[key]
	if !ok || now.Sub(state.fetchedAt) >= recipientStateTTL {
		return false, false
	}
	return state.isNew, true
}

func (c *recipientStateCache) set(key recipientStateKey, isNew bool, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[recipientStateKey]recipientState)
	}
	// keep the map bounded
	for k, state := range c.entries {
		if now.Sub(state.fetchedAt) >= recipientStateTTL {
			delete(c.entries, k)
		}
	}
	c.entries[key] = recipientState{isNew: isNew, fetchedAt: now}
}
