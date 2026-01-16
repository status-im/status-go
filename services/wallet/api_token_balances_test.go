package wallet_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/services/wallet/mock_tokenbalances"
	"github.com/status-im/status-go/services/wallet/multistandardbalance"
	"github.com/status-im/status-go/services/wallet/tokenbalances"
)

// TestTokenBalancesServiceCreation verifies the tokenbalances service can be created
func TestTokenBalancesServiceCreation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_tokenbalances.NewMockStorage(ctrl)
	publisher := pubsub.NewPublisher()
	controller := &multistandardbalance.Controller{}
	tokenBalancesService := tokenbalances.NewService(mockStorage, publisher, controller)

	// Verify service was created successfully
	assert.NotNil(t, tokenBalancesService)
	assert.NotNil(t, tokenBalancesService.GetPublisher())
}

// Note: Full integration tests for GetAllBalances require a complete
// wallet service setup with token manager, balance storage, etc. These are better
// suited for integration test suites rather than unit tests.
