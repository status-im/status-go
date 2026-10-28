package transferdetector_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/status-im/go-wallet-sdk/pkg/eventfilter"
	"github.com/status-im/go-wallet-sdk/pkg/eventlog"
	"github.com/status-im/go-wallet-sdk/pkg/httptraffic"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/traffic"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/wallet/transferdetector"
	mock_transferdetector "github.com/status-im/status-go/pkg/services/wallet/transferdetector/mock"
)

func TestController_TagsItsRequestsAsTransferDetection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	recorder := httptraffic.NewRecorder()
	client := &http.Client{Transport: recorder.Instrument(nil)}
	request := func(ctx context.Context) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
	}

	ctrl := gomock.NewController(t)
	fetcher := mock_transferdetector.NewMockFilterProvider(ctrl)
	accountsProvider := mock_transferdetector.NewMockAccountsProvider(ctrl)
	networksProvider := mock_transferdetector.NewMockNetworksProvider(ctrl)
	lastBlockProvider := mock_transferdetector.NewMockLastBlockProvider(ctrl)

	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{{0x11}}, nil).AnyTimes()
	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{{ChainID: 1}}, nil).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(pubsub.NewPublisher()).AnyTimes()
	var block atomic.Uint64
	block.Store(100)
	lastBlockProvider.EXPECT().GetReadBlocks(uint64(1)).DoAndReturn(
		func(uint64) (uint64, uint64, bool) {
			return 100, block.Add(10), true
		}).AnyTimes()
	var filtered atomic.Bool
	fetcher.EXPECT().FilterTransfers(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, _ uint64, _ eventfilter.TransferQueryConfig) ([]eventlog.Event, error) {
			request(ctx)
			filtered.Store(true)
			return nil, nil
		}).AnyTimes()

	controller := transferdetector.NewController(
		transferdetector.ControllerConfig{FetchDebounceTime: 10 * time.Millisecond},
		fetcher, accountsProvider, pubsub.NewPublisher(), networksProvider, lastBlockProvider, zap.NewNop())
	controller.Start()
	require.Eventually(t, filtered.Load, time.Second, 10*time.Millisecond)
	controller.Stop()

	endpoints := recorder.Snapshot().Endpoints
	require.NotEmpty(t, endpoints)
	for _, endpoint := range endpoints {
		require.Equal(t, traffic.TransferDetection, endpoint.Source)
	}
}
