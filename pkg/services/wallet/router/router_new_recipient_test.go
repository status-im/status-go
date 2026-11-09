package router

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor"
	pathprocessorcommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
	mockpathprocessor "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/mock"
	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

func TestNewRecipientRaisesGas(t *testing.T) {
	logger := logutils.ZapLogger().Named("router-test")
	sender := common.HexToAddress("0x1")
	recipient := common.HexToAddress("0x2")
	usdc := &tokentypes.Token{Token: &types.Token{Symbol: "USDC", Address: common.HexToAddress("0xa0b8"), ChainID: 1}}
	eth := &tokentypes.Token{Token: &types.Token{Symbol: "ETH", Address: common.Address{}, ChainID: 1}}

	inputParams := func(token *tokentypes.Token, to common.Address) pathprocessor.ProcessorInputParams {
		return pathprocessor.ProcessorInputParams{
			FromChain: &params.Network{ChainID: 1},
			FromToken: token,
			FromAddr:  sender,
			ToAddr:    to,
			AmountIn:  big.NewInt(100),
		}
	}
	isNew := func(v bool) func() (bool, error) { return func() (bool, error) { return v, nil } }

	testCases := []struct {
		name         string
		processor    string
		token        *tokentypes.Token
		to           common.Address
		estimatedGas uint64
		recipientNew func() (bool, error)
		selfGas      uint64 // the transfer to the sender's own address; 0 = not estimated
		selfErr      error
		expected     bool
	}{
		// after Glamsterdam: EIP-8037 state gas for the new account / storage slot
		{"ETH to a new account costs more", pathprocessorcommon.ProcessorTransferName, eth, recipient, 204600, isNew(true), 12000, nil, true},
		{"ERC-20 to a first-time holder costs more", pathprocessorcommon.ProcessorTransferName, usdc, recipient, 154100, isNew(true), 37000, nil, true},
		// before the fork a new ETH account costs nothing extra: no hint
		{"ETH to a new account costing the same", pathprocessorcommon.ProcessorTransferName, eth, recipient, 21000, isNew(true), 21000, nil, false},
		{"existing recipient: no second estimate", pathprocessorcommon.ProcessorTransferName, eth, recipient, 21000, isNew(false), 0, nil, false},
		{"recipient state unknown", pathprocessorcommon.ProcessorTransferName, eth, recipient, 204600, func() (bool, error) { return false, errors.New("rpc down") }, 0, nil, false},
		{"estimate to self failed", pathprocessorcommon.ProcessorTransferName, usdc, recipient, 154100, isNew(true), 0, errors.New("reverted"), false},
		{"self-send", pathprocessorcommon.ProcessorTransferName, eth, sender, 12000, isNew(true), 0, nil, false},
		{"not a plain transfer", pathprocessorcommon.ProcessorERC721Name, eth, recipient, 204600, isNew(true), 0, nil, false},
		{"no estimate", pathprocessorcommon.ProcessorTransferName, eth, recipient, 0, isNew(true), 0, nil, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			processor := mockpathprocessor.NewMockPathProcessor(ctrl)
			processor.EXPECT().Name().Return(tc.processor).AnyTimes()
			if tc.selfGas != 0 || tc.selfErr != nil {
				processor.EXPECT().PackTxInputData(gomock.Any()).DoAndReturn(func(p pathprocessor.ProcessorInputParams) ([]byte, error) {
					require.Equal(t, sender, p.ToAddr, "estimated to the sender's own address")
					return []byte{0x01}, nil
				})
				processor.EXPECT().EstimateGas(gomock.Any(), []byte{0x01}).Return(tc.selfGas, tc.selfErr)
			}

			got := newRecipientRaisesGas(processor, inputParams(tc.token, tc.to), tc.estimatedGas, tc.recipientNew, logger)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestRecipientStateCache(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	key := recipientStateKey{chainID: 1, recipient: common.HexToAddress("0x2")}
	other := recipientStateKey{chainID: 10, recipient: common.HexToAddress("0x2")}

	var cache recipientStateCache
	_, ok := cache.get(key, now)
	require.False(t, ok, "the zero value is empty")

	cache.set(key, true, now)
	isNew, ok := cache.get(key, now.Add(recipientStateTTL-time.Second))
	require.True(t, ok)
	require.True(t, isNew)
	_, ok = cache.get(other, now)
	require.False(t, ok, "per chain, token and recipient")

	_, ok = cache.get(key, now.Add(recipientStateTTL))
	require.False(t, ok, "expired")

	// setting drops expired entries
	cache.set(other, false, now.Add(recipientStateTTL))
	require.Len(t, cache.entries, 1)
}

func TestCheckRecipientIsNew(t *testing.T) {
	ctrl := gomock.NewController(t)
	transfer := mockpathprocessor.NewMockPathProcessor(ctrl)
	transfer.EXPECT().Name().Return(pathprocessorcommon.ProcessorTransferName).AnyTimes()
	bridge := mockpathprocessor.NewMockPathProcessor(ctrl)
	bridge.EXPECT().Name().Return(pathprocessorcommon.ProcessorRelayName).AnyTimes()

	sender, recipient := common.HexToAddress("0x1"), common.HexToAddress("0x2")
	usdc := &tokentypes.Token{Token: &types.Token{Symbol: "USDC", Address: common.HexToAddress("0xa0b8"), ChainID: 1}}
	inputParams := pathprocessor.ProcessorInputParams{
		FromChain: &params.Network{ChainID: 1}, FromToken: usdc, FromAddr: sender, ToAddr: recipient,
	}

	// a cached answer: no balance fetch, no RPC (the router has no clients here)
	r := &Router{logger: logutils.ZapLogger().Named("router-test")}
	r.newRecipients.set(recipientStateKey{chainID: 1, token: usdc.Address, recipient: recipient}, true, time.Now())

	isNew, err := r.checkRecipientIsNew(context.Background(), transfer, inputParams)()
	require.NoError(t, err)
	require.True(t, isNew)

	isNew, err = r.checkRecipientIsNew(context.Background(), bridge, inputParams)()
	require.NoError(t, err)
	require.False(t, isNew, "not a plain transfer: not checked")
}

// fakeBatchCaller answers eth_getTransactionCount and eth_getCode.
type fakeBatchCaller struct {
	nonce   hexutil.Uint64
	code    hexutil.Bytes
	elemErr error
	calls   int
}

func (f *fakeBatchCaller) BatchCallContext(_ context.Context, batch []gethrpc.BatchElem) error {
	f.calls++
	for i := range batch {
		switch result := batch[i].Result.(type) {
		case *hexutil.Uint64:
			*result = f.nonce
		case *hexutil.Bytes:
			*result = f.code
		}
		batch[i].Error = f.elemErr
	}
	return nil
}

func TestFetchRecipientIsNew(t *testing.T) {
	recipient := common.HexToAddress("0x2")
	usdc := &tokentypes.Token{Token: &types.Token{Symbol: "USDC", Address: common.HexToAddress("0xa0b8"), ChainID: 1}}
	eth := &tokentypes.Token{Token: &types.Token{Symbol: "ETH", Address: common.Address{}, ChainID: 1}}

	for _, tc := range []struct {
		name        string
		token       *tokentypes.Token
		balance     int64
		balanceErr  error
		nonce       hexutil.Uint64
		code        hexutil.Bytes
		elemErr     error
		expected    bool
		wantErr     bool
		wantRPCCall bool
	}{
		{"funded recipient", eth, 1, nil, 0, nil, nil, false, false, false},
		{"ERC-20 first-time holder: no nonce/code lookup", usdc, 0, nil, 0, nil, nil, true, false, false},
		{"ERC-20 holder", usdc, 5, nil, 0, nil, nil, false, false, false},
		{"empty EOA", eth, 0, nil, 0, nil, nil, true, false, true},
		{"zero-balance EOA with nonce", eth, 0, nil, 3, nil, nil, false, false, true},
		{"zero-balance contract", eth, 0, nil, 0, hexutil.Bytes{0x60}, nil, false, false, true},
		{"batch element error", eth, 0, nil, 0, nil, errors.New("boom"), false, true, true},
		{"balance unknown", eth, 0, errors.New("rpc down"), 0, nil, nil, false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakeBatchCaller{nonce: tc.nonce, code: tc.code, elemErr: tc.elemErr}
			getBalance := func(_ context.Context, chainID uint64, token *tokentypes.Token, account common.Address) (*big.Int, error) {
				require.Equal(t, uint64(1), chainID)
				require.Equal(t, tc.token, token)
				require.Equal(t, recipient, account)
				return big.NewInt(tc.balance), tc.balanceErr
			}
			inputParams := pathprocessor.ProcessorInputParams{
				FromChain: &params.Network{ChainID: 1}, FromToken: tc.token, FromAddr: common.HexToAddress("0x1"), ToAddr: recipient,
			}

			isNew, err := fetchRecipientIsNew(context.Background(), inputParams, getBalance,
				func(uint64) (batchCaller, error) { return client, nil })

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.expected, isNew)
			require.Equal(t, tc.wantRPCCall, client.calls == 1)
		})
	}
}
