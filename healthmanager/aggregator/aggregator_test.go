package aggregator

import (
	"sync"
	"testing"
	"time"

	"github.com/status-im/status-go/healthmanager/rpcstatus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// StatusAggregatorTestSuite defines the test suite for Aggregator.
type StatusAggregatorTestSuite struct {
	suite.Suite
	aggregator *Aggregator
}

// SetupTest runs before each test in the suite.
func (suite *StatusAggregatorTestSuite) SetupTest() {
	suite.aggregator = NewAggregator("TestAggregator")
}

// TestNewAggregator verifies that a new Aggregator is initialized correctly.
func (suite *StatusAggregatorTestSuite) TestNewAggregator() {
	assert.Equal(suite.T(), "TestAggregator", suite.aggregator.Name, "Aggregator name should be set correctly")
	assert.Empty(suite.T(), suite.aggregator.providerStatuses, "Aggregator should have no providers initially")
}

// TestRegisterProvider verifies that providers are registered correctly.
func (suite *StatusAggregatorTestSuite) TestRegisterProvider() {
	providerName := "Provider1"
	suite.aggregator.RegisterProvider(providerName)

	assert.Len(suite.T(), suite.aggregator.providerStatuses, 1, "Expected 1 provider after registration")
	_, exists := suite.aggregator.providerStatuses[providerName]
	assert.True(suite.T(), exists, "Provider1 should be registered")

	// Attempt to register the same provider again
	suite.aggregator.RegisterProvider(providerName)
	assert.Len(suite.T(), suite.aggregator.providerStatuses, 1, "Duplicate registration should not increase provider count")
}

// TestUpdate verifies that updating a provider's status works correctly.
func (suite *StatusAggregatorTestSuite) TestUpdate() {
	providerName := "Provider1"
	suite.aggregator.RegisterProvider(providerName)

	now := time.Now()

	// Update existing provider to up
	statusUp := rpcstatus.ProviderStatus{
		Name:          providerName,
		Status:        rpcstatus.StatusUp,
		LastSuccessAt: now,
	}
	suite.aggregator.Update(statusUp)

	ps, exists := suite.aggregator.providerStatuses[providerName]
	assert.True(suite.T(), exists, "Provider1 should exist after update")
	assert.Equal(suite.T(), rpcstatus.StatusUp, ps.Status, "Provider1 status should be 'up'")
	assert.Equal(suite.T(), now, ps.LastSuccessAt, "Provider1 LastSuccessAt should be updated")

	// Update existing provider to down
	nowDown := now.Add(1 * time.Hour)
	statusDown := rpcstatus.ProviderStatus{
		Name:        providerName,
		Status:      rpcstatus.StatusDown,
		LastErrorAt: nowDown,
	}
	suite.aggregator.Update(statusDown)

	ps, exists = suite.aggregator.providerStatuses[providerName]
	assert.True(suite.T(), exists, "Provider1 should exist after second update")
	assert.Equal(suite.T(), rpcstatus.StatusDown, ps.Status, "Provider1 status should be 'down'")
	assert.Equal(suite.T(), nowDown, ps.LastErrorAt, "Provider1 LastErrorAt should be updated")

	// Update a non-registered provider via Update (should add it)
	provider2 := "Provider2"
	statusUp2 := rpcstatus.ProviderStatus{
		Name:          provider2,
		Status:        rpcstatus.StatusUp,
		LastSuccessAt: now,
	}
	suite.aggregator.Update(statusUp2)

	assert.Len(suite.T(), suite.aggregator.providerStatuses, 2, "Expected 2 providers after updating a new provider")
	ps2, exists := suite.aggregator.providerStatuses[provider2]
	assert.True(suite.T(), exists, "Provider2 should be added via Update")
	assert.Equal(suite.T(), rpcstatus.StatusUp, ps2.Status, "Provider2 status should be 'up'")
}

// TestComputeAggregatedStatus_NoProviders verifies aggregated status when no providers are registered.
func (suite *StatusAggregatorTestSuite) TestComputeAggregatedStatus_NoProviders() {
	aggStatus := suite.aggregator.ComputeAggregatedStatus()

	assert.Equal(suite.T(), rpcstatus.StatusUnknown, aggStatus.Status, "Aggregated status should be 'unknown' when no providers are registered")
	assert.True(suite.T(), aggStatus.LastSuccessAt.IsZero(), "LastSuccessAt should be zero when no providers are registered")
	assert.True(suite.T(), aggStatus.LastErrorAt.IsZero(), "LastErrorAt should be zero when no providers are registered")
}

// TestComputeAggregatedStatus_AllUnknown verifies aggregated status when all providers are unknown.
func (suite *StatusAggregatorTestSuite) TestComputeAggregatedStatus_AllUnknown() {
	// Register multiple providers with unknown status
	suite.aggregator.RegisterProvider("Provider1")
	suite.aggregator.RegisterProvider("Provider2")
	suite.aggregator.RegisterProvider("Provider3")

	aggStatus := suite.aggregator.ComputeAggregatedStatus()

	assert.Equal(suite.T(), rpcstatus.StatusUnknown, aggStatus.Status, "Aggregated status should be 'unknown' when all providers are unknown")
	assert.True(suite.T(), aggStatus.LastSuccessAt.IsZero(), "LastSuccessAt should be zero when all providers are unknown")
	assert.True(suite.T(), aggStatus.LastErrorAt.IsZero(), "LastErrorAt should be zero when all providers are unknown")
}

// TestComputeAggregatedStatus_AllUp verifies aggregated status when all providers are up.
func (suite *StatusAggregatorTestSuite) TestComputeAggregatedStatus_AllUp() {
	// Register providers
	suite.aggregator.RegisterProvider("Provider1")
	suite.aggregator.RegisterProvider("Provider2")

	now1 := time.Now()
	now2 := now1.Add(1 * time.Hour)

	// Update all providers to up
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:          "Provider1",
		Status:        rpcstatus.StatusUp,
		LastSuccessAt: now1,
	})
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:          "Provider2",
		Status:        rpcstatus.StatusUp,
		LastSuccessAt: now2,
	})

	aggStatus := suite.aggregator.ComputeAggregatedStatus()

	assert.Equal(suite.T(), rpcstatus.StatusUp, aggStatus.Status, "Aggregated status should be 'up' when all providers are up")
	assert.Equal(suite.T(), now2, aggStatus.LastSuccessAt, "LastSuccessAt should reflect the latest success time")
	assert.True(suite.T(), aggStatus.LastErrorAt.IsZero(), "LastErrorAt should be zero when all providers are up")
}

