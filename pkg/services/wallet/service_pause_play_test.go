package wallet

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/db/walletdb"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/networks"
	"github.com/status-im/status-go/pkg/services/wallet/collectibles"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/market"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
	"github.com/status-im/status-go/pkg/services/wallet/walletevent"
)

func TestServicePauseNoopWhenNotStarted(t *testing.T) {
	svc := &Service{}
	require.NoError(t, svc.Pause())
}

func TestServicePauseNoopWhenAlreadyPaused(t *testing.T) {
	svc := &Service{
		started: true,
		paused:  true,
	}
	require.NoError(t, svc.Pause())
}

func TestServiceResumeNoopWhenNotStarted(t *testing.T) {
	svc := &Service{}
	require.NoError(t, svc.Resume())
}

func TestServiceResumeNoopWhenNotPaused(t *testing.T) {
	svc := &Service{
		started: true,
		paused:  false,
	}
	require.NoError(t, svc.Resume())
}

var errProviderDown = errors.New("dial tcp: lookup provider.example: no such host")

type failingProvider struct {
	thirdparty.MarketDataProvider
	thirdparty.CollectibleAccountOwnershipProvider
}

func (failingProvider) ID() string                                 { return "failing" }
func (failingProvider) IsChainSupported(walletCommon.ChainID) bool { return true }
func (failingProvider) FetchPrices([]*tokentypes.Token, []string) (map[string]map[string]float64, error) {
	return nil, errProviderDown
}
func (failingProvider) FetchAllAssetsByOwnerAndContractAddress(context.Context, walletCommon.ChainID, common.Address, []common.Address, string, int) (*thirdparty.FullCollectibleDataContainer, error) {
	return nil, errProviderDown
}

type noMarketTokens struct{ market.TokenManagerInterface }

func (noMarketTokens) GetTokensByKeysForFetchingMarketData([]string) ([]*tokentypes.Token, error) {
	return nil, nil
}

func TestServicePauseCancelsPendingDowns(t *testing.T) {
	appDB, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	accountsDb, err := accounts.NewDB(appDB)
	require.NoError(t, err)
	db, err := testutils.SetupTestMemorySQLDB(walletdb.DbInitializer{})
	require.NoError(t, err)
	accountsPublisher := pubsub.NewPublisher()
	networkManager := networks.NewManager(appDB, nil)
	require.NoError(t, networkManager.InitEmbeddedNetworks(nil))
	rpcClient, err := rpc.NewClient(rpc.ClientConfig{NetworkManager: networkManager, AccountsPublisher: accountsPublisher})
	require.NoError(t, err)
	svc, err := NewService(db, accountsDb, appDB, rpcClient, accountsPublisher, nil, nil, &params.NodeConfig{}, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	feed := new(event.Feed)
	events := make(chan walletevent.Event, 10)
	sub := feed.Subscribe(events)
	defer sub.Unsubscribe()

	svc.marketManager = market.NewManager([]thirdparty.MarketDataProvider{failingProvider{}}, noMarketTokens{}, feed)
	svc.marketManager.SetDownDebounce(100 * time.Millisecond)
	svc.collectiblesManager = collectibles.NewManager(db, nil, nil, thirdparty.CollectibleProviders{
		AccountOwnershipProviders: []thirdparty.CollectibleAccountOwnershipProvider{failingProvider{}},
	}, nil, feed)
	svc.collectiblesManager.SetDownDebounce(100 * time.Millisecond)

	_, err = svc.marketManager.FetchPrices([]string{"eth"}, []string{"usd"})
	require.Error(t, err)
	_, err = svc.collectiblesManager.FetchAllAssetsByOwnerAndContractAddress(context.Background(), walletCommon.ChainID(1), common.Address{}, nil, "", 1, thirdparty.FetchFromAnyProvider)
	require.Error(t, err)

	svc.started = true
	require.NoError(t, svc.Pause())

	time.Sleep(500 * time.Millisecond)
	var got []walletevent.EventType
	for len(events) > 0 {
		got = append(got, (<-events).Type)
	}
	require.Empty(t, got, "pause must cancel pending downs")
}
