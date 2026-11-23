package market

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
	mock_thirdparty "github.com/status-im/status-go/pkg/services/wallet/thirdparty/mock"
)

// circuitProvider answers with its name, or fails with err, and counts its calls.
type circuitProvider struct {
	mock_thirdparty.MockMarketDataProvider
	id    string
	name  string
	err   error
	calls atomic.Int32
}

func (p *circuitProvider) ID() string {
	return p.id
}

func (p *circuitProvider) answer() (interface{}, error) {
	p.calls.Add(1)
	if p.err != nil {
		return nil, p.err
	}
	return p.name, nil
}

// The prod proxy, direct CoinGecko and the test proxy share an ID. When the
// prod proxy fails often enough to open its circuit, direct CoinGecko still
// answers: it is not shut out with the prod proxy and left to the test proxy.
func TestMakeCallOpenCircuitSkipsOnlyTheFailingProvider(t *testing.T) {
	// Circuits live in a process-wide registry: an ID of this test's own
	// keeps them apart from other tests.
	const sharedID = "coingecko-circuit-test"
	prodProxy := &circuitProvider{id: sharedID, name: "prod proxy", err: errors.New("prod proxy is down")}
	direct := &circuitProvider{id: sharedID, name: "direct"}
	testProxy := &circuitProvider{id: sharedID, name: "test proxy"}
	providers := []thirdparty.MarketDataProvider{prodProxy, direct, testProxy}
	manager := setupMarketManager(t, providers, &event.Feed{})
	defer manager.Stop()

	call := func() interface{} {
		result, err := manager.makeCall(manager.providers, func(provider thirdparty.MarketDataProvider) (interface{}, error) {
			return provider.(*circuitProvider).answer()
		})
		require.NoError(t, err)
		return result
	}

	// Call until the prod proxy's circuit opens and it is no longer asked.
	var servedBy interface{}
	require.Eventually(t, func() bool {
		before := prodProxy.calls.Load()
		servedBy = call()
		return prodProxy.calls.Load() == before
	}, 10*time.Second, 10*time.Millisecond, "the prod proxy's circuit never opened")

	require.Equal(t, "direct", servedBy)
	require.Zero(t, testProxy.calls.Load())
}