// TestComputeAggregatedStatus_AllDown verifies aggregated status when all providers are down.
func (suite *StatusAggregatorTestSuite) TestComputeAggregatedStatus_AllDown() {
	// Register providers
	suite.aggregator.RegisterProvider("Provider1")
	suite.aggregator.RegisterProvider("Provider2")

	now1 := time.Now()
	now2 := now1.Add(1 * time.Hour)

	// Update all providers to down
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:        "Provider1",
		Status:      rpcstatus.StatusDown,
		LastErrorAt: now1,
	})
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:        "Provider2",
		Status:      rpcstatus.StatusDown,
		LastErrorAt: now2,
	})

	aggStatus := suite.aggregator.ComputeAggregatedStatus()

	assert.Equal(suite.T(), rpcstatus.StatusDown, aggStatus.Status, "Aggregated status should be 'down' when all providers are down")
	assert.Equal(suite.T(), now2, aggStatus.LastErrorAt, "LastErrorAt should reflect the latest error time")
	assert.True(suite.T(), aggStatus.LastSuccessAt.IsZero(), "LastSuccessAt should be zero when all providers are down")
}

// TestComputeAggregatedStatus_MixedUpAndUnknown verifies aggregated status with mixed up and unknown providers.
func (suite *StatusAggregatorTestSuite) TestComputeAggregatedStatus_MixedUpAndUnknown() {
	// Register providers
	suite.aggregator.RegisterProvider("Provider1") // up
	suite.aggregator.RegisterProvider("Provider2") // unknown
	suite.aggregator.RegisterProvider("Provider3") // up

	now1 := time.Now()
	now2 := now1.Add(30 * time.Minute)

	// Update some providers to up
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:          "Provider1",
		Status:        rpcstatus.StatusUp,
		LastSuccessAt: now1,
	})
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:          "Provider3",
		Status:        rpcstatus.StatusUp,
		LastSuccessAt: now2,
	})

	aggStatus := suite.aggregator.ComputeAggregatedStatus()

	assert.Equal(suite.T(), rpcstatus.StatusUp, aggStatus.Status, "Aggregated status should be 'up' when at least one provider is up")
	assert.Equal(suite.T(), now2, aggStatus.LastSuccessAt, "LastSuccessAt should reflect the latest success time")
	assert.True(suite.T(), aggStatus.LastErrorAt.IsZero(), "LastErrorAt should be zero when no providers are down")
}

