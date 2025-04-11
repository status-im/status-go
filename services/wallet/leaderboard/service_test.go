package leaderboard

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/stretchr/testify/require"
)

func setupMarketDatadService(t *testing.T, config ServiceConfig) *MarketDataService {
	config.Validate()
	return NewMarketDataService(config, &event.Feed{})
}

func TestServiceStartStop(t *testing.T) {
	config := ServiceConfig{}

	service := setupMarketDatadService(t, config)
	require.NotNil(t, service)

	service.Start(context.Background())
	service.Stop()
}

func TestUnsubscribeWhenNotSubscribed(t *testing.T) {
	config := ServiceConfig{}
	service := setupMarketDatadService(t, config)

	// Unsubscribe should not panic or error
	err := service.UnsubscribeFromLeaderboard()
	require.Error(t, err)
}

func TestSubsribe(t *testing.T) {
	config := ServiceConfig{}
	service := setupMarketDatadService(t, config)

	// Subscribe should not panic or error
	service.FetchLeaderboardPageAsync(0, 0, 0, "usd")

	time.Sleep(3 * time.Second) // Wait for the async operation to complete and events to be sent

	// TODO check for sent events

	service.UnsubscribeFromLeaderboard() // Unsubscribe after the test
}
