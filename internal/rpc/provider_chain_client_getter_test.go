package rpc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mock_ethclient "github.com/status-im/status-go/internal/rpc/chain/ethclient/mock/client/ethclient"
	mock_rpcclient "github.com/status-im/status-go/internal/rpc/mock/client"
)

func TestPreferredProviderChainClientGetter(t *testing.T) {
	const chainID = uint64(10)

	t.Run("first available provider wins", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		client := mock_rpcclient.NewMockClientInterface(ctrl)
		direct := mock_ethclient.NewMockEthClientInterface(ctrl)
		client.EXPECT().EthClientWithProvider(chainID, "direct").Return(direct, nil)

		ethClient, err := NewPreferredProviderChainClientGetter(client, "direct", "proxy").EthClient(chainID)
		require.NoError(t, err)
		require.Equal(t, direct, ethClient)
	})

	t.Run("falls back to the next provider", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		client := mock_rpcclient.NewMockClientInterface(ctrl)
		proxy := mock_ethclient.NewMockEthClientInterface(ctrl)
		client.EXPECT().EthClientWithProvider(chainID, "direct").Return(nil, nil)
		client.EXPECT().EthClientWithProvider(chainID, "proxy").Return(proxy, nil)

		ethClient, err := NewPreferredProviderChainClientGetter(client, "direct", "proxy").EthClient(chainID)
		require.NoError(t, err)
		require.Equal(t, proxy, ethClient)
	})

	t.Run("no provider gives a nil client", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		client := mock_rpcclient.NewMockClientInterface(ctrl)
		client.EXPECT().EthClientWithProvider(chainID, gomock.Any()).Return(nil, nil).Times(2)

		ethClient, err := NewPreferredProviderChainClientGetter(client, "direct", "proxy").EthClient(chainID)
		require.NoError(t, err)
		require.Nil(t, ethClient)
	})

	t.Run("unknown chain returns the error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		client := mock_rpcclient.NewMockClientInterface(ctrl)
		client.EXPECT().EthClientWithProvider(chainID, "direct").Return(nil, errors.New("no chain"))

		_, err := NewPreferredProviderChainClientGetter(client, "direct", "proxy").EthClient(chainID)
		require.Error(t, err)
	})
}
