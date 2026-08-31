package router

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	mock_contracts "github.com/status-im/status-go/internal/contracts/mock"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/params"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/pkg/services/wallet/router/sendtype"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

// fakeERC20 satisfies ierc20.IERC20Iface for the allowance checks.
type fakeERC20 struct {
	allowance *big.Int
	err       error
}

func (f *fakeERC20) BalanceOf(*bind.CallOpts, common.Address) (*big.Int, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeERC20) Name(*bind.CallOpts) (string, error)    { return "", errors.New("not implemented") }
func (f *fakeERC20) Symbol(*bind.CallOpts) (string, error)  { return "", errors.New("not implemented") }
func (f *fakeERC20) Decimals(*bind.CallOpts) (uint8, error) { return 0, errors.New("not implemented") }
func (f *fakeERC20) Allowance(*bind.CallOpts, common.Address, common.Address) (*big.Int, error) {
	return f.allowance, f.err
}

func TestRequireApproval(t *testing.T) {
	mainnet := &params.Network{ChainID: walletCommon.EthereumMainnet}
	fromAddr := common.HexToAddress("0x0000000000000000000000000000000000000A01")
	spender := common.HexToAddress("0x0000000000000000000000000000000000000A02")
	amountIn := big.NewInt(1000)

	erc20Token := &tokentypes.Token{Token: &types.Token{
		ChainID: walletCommon.EthereumMainnet,
		Address: common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
		Symbol:  walletCommon.UsdcSymbolEVM,
	}}
	nativeToken := &tokentypes.Token{Token: &types.Token{
		ChainID: walletCommon.EthereumMainnet,
		Symbol:  walletCommon.EthSymbol,
	}}

	newRouterWithMaker := func(t *testing.T) (*Router, *mock_contracts.MockContractMakerIface) {
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		maker := mock_contracts.NewMockContractMakerIface(ctrl)
		return &Router{
			logger:        logutils.ZapLogger().Named("router-test"),
			contractMaker: maker,
		}, maker
	}

	params := pathprocessor.ProcessorInputParams{
		FromChain: mainnet,
		FromAddr:  fromAddr,
		FromToken: erc20Token,
		AmountIn:  amountIn,
	}

	t.Run("not required for collectibles, ens and stickers transfers", func(t *testing.T) {
		router, _ := newRouterWithMaker(t)

		for _, st := range []sendtype.SendType{sendtype.ERC721Transfer, sendtype.ENSRegister, sendtype.StickersBuy} {
			required, amount, err := router.requireApproval(context.Background(), st, &spender, params)
			require.NoError(t, err)
			assert.False(t, required)
			assert.Nil(t, amount)
		}
	})

	t.Run("not required for the native token", func(t *testing.T) {
		router, _ := newRouterWithMaker(t)

		nativeParams := params
		nativeParams.FromToken = nativeToken
		required, amount, err := router.requireApproval(context.Background(), sendtype.Transfer, &spender, nativeParams)
		require.NoError(t, err)
		assert.False(t, required)
		assert.Nil(t, amount)
	})

	t.Run("not required without an approval contract address", func(t *testing.T) {
		router, maker := newRouterWithMaker(t)
		maker.EXPECT().NewERC20(walletCommon.EthereumMainnet, erc20Token.Address).Return(&fakeERC20{}, nil).Times(2)

		required, _, err := router.requireApproval(context.Background(), sendtype.Transfer, nil, params)
		require.NoError(t, err)
		assert.False(t, required)

		zeroAddress := walletCommon.ZeroAddress()
		required, _, err = router.requireApproval(context.Background(), sendtype.Transfer, &zeroAddress, params)
		require.NoError(t, err)
		assert.False(t, required)
	})

	t.Run("not required when the allowance covers the amount", func(t *testing.T) {
		router, maker := newRouterWithMaker(t)
		maker.EXPECT().NewERC20(walletCommon.EthereumMainnet, erc20Token.Address).
			Return(&fakeERC20{allowance: big.NewInt(1000)}, nil)

		required, amount, err := router.requireApproval(context.Background(), sendtype.Transfer, &spender, params)
		require.NoError(t, err)
		assert.False(t, required)
		assert.Nil(t, amount)
	})

	t.Run("required when the allowance is insufficient", func(t *testing.T) {
		router, maker := newRouterWithMaker(t)
		maker.EXPECT().NewERC20(walletCommon.EthereumMainnet, erc20Token.Address).
			Return(&fakeERC20{allowance: big.NewInt(999)}, nil)

		required, amount, err := router.requireApproval(context.Background(), sendtype.Transfer, &spender, params)
		require.NoError(t, err)
		assert.True(t, required)
		assert.Equal(t, amountIn, amount)
	})

	t.Run("contract instantiation error is propagated", func(t *testing.T) {
		router, maker := newRouterWithMaker(t)
		maker.EXPECT().NewERC20(walletCommon.EthereumMainnet, erc20Token.Address).
			Return(nil, errors.New("no eth client"))

		_, _, err := router.requireApproval(context.Background(), sendtype.Transfer, &spender, params)
		assert.Error(t, err)
	})

	t.Run("allowance read error is propagated", func(t *testing.T) {
		router, maker := newRouterWithMaker(t)
		maker.EXPECT().NewERC20(walletCommon.EthereumMainnet, erc20Token.Address).
			Return(&fakeERC20{err: errors.New("call reverted")}, nil)

		_, _, err := router.requireApproval(context.Background(), sendtype.Transfer, &spender, params)
		assert.Error(t, err)
	})
}
