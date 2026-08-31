package pathprocessor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	mock_ethclient "github.com/status-im/status-go/internal/rpc/chain/ethclient/mock/client/ethclient"
	mock_rpcclient "github.com/status-im/status-go/internal/rpc/mock/client"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
)

// fakeEnsResolver satisfies ensResolverIface for tests.
type fakeEnsResolver struct {
	registrarAddress common.Address
	registrarErr     error
	resolverAddress  *common.Address
	resolverErr      error
}

func (f *fakeEnsResolver) GetRegistrarAddress(_ context.Context, _ uint64) (common.Address, error) {
	return f.registrarAddress, f.registrarErr
}

func (f *fakeEnsResolver) Resolver(_ context.Context, _ uint64, _ string) (*common.Address, error) {
	return f.resolverAddress, f.resolverErr
}

func TestENSReleaseProcessor_EstimateGas(t *testing.T) {
	registrarAddress := common.HexToAddress("0x0000000000000000000000000000000000000E15")
	params := ProcessorInputParams{
		FromChain: &mainnet,
		FromAddr:  testFromAddr,
		Username:  "myname.stateofus.eth",
	}

	t.Run("estimates against the registrar contract", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		mockEthClient := mock_ethclient.NewMockEthClientInterface(ctrl)
		processor := NewENSReleaseProcessor(mockRPCClient, nil, nil)
		processor.ensResolver = &fakeEnsResolver{registrarAddress: registrarAddress}

		input, err := processor.PackTxInputData(params)
		require.NoError(t, err)
		require.NotEmpty(t, input)

		mockRPCClient.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(mockEthClient, nil)
		mockEthClient.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, msg ethereum.CallMsg) (uint64, error) {
				assert.Equal(t, &registrarAddress, msg.To)
				assert.Equal(t, input, msg.Data)
				return uint64(60000), nil
			})

		estimation, err := processor.EstimateGas(params, input)
		require.NoError(t, err)
		assert.Equal(t, uint64(float64(60000)*pathProcessorCommon.IncreaseEstimatedGasFactor), estimation)
	})

	t.Run("resolver error is propagated", func(t *testing.T) {
		processor := NewENSReleaseProcessor(nil, nil, nil)
		processor.ensResolver = &fakeEnsResolver{registrarErr: errors.New("registrar not found")}

		_, err := processor.EstimateGas(params, []byte{})
		assert.Error(t, err)
	})

	t.Run("eth client getter error is propagated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		processor := NewENSReleaseProcessor(mockRPCClient, nil, nil)
		processor.ensResolver = &fakeEnsResolver{registrarAddress: registrarAddress}

		mockRPCClient.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(nil, errors.New("no client"))

		_, err := processor.EstimateGas(params, []byte{})
		assert.Error(t, err)
	})
}

func TestENSPublicKeyProcessor_EstimateGas(t *testing.T) {
	resolverAddress := common.HexToAddress("0x0000000000000000000000000000000000000E20")
	params := ProcessorInputParams{
		FromChain: &mainnet,
		FromAddr:  testFromAddr,
		Username:  "myname.eth",
		PublicKey: "0x04bb2024ce5d72e45d4a4f8589ae657ef9745855006996115a23a1af88d5708fbd652d0bc616376bf90a87f7d4e3ca73043f3746e017119a3f36cbf711b8c0dd25",
	}

	t.Run("estimates against the resolver contract", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		mockEthClient := mock_ethclient.NewMockEthClientInterface(ctrl)
		processor := NewENSPublicKeyProcessor(mockRPCClient, nil, nil)
		processor.ensResolver = &fakeEnsResolver{resolverAddress: &resolverAddress}

		input, err := processor.PackTxInputData(params)
		require.NoError(t, err)
		require.NotEmpty(t, input)

		mockRPCClient.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(mockEthClient, nil)
		mockEthClient.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, msg ethereum.CallMsg) (uint64, error) {
				assert.Equal(t, &resolverAddress, msg.To)
				return uint64(70000), nil
			})

		estimation, err := processor.EstimateGas(params, input)
		require.NoError(t, err)
		assert.Equal(t, uint64(float64(70000)*pathProcessorCommon.IncreaseEstimatedGasFactor), estimation)
	})

	t.Run("zero resolver address is an error", func(t *testing.T) {
		zeroAddress := walletCommon.ZeroAddress()
		processor := NewENSPublicKeyProcessor(nil, nil, nil)
		processor.ensResolver = &fakeEnsResolver{resolverAddress: &zeroAddress}

		_, err := processor.EstimateGas(params, []byte{})
		assert.Error(t, err)
	})

	t.Run("resolver error is propagated", func(t *testing.T) {
		processor := NewENSPublicKeyProcessor(nil, nil, nil)
		processor.ensResolver = &fakeEnsResolver{resolverErr: errors.New("resolver lookup failed")}

		_, err := processor.EstimateGas(params, []byte{})
		assert.Error(t, err)
	})
}

func TestENSRegisterProcessor_EstimateGas(t *testing.T) {
	params := ProcessorInputParams{
		FromChain: &mainnet,
		FromAddr:  testFromAddr,
		Username:  "myname.stateofus.eth",
	}

	t.Run("estimates against the SNT contract", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		mockEthClient := mock_ethclient.NewMockEthClientInterface(ctrl)
		processor := NewENSRegisterProcessor(mockRPCClient, nil, nil)

		// On mainnet the contract address resolves statically from the SNT tables, no resolver needed.
		mockRPCClient.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(mockEthClient, nil)
		mockEthClient.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(uint64(90000), nil)

		estimation, err := processor.EstimateGas(params, []byte{0x01})
		require.NoError(t, err)
		assert.Equal(t, uint64(float64(90000)*pathProcessorCommon.IncreaseEstimatedGasFactor), estimation)
	})

	t.Run("estimation error is propagated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		mockEthClient := mock_ethclient.NewMockEthClientInterface(ctrl)
		processor := NewENSRegisterProcessor(mockRPCClient, nil, nil)

		mockRPCClient.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(mockEthClient, nil)
		mockEthClient.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(uint64(0), errors.New("estimation failed"))

		_, err := processor.EstimateGas(params, []byte{0x01})
		assert.Error(t, err)
	})
}