// TestComputeAggregatedStatus_MixedUpAndDown verifies aggregated status with mixed up and down providers.
func (suite *StatusAggregatorTestSuite) TestComputeAggregatedStatus_MixedUpAndDown() {
	// Register providers
	suite.aggregator.RegisterProvider("Provider1") // up
	suite.aggregator.RegisterProvider("Provider2") // down
	suite.aggregator.RegisterProvider("Provider3") // up

	now1 := time.Now()
	now2 := now1.Add(15 * time.Minute)

	// Update providers
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:          "Provider1",
		Status:        rpcstatus.StatusUp,
		LastSuccessAt: now1,
	})
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:        "Provider2",
		Status:      rpcstatus.StatusDown,
		LastErrorAt: now2,
	})
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:          "Provider3",
		Status:        rpcstatus.StatusUp,
		LastSuccessAt: now1,
	})

	aggStatus := suite.aggregator.ComputeAggregatedStatus()

	assert.Equal(suite.T(), rpcstatus.StatusUp, aggStatus.Status, "Aggregated status should be 'up' when at least one provider is up")
	assert.Equal(suite.T(), now1, aggStatus.LastSuccessAt, "LastSuccessAt should reflect the latest success time")
	assert.Equal(suite.T(), now2, aggStatus.LastErrorAt, "LastErrorAt should reflect the latest error time")
}

// TestGetAggregatedStatus verifies that GetAggregatedStatus returns the correct aggregated status.
func (suite *StatusAggregatorTestSuite) TestGetAggregatedStatus() {
	// Register and update providers
	suite.aggregator.RegisterProvider("Provider1")
	suite.aggregator.RegisterProvider("Provider2")

	now := time.Now()

	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:          "Provider1",
		Status:        rpcstatus.StatusUp,
		LastSuccessAt: now,
	})
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:        "Provider2",
		Status:      rpcstatus.StatusDown,
		LastErrorAt: now.Add(1 * time.Hour),
	})

	aggStatus := suite.aggregator.GetAggregatedStatus()

	assert.Equal(suite.T(), rpcstatus.StatusUp, aggStatus.Status, "Aggregated status should be 'up' when at least one provider is up")
	assert.Equal(suite.T(), now, aggStatus.LastSuccessAt, "LastSuccessAt should reflect the provider's success time")
	assert.Equal(suite.T(), now.Add(1*time.Hour), aggStatus.LastErrorAt, "LastErrorAt should reflect the provider's error time")
}

// TestConcurrentAccess verifies that the Aggregator is safe for concurrent use.
func (suite *StatusAggregatorTestSuite) TestConcurrentAccess() {
	// Register multiple providers
	providers := []string{"Provider1", "Provider2", "Provider3", "Provider4", "Provider5"}
	for _, p := range providers {
		suite.aggregator.RegisterProvider(p)
	}

	var wg sync.WaitGroup

	// Concurrently update providers
	for _, p := range providers {
		wg.Add(1)
		go func(providerName string) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				suite.aggregator.Update(rpcstatus.ProviderStatus{
					Name:          providerName,
					Status:        rpcstatus.StatusUp,
					LastSuccessAt: time.Now(),
				})
				suite.aggregator.Update(rpcstatus.ProviderStatus{
					Name:        providerName,
					Status:      rpcstatus.StatusDown,
					LastErrorAt: time.Now(),
				})
			}
		}(p)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Set all providers to down to ensure deterministic aggregated status
	now := time.Now()
	for _, p := range providers {
		suite.aggregator.Update(rpcstatus.ProviderStatus{
			Name:        p,
			Status:      rpcstatus.StatusDown,
			LastErrorAt: now,
		})
	}

	aggStatus := suite.aggregator.GetAggregatedStatus()
	assert.Equal(suite.T(), rpcstatus.StatusDown, aggStatus.Status, "Aggregated status should be 'down' after setting all providers to down")
}

// TestStatusAggregatorTestSuite runs the test suite.
func TestStatusAggregatorTestSuite(t *testing.T) {
	suite.Run(t, new(StatusAggregatorTestSuite))
}
