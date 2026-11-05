package router

import (
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/common"

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
